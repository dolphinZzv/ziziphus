import Foundation

/// Simple JSON-file backed cache for conversations.
/// Avoids CoreData complexity in SPM by using Codable + file storage.
public class ConversationCache: @unchecked Sendable {
    public static let shared = ConversationCache()

    private var cache: [String: ConvListItem] = [:]
    private let queue = DispatchQueue(label: "com.im.convcache")
    private let fileURL: URL

    private init() {
        let dir = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first!
        fileURL = dir.appendingPathComponent("im_conversations.json")
        _ = loadFromDisk()
    }

    // MARK: - Read
    public func getAllConversations() -> [ConvListItem] {
        queue.sync {
            cache.values.sorted { $0.lastMsgAt > $1.lastMsgAt }
        }
    }

    public func getConversation(convID: String) -> ConvListItem? {
        queue.sync { cache[convID] }
    }

    // MARK: - Write
    public func upsertConversations(_ conversations: [ConvListItem]) {
        queue.async(flags: .barrier) { [weak self] in
            guard let self else { return }
            for conv in conversations {
                self.cache[conv.convID] = conv
            }
            self.saveToDisk()
        }
    }

    public func updateUnreadCount(convID: String, count: Int) {
        queue.async(flags: .barrier) { [weak self] in
            guard let self, var existing = self.cache[convID] else { return }
            existing = ConvListItem(
                convID: existing.convID,
                type: existing.type,
                name: existing.name,
                avatar: existing.avatar,
                unreadCount: count,
                lastMessage: existing.lastMessage,
                lastMsgAt: existing.lastMsgAt,
                role: existing.role,
                mute: existing.mute,
                mentionMe: existing.mentionMe
            )
            self.cache[convID] = existing
            self.saveToDisk()
        }
    }

    public func deleteAll() {
        queue.async(flags: .barrier) { [weak self] in
            self?.cache.removeAll()
            self?.saveToDisk()
        }
    }

    // MARK: - Persistence
    private func saveToDisk() {
        guard let data = try? JSONEncoder().encode(cache) else { return }
        try? data.write(to: fileURL, options: .atomic)
    }

    private func loadFromDisk() -> [String: ConvListItem] {
        guard let data = try? Data(contentsOf: fileURL),
              let items = try? JSONDecoder().decode([String: ConvListItem].self, from: data) else {
            return [:]
        }
        cache = items
        return items
    }
}
