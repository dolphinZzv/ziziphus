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
    @Published public var members: [ConvMember] = []

    private let senderInfoTTL: TimeInterval = 5 * 60 // 5 minutes

    public init(convID: String, convName: String = "", convType: ConvType = .p2p) {
        self.convID = convID

        // Handle incoming messages
        ws.on(.msgPush) { [weak self] frame in
            guard let self, let msg = try? JSONDecoder().decode(Message.self, from: frame.payload) else { return }
            if msg.convID == self.convID {
                Task { @MainActor in
                    // Check if it's an agent timeline append (merge into parent)
                    if msg.contentType == 9,
                       let data = msg.body.data(using: .utf8),
                       let timeline = try? JSONDecoder().decode(AgentTimelineBody.self, from: data),
                       timeline.parentMsgID > 0 {
                        self.appendAgentTimelineEntries(timeline, toParent: timeline.parentMsgID)
                    } else {
                        self.insertMessageInOrder(msg)
                        self.cache.insertMessage(msg)
                    }
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

        loadInitialMessages()
    }

    deinit {
        typingTimer?.invalidate()
        logToFile("[ChatVM] deinit conv=\(convID)")
    }

    // MARK: - Message Loading
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
            // Mark as read AFTER messages are loaded, not before (race condition fix)
            self.markAsReadIfActive()
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
                }
                if msgs.count < 50 {
                    allHistoryLoaded = true
                }
            } catch {
                logToFile("[ChatVM] load more error: \(error)")
            }
            isLoadingHistory = false
        }
    }

    // MARK: - Sender Info
    public func loadSenderInfo() {
        let ids = Set(messages.filter { $0.senderID != AuthManager.shared.currentUser?.userID }.map(\.senderID))
        guard !ids.isEmpty else { return }
        Task {
            do {
                let users = try await userService.batchGet(userIDs: Array(ids))
                self.senderInfo = users
            } catch {
                logToFile("[ChatVM] load sender info error: \(error)")
            }
        }
    }

    public func loadMembers() {
        Task {
            do {
                self.members = try await convService.getMembers(convID: convID)
            } catch {
                logToFile("[ChatVM] load members error: \(error)")
            }
        }
    }

    public func loadContextAround(msgID: Int64) {
        Task {
            do {
                let msgs = try await msgService.getHistory(convID: convID, aroundMsgID: msgID, limit: 50)
                for msg in msgs {
                    cache.insertMessage(msg)
                }
                messages = cache.getMessages(convID: convID)
                sortMessages()
            } catch {
                logToFile("[ChatVM] load context error: \(error)")
            }
        }
    }

    // MARK: - Send

    public func sendTextMessage(_ text: String, replyTo: Int64 = 0, mention: [String] = []) {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        sendMessage(body: trimmed, contentType: .text, replyTo: replyTo, mention: mention)
    }

    public func sendFileMessage(fileData: Data, fileName: String, fileType: Int = 1) {
        Task {
            let uploadResult = try? await APIClient.shared.uploadFile(fileData: fileData, fileName: fileName, fileType: fileType)
            guard let finfo = uploadResult else { return }
            let fileBody = FileMessageBody(fileID: finfo.fileID, url: finfo.url, name: finfo.name, size: finfo.size)
            let bodyData = try JSONEncoder().encode(fileBody)
            let bodyStr = String(data: bodyData, encoding: .utf8) ?? finfo.url
            sendMessage(body: bodyStr, contentType: fileType == 0 ? .image : .file)
        }
    }

    public func sendImage(fileData: Data, fileName: String) {
        sendFileMessage(fileData: fileData, fileName: fileName, fileType: 0)
    }

    public func sendMessage(body: String, contentType: ContentType = .text, localMsgID: Int64 = 0, replyTo: Int64 = 0, mention: [String] = []) {
        let me = AuthManager.shared.currentUser
        let now = Int64(Date().timeIntervalSince1970 * 1000)
        let localID = localMsgID > 0 ? localMsgID : Int64(arc4random_uniform(UInt32.max))

        var finalBody = body
        if replyTo > 0,
           let replyMsg = messages.first(where: { $0.msgID == replyTo }) {
            finalBody = "[reply:\(replyTo)]\(finalBody)"
        }

        let msg = Message(
            msgID: -localID,
            convID: convID,
            senderID: me?.userID ?? "",
            senderName: me?.name ?? "",
            senderSessionID: "",
            contentType: contentType,
            body: finalBody,
            timestamp: now,
            clientSeq: 0,
            convSeq: 0,
            status: .sending
        )
        messages.append(msg)
        sortMessages()

        let payload = MessageSendPayload(
            convID: convID,
            body: finalBody,
            contentType: contentType,
            replyTo: replyTo,
            mention: mention,
            clientSeq: Int64(arc4random()),
            localMsgID: localID
        )
        guard let data = try? JSONEncoder().encode(payload) else { return }
        ws.send(frame: WSFrame(type: .msgSend, payload: data))
    }

    // MARK: - Mark Read
    public func markAsReadIfActive() {
        guard let last = messages.last(where: { $0.senderID != AuthManager.shared.currentUser?.userID }) else { return }
        Task {
            do {
                try await convService.markRead(convID: convID, msgID: last.msgID)
            } catch {
                logToFile("[ChatVM] markRead failed: \(error)")
            }
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
    // Stored property rebuilt on messages change, not computed on every view evaluation
    @Published public var chatItems: [ChatItem] = []
    @Published public var chatVersion = 0

    /// Merge agent timeline append messages into their parent, removing standalone appends.
    private func buildChatItems() -> [ChatItem] {
        var items: [ChatItem] = []
        var lastTimestamp: Int64 = 0
        var mergedParents: Set<Int64> = []

        for msg in messages {
            // Agent timeline append — merge into parent
            if msg.contentType == 9,
               let data = msg.body.data(using: .utf8),
               let timeline = try? JSONDecoder().decode(AgentTimelineBody.self, from: data),
               timeline.parentMsgID > 0 {
                mergedParents.insert(timeline.parentMsgID)
                continue
            }

            // Date separator
            if !Calendar.current.isDate(msg.timestampDate, inSameDayAs: Date(timeIntervalSince1970: Double(lastTimestamp) / 1000)) {
                items.append(.dateSeparator(msg.timestampDate))
            }
            lastTimestamp = msg.timestamp

            // Mark unread separator
            if msg.status == .sent && msg.convSeq > 0 {
                items.append(.unreadSeparator)
            }

            let isMergedAgent = mergedParents.contains(msg.msgID)
            items.append(.message(msg, isMergedAgent: isMergedAgent))
        }
        return items
    }

    public func rebuildChatItems() {
        chatItems = buildChatItems()
        chatVersion += 1
    }

    private func sortMessages() {
        messages.sort { $0.timestamp < $1.timestamp || ($0.timestamp == $1.timestamp && $0.msgID < $1.msgID) }
        while messages.count > maxMessages {
            messages.removeFirst()
        }
        rebuildChatItems()
    }

    private func insertMessageInOrder(_ msg: Message) {
        // Deduplicate
        if let idx = messages.firstIndex(where: { $0.msgID == msg.msgID || ($0.msgID < 0 && msg.msgID < 0 && $0.msgID == msg.msgID) }) {
            // Update existing local message with server data
            if msg.status == .delivered || msg.status == .read {
                messages[idx] = msg
            } else if msg.msgID > 0 && messages[idx].msgID < 0 {
                messages[idx] = msg
            }
        } else {
            messages.append(msg)
        }
        sortMessages()
    }

    // MARK: - Agent Timeline Append Handling
    public var appendTimeline: AgentTimelineBody? // temporarily stored incoming append
    private var typingTimer: Timer?
    private var lastTypingSend: TimeInterval = 0
    private let typingInterval: TimeInterval = 3.0

    private func appendAgentTimelineEntries(_ timeline: AgentTimelineBody, toParent parentMsgID: Int64) {
        guard let parentIdx = messages.firstIndex(where: { $0.msgID == parentMsgID }) else {
            // Parent not yet loaded; store for later
            appendTimeline = timeline
            return
        }
        let parent = messages[parentIdx]
        guard let parentData = parent.body.data(using: .utf8),
              var parentTimeline = try? JSONDecoder().decode(AgentTimelineBody.self, from: parentData) else { return }

        // Append new entries
        parentTimeline.entries.append(contentsOf: timeline.entries)
        if let newData = try? JSONEncoder().encode(parentTimeline),
           let newBody = String(data: newData, encoding: .utf8) {
            messages[parentIdx] = Message(
                msgID: parent.msgID,
                convID: parent.convID,
                senderID: parent.senderID,
                senderName: parent.senderName,
                senderSessionID: parent.senderSessionID,
                contentType: parent.contentType,
                body: newBody,
                timestamp: parent.timestamp,
                clientSeq: parent.clientSeq,
                convSeq: parent.convSeq,
                status: parent.status
            )
            cache.insertMessage(messages[parentIdx])
            rebuildChatItems()
        }
    }
}

// MARK: - Models

public struct MessageSendPayload: Codable {
    public let convID: String
    public let body: String
    public let contentType: ContentType
    public let replyTo: Int64
    public let mention: [String]
    public let clientSeq: Int64
    public let localMsgID: Int64

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case body
        case contentType = "content_type"
        case replyTo = "reply_to"
        case mention
        case clientSeq = "client_seq"
        case localMsgID = "local_msg_id"
    }
}

public struct FileMessageBody: Codable, Sendable {
    public let fileID: String
    public let url: String
    public let name: String
    public let size: Int64

    enum CodingKeys: String, CodingKey {
        case fileID = "file_id"
        case url
        case name
        case size
    }

    public init(fileID: String, url: String, name: String, size: Int64?) {
        self.fileID = fileID
        self.url = url
        self.name = name
        self.size = size ?? 0
    }
}

public struct AgentTimelineBody: Codable, Sendable {
    public let title: String?
    public let parentMsgID: Int64
    public var entries: [AgentStep]

    enum CodingKeys: String, CodingKey {
        case title
        case parentMsgID = "parent_msg_id"
        case entries
    }
}

public struct AgentStep: Codable, Sendable {
    public let type: String
    public let content: String
    public let title: String?
}

public enum ChatItem: Identifiable, Hashable {
    case message(Message, isMergedAgent: Bool)
    case dateSeparator(Date)
    case unreadSeparator

    public var id: String {
        switch self {
        case .message(let msg, _): return "msg_\(msg.msgID)"
        case .dateSeparator(let date): return "date_\(date.timeIntervalSince1970)"
        case .unreadSeparator: return "unread"
        }
    }

    public static func == (lhs: ChatItem, rhs: ChatItem) -> Bool {
        lhs.id == rhs.id
    }

    public func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}
