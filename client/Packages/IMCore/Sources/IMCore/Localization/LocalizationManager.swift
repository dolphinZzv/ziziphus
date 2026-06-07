import Foundation

@MainActor
public class LocalizationManager: ObservableObject {
    public static let shared = LocalizationManager()

    @Published public var currentLanguage: Language {
        didSet {
            UserDefaults.standard.set(currentLanguage.rawValue, forKey: "app_language")
        }
    }

    private init() {
        let saved = UserDefaults.standard.string(forKey: "app_language")
        if let saved, let lang = Language(rawValue: saved) {
            currentLanguage = lang
        } else {
            let preferred = Locale.preferredLanguages.first ?? "zh-Hans"
            if preferred.hasPrefix("en") {
                currentLanguage = .en
            } else {
                currentLanguage = .zhHans
            }
        }
    }

    public func localized(_ key: String, _ args: CVarArg...) -> String {
        Self.lookup(key, args)
    }

    /// Non-isolated static lookup for use from global `loc()` function.
    nonisolated public static func lookup(_ key: String, _ args: CVarArg...) -> String {
        let rawLang = UserDefaults.standard.string(forKey: "app_language")
        let lang = Language(rawValue: rawLang ?? "") ?? .zhHans
        let entry = StringCatalog[key]?[lang]
            ?? StringCatalog[key]?[.zhHans]
            ?? key
        if args.isEmpty {
            return entry
        }
        return String(format: entry, arguments: args)
    }

    public func setLanguage(_ language: Language) {
        currentLanguage = language
    }
}
