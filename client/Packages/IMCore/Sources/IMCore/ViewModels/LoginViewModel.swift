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

    // MFA challenge state
    @Published public var mfaRequired = false
    @Published public var mfaToken = ""
    @Published public var mfaType: Int = 0
    @Published public var maskedEmail = ""
    @Published public var mfaCode = ""
    @Published public var mfaError: String?
    private var mfaUserID = ""

    private let authService = AuthService.shared
    private let wsClient = WebSocketClient.shared

    private let rememberedAccountsKey = "com.im.remembered_accounts"

    @Published public var rememberedAccounts: [String] = []

    public func loadRememberedAccounts() {
        rememberedAccounts = UserDefaults.standard.stringArray(forKey: rememberedAccountsKey) ?? []
    }

    public func saveRememberedAccount() {
        guard !account.isEmpty else { return }
        if !rememberedAccounts.contains(account) {
            rememberedAccounts.append(account)
        }
        UserDefaults.standard.set(rememberedAccounts, forKey: rememberedAccountsKey)
    }

    public func removeRememberedAccount(_ account: String) {
        rememberedAccounts.removeAll { $0 == account }
        UserDefaults.standard.set(rememberedAccounts, forKey: rememberedAccountsKey)
    }

    public func selectAccount(_ account: String) {
        self.account = account
        self.password = ""
        showAccountPicker = false
    }

    public init() {
        loadRememberedAccounts()
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
        mfaError = nil
        Task {
            do {
                _ = try await authService.login(account: account, password: password)
                isLoggedIn = true
                wsClient.connect()
                finishLoginCleanup()
            } catch let error as LoginError {
                if case .mfaRequired(let uid, let mfaTok, let mfaT, let email) = error {
                    mfaRequired = true
                    mfaUserID = uid
                    mfaToken = mfaTok
                    mfaType = mfaT
                    maskedEmail = email
                    mfaCode = ""
                }
                isLoading = false
            } catch {
                errorMessage = error.localizedDescription
                isLoading = false
            }
        }
    }

    public func verifyMFA() {
        guard !mfaCode.isEmpty else { return }
        isLoading = true
        mfaError = nil
        Task {
            do {
                _ = try await authService.mfaVerify(userID: mfaUserID, mfaToken: mfaToken, code: mfaCode)
                isLoggedIn = true
                wsClient.connect()
                finishLoginCleanup()
                resetMFAState()
            } catch {
                mfaError = error.localizedDescription
                isLoading = false
            }
        }
    }

    public func cancelMFA() {
        resetMFAState()
        isLoading = false
    }

    private func resetMFAState() {
        mfaRequired = false
        mfaToken = ""
        mfaType = 0
        maskedEmail = ""
        mfaCode = ""
        mfaError = nil
    }

    private func finishLoginCleanup() {
        if rememberAccount {
            saveRememberedAccount()
            password = ""
        } else {
            account = ""
            password = ""
        }
        isLoading = false
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
