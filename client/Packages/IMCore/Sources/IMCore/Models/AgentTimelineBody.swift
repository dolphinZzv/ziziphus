import Foundation

public struct AgentTimelineBody: Codable, Sendable {
    public enum EntryType: String, Codable, Sendable {
        case thinking
        case toolCall
        case toolResult
        case response
    }

    public struct Entry: Codable, Sendable, Identifiable {
        public var id: String
        public var type: EntryType
        public var content: String
        public var toolName: String?
        public var toolInput: String?
        public var status: String?
        public var timestamp: Int64

        public init(id: String = UUID().uuidString, type: EntryType, content: String,
                    toolName: String? = nil, toolInput: String? = nil,
                    status: String? = nil, timestamp: Int64 = 0) {
            self.id = id
            self.type = type
            self.content = content
            self.toolName = toolName
            self.toolInput = toolInput
            self.status = status
            self.timestamp = timestamp
        }
    }

    public var title: String?
    public var entries: [Entry]
    public var status: String
    /// When > 0, append entries to the existing message with this msgID instead of creating a new one.
    public var parentMsgID: Int64

    public init(title: String? = nil, entries: [Entry], status: String = "running",
                parentMsgID: Int64 = 0) {
        self.title = title
        self.entries = entries
        self.status = status
        self.parentMsgID = parentMsgID
    }
}
