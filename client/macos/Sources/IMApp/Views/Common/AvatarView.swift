import AppKit
import CryptoKit
import SwiftUI
import IMCore

// MARK: - Image Cache

/// Memory + disk image cache with automatic downsizing.
@MainActor
final class ImageCache {
    static let shared = ImageCache()

    private let memoryCache = NSCache<NSURL, NSImage>()
    private let maxDimension: CGFloat = 256
    private let fileManager = FileManager.default

    private var diskCacheDir: URL {
        fileManager.urls(for: .cachesDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("ImageCache", isDirectory: true)
    }

    private init() {
        memoryCache.countLimit = 200
        memoryCache.totalCostLimit = 20 * 1024 * 1024
        try? fileManager.createDirectory(at: diskCacheDir, withIntermediateDirectories: true)
    }

    func load(_ url: URL) async -> NSImage? {
        // 1. Memory
        if let cached = memoryCache.object(forKey: url as NSURL) {
            return cached
        }

        let diskURL = diskCacheURL(for: url)

        // 2. Disk
        if let diskImage = NSImage(contentsOf: diskURL) {
            memoryCache.setObject(diskImage, forKey: url as NSURL)
            return diskImage
        }

        // 3. Network
        guard let (data, _) = try? await URLSession.shared.data(from: url),
              let source = NSImage(data: data) ?? NSBitmapImageRep(data: data).flatMap({ rep in
                  let img = NSImage(size: rep.size)
                  img.addRepresentation(rep)
                  return img
              })
        else { return nil }

        // 4. Resize (max 256pt on longest side)
        let final = resize(source, maxDimension: maxDimension) ?? source

        // 5. Store in memory
        memoryCache.setObject(final, forKey: url as NSURL)

        // 6. Store to disk as PNG
        if let tiff = final.tiffRepresentation,
           let rep = NSBitmapImageRep(data: tiff),
           let png = rep.representation(using: .png, properties: [:]) {
            try? png.write(to: diskURL, options: .atomic)
        }

        return final
    }

    func clearDiskCache() {
        try? fileManager.removeItem(at: diskCacheDir)
        try? fileManager.createDirectory(at: diskCacheDir, withIntermediateDirectories: true)
    }

    // MARK: Private

    private func diskCacheURL(for url: URL) -> URL {
        let hash = SHA256.hash(data: Data(url.absoluteString.utf8))
        let hex = hash.compactMap { String(format: "%02x", $0) }.joined()
        return diskCacheDir.appendingPathComponent(hex, isDirectory: false)
    }

    private func resize(_ image: NSImage, maxDimension: CGFloat) -> NSImage? {
        let size = image.size
        guard size.width > 0, size.height > 0 else { return nil }
        let longest = max(size.width, size.height)
        guard longest > maxDimension else { return nil }

        let scale = maxDimension / longest
        let newSize = NSSize(width: size.width * scale, height: size.height * scale)

        let resized = NSImage(size: newSize)
        resized.lockFocus()
        NSGraphicsContext.current?.imageInterpolation = .high
        image.draw(in: NSRect(origin: .zero, size: newSize),
                   from: NSRect(origin: .zero, size: size),
                   operation: .copy, fraction: 1)
        resized.unlockFocus()
        return resized
    }
}

// MARK: - Avatar View

struct AvatarView: View {
    let name: String
    let url: String
    var size: CGFloat = 40
    var primaryColor: String = ""
    var secondaryColor: String = ""

    @State private var loadedImage: NSImage?
    @State private var loadFailed = false

    private var resolvedURL: URL? {
        if url.isEmpty { return nil }
        if url.hasPrefix("http://") || url.hasPrefix("https://") {
            return URL(string: url)
        }
        return URL(string: url, relativeTo: URL(string: APIClient.shared.baseURL))
    }

    var body: some View {
        Group {
            if let imageURL = resolvedURL, let nsImage = loadedImage {
                Image(nsImage: nsImage)
                    .resizable()
                    .scaledToFill()
                    .frame(width: size, height: size)
                    .clipShape(Circle())
            } else if resolvedURL != nil && !loadFailed {
                ProgressView()
                    .frame(width: size, height: size)
            } else {
                letterFallback
            }
        }
        .frame(width: size, height: size)
        .task(id: resolvedURL) {
            loadFailed = false
            loadedImage = nil
            guard let url = resolvedURL else { return }
            if let cached = await ImageCache.shared.load(url) {
                loadedImage = cached
            } else {
                loadFailed = true
            }
        }
    }

    @ViewBuilder
    private var letterFallback: some View {
        let hasGradient = !primaryColor.isEmpty || !secondaryColor.isEmpty

        Circle()
            .fill(hasGradient ? .clear : Color.blue.opacity(0.2))
            .applyIf(hasGradient) { view in
                view.overlay(
                    LinearGradient(
                        colors: [
                            Color(hex: primaryColor.isEmpty ? "#007AFF" : primaryColor),
                            Color(hex: secondaryColor.isEmpty ? "#007AFF" : secondaryColor),
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                    .clipShape(Circle())
                )
            }
            .frame(width: size, height: size)
            .overlay {
                Text(String(name.prefix(1)).uppercased())
                    .font(.system(size: size * 0.4))
                    .fontWeight(.semibold)
                    .foregroundColor(.white)
            }
    }
}

extension View {
    @ViewBuilder
    func applyIf<Content: View>(_ condition: Bool, transform: (Self) -> Content) -> some View {
        if condition {
            transform(self)
        } else {
            self
        }
    }
}
