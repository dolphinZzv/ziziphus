import Foundation

/// Simple JSON-file backed cache for messages.
public class MessageCache: @unchecked Sendable {
    public static let shared = MessageCache()

    private var cache: [String: [Message]] = [:] // convID -> messages
    private let queue = DispatchQueue(label: "com.im.msgcache")
    private let fileURL: URL

    private init() {
        let dir = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first!
        fileURL = dir.appendingPathComponent("im_messages.json")
        _ = loadFromDisk()
    }

    // MARK: - Read
    public func getMessages(convID: String, limit: Int = 50, before msgID: Int64? = nil) -> [Message] {
        queue.sync {
            guard let messages = cache[convID] else { return [] }
            let sorted = messages.sorted { $0.convSeq < $1.convSeq }
            if let beforeID = msgID {
                let filtered = sorted.filter { $0.msgID < beforeID }
                return Array(filtered.suffix(limit))
            }
            return Array(sorted.suffix(limit))
        }
    }

    public func getLastConvSeq(convID: String) -> Int64 {
        queue.sync {
            guard let messages = cache[convID] else { return 0 }
            return messages.map(\.convSeq).max() ?? 0
        }
    }

    // MARK: - Write
    public func insertMessage(_ message: Message) {
        queue.async(flags: .barrier) { [weak self] in
            guard let self else { return }
            var messages = self.cache[message.convID] ?? []
            if !messages.contains(where: { $0.msgID == message.msgID && $0.msgID > 0 }) {
                messages.append(message)
                self.cache[message.convID] = messages
                self.saveToDisk()
            }
        }
    }

    public func insertMessages(_ messages: [Message]) {
        queue.async(flags: .barrier) { [weak self] in
            guard let self else { return }
            for msg in messages {
                var existing = self.cache[msg.convID] ?? []
                if !existing.contains(where: { $0.msgID == msg.msgID && $0.msgID > 0 }) {
                    existing.append(msg)
                    self.cache[msg.convID] = existing
                }
            }
            self.saveToDisk()
        }
    }

    public func updateMessageStatus(msgID: Int64, status: MsgStatus) {
        queue.async(flags: .barrier) { [weak self] in
            guard let self else { return }
            for (convID, messages) in self.cache {
                if let idx = messages.firstIndex(where: { $0.msgID == msgID }) {
                    var updated = messages[idx]
                    updated.status = status
                    var newMessages = messages
                    newMessages[idx] = updated
                    self.cache[convID] = newMessages
                    self.saveToDisk()
                    return
                }
            }
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

    private func loadFromDisk() -> [String: [Message]] {
        guard let data = try? Data(contentsOf: fileURL),
              let items = try? JSONDecoder().decode([String: [Message]].self, from: data) else {
            return [:]
        }
        return items
    }
}
