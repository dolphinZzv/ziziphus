import Foundation
import Combine

@MainActor
public class GroupManagementViewModel: ObservableObject {
    @Published public var members: [ConvMember] = []
    @Published public var membersInfo: [String: User] = [:]
    @Published public var isLoading = false
    @Published public var isCreating = false

    // Add member search
    @Published public var searchQuery = ""
    @Published public var searchResults: [User] = []
    @Published public var isSearching = false

    // Join requests
    @Published public var joinRequests: [JoinRequest] = []
    @Published public var isLoadingRequests = false
    @Published public var joinRequestSent = false
    @Published public var isAdmin = false
    @Published public var conversationAvatar = ""
    @Published public var errorMessage: String?

    private let convService = ConversationService.shared
    private let contactService = ContactService.shared
    private var searchTask: Task<Void, Never>?

    public init() {}

    public func loadDetail(convID: String) {
        guard !isLoading else { return }
        isLoading = true
        Task {
            do {
                let detail = try await convService.getConversationDetail(convID: convID)
                members = detail.members
                conversationAvatar = detail.avatar
                isAdmin = members.first(where: { $0.userID == AuthManager.shared.currentUser?.userID })?.role.rawValue ?? 0 >= ConvRole.admin.rawValue
                let userIDs = members.map(\.userID)
                let users = try await contactService.batchGetUsers(userIDs: userIDs)
                membersInfo = users
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    public func addMember(convID: String, userID: String) async throws {
        try await convService.addMembers(convID: convID, userIDs: [userID])
        loadDetail(convID: convID)
    }

    public func removeMember(convID: String, userID: String) async throws {
        try await convService.removeMember(convID: convID, userID: userID)
        loadDetail(convID: convID)
    }

    public func leaveGroup(convID: String) async throws {
        try await convService.leaveGroup(convID: convID)
    }

    public func loadJoinRequests(convID: String) {
        isLoadingRequests = true
        Task {
            do {
                joinRequests = try await convService.listJoinRequests(convID: convID)
            } catch {
                joinRequests = []
                errorMessage = error.localizedDescription
            }
            isLoadingRequests = false
        }
    }

    public func requestJoin(convID: String) async throws {
        try await convService.requestJoin(convID: convID)
        joinRequestSent = true
    }

    public func approveJoinRequest(convID: String, userID: String) async throws {
        try await convService.approveJoinRequest(convID: convID, userID: userID)
        loadJoinRequests(convID: convID)
        loadDetail(convID: convID)
    }

    public func rejectJoinRequest(convID: String, userID: String) async throws {
        try await convService.rejectJoinRequest(convID: convID, userID: userID)
        loadJoinRequests(convID: convID)
    }

    public func createGroup(name: String, memberIDs: [String]) async throws -> Conversation {
        isCreating = true
        defer { isCreating = false }
        return try await convService.createGroup(name: name, memberIDs: memberIDs)
    }

    public func searchUsers() {
        searchTask?.cancel()
        let q = searchQuery.trimmingCharacters(in: .whitespaces)
        guard !q.isEmpty else {
            searchResults = []
            return
        }
        isSearching = true
        searchTask = Task {
            do {
                let users = try await contactService.searchUsers(query: q)
                searchResults = users
            } catch {
                searchResults = []
                errorMessage = error.localizedDescription
            }
            isSearching = false
        }
    }
}
