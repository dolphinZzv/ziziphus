import SwiftUI
import IMCore

struct AvatarView: View {
    let name: String
    let url: String
    var size: CGFloat = 40
    var primaryColor: String = ""
    var secondaryColor: String = ""

    private var resolvedURL: URL? {
        if url.isEmpty { return nil }
        if url.hasPrefix("http://") || url.hasPrefix("https://") {
            return URL(string: url)
        }
        return URL(string: url, relativeTo: URL(string: APIClient.shared.baseURL))
    }

    var body: some View {
        if let imageURL = resolvedURL {
            AsyncImage(url: imageURL) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .scaledToFill()
                        .frame(width: size, height: size)
                        .clipShape(Circle())
                case .failure:
                    letterFallback
                case .empty:
                    ProgressView()
                        .frame(width: size, height: size)
                @unknown default:
                    letterFallback
                }
            }
            .frame(width: size, height: size)
        } else {
            letterFallback
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
