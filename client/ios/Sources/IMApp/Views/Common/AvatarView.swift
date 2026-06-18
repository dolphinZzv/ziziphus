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
            CachedAsyncImage(url: imageURL) { image in
                image
                    .resizable()
                    .scaledToFill()
                    .frame(width: size, height: size)
                    .clipShape(Circle())
            } placeholder: {
                ProgressView()
                    .frame(width: size, height: size)
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

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)
        let r, g, b: UInt64
        switch hex.count {
        case 6:
            (r, g, b) = ((int >> 16) & 0xFF, (int >> 8) & 0xFF, int & 0xFF)
        default:
            (r, g, b) = (0, 0x7A, 0xFF)
        }
        self.init(
            .sRGB,
            red: Double(r) / 255,
            green: Double(g) / 255,
            blue: Double(b) / 255,
            opacity: 1
        )
    }
}
