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

    // MARK: - Agents
    public func listAgents() async throws -> [User] {
        let agents: [User] = try await api.request("/api/v1/users/me/agents")
        return agents
    }

    public func createAgent(name: String, avatar: String = "", primaryColor: String = "", secondaryColor: String = "", wakeMode: Int = 0) async throws -> User {
        struct CreateAgentReq: Codable, Sendable {
            let name: String
            let avatar: String
            let primaryColor: String
            let secondaryColor: String
            let wakeMode: Int

            enum CodingKeys: String, CodingKey {
                case name, avatar
                case primaryColor = "primary_color"
                case secondaryColor = "secondary_color"
                case wakeMode = "wake_mode"
            }
        }

        let agent: User = try await api.request(
            "/api/v1/users/me/agents",
            method: .post,
            body: CreateAgentReq(name: name, avatar: avatar, primaryColor: primaryColor, secondaryColor: secondaryColor, wakeMode: wakeMode)
        )
        return agent
    }

    public func updateAgent(agentID: String, name: String, avatar: String = "", primaryColor: String = "", secondaryColor: String = "", wakeMode: Int = 0) async throws {
        struct UpdateAgentReq: Codable, Sendable {
            let name: String
            let avatar: String
            let primaryColor: String
            let secondaryColor: String
            let wakeMode: Int

            enum CodingKeys: String, CodingKey {
                case name, avatar
                case primaryColor = "primary_color"
                case secondaryColor = "secondary_color"
                case wakeMode = "wake_mode"
            }
        }

        let _: [String: String] = try await api.request(
            "/api/v1/users/me/agents/\(agentID)",
            method: .put,
            body: UpdateAgentReq(name: name, avatar: avatar, primaryColor: primaryColor, secondaryColor: secondaryColor, wakeMode: wakeMode)
        )
    }

    public func regenerateAgentKey(agentID: String) async throws -> String {
        struct RegenerateResp: Codable, Sendable {
            let apiKey: String
            enum CodingKeys: String, CodingKey {
                case apiKey = "api_key"
            }
        }
        let resp: RegenerateResp = try await api.request(
            "/api/v1/users/me/agents/\(agentID)/regenerate-key",
            method: .put
        )
        return resp.apiKey
    }

    public func deleteAgent(agentID: String) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/users/me/agents/\(agentID)",
            method: .delete
        )
    }
}
