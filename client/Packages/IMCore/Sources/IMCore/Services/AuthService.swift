import Foundation

@MainActor
public class AuthService {
    public static let shared = AuthService()
    private let api = APIClient.shared

    private init() {}

    // MARK: - Register
    public func register(name: String, password: String) async throws -> User {
        struct RegisterReq: Codable, Sendable {
            let name: String
            let password: String
        }
        struct RegisterResp: Codable, Sendable {
            let userID: String
            let name: String
            let token: String

            enum CodingKeys: String, CodingKey {
                case userID = "user_id"
                case name, token
            }
        }

        let resp: RegisterResp = try await api.request(
            "/api/v1/users/register",
            method: .post,
            body: RegisterReq(name: name, password: password)
        )

        AuthManager.shared.saveToken(resp.token)
        let user = User(userID: resp.userID, name: resp.name)
        AuthManager.shared.setLoggedIn(user: user)
        return user
    }

    // MARK: - Login
    public func login(userID: String, password: String) async throws -> User {
        struct LoginReq: Codable, Sendable {
            let userID: String
            let password: String

            enum CodingKeys: String, CodingKey {
                case userID = "user_id"
                case password
            }
        }
        struct LoginResp: Codable, Sendable {
            let userID: String
            let name: String
            let token: String
            let expiresAt: Int64

            enum CodingKeys: String, CodingKey {
                case userID = "user_id"
                case name, token
                case expiresAt = "expires_at"
            }
        }

        let resp: LoginResp = try await api.request(
            "/api/v1/users/login",
            method: .post,
            body: LoginReq(userID: userID, password: password)
        )

        AuthManager.shared.saveToken(resp.token)
        let user = User(userID: resp.userID, name: resp.name)
        AuthManager.shared.setLoggedIn(user: user)
        return user
    }

    // MARK: - Get Me
    public func getMe() async throws -> User {
        let user: User = try await api.request("/api/v1/users/me")
        AuthManager.shared.setLoggedIn(user: user)
        return user
    }
}
