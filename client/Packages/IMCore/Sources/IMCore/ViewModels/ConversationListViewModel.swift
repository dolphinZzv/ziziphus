import Foundation
import Combine

@MainActor
public class ConversationListViewModel: ObservableObject {
    @Published public var conversations: [ConvListItem] = []
    @Published public var isLoading = false
    @Published public var isRefreshing = false
    @Published public var connectionStatus: ConnectionStatus = .disconnected
    @Published public var errorMessage: String?

    private let convService = ConversationService.shared
    private let wsClient = WebSocketClient.shared
    private let cache = ConversationCache.shared
    private var cancellables: Set<AnyCancellable> = []
    private var connectionStatusTask: Task<Void, Never>?
    private var refreshTask: Task<Void, Never>?

    public init() {
        setupSubscriptions()
    }

    deinit {
        connectionStatusTask?.cancel()
    }

    private func setupSubscriptions() {
        // Observe connection status
        connectionStatusTask = Task { [weak self, weak wsClient] in
            guard let wsClient else { return }
            for await status in wsClient.$connectionStatus.values {
                guard !Task.isCancelled else { break }
                self?.connectionStatus = status
            }
        }

        // Refresh when messages are marked as read
        NotificationCenter.default.publisher(for: .init("didMarkRead"))
            .sink { [weak self] _ in self?.refresh() }
            .store(in: &cancellables)

        // Register WS handlers
        wsClient.on(.msgPush) { [weak self] frame in
            guard let self else { return }
            if let payload = try? JSONDecoder().decode(MsgPushPayload.self, from: frame.payload) {
                self.handlePush(payload: payload, frame: frame)
            }
        }

        wsClient.on(.sessionOnline) { [weak self] _ in
            self?.refresh()
        }

        wsClient.on(.sessionOffline) { [weak self] _ in
            self?.refresh()
        }
    }

    public func loadConversations() {
        guard !isLoading else { return }
        isLoading = true

        // Load from cache first
        conversations = cache.getAllConversations()

        Task {
            do {
                let items = try await convService.listConversations()
                self.conversations = items
                self.cache.upsertConversations(items)
            } catch {
                errorMessage = error.localizedDescription
            }
            self.isLoading = false
        }
    }

    public func refresh() {
        isRefreshing = true
        Task {
            do {
                let items = try await convService.listConversations()
                self.conversations = items
                self.cache.upsertConversations(items)
                errorMessage = nil
            } catch {
                errorMessage = error.localizedDescription
            }
            self.isRefreshing = false
        }
    }

    public func connectWebSocket() {
        wsClient.connect()
    }

    private func handlePush(payload: MsgPushPayload, frame: WSFrame) {
        // Throttle: cancel any pending refresh and debounce to avoid redundant requests
        refreshTask?.cancel()
        refreshTask = Task { [weak self] in
            guard let self else { return }
            try? await Task.sleep(nanoseconds: 500_000_000)  // 500ms debounce
            guard !Task.isCancelled else { return }
            do {
                let items = try await convService.listConversations()
                self.conversations = items
                self.cache.upsertConversations(items)
            } catch {
                self.errorMessage = error.localizedDescription
            }
        }
    }
}
