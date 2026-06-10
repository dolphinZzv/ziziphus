import Foundation

/// Orchestrates post-reconnect sync across all conversations.
@MainActor
public class SyncService {
    public static let shared = SyncService()
    private let msgService = MessageService.shared
    private let cache = MessageCache.shared
    private let convCache = ConversationCache.shared

    private init() {}

    /// Sync all cached conversations
    public func performFullSync() async throws {
        let conversations = convCache.getAllConversations()
        var errors: [Error] = []
        for conv in conversations {
            do {
                let lastSeq = cache.getLastConvSeq(convID: conv.convID)
                try await syncConversation(convID: conv.convID, lastConvSeq: lastSeq)
            } catch {
                logToFile("[Sync] failed for conv \(conv.convID): \(error)")
                errors.append(error)
            }
        }
        if !errors.isEmpty {
            throw APIError.server(code: -1, message: "\(errors.count) conversation(s) failed to sync")
        }
    }

    /// Sync a single conversation
    public func syncConversation(convID: String, lastConvSeq: Int64) async throws {
        var seq = lastConvSeq
        var hasMore = true
        var emptyBatches = 0
        while hasMore {
            let res = try await msgService.syncMessages(convID: convID, lastConvSeq: seq, limit: 50)
            let messages = res.messages.map { msg in
                Message(
                    msgID: msg.msgID,
                    convID: convID,
                    senderID: msg.senderID,
                    contentType: ContentType(rawValue: msg.contentType) ?? .text,
                    body: msg.body,
                    timestamp: msg.timestamp,
                    convSeq: msg.convSeq,
                    status: .delivered
                )
            }
            guard !messages.isEmpty else {
                emptyBatches += 1
                if emptyBatches >= 3 {
                    break
                }
                // Still advance seq to avoid re-requesting the same range
                seq += 1
                hasMore = res.hasMore
                continue
            }
            emptyBatches = 0
            cache.insertMessages(messages)
            seq = messages.map(\.convSeq).max() ?? seq
            hasMore = res.hasMore
        }
    }
}
