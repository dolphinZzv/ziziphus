import Foundation

@MainActor
public class ContactService {
    public static let shared = ContactService()
    private let api = APIClient.shared

    private init() {}

    // MARK: - List
    public func listContacts(page: Int = 1, size: Int = 20) async throws -> [Contact] {
        let result: PaginatedData<Contact> = try await api.request(
            "/api/v1/contacts",
            query: ["page": "\(page)", "size": "\(size)"]
        )
        return result.items
    }

    // MARK: - Add
    public func addContact(userID: String, nickname: String? = nil) async throws {
        struct AddReq: Codable, Sendable {
            let userID: String
            let nickname: String?

            enum CodingKeys: String, CodingKey {
                case userID = "user_id"
                case nickname
            }
        }
        let _: [String: String] = try await api.request(
            "/api/v1/contacts",
            method: .post,
            body: AddReq(userID: userID, nickname: nickname)
        )
    }

    // MARK: - Remove
    public func removeContact(userID: String) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/contacts/\(userID)",
            method: .delete
        )
    }

    // MARK: - Update Nickname
    public func updateNickname(userID: String, nickname: String) async throws {
        struct UpdateReq: Codable, Sendable {
            let nickname: String
        }
        let _: [String: String] = try await api.request(
            "/api/v1/contacts/\(userID)",
            method: .put,
            body: UpdateReq(nickname: nickname)
        )
    }

    // MARK: - Search Users
    public func searchUsers(query: String) async throws -> [User] {
        struct SearchResult: Codable, Sendable {
            let items: [User]
            let total: Int
        }
        let result: SearchResult = try await api.request(
            "/api/v1/users/search",
            query: ["q": query]
        )
        return result.items
    }

    // MARK: - Batch Get Users
    public func batchGetUsers(userIDs: [String]) async throws -> [String: User] {
        struct BatchReq: Codable, Sendable {
            let userIDs: [String]

            enum CodingKeys: String, CodingKey {
                case userIDs = "user_ids"
            }
        }
        struct BatchResp: Codable, Sendable {
            let users: [String: User]
        }
        let resp: BatchResp = try await api.request(
            "/api/v1/users/batch",
            method: .post,
            body: BatchReq(userIDs: userIDs)
        )
        return resp.users
    }
}
