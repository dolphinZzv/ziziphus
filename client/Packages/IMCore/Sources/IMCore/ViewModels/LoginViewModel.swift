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
    @Published public var isCheckingSession = true
    @Published public var rememberAccount = false
    @Published public var showAccountPicker = false

    private let authService = AuthService.shared
    private let wsClient = WebSocketClient.shared

    private let rememberedAccountsKey = "com.im.remembered_accounts"

    public var rememberedAccounts: [String] {
        UserDefaults.standard.stringArray(forKey: rememberedAccountsKey) ?? []
    }

    public func saveRememberedAccount() {
        guard !account.isEmpty else { return }
        var accounts = rememberedAccounts
        if !accounts.contains(account) {
            accounts.append(account)
        }
        UserDefaults.standard.set(accounts, forKey: rememberedAccountsKey)
    }

    public func removeRememberedAccount(_ account: String) {
        var accounts = rememberedAccounts
        accounts.removeAll { $0 == account }
        UserDefaults.standard.set(accounts, forKey: rememberedAccountsKey)
    }

    public func selectAccount(_ account: String) {
        self.account = account
        self.password = ""
        showAccountPicker = false
    }

    public init() {
        NotificationCenter.default.addObserver(
            forName: .init("kicked"),
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                self?.isLoggedIn = false
                self?.isCheckingSession = false
            }
        }
    }

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
                if rememberAccount {
                    saveRememberedAccount()
                    password = ""
                } else {
                    account = ""
                    password = ""
                }
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
        defer { isCheckingSession = false }
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
