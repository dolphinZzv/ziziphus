import Foundation

@MainActor
public class ConversationSettings {
    public static let shared = ConversationSettings()
    private let defaults = UserDefaults.standard
    private let prefix = "conv_settings_"

    private init() {}

    public func showAgentResponseOnly(convID: String) -> Bool {
        defaults.bool(forKey: prefix + convID)
    }

    public func setShowAgentResponseOnly(convID: String, value: Bool) {
        defaults.set(value, forKey: prefix + convID)
    }

    public func toggleAgentResponseOnly(convID: String) {
        let current = showAgentResponseOnly(convID: convID)
        setShowAgentResponseOnly(convID: convID, value: !current)
    }
}
