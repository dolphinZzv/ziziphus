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
        UserDefaults.standard.string(forKey: "server_url") ?? "http://192.168.2.111:8080"
    }

    private init() {
        let hex = UserDefaults.standard.string(forKey: "bubble_color") ?? "#d5e3f8"
        bubbleColorHex = hex
        let saved = UserDefaults.standard.string(forKey: "server_url")
        serverURL = saved ?? "http://192.168.2.111:8080"
        APIClient.shared.baseURL = serverURL
    }
}
