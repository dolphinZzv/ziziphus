import Foundation
import Security
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

    // MARK: - Token
    public func saveToken(_ token: String) {
        let data = Data(token.utf8)
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: tokenKey,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
        ]
        SecItemDelete(query as CFDictionary)
        SecItemAdd(query as CFDictionary, nil)
    }

    public func readToken() -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: tokenKey,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else {
            return nil
        }
        return String(data: data, encoding: .utf8)
    }

    public func clearToken() {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: tokenKey,
        ]
        SecItemDelete(query as CFDictionary)
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
    }

    // MARK: - Helpers
    public func p2pConvID(userA: String, userB: String) -> String {
        [userA, userB].sorted().joined(separator: ":")
    }
}
