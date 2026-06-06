import Foundation
import Combine

@MainActor
public class LoginViewModel: ObservableObject {
    @Published public var name = ""
    @Published public var userID = ""
    @Published public var password = ""
    @Published public var isRegistering = false
    @Published public var isLoading = false
    @Published public var errorMessage: String?
    @Published public var isLoggedIn = false

    private let authService = AuthService.shared
    private let wsClient = WebSocketClient.shared

    public init() {}

    public func login() {
        guard !userID.isEmpty, !password.isEmpty else {
            errorMessage = "请输入用户ID和密码"
            return
        }
        isLoading = true
        errorMessage = nil
        Task {
            do {
                _ = try await authService.login(userID: userID, password: password)
                isLoggedIn = true
                wsClient.connect()
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    public func register() {
        guard !name.isEmpty, !password.isEmpty else {
            errorMessage = "请输入名称和密码"
            return
        }
        isLoading = true
        errorMessage = nil
        Task {
            do {
                _ = try await authService.register(name: name, password: password)
                isLoggedIn = true
                wsClient.connect()
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    public func switchMode() {
        isRegistering.toggle()
        errorMessage = nil
    }

    public func checkExistingSession() async {
        if AuthManager.shared.isLoggedIn, AuthManager.shared.readToken() != nil {
            // token exists, try to validate
            do {
                _ = try await authService.getMe()
                isLoggedIn = true
                wsClient.connect()
            } catch {
                AuthManager.shared.logout()
            }
        }
    }
}
