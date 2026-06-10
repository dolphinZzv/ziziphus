import Foundation
import Combine

/// Append a timestamped line to /tmp/imcore.log for debugging.
nonisolated func logToFile(_ msg: String) {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    let ts = formatter.string(from: Date())
    let line = "\(ts) \(msg)\n"
    guard let data = line.data(using: .utf8) else { return }
    let url = URL(fileURLWithPath: "/tmp/imcore.log")
    if FileManager.default.fileExists(atPath: url.path) {
        if let handle = try? FileHandle(forWritingTo: url) {
            try? handle.seekToEnd()
            try? handle.write(contentsOf: data)
            try? handle.close()
        }
    } else {
        try? data.write(to: url, options: .atomic)
    }
}

@MainActor
public class ChatViewModel: ObservableObject {
    @Published public var messages: [Message] = []
    @Published public var inputText = ""
    @Published public var isLoadingHistory = false
    @Published public var allHistoryLoaded = false
    @Published public var isSending = false
    @Published public var replyingToMsg: Message?
    @Published public var isTyping = false
    @Published public var peerOnline = false
    @Published public var sendErrorMessage: String?
    @Published public var errorMessage: String?
    @Published public var uploadProgress: [Int64: Double] = [:]

    private let maxBodyBytes = 10_240
    private let maxMessages = 500

    public let convID: String
    private let msgService = MessageService.shared
    private let ws = WebSocketClient.shared
    private let cache = MessageCache.shared
    private let convService = ConversationService.shared
    private let contactService = ContactService.shared

    @Published public var senderInfo: [String: User] = [:]

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
            let msg = MessageService.shared.handlePush(frame: frame)
            if let msg, msg.convID == self.convID {
                Task { @MainActor in
                    logToFile("[ChatVM] push msg arrived: msgID=\(msg.msgID) body=\(msg.body.prefix(20))")
                    self.insertMessageInOrder(msg)
                    self.cache.insertMessage(msg)
                    self.markAsReadIfActive()
                    self.loadSenderInfo()
                }
            } else if msg != nil {
                logToFile("[ChatVM] push for different conv: \(msg!.convID) != \(self.convID)")
            } else {
                logToFile("[ChatVM] push decode failed")
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
        // Show cached messages instantly
        messages = cache.getMessages(convID: convID)
        sortMessages()

        Task {
            do {
                let msgs = try await msgService.getHistory(convID: convID, limit: 50)
                for msg in msgs {
                    cache.insertMessage(msg)
                }
                // Check if there are truly new messages before replacing
                let updated = cache.getMessages(convID: convID)
                if updated.map(\.msgID) != messages.map(\.msgID) {
                    messages = updated
                    sortMessages()
                }
                if msgs.count < 50 {
                    allHistoryLoaded = true
                }
            } catch {
                logToFile("[ChatVM] load history error: \(error)")
                errorMessage = error.localizedDescription
            }
            loadSenderInfo()
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
                // Prepend new messages directly — don't use cache.getMessages (returns only last 50)
                let existingIDs = Set(messages.filter { $0.msgID > 0 }.map(\.msgID))
                let newMsgs = msgs.filter { !existingIDs.contains($0.msgID) }
                if !newMsgs.isEmpty {
                    messages.append(contentsOf: newMsgs)
                    sortMessages()
                    // Cap to prevent unbounded growth.
                    // When capping discards the oldest messages, mark allHistoryLoaded
                    // to prevent the next scroll-to-top from requesting a gap.
                    if messages.count > maxMessages {
                        messages = Array(messages.suffix(maxMessages))
                        allHistoryLoaded = true
                    }
                }
                if msgs.count < 50 {
                    allHistoryLoaded = true
                }
            } catch {
                logToFile("[ChatVM] load more history error: \(error)")
                errorMessage = error.localizedDescription
            }
            loadSenderInfo()
            isLoadingHistory = false
        }
    }

    public func loadSenderInfo() {
        let ids = Set(messages.map(\.senderID))
            .subtracting([AuthManager.shared.currentUser?.userID ?? ""])
            .subtracting(senderInfo.keys)
        guard !ids.isEmpty else { return }
        Task {
            do {
                let info = try await contactService.batchGetUsers(userIDs: Array(ids))
                senderInfo.merge(info) { _, new in new }
            } catch {
                logToFile("[ChatVM] loadSenderInfo error: \(error)")
            }
        }
    }

    // MARK: - Load Context Around Message
    public func loadContextAround(msgID: Int64) {
        Task {
            do {
                let msgs = try await ConversationService.shared.getHistory(convID: convID, aroundMsgID: msgID, limit: 50)
                for msg in msgs {
                    cache.insertMessage(msg)
                }
                messages = msgs
                sortMessages()
                allHistoryLoaded = false
            } catch {
                logToFile("[ChatVM] load context error: \(error)")
            }
            loadSenderInfo()
        }
    }

    // MARK: - Send Message
    public func sendMessage() {
        let text = inputText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else { return }
        guard text.utf8.count <= maxBodyBytes else {
            sendErrorMessage = loc("chat.message_too_long")
            return
        }
        guard !isSending else { return }

        sendErrorMessage = nil
        inputText = ""
        replyingToMsg = nil
        isSending = true

        let clientSeq = AuthManager.shared.nextClientSeq()
        let localMsg = Message(
            convID: convID,
            senderID: AuthManager.shared.currentUser?.userID ?? "",
            body: text,
            replyTo: replyingToMsg?.msgID ?? 0,
            timestamp: Int64(Date().timeIntervalSince1970 * 1000),
            clientSeq: clientSeq,
            status: .sending
        )
        messages.append(localMsg)
        sortMessages()

        Task {
            do {
                let replyTo = replyingToMsg?.msgID ?? 0
                let ack = try await msgService.sendMessage(convID: convID, body: text, clientSeq: clientSeq, replyTo: replyTo)
                replyingToMsg = nil
                if let idx = messages.firstIndex(where: { $0.clientSeq == ack.clientSeq && $0.msgID == 0 }) {
                    messages[idx].msgID = ack.msgID
                    messages[idx].timestamp = ack.timestamp
                    messages[idx].status = .sent
                    cache.insertMessage(messages[idx])
                }
            } catch {
                logToFile("[ChatVM] sendMessage error: \(error)")
                if let idx = messages.firstIndex(where: { $0.clientSeq == clientSeq }) {
                    messages[idx].status = .failed
                }
                sendErrorMessage = loc("chat.send_failed")
            }
            isSending = false
        }
    }

    // MARK: - Send File/Image
    public func sendFile(fileData: Data, fileName: String, fileType: Int = 1) {
        guard !isSending else { return }
        sendErrorMessage = nil
        isSending = true

        let clientSeq = AuthManager.shared.nextClientSeq()

        // Create placeholder message before upload so progress is visible
        let placeholderMsg = Message(
            convID: convID,
            senderID: AuthManager.shared.currentUser?.userID ?? "",
            contentType: fileType == 0 ? .image : .file,
            body: fileName,
            timestamp: Int64(Date().timeIntervalSince1970 * 1000),
            clientSeq: clientSeq,
            status: .sending
        )
        messages.append(placeholderMsg)
        sortMessages()
        uploadProgress[clientSeq] = 0

        // Use detached task to avoid @MainActor isolation inheritance for the progress closure
        Task.detached { [weak self] in
            guard let self else { return }
            do {
                let finfo = try await APIClient.shared.uploadFile(
                    fileData: fileData,
                    fileName: fileName,
                    fileType: fileType,
                    onProgress: { progress in
                        Task { @MainActor in
                            self.uploadProgress[clientSeq] = progress
                        }
                    }
                )
                let fileBody = FileMessageBody(fileID: finfo.fileID, url: finfo.url, name: finfo.name, size: finfo.size)
                let bodyData = try JSONEncoder().encode(fileBody)
                let bodyStr = String(data: bodyData, encoding: .utf8) ?? finfo.url

                await MainActor.run {
                    // Update placeholder with real body
                    if let idx = self.messages.firstIndex(where: { $0.clientSeq == clientSeq }) {
                        self.messages[idx].body = bodyStr
                    }
                }

                let ct = fileType == 0 ? 1 : 2
                let ack = try await self.msgService.sendMessage(convID: convID, body: bodyStr, clientSeq: clientSeq, contentType: ct, replyTo: 0)
                await MainActor.run {
                    if let idx = self.messages.firstIndex(where: { $0.clientSeq == ack.clientSeq && $0.msgID == 0 }) {
                        self.messages[idx].msgID = ack.msgID
                        self.messages[idx].timestamp = ack.timestamp
                        self.messages[idx].status = .sent
                        self.cache.insertMessage(self.messages[idx])
                    }
                    self.uploadProgress.removeValue(forKey: clientSeq)
                    self.isSending = false
                }
            } catch {
                logToFile("[ChatVM] sendFile error: \(error)")
                await MainActor.run {
                    if let idx = self.messages.firstIndex(where: { $0.clientSeq == clientSeq }) {
                        self.messages[idx].status = .failed
                    }
                    self.sendErrorMessage = loc("chat.send_failed")
                    self.uploadProgress.removeValue(forKey: clientSeq)
                    self.isSending = false
                }
            }
        }
    }

    public func sendImage(fileData: Data, fileName: String) {
        sendFile(fileData: fileData, fileName: fileName, fileType: 0)
    }

    public func retryMessage(clientSeq: Int64) {
        guard !isSending else { return }
        guard let idx = messages.firstIndex(where: { $0.clientSeq == clientSeq && $0.status == .failed }),
              idx < messages.count else { return }
        let failedMsg = messages[idx]
        messages[idx].status = .sending

        Task {
            do {
                let ack = try await msgService.sendMessage(convID: convID, body: failedMsg.body, clientSeq: clientSeq, replyTo: failedMsg.replyTo)
                if let updateIdx = messages.firstIndex(where: { $0.clientSeq == ack.clientSeq && $0.msgID == 0 }) {
                    messages[updateIdx].msgID = ack.msgID
                    messages[updateIdx].timestamp = ack.timestamp
                    messages[updateIdx].status = .sent
                    cache.insertMessage(messages[updateIdx])
                }
            } catch {
                logToFile("[ChatVM] retryMessage error: \(error)")
                if let updateIdx = messages.firstIndex(where: { $0.clientSeq == clientSeq }) {
                    messages[updateIdx].status = .failed
                }
                sendErrorMessage = loc("chat.send_failed")
            }
        }
    }

    // MARK: - Mark Read
    public func markAsReadIfActive() {
        guard let last = messages.last(where: { $0.senderID != AuthManager.shared.currentUser?.userID }) else { return }
        Task {
            try? await convService.markRead(convID: convID, msgID: last.msgID)
            NotificationCenter.default.post(name: .init("didMarkRead"), object: nil)
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

    // MARK: - Chat Items (with date grouping & bubble merging)
    public var chatItems: [ChatItem] {
        guard !messages.isEmpty else { return [] }
        var items: [ChatItem] = []
        let calendar = Calendar.current
        let currentUserID = AuthManager.shared.currentUser?.userID ?? ""

        for i in messages.indices {
            let msg = messages[i]
            let msgDate = Date(timeIntervalSince1970: Double(msg.timestamp) / 1000)

            // Date separator
            if i == 0 {
                items.append(.dateSeparator(msgDate))
            } else {
                let prevDate = Date(timeIntervalSince1970: Double(messages[i - 1].timestamp) / 1000)
                if !calendar.isDate(msgDate, inSameDayAs: prevDate) {
                    items.append(.dateSeparator(msgDate))
                }
            }

            // Group detection: same sender, within 5 minutes
            let isSameSender = msg.senderID == currentUserID
                ? (i > 0 && messages[i - 1].senderID == currentUserID)
                : (i > 0 && messages[i - 1].senderID == msg.senderID)
            let timeDiff: Double = i > 0 ? Double(msg.timestamp - messages[i - 1].timestamp) / 1000 : 999
            let inSameGroup = isSameSender && timeDiff < 300

            // Check if this message continues a group
            let isFirstInGroup = !inSameGroup

            // Check if next message continues the group
            let isNextSameSender: Bool
            let nextTimeDiff: Double
            if i + 1 < messages.count {
                let next = messages[i + 1]
                isNextSameSender = msg.senderID == currentUserID
                    ? next.senderID == currentUserID
                    : next.senderID == msg.senderID
                nextTimeDiff = Double(next.timestamp - msg.timestamp) / 1000
            } else {
                isNextSameSender = false
                nextTimeDiff = 999
            }
            let isLastInGroup = !(isNextSameSender && nextTimeDiff < 300)

            items.append(.message(msg, isFirstInGroup: isFirstInGroup, isLastInGroup: isLastInGroup))
        }
        return items
    }

    // MARK: - Helpers
    private func insertMessageInOrder(_ msg: Message) {
        // Binary search to find the correct insertion index, keeping messages sorted by timestamp then convSeq.
        // All pushed messages have msgID != 0, so no need to handle the "unsent local" edge case here.
        var lo = 0, hi = messages.count
        while lo < hi {
            let mid = (lo + hi) / 2
            let existing = messages[mid]
            if existing.timestamp < msg.timestamp || (existing.timestamp == msg.timestamp && existing.convSeq < msg.convSeq) {
                lo = mid + 1
            } else {
                hi = mid
            }
        }
        messages.insert(msg, at: lo)
    }

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
