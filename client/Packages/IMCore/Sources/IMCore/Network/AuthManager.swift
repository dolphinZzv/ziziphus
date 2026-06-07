import Foundation
import Combine

public class AuthManager: ObservableObject, @unchecked Sendable {
    public static let shared = AuthManager()

    @Published public var currentUser: User?
    @Published public var isLoggedIn = false

    private let tokenKey = "com.im.token"
    private let sessionIDKey = "com.im.session_id"
    private let clientSeqKey = "com.im.client_seq"

    private init() {
        if let _ = readToken() {
            isLoggedIn = true
        }
    }

    // MARK: - Token (stored in UserDefaults to avoid Keychain prompts without code signing)
    public func saveToken(_ token: String) {
        UserDefaults.standard.set(token, forKey: tokenKey)
    }

    public func readToken() -> String? {
        UserDefaults.standard.string(forKey: tokenKey)
    }

    public func clearToken() {
        UserDefaults.standard.removeObject(forKey: tokenKey)
    }

    // MARK: - Session ID
    public var sessionID: String? {
        get { UserDefaults.standard.string(forKey: sessionIDKey) }
        set { UserDefaults.standard.set(newValue, forKey: sessionIDKey) }
    }

    // MARK: - Client Seq
    public func nextClientSeq() -> Int64 {
        let seq = UserDefaults.standard.integer(forKey: clientSeqKey)
        let next = Int64(seq) + 1
        UserDefaults.standard.set(next, forKey: clientSeqKey)
        return next
    }

    // MARK: - Session
    public func setLoggedIn(user: User) {
        currentUser = user
        isLoggedIn = true
    }

    public func logout() {
        clearToken()
        currentUser = nil
        isLoggedIn = false
        sessionID = nil
        ConversationCache.shared.deleteAll()
        MessageCache.shared.deleteAll()
    }

    // MARK: - Helpers
    public func p2pConvID(userA: String, userB: String) -> String {
        [userA, userB].sorted().joined(separator: ":")
    }
}
