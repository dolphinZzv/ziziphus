import Foundation
import Combine

@MainActor
public class ContactListViewModel: ObservableObject {
    @Published public var contacts: [Contact] = []
    @Published public var isLoading = false
    @Published public var isRefreshing = false
    @Published public var searchQuery = ""
    @Published public var searchResults: [User] = []
    @Published public var isSearching = false
    @Published public var errorMessage: String?

    private let contactService = ContactService.shared
    private let ws = WebSocketClient.shared

    public init() {
        setupSubscriptions()
    }

    private func setupSubscriptions() {
        ws.on(.sessionOnline) { [weak self] frame in
            guard let self else { return }
            if let payload = try? JSONDecoder().decode(SessionEventPayload.self, from: frame.payload) {
                Task { @MainActor in
                    if let idx = self.contacts.firstIndex(where: { $0.userID == payload.userID }) {
                        self.contacts[idx].status = .online
                    }
                }
            }
        }

        ws.on(.sessionOffline) { [weak self] frame in
            guard let self else { return }
            if let payload = try? JSONDecoder().decode(SessionEventPayload.self, from: frame.payload) {
                Task { @MainActor in
                    if let idx = self.contacts.firstIndex(where: { $0.userID == payload.userID }) {
                        self.contacts[idx].status = .offline
                    }
                }
            }
        }
    }

    public func loadContacts() {
        guard !isLoading else { return }
        isLoading = true
        Task {
            do {
                contacts = try await contactService.listContacts()
                errorMessage = nil
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    public func refresh() {
        isRefreshing = true
        Task {
            do {
                contacts = try await contactService.listContacts()
                errorMessage = nil
            } catch {
                errorMessage = error.localizedDescription
            }
            isRefreshing = false
        }
    }

    public func addContact(userID: String, nickname: String? = nil) async throws {
        try await contactService.addContact(userID: userID, nickname: nickname)
        loadContacts()
    }

    public func removeContact(userID: String) async throws {
        try await contactService.removeContact(userID: userID)
        loadContacts()
    }

    public func searchUsers() {
        guard !searchQuery.trimmingCharacters(in: .whitespaces).isEmpty else {
            searchResults = []
            return
        }
        isSearching = true
        Task {
            do {
                searchResults = try await contactService.searchUsers(query: searchQuery)
            } catch {
                searchResults = []
                errorMessage = error.localizedDescription
            }
            isSearching = false
        }
    }
}
