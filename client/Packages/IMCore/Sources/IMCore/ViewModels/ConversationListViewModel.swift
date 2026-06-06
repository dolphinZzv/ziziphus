import Foundation
import Combine

@MainActor
public class ConversationListViewModel: ObservableObject {
    @Published public var conversations: [ConvListItem] = []
    @Published public var isLoading = false
    @Published public var isRefreshing = false
    @Published public var connectionStatus: ConnectionStatus = .disconnected

    private let convService = ConversationService.shared
    private let wsClient = WebSocketClient.shared
    private let cache = ConversationCache.shared

    public init() {
        setupSubscriptions()
    }

    private func setupSubscriptions() {
        // Observe connection status
        Task { [weak self, weak wsClient] in
            guard let wsClient else { return }
            for await status in wsClient.$connectionStatus.values {
                self?.connectionStatus = status
            }
        }

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
                // cache data already loaded
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
            } catch {
                // keep existing data
            }
            self.isRefreshing = false
        }
    }

    public func connectWebSocket() {
        wsClient.connect()
    }

    private func handlePush(payload: MsgPushPayload, frame: WSFrame) {
        // Update conversation list when new message arrives
        Task {
            do {
                let items = try await convService.listConversations()
                self.conversations = items
                self.cache.upsertConversations(items)
            } catch {
                // silently fail
            }
        }
    }
}
