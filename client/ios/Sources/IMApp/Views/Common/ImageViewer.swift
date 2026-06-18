import SwiftUI

struct ImageViewer: View {
    let images: [URL]
    let initialIndex: Int

    @Environment(\.dismiss) private var dismiss
    @State private var currentIndex: Int
    @State private var scale: CGFloat = 1
    @State private var lastScale: CGFloat = 1
    @State private var offset: CGSize = .zero
    @State private var lastOffset: CGSize = .zero

    init(images: [URL], initialIndex: Int = 0) {
        self.images = images
        self.initialIndex = initialIndex
        _currentIndex = State(initialValue: initialIndex)
    }

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()

            if !images.isEmpty {
                TabView(selection: $currentIndex) {
                    ForEach(Array(images.enumerated()), id: \.offset) { _, url in
                        zoomableImage(url: url)
                    }
                }
                .tabViewStyle(.page(indexDisplayMode: .always))
            }
        }
        .overlay(alignment: .topTrailing) {
            HStack(spacing: 20) {
                if !images.isEmpty {
                    Button {
                        shareImage(at: currentIndex)
                    } label: {
                        Image(systemName: "square.and.arrow.up")
                            .font(.title3)
                            .foregroundColor(.white)
                            .padding(10)
                            .background(.ultraThinMaterial)
                            .clipShape(Circle())
                    }

                    Button {
                        saveImage(at: currentIndex)
                    } label: {
                        Image(systemName: "square.and.arrow.down")
                            .font(.title3)
                            .foregroundColor(.white)
                            .padding(10)
                            .background(.ultraThinMaterial)
                            .clipShape(Circle())
                    }
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
        .statusBarHidden()
    }

    @ViewBuilder
    private func zoomableImage(url: URL) -> some View {
        CachedAsyncImage(url: url) { image in
            image
                .resizable()
                .aspectRatio(contentMode: .fit)
                .scaleEffect(scale)
                .offset(offset)
                .gesture(
                    MagnificationGesture()
                        .onChanged { value in
                            let delta = value / lastScale
                            lastScale = value
                            scale = min(max(scale * delta, 0.5), 5)
                        }
                        .onEnded { _ in
                            lastScale = 1
                            if scale < 1 {
                                withAnimation { scale = 1; offset = .zero }
                            }
                        }
                )
                .gesture(
                    DragGesture()
                        .onChanged { value in
                            offset = CGSize(
                                width: lastOffset.width + value.translation.width,
                                height: lastOffset.height + value.translation.height
                            )
                        }
                        .onEnded { _ in
                            lastOffset = offset
                            if scale <= 1 {
                                withAnimation { offset = .zero; lastOffset = .zero }
                            }
                        }
                )
                .onTapGesture(count: 2) {
                    withAnimation {
                        if scale > 1 {
                            scale = 1
                            offset = .zero
                            lastOffset = .zero
                        } else {
                            scale = 2
                        }
                    }
                }
        } placeholder: {
            ProgressView()
                .tint(.white)
        }
    }

    private func shareImage(at index: Int) {
        guard index < images.count else { return }
        let url = images[index]
        let av = UIActivityViewController(activityItems: [url], applicationActivities: nil)
        if let scene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
           let root = scene.windows.first?.rootViewController {
            root.present(av, animated: true)
        }
    }

    private func saveImage(at index: Int) {
        guard index < images.count else { return }
        Task {
            do {
                let (data, _) = try await URLSession.shared.data(from: images[index])
                if let image = UIImage(data: data) {
                    UIImageWriteToSavedPhotosAlbum(image, nil, nil, nil)
                }
            } catch {}
        }
    }
}
