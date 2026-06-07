import SwiftUI

public enum AppTheme: String, CaseIterable, Codable {
    case system
    case light
    case dark

    public var displayName: String {
        switch self {
        case .system: return loc("settings.theme_system")
        case .light: return loc("settings.theme_light")
        case .dark: return loc("settings.theme_dark")
        }
    }
}

@MainActor
public class ThemeManager: ObservableObject {
    public static let shared = ThemeManager()

    @Published public var currentTheme: AppTheme {
        didSet {
            UserDefaults.standard.set(currentTheme.rawValue, forKey: "app_theme")
        }
    }

    public var resolvedColorScheme: ColorScheme? {
        switch currentTheme {
        case .system: return nil
        case .light: return .light
        case .dark: return .dark
        }
    }

    private init() {
        let saved = UserDefaults.standard.string(forKey: "app_theme")
        currentTheme = AppTheme(rawValue: saved ?? "") ?? .system
    }

    public func setTheme(_ theme: AppTheme) {
        currentTheme = theme
    }
}
