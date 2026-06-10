import Foundation

@MainActor
public class AuthService {
    public static let shared = AuthService()
    private let api = APIClient.shared

    private init() {}

    // MARK: - Register
    public func register(account: String, name: String, password: String) async throws -> User {
        struct RegisterReq: Codable, Sendable {
            let account: String
            let name: String
            let password: String
        }
        struct RegisterResp: Codable, Sendable {
            let userID: String
            let account: String
            let name: String

            enum CodingKeys: String, CodingKey {
                case userID = "user_id"
                case account, name
            }
        }

        let resp: RegisterResp = try await api.request(
            "/api/v1/users/register",
            method: .post,
            body: RegisterReq(account: account, name: name, password: password)
        )

        return User(userID: resp.userID, account: resp.account, name: resp.name)
    }

    // MARK: - Login
    public func login(account: String, password: String) async throws -> User {
        struct LoginReq: Codable, Sendable {
            let account: String
            let password: String
        }
        struct LoginResp: Codable, Sendable {
            let userID: String
            let account: String
            let name: String
            let token: String
            let expiresAt: Int64

            enum CodingKeys: String, CodingKey {
                case userID = "user_id"
                case account, name, token
                case expiresAt = "expires_at"
            }
        }

        let resp: LoginResp = try await api.request(
            "/api/v1/users/login",
            method: .post,
            body: LoginReq(account: account, password: password)
        )

        AuthManager.shared.saveToken(resp.token)
        let user = User(userID: resp.userID, account: resp.account, name: resp.name)
        AuthManager.shared.setLoggedIn(user: user)
        return user
    }

    // MARK: - Get Me
    public func getMe() async throws -> User {
        let user: User = try await api.request("/api/v1/users/me")
        AuthManager.shared.setLoggedIn(user: user)
        return user
    }

    // MARK: - Update Profile
    public func updateProfile(name: String? = nil, avatar: String? = nil, primaryColor: String? = nil, secondaryColor: String? = nil) async throws -> User {
        struct UpdateProfileReq: Codable, Sendable {
            let name: String?
            let avatar: String?
            let primaryColor: String?
            let secondaryColor: String?

            enum CodingKeys: String, CodingKey {
                case name, avatar
                case primaryColor = "primary_color"
                case secondaryColor = "secondary_color"
            }
        }

        let body = UpdateProfileReq(name: name, avatar: avatar, primaryColor: primaryColor, secondaryColor: secondaryColor)
        let _: [String: String] = try await api.request("/api/v1/users/me", method: .put, body: body)
        // Fetch full user after update
        return try await getMe()
    }
}
