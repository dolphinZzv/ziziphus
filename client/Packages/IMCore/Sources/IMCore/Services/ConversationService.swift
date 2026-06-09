import Foundation

@MainActor
public class ConversationService {
    public static let shared = ConversationService()
    private let api = APIClient.shared

    private init() {}

    // MARK: - List
    public func listConversations(page: Int = 1, size: Int = 20) async throws -> [ConvListItem] {
        let result: PaginatedData<ConvListItem> = try await api.request(
            "/api/v1/conversations",
            query: ["page": "\(page)", "size": "\(size)"]
        )
        return result.items
    }

    // MARK: - Detail
    public func getConversationDetail(convID: String) async throws -> ConversationDetail {
        let detail: ConversationDetail = try await api.request("/api/v1/conversations/\(convID)")
        return detail
    }

    // MARK: - Create P2P
    public func createP2P(userID: String) async throws -> (convID: String, name: String) {
        struct P2PReq: Codable, Sendable {
            let userID: String
            enum CodingKeys: String, CodingKey {
                case userID = "user_id"
            }
        }
        struct P2PResp: Codable, Sendable {
            let convID: String
            let name: String
            let type: ConvType
            enum CodingKeys: String, CodingKey {
                case convID = "conv_id"
                case name, type
            }
        }
        let resp: P2PResp = try await api.request(
            "/api/v1/conversations/p2p",
            method: .post,
            body: P2PReq(userID: userID)
        )
        return (resp.convID, resp.name)
    }

    // MARK: - Create Group
    public func createGroup(name: String, memberIDs: [String]) async throws -> Conversation {
        let detail: Conversation = try await api.request(
            "/api/v1/conversations/group",
            method: .post,
            body: CreateGroupReq(name: name, memberIDs: memberIDs)
        )
        return detail
    }

    // MARK: - Mark Read
    public func markRead(convID: String, msgID: Int64) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/conversations/\(convID)/read",
            method: .post,
            body: MarkReadReq(msgID: msgID)
        )
    }

    // MARK: - Members
    public func addMembers(convID: String, userIDs: [String]) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/conversations/\(convID)/members",
            method: .post,
            body: AddMembersReq(userIDs: userIDs)
        )
    }

    public func removeMember(convID: String, userID: String) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/conversations/\(convID)/members/\(userID)",
            method: .delete
        )
    }

    public func leaveGroup(convID: String) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/conversations/\(convID)/leave",
            method: .post
        )
    }

    // MARK: - Group Search
    public func searchGroups(query: String, page: Int = 1, size: Int = 20) async throws -> [GroupSearchItem] {
        let result: PaginatedData<GroupSearchItem> = try await api.request(
            "/api/v1/groups/search",
            query: ["q": query, "page": "\(page)", "size": "\(size)"]
        )
        return result.items
    }

    // MARK: - Join Requests
    public func requestJoin(convID: String) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/conversations/\(convID)/join-requests",
            method: .post
        )
    }

    public func listJoinRequests(convID: String) async throws -> [JoinRequest] {
        let requests: [JoinRequest] = try await api.request(
            "/api/v1/conversations/\(convID)/join-requests"
        )
        return requests
    }

    public func approveJoinRequest(convID: String, userID: String) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/conversations/\(convID)/join-requests/\(userID)/approve",
            method: .post
        )
    }

    public func rejectJoinRequest(convID: String, userID: String) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/conversations/\(convID)/join-requests/\(userID)/reject",
            method: .post
        )
    }

    // MARK: - Requests
    private struct CreateGroupReq: Codable, Sendable {
        let name: String
        let memberIDs: [String]

        enum CodingKeys: String, CodingKey {
            case name
            case memberIDs = "member_ids"
        }
    }

    private struct MarkReadReq: Codable, Sendable {
        let msgID: Int64

        enum CodingKeys: String, CodingKey {
            case msgID = "msg_id"
        }
    }

    private struct AddMembersReq: Codable, Sendable {
        let userIDs: [String]

        enum CodingKeys: String, CodingKey {
            case userIDs = "user_ids"
        }
    }
}
