import SwiftUI
import IMCore

// MARK: - Apple Design Tokens (from DESIGN.md)

enum AppleDesign {
    // Colors
    enum Colors {
        static var actionBlue: Color { Color(hex: AuthManager.shared.currentUser?.primaryColor ?? "#0066cc") }
        static var focusBlue: Color { Color(hex: AuthManager.shared.currentUser?.primaryColor ?? "#0071e3") }
        static let skyBlue = Color(hex: "#2997ff")
        static let parchment = Color.adaptive(light: "#f5f5f7", dark: "#1c1c1e")
        static let ink = Color.adaptive(light: "#1d1d1f", dark: "#f5f5f7")
        static let inkMuted = Color.adaptive(light: "#7a7a7a", dark: "#98989d")
        static let hairline = Color.adaptive(light: "#e0e0e0", dark: "#38383a")
        static let chatGray = Color.adaptive(light: "#e5e5ea", dark: "#2c2c2e")
        static let chatBubbleOther = Color.adaptive(light: "#d5e3f8", dark: "#2c2c2e")
        static let dividerSoft = Color.adaptive(light: "#f0f0f0", dark: "#2c2c2e")
        static let pearl = Color.adaptive(light: "#fafafc", dark: "#1c1c1e")
        static let surfaceBlack = Color.black
        static let nearBlackTile = Color(hex: "#272729")
    }

    // Typography
    enum Typography {
        static let bodySize: CGFloat = 17
        static let captionSize: CGFloat = 14
        static let finePrintSize: CGFloat = 12
        static let navLinkSize: CGFloat = 12
    }

    // Spacing (base unit 8px)
    enum Spacing {
        static let xs: CGFloat = 8
        static let sm: CGFloat = 12
        static let md: CGFloat = 17
        static let lg: CGFloat = 24
        static let xl: CGFloat = 32
        static let xxl: CGFloat = 48
        static let section: CGFloat = 80
    }

    // Border Radius
    enum Radius {
        static let sm: CGFloat = 8
        static let md: CGFloat = 11
        static let lg: CGFloat = 18
        // .clipShape(Capsule()) for pill
    }
}

// MARK: - Standard Fonts

extension Font {
    /// 40px / 600 — login/register headline
    static let appleDisplay = Font.system(size: 40, weight: .semibold)

    /// 20px / 600 — profile user name
    static let appleTitle = Font.system(size: 20, weight: .semibold)

    /// 17px / 600 — conversation name, headers
    static let appleBodySemibold = Font.system(size: AppleDesign.Typography.bodySize, weight: .semibold)

    /// 17px / 400 — default body, buttons, input text
    static let appleBody = Font.system(size: AppleDesign.Typography.bodySize, weight: .regular)

    /// 14px / 400 — captions, secondary labels
    static let appleCaption = Font.system(size: AppleDesign.Typography.captionSize, weight: .regular)

    /// 14px / 600 — emphasized captions
    static let appleCaptionSemibold = Font.system(size: AppleDesign.Typography.captionSize, weight: .semibold)

    /// 12px / 400 — timestamps, fine print
    static let appleFinePrint = Font.system(size: AppleDesign.Typography.finePrintSize, weight: .regular)

    /// 12px / 600 — badge counts
    static let appleFinePrintSemibold = Font.system(size: AppleDesign.Typography.finePrintSize, weight: .semibold)
}

// MARK: - Button Styles

struct ApplePrimaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: AppleDesign.Typography.bodySize, weight: .regular))
            .foregroundColor(.white)
            .padding(.horizontal, 22)
            .frame(height: 44)
            .background(AppleDesign.Colors.actionBlue)
            .clipShape(Capsule())
            .scaleEffect(configuration.isPressed ? 0.95 : 1)
            .animation(.easeOut(duration: 0.1), value: configuration.isPressed)
    }
}

struct AppleSecondaryPillStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: AppleDesign.Typography.captionSize, weight: .regular))
            .foregroundColor(AppleDesign.Colors.actionBlue)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .overlay(
                Capsule()
                    .stroke(AppleDesign.Colors.actionBlue, lineWidth: 1)
            )
            .clipShape(Capsule())
            .scaleEffect(configuration.isPressed ? 0.95 : 1)
            .animation(.easeOut(duration: 0.1), value: configuration.isPressed)
    }
}

// MARK: - Hex Parsing

private func hexToRGBA(_ hex: String) -> (r: Double, g: Double, b: Double) {
    let h = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
    var int: UInt64 = 0
    Scanner(string: h).scanHexInt64(&int)
    switch h.count {
    case 6:
        return (
            Double((int >> 16) & 0xFF) / 255,
            Double((int >> 8) & 0xFF) / 255,
            Double(int & 0xFF) / 255
        )
    default:
        return (0, 0, 0)
    }
}

// MARK: - Adaptive Color (light/dark)

extension Color {
    /// Returns a color that switches between light/dark variants automatically.
    static func adaptive(light: String, dark: String) -> Color {
        #if os(macOS)
        Color(nsColor: NSColor(name: nil) { appearance in
            let isDark = appearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
            let (r, g, b) = isDark ? hexToRGBA(dark) : hexToRGBA(light)
            return NSColor(srgbRed: r, green: g, blue: b, alpha: 1)
        })
        #else
        Color(UIColor { trait in
            let isDark = trait.userInterfaceStyle == .dark
            let (r, g, b) = isDark ? hexToRGBA(dark) : hexToRGBA(light)
            return UIColor(displayP3Red: r, green: g, blue: b, alpha: 1)
        })
        #endif
    }
}

// MARK: - Color Hex Extension

extension Color {
    init(hex: String) {
        let (r, g, b) = hexToRGBA(hex)
        self.init(.displayP3, red: r, green: g, blue: b)
    }
}
