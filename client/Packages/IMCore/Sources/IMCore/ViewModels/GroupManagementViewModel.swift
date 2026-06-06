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
                let userIDs = members.map(\.userID)
                let users = try await contactService.batchGetUsers(userIDs: userIDs)
                membersInfo = users
            } catch {
                // keep existing data
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
            }
            isSearching = false
        }
    }
}
