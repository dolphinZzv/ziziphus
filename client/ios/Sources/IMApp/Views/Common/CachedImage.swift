import SwiftUI
import UIKit

final class ImageCache: @unchecked Sendable {
    static let shared = ImageCache()
    private let cache = NSCache<NSURL, UIImage>()

    private init() {
        cache.countLimit = 200
        cache.totalCostLimit = 50 * 1024 * 1024 // 50 MB
    }

    func get(_ url: URL) -> UIImage? {
        cache.object(forKey: url as NSURL)
    }

    func set(_ image: UIImage, for url: URL) {
        let cost = Int(image.size.width * image.size.height * 4)
        cache.setObject(image, forKey: url as NSURL, cost: cost)
    }
}

struct CachedAsyncImage<Content: View, Placeholder: View>: View {
    let url: URL?
    @ViewBuilder let content: (Image) -> Content
    @ViewBuilder let placeholder: () -> Placeholder

    @State private var loadedImage: UIImage?

    var body: some View {
        if let image = loadedImage {
            content(Image(uiImage: image))
        } else {
            placeholder()
                .task {
                    await loadImage()
                }
        }
    }

    private func loadImage() async {
        guard let url else { return }
        if let cached = ImageCache.shared.get(url) {
            loadedImage = cached
            return
        }
        do {
            var request = URLRequest(url: url)
            // Only attach auth token for requests to our own server
            if let serverURL = URL(string: AppSettings.shared.serverURL),
               url.host == serverURL.host {
                if let token = AuthManager.shared.readToken() {
                    request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
                }
            }
            let (data, _) = try await URLSession.shared.data(for: request)
            if let image = UIImage(data: data) {
                ImageCache.shared.set(image, for: url)
                loadedImage = image
            }
        } catch {}
    }
}
