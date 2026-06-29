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
    private var senderInfoFetchTimes: [String: Date] = [:]

    private var typingTimer: Timer?
    private var lastTypingSend: TimeInterval = 0
    private let typingInterval: TimeInterval = 3
    private var cancellables = Set<AnyCancellable>()

    public init(convID: String) {
        self.convID = convID
        setupSubscriptions()
        $messages
            .sink { [weak self] newMsgs in
                self?.rebuildChatItems(from: newMsgs)
            }
            .store(in: &cancellables)
    }

    private func setupSubscriptions() {
        // Handle incoming pushes
        ws.on(.msgPush) { [weak self] frame in
            guard let self else { return }
            let msg = MessageService.shared.handlePush(frame: frame)
            if let msg, msg.convID == self.convID {
                Task { @MainActor in
                    logToFile("[ChatVM] push msg arrived: msgID=\(msg.msgID) body=\(msg.body.prefix(20))")
                    // agentTimeline with parentMsgID > 0: append entries to parent
                    if msg.contentType == .agentTimeline,
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
        let now = Date()
        let staleIDs = Set(messages.map(\.senderID))
            .subtracting([AuthManager.shared.currentUser?.userID ?? ""])
            .filter { id in
                guard let fetchTime = senderInfoFetchTimes[id] else { return true }
                return now.timeIntervalSince(fetchTime) > senderInfoTTL
            }
        guard !staleIDs.isEmpty else { return }
        Task {
            do {
                let info = try await contactService.batchGetUsers(userIDs: Array(staleIDs))
                for (id, user) in info {
                    senderInfo[id] = user
                    senderInfoFetchTimes[id] = now
                }
            } catch {
                logToFile("[ChatVM] loadSenderInfo error: \(error)")
            }
        }
    }

    public func loadMembers() {
        guard members.isEmpty else { return }
        Task {
            do {
                let detail = try await convService.getConversationDetail(convID: convID)
                members = detail.members.filter { $0.userID != AuthManager.shared.currentUser?.userID }
            } catch {
                logToFile("[ChatVM] loadMembers error: \(error)")
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

        let mentionIDs = extractMentionIDs(from: text)

        let clientSeq = AuthManager.shared.nextClientSeq()
        let localMsg = Message(
            convID: convID,
            senderID: AuthManager.shared.currentUser?.userID ?? "",
            body: text,
            replyTo: replyingToMsg?.msgID ?? 0,
            mention: mentionIDs,
            timestamp: Int64(Date().timeIntervalSince1970 * 1000),
            clientSeq: clientSeq,
            status: .sending
        )
        messages.append(localMsg)
        sortMessages()

        Task {
            do {
                let replyTo = replyingToMsg?.msgID ?? 0
                let ack = try await msgService.sendMessage(convID: convID, body: text, clientSeq: clientSeq, replyTo: replyTo, mention: mentionIDs)
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

    // MARK: - Send Agent Timeline
    /// - Returns: The server-assigned msgID for the first message (parentMsgID == 0),
    ///            or nil for append messages / errors.
    /// Sends a form response message (content_type=11).
    /// Called when the user clicks approve/reject on a form bubble.
    @discardableResult
    public func sendFormResponse(body: FormResponseBody, convID: String, replyTo: Int64) async -> Bool {
        let clientSeq = AuthManager.shared.nextClientSeq()
        guard let bodyData = try? JSONEncoder().encode(body),
              let bodyStr = String(data: bodyData, encoding: .utf8) else { return false }

        let localMsg = Message(
            convID: convID,
            senderID: AuthManager.shared.currentUser?.userID ?? "",
            contentType: .formResponse,
            body: bodyStr,
            replyTo: replyTo,
            timestamp: Int64(Date().timeIntervalSince1970 * 1000),
            clientSeq: clientSeq,
            status: .sending
        )
        messages.append(localMsg)
        sortMessages()

        do {
            let ack = try await msgService.sendMessage(convID: convID, body: bodyStr, clientSeq: clientSeq, contentType: 11, replyTo: replyTo)
            if let idx = messages.firstIndex(where: { $0.clientSeq == ack.clientSeq && $0.msgID == 0 }) {
                messages[idx].msgID = ack.msgID
                messages[idx].timestamp = ack.timestamp
                messages[idx].status = .sent
                cache.insertMessage(messages[idx])
            }
            return true
        } catch {
            if let idx = messages.firstIndex(where: { $0.clientSeq == clientSeq }) {
                messages[idx].status = .failed
            }
            return false
        }
    }

    @discardableResult
    public func sendAgentTimeline(body: AgentTimelineBody) async -> Int64? {
        let clientSeq = AuthManager.shared.nextClientSeq()
        guard let bodyData = try? JSONEncoder().encode(body),
              let bodyStr = String(data: bodyData, encoding: .utf8) else { return nil }

        // If appending to an existing message, merge locally before sending
        if body.parentMsgID > 0 {
            appendAgentTimelineEntries(body, toParent: body.parentMsgID)
        }

        let localMsg = Message(
            convID: convID,
            senderID: AuthManager.shared.currentUser?.userID ?? "",
            contentType: .agentTimeline,
            body: bodyStr,
            timestamp: Int64(Date().timeIntervalSince1970 * 1000),
            clientSeq: clientSeq,
            status: .sending
        )
        // Only add a new bubble if this is NOT an append
        if body.parentMsgID == 0 {
            messages.append(localMsg)
            sortMessages()
        }

        do {
            let ack = try await msgService.sendMessage(convID: convID, body: bodyStr, clientSeq: clientSeq, contentType: 9)
            if body.parentMsgID == 0 {
                if let idx = messages.firstIndex(where: { $0.clientSeq == ack.clientSeq && $0.msgID == 0 }) {
                    messages[idx].msgID = ack.msgID
                    messages[idx].timestamp = ack.timestamp
                    messages[idx].status = .sent
                    cache.insertMessage(messages[idx])
                }
                return ack.msgID
            }
            return nil
        } catch {
            logToFile("[ChatVM] sendAgentTimeline error: \(error)")
            if let idx = messages.firstIndex(where: { $0.clientSeq == clientSeq }) {
                messages[idx].status = .failed
            }
            sendErrorMessage = loc("chat.send_failed")
            return nil
        }
    }

    // MARK: - Append Agent Timeline Entries
    private func appendAgentTimelineEntries(_ timeline: AgentTimelineBody, toParent parentMsgID: Int64) {
        guard let idx = messages.firstIndex(where: { $0.msgID == parentMsgID }) else {
            logToFile("[ChatVM] appendAgentTimeline: parent msgID=\(parentMsgID) not found")
            return
        }
        var parent = messages[idx]
        guard var existing = try? JSONDecoder().decode(AgentTimelineBody.self, from: parent.body.data(using: .utf8) ?? Data()) else {
            logToFile("[ChatVM] appendAgentTimeline: failed to decode parent body")
            return
        }
        // Append new entries, deduplicating by id
        let existingIDs = Set(existing.entries.map(\.id))
        let newEntries = timeline.entries.filter { !existingIDs.contains($0.id) }
        existing.entries.append(contentsOf: newEntries)
        existing.status = timeline.status
        // Re-encode
        guard let updatedData = try? JSONEncoder().encode(existing),
              let updatedBody = String(data: updatedData, encoding: .utf8) else { return }
        parent.body = updatedBody
        messages[idx] = parent
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
    // Stored property rebuilt on messages change, not computed on every view evaluation
    @Published public var chatItems: [ChatItem] = []
    @Published public var chatVersion = 0

    /// Merge agent timeline append messages into their parent, removing standalone appends.
    private func mergeAgentTimelineAppends(_ msgs: [Message]) -> [Message] {
        var result: [Message] = []
        var appendEntries: [Int64: [AgentTimelineBody.Entry]] = [:]
        var appendStatus: [Int64: String] = [:]

        // Partition: normal messages go to result, appends are collected for merging
        for msg in msgs {
            guard msg.contentType == .agentTimeline,
                  let data = msg.body.data(using: .utf8),
                  let timeline = try? JSONDecoder().decode(AgentTimelineBody.self, from: data),
                  timeline.parentMsgID > 0
            else {
                result.append(msg)
                continue
            }
            appendEntries[timeline.parentMsgID, default: []].append(contentsOf: timeline.entries)
            if timeline.status == "completed" || timeline.status == "error" {
                appendStatus[timeline.parentMsgID] = timeline.status
            }
        }

        guard !appendEntries.isEmpty else { return msgs }

        // Merge collected entries into parent messages
        for i in result.indices {
            let msg = result[i]
            guard msg.msgID > 0,
                  let entries = appendEntries[msg.msgID],
                  var body = try? JSONDecoder().decode(AgentTimelineBody.self, from: (result[i].body.data(using: .utf8) ?? Data()))
            else { continue }

            let existingIDs = Set(body.entries.map(\.id))
            body.entries.append(contentsOf: entries.filter { !existingIDs.contains($0.id) })
            if let status = appendStatus[msg.msgID] {
                body.status = status
            }

            guard let data = try? JSONEncoder().encode(body),
                  let updatedBody = String(data: data, encoding: .utf8) else { continue }
            var updated = result[i]
            updated.body = updatedBody
            result[i] = updated
        }

        return result
    }

    private func rebuildChatItems(from msgs: [Message]) {
        let merged = mergeAgentTimelineAppends(msgs)
        guard !merged.isEmpty else {
            chatItems = []
            return
        }
        var items: [ChatItem] = []
        let calendar = Calendar.current
        let currentUserID = AuthManager.shared.currentUser?.userID ?? ""

        for i in merged.indices {
            let msg = merged[i]
            let msgDate = Date(timeIntervalSince1970: Double(msg.timestamp) / 1000)

            if i == 0 {
                items.append(.dateSeparator(msgDate))
            } else {
                let prevDate = Date(timeIntervalSince1970: Double(merged[i - 1].timestamp) / 1000)
                if !calendar.isDate(msgDate, inSameDayAs: prevDate) {
                    items.append(.dateSeparator(msgDate))
                }
            }

            // agentTimeline messages always stand alone — never merge with adjacent messages
            let prevIsAgentTimeline = i > 0 && merged[i - 1].contentType == .agentTimeline
            let isAgentTimeline = msg.contentType == .agentTimeline

            let isSameSender = msg.senderID == currentUserID
                ? (i > 0 && merged[i - 1].senderID == currentUserID)
                : (i > 0 && merged[i - 1].senderID == msg.senderID)
            let timeDiff: Double = i > 0 ? Double(msg.timestamp - merged[i - 1].timestamp) / 1000 : 999
            let inSameGroup = !isAgentTimeline && !prevIsAgentTimeline && isSameSender && timeDiff < 300
            let isFirstInGroup = !inSameGroup

            let isLastInGroup: Bool
            if i + 1 < merged.count {
                let next = merged[i + 1]
                let nextIsAgentTimeline = next.contentType == .agentTimeline
                let isNextSameSender = msg.senderID == currentUserID
                    ? next.senderID == currentUserID
                    : next.senderID == msg.senderID
                let nextTimeDiff = Double(next.timestamp - msg.timestamp) / 1000
                isLastInGroup = isAgentTimeline || nextIsAgentTimeline || !(isNextSameSender && nextTimeDiff < 300)
            } else {
                isLastInGroup = true
            }

            items.append(.message(msg, isFirstInGroup: isFirstInGroup, isLastInGroup: isLastInGroup))
        }
        chatItems = items
        chatVersion &+= 1
    }

    // MARK: - Helpers

    /// Parse `@name` patterns from message text and return matching member user IDs.
    private func extractMentionIDs(from text: String) -> [String] {
        let pattern = /@(\S+)/
        let matches = text.matches(of: pattern)
        var ids: [String] = []
        for match in matches {
            let name = String(match.1)
            // Match against member nicknames or sender names
            if let member = members.first(where: {
                ($0.nickname ?? "") == name || senderInfo[$0.userID]?.name == name
            }) {
                if !ids.contains(member.userID) {
                    ids.append(member.userID)
                }
            }
        }
        return ids
    }

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
