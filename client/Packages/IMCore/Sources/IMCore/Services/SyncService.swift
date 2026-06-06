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
        for conv in conversations {
            let lastSeq = cache.getLastConvSeq(convID: conv.convID)
            try await syncConversation(convID: conv.convID, lastConvSeq: lastSeq)
        }
    }

    /// Sync a single conversation
    public func syncConversation(convID: String, lastConvSeq: Int64) async throws {
        var seq = lastConvSeq
        var hasMore = true
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
            cache.insertMessages(messages)
            seq = messages.map(\.convSeq).max() ?? seq
            hasMore = res.hasMore
        }
    }
}
