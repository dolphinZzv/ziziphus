import Foundation

@MainActor
public class AppSettings: ObservableObject {
    public static let shared = AppSettings()

    @Published public var bubbleColorHex: String {
        didSet {
            UserDefaults.standard.set(bubbleColorHex, forKey: "bubble_color")
        }
    }

    @Published public var serverURL: String {
        didSet {
            UserDefaults.standard.set(serverURL, forKey: "server_url")
            APIClient.shared.baseURL = serverURL
        }
    }

    nonisolated public static func serverBaseURL() -> String {
        UserDefaults.standard.string(forKey: "server_url") ?? "http://47.95.200.101:10011"
    }

    private init() {
        let hex = UserDefaults.standard.string(forKey: "bubble_color") ?? "#d5e3f8"
        bubbleColorHex = hex
        let saved = UserDefaults.standard.string(forKey: "server_url")
        serverURL = saved ?? "http://47.95.200.101:10011"
        APIClient.shared.baseURL = serverURL
    }
}
