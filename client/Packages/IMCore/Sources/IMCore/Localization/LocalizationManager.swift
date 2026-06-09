import Foundation

@MainActor
public class LocalizationManager: ObservableObject {
    public static let shared = LocalizationManager()

    @Published public var currentLanguage: Language {
        didSet {
            UserDefaults.standard.set(currentLanguage.rawValue, forKey: "app_language")
        }
    }

    /// Incremented when the effective language changes (user selection or system change).
    @Published public var refreshVersion = 0

    private init() {
        let saved = UserDefaults.standard.string(forKey: "app_language")
        if let saved, let lang = Language(rawValue: saved) {
            currentLanguage = lang
        } else {
            currentLanguage = .system
        }
        observeSystemLanguage()
    }

    private func observeSystemLanguage() {
        NotificationCenter.default.addObserver(
            forName: NSLocale.currentLocaleDidChangeNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self, self.currentLanguage == .system else { return }
                self.refreshVersion += 1
            }
        }
    }

    public func localized(_ key: String, _ args: CVarArg...) -> String {
        Self.lookup(key, args)
    }

    /// Non-isolated static lookup for use from global `loc()` function.
    nonisolated public static func lookup(_ key: String, _ args: CVarArg...) -> String {
        let rawLang = UserDefaults.standard.string(forKey: "app_language")
        let saved = Language(rawValue: rawLang ?? "")
        let lang = saved?.effectiveLanguage ?? .zhHans
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
