import Foundation

public enum Language: String, CaseIterable, Codable, Sendable {
    case system = "system"
    case zhHans = "zh-Hans"
    case en = "en"

    public var displayName: String {
        switch self {
        case .system: return loc("settings.language.follow_system")
        case .zhHans: return "中文"
        case .en: return "English"
        }
    }

    public var effectiveLanguage: Language {
        guard self == .system else { return self }
        let preferred = Locale.preferredLanguages.first ?? "zh-Hans"
        if preferred.hasPrefix("en") {
            return .en
        }
        return .zhHans
    }
}
