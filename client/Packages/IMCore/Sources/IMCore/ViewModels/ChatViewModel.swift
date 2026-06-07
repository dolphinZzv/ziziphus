import Foundation
import Combine

@MainActor
public class ChatViewModel: ObservableObject {
    @Published public var messages: [Message] = []
    @Published public var inputText = ""
    @Published public var isLoadingHistory = false
    @Published public var allHistoryLoaded = false
    @Published public var isSending = false
    @Published public var isTyping = false
    @Published public var peerOnline = false

    public let convID: String
    private let msgService = MessageService.shared
    private let ws = WebSocketClient.shared
    private let cache = MessageCache.shared
    private let convService = ConversationService.shared

    private var typingTimer: Timer?
    private var lastTypingSend: TimeInterval = 0
    private let typingInterval: TimeInterval = 3

    public init(convID: String) {
        self.convID = convID
        setupSubscriptions()
    }

    private func setupSubscriptions() {
        // Handle incoming pushes
        ws.on(.msgPush) { [weak self] frame in
            guard let self else { return }
            if let msg = MessageService.shared.handlePush(frame: frame), msg.convID == self.convID {
                Task { @MainActor in
                    self.messages.append(msg)
                    self.sortMessages()
                    self.cache.insertMessage(msg)
                    self.markAsReadIfActive()
                }
            }
        }

        // Handle read notifications
        ws.on(.msgReadNotify) { [weak self] frame in
            guard let self else { return }
            if let payload = try? JSONDecoder().decode(MsgReadNotifyPayload.self, from: frame.payload),
               payload.convID == self.convID {
                Task { @MainActor in
                    if let idx = self.messages.firstIndex(where: { $0.msgID == payload.msgID }) {
                        self.messages[idx].status = .read
                    }
                }
            }
        }

        // Typing indicator
        ws.on(.typing) { [weak self] frame in
            guard let self else { return }
            if let payload = try? JSONDecoder().decode(TypingPayload.self, from: frame.payload),
               payload.convID == self.convID {
                Task { @MainActor in
                    self.isTyping = true
                    self.typingTimer?.invalidate()
                    self.typingTimer = Timer.scheduledTimer(withTimeInterval: 10, repeats: false) { [weak self] _ in
                        Task { @MainActor in
                            self?.isTyping = false
                        }
                    }
                }
            }
        }
    }

    // MARK: - Load Messages
    public func loadInitialMessages() {
        messages = cache.getMessages(convID: convID)
        sortMessages()

        Task {
            do {
                let msgs = try await msgService.getHistory(convID: convID, limit: 50)
                for msg in msgs {
                    cache.insertMessage(msg)
                }
                messages = cache.getMessages(convID: convID)
                sortMessages()
                if msgs.count < 50 {
                    allHistoryLoaded = true
                }
            } catch {
                // cache data already loaded
            }
        }
    }

    public func loadMoreHistory() {
        guard !isLoadingHistory, !allHistoryLoaded, let first = messages.first else { return }
        isLoadingHistory = true
        Task {
            do {
                let msgs = try await msgService.getHistory(convID: convID, beforeMsgID: first.msgID, limit: 50)
                for msg in msgs {
                    cache.insertMessage(msg)
                }
                messages = cache.getMessages(convID: convID)
                sortMessages()
                if msgs.count < 50 {
                    allHistoryLoaded = true
                }
            } catch {
                // silently fail
            }
            isLoadingHistory = false
        }
    }

    // MARK: - Send Message
    public func sendMessage() {
        let text = inputText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else { return }

        inputText = ""
        isSending = true

        let clientSeq = AuthManager.shared.nextClientSeq()
        let localMsg = Message(
            convID: convID,
            senderID: AuthManager.shared.currentUser?.userID ?? "",
            body: text,
            timestamp: Int64(Date().timeIntervalSince1970 * 1000),
            clientSeq: clientSeq,
            status: .sending
        )
        messages.append(localMsg)
        sortMessages()

        Task {
            do {
                let ack = try await msgService.sendMessage(convID: convID, body: text, clientSeq: clientSeq)
                if let idx = messages.firstIndex(where: { $0.clientSeq == ack.clientSeq && $0.msgID == 0 }) {
                    messages[idx].msgID = ack.msgID
                    messages[idx].timestamp = ack.timestamp
                    messages[idx].status = .sent
                    cache.insertMessage(messages[idx])
                }
            } catch {
                if let idx = messages.firstIndex(where: { $0.clientSeq == clientSeq }) {
                    messages.remove(at: idx)
                }
            }
            isSending = false
        }
    }

    // MARK: - Mark Read
    public func markAsReadIfActive() {
        guard let last = messages.last(where: { $0.senderID != AuthManager.shared.currentUser?.userID }) else { return }
        Task {
            try? await convService.markRead(convID: convID, msgID: last.msgID)
        }
    }

    // MARK: - Typing
    public func userDidStartTyping() {
        let now = Date().timeIntervalSince1970
        guard now - lastTypingSend > typingInterval else { return }
        lastTypingSend = now

        let payload = TypingPayload(convID: convID, userID: AuthManager.shared.currentUser?.userID ?? "")
        if let data = try? JSONEncoder().encode(payload) {
            ws.send(frame: WSFrame(type: .typing, payload: data))
        }
    }

    // MARK: - Helpers
    private func sortMessages() {
        messages.sort { a, b in
            // Unsent local messages (msgID == 0) go to the end
            if a.msgID == 0 && b.msgID != 0 { return false }
            if a.msgID != 0 && b.msgID == 0 { return true }
            // Sort by timestamp ascending (oldest first)
            if a.timestamp != b.timestamp { return a.timestamp < b.timestamp }
            return a.convSeq < b.convSeq
        }
    }
}
