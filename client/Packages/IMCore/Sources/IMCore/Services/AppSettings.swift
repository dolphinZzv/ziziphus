import Foundation

@MainActor
public class AppSettings: ObservableObject {
    public static let shared = AppSettings()

    @Published public var serverURL: String {
        didSet {
            UserDefaults.standard.set(serverURL, forKey: "server_url")
            APIClient.shared.baseURL = serverURL
        }
    }

    private init() {
        let saved = UserDefaults.standard.string(forKey: "server_url")
        serverURL = saved ?? "http://192.168.2.111:8080"
        APIClient.shared.baseURL = serverURL
    }
}
