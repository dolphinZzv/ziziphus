import SwiftUI
import IMCore

struct ImageViewer: View {
    let images: [URL]
    let initialIndex: Int

    @Environment(\.dismiss) private var dismiss
    @State private var currentIndex: Int
    @State private var scale: CGFloat = 1
    @State private var offset: CGSize = .zero

    init(images: [URL], initialIndex: Int = 0) {
        self.images = images
        self.initialIndex = initialIndex
        _currentIndex = State(initialValue: initialIndex)
    }

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()

            if !images.isEmpty {
                zoomableImage(url: images[currentIndex])
            }
        }
        .overlay(alignment: .topTrailing) {
            HStack(spacing: 16) {
                if images.count > 1 {
                    Button {
                        currentIndex = max(0, currentIndex - 1)
                        resetZoom()
                    } label: {
                        Image(systemName: "chevron.left")
                            .font(.title3)
                            .foregroundColor(.white)
                            .padding(10)
                            .background(.ultraThinMaterial)
                            .clipShape(Circle())
                    }
                    .keyboardShortcut(.leftArrow, modifiers: [])

                    Button {
                        currentIndex = min(images.count - 1, currentIndex + 1)
                        resetZoom()
                    } label: {
                        Image(systemName: "chevron.right")
                            .font(.title3)
                            .foregroundColor(.white)
                            .padding(10)
                            .background(.ultraThinMaterial)
                            .clipShape(Circle())
                    }
                    .keyboardShortcut(.rightArrow, modifiers: [])
                }

                Text("\(currentIndex + 1) / \(images.count)")
                    .font(.caption)
                    .foregroundColor(.white)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(.ultraThinMaterial)
                    .clipShape(Capsule())

                Button {
                    saveImage()
                } label: {
                    Image(systemName: "square.and.arrow.down")
                        .font(.title3)
                        .foregroundColor(.white)
                        .padding(10)
                        .background(.ultraThinMaterial)
                        .clipShape(Circle())
                }

                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                        .font(.title3)
                        .foregroundColor(.white)
                        .padding(10)
                        .background(.ultraThinMaterial)
                        .clipShape(Circle())
                }
            }
            .padding()
        }
    }

    @ViewBuilder
    private func zoomableImage(url: URL) -> some View {
        MacImageViewerImage(url: url, scale: $scale, offset: $offset, onDismiss: { dismiss() })
    }

    private func resetZoom() {
        withAnimation {
            scale = 1
            offset = .zero
        }
    }

    private func saveImage() {
        guard currentIndex < images.count else { return }
        Task {
            do {
                let (data, _) = try await URLSession.shared.data(from: images[currentIndex])
                if let image = NSImage(data: data) {
                    let downloads = FileManager.default.urls(for: .downloadsDirectory, in: .userDomainMask).first!
                    let filename = images[currentIndex].lastPathComponent
                    let dest = downloads.appendingPathComponent(filename)
                    try? FileManager.default.removeItem(at: dest)
                    if let tiff = image.tiffRepresentation,
                       let rep = NSBitmapImageRep(data: tiff),
                       let png = rep.representation(using: .png, properties: [:]) {
                        try png.write(to: dest)
                        NSWorkspace.shared.activateFileViewerSelecting([dest])
                    }
                }
            } catch {}
        }
    }
}

// MARK: - Image Loader View

private struct MacImageViewerImage: View {
    let url: URL
    @Binding var scale: CGFloat
    @Binding var offset: CGSize
    let onDismiss: () -> Void

    @State private var loadedImage: NSImage?
    @State private var loadFailed = false

    var body: some View {
        Group {
            if let image = loadedImage {
                Image(nsImage: image)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .scaleEffect(scale)
                    .offset(offset)
                    .gesture(
                        MagnificationGesture()
                            .onChanged { value in
                                scale = min(max(value, 0.5), 5)
                            }
                            .onEnded { _ in
                                if scale < 1 {
                                    withAnimation { scale = 1; offset = .zero }
                                }
                            }
                    )
                    .gesture(
                        DragGesture()
                            .onChanged { value in
                                offset = value.translation
                            }
                            .onEnded { _ in
                                if scale <= 1 {
                                    withAnimation { offset = .zero }
                                }
                            }
                    )
                    .onTapGesture(count: 2) {
                        withAnimation {
                            if scale > 1 {
                                scale = 1
                                offset = .zero
                            } else {
                                scale = 2
                            }
                        }
                    }
                    .onTapGesture(count: 1) {
                        onDismiss()
                    }
            } else if loadFailed {
                VStack(spacing: 8) {
                    Image(systemName: "photo.badge.exclamationmark")
                        .font(.largeTitle)
                    Text(loc("common.load_failed"))
                }
                .foregroundColor(.white)
            } else {
                ProgressView()
                    .scaleEffect(1.5)
            }
        }
        .task { await load() }
    }

    private func load() async {
        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            if let image = NSImage(data: data) {
                await MainActor.run { loadedImage = image }
            } else {
                await MainActor.run { loadFailed = true }
            }
        } catch {
            await MainActor.run { loadFailed = true }
        }
    }
}
