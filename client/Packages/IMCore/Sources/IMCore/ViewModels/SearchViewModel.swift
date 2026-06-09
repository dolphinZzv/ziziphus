import Foundation
import Combine

@MainActor
public class SearchViewModel: ObservableObject {
    @Published public var query = ""
    @Published public var results: [User] = []
    @Published public var isSearching = false
    @Published public var errorMessage: String?

    private let contactService = ContactService.shared
    private var searchTask: Task<Void, Never>?

    public init() {
        setupDebounce()
    }

    private func setupDebounce() {
        $query
            .debounce(for: .milliseconds(300), scheduler: DispatchQueue.main)
            .sink { [weak self] q in
                guard let self else { return }
                if q.trimmingCharacters(in: .whitespaces).isEmpty {
                    self.results = []
                } else {
                    self.search()
                }
            }
            .store(in: &cancellables)
    }

    private var cancellables = Set<AnyCancellable>()

    public func search() {
        searchTask?.cancel()
        isSearching = true
        searchTask = Task {
            do {
                let users = try await contactService.searchUsers(query: query)
                results = users
            } catch {
                results = []
                errorMessage = error.localizedDescription
            }
            isSearching = false
        }
    }
}
