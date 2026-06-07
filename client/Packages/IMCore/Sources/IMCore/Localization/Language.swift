public enum Language: String, CaseIterable, Codable, Sendable {
    case zhHans = "zh-Hans"
    case en = "en"

    public var displayName: String {
        switch self {
        case .zhHans: return "中文"
        case .en: return "English"
        }
    }
}
