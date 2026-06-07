import Foundation
import Combine

@MainActor
public class LoginViewModel: ObservableObject {
    @Published public var name = ""
    @Published public var account = ""
    @Published public var password = ""
    @Published public var isRegistering = false
    @Published public var isLoading = false
    @Published public var errorMessage: String?
    @Published public var isLoggedIn = false

    private let authService = AuthService.shared
    private let wsClient = WebSocketClient.shared

    public init() {}

    public func login() {
        guard !account.isEmpty, !password.isEmpty else {
            errorMessage = loc("login.account_password_required")
            return
        }
        ConversationCache.shared.deleteAll()
        MessageCache.shared.deleteAll()
        isLoading = true
        errorMessage = nil
        Task {
            do {
                _ = try await authService.login(account: account, password: password)
                isLoggedIn = true
                wsClient.connect()
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    public func register() {
        guard !account.isEmpty, !name.isEmpty, !password.isEmpty else {
            errorMessage = loc("login.all_required")
            return
        }
        ConversationCache.shared.deleteAll()
        MessageCache.shared.deleteAll()
        isLoading = true
        errorMessage = nil
        Task {
            do {
                _ = try await authService.register(account: account, name: name, password: password)
                // Registration succeeded, switch to login
                password = ""
                isRegistering = false
                errorMessage = loc("login.register_success")
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
