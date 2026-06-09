import Foundation

public enum ContentType: Int, Codable, Sendable {
    case text = 0
    case image = 1
    case file = 2
    case audio = 3
    case video = 4
    case system = 5
    case recall = 6
    case edit = 7
    case custom = 8
}

public enum MsgStatus: Int, Codable, Sendable {
    case sending = 0
    case sent = 1
    case delivered = 2
    case read = 3
    case failed = 4
}

public struct Message: Codable, Sendable, Identifiable, Hashable {
    public var msgID: Int64
    public var convID: String
    public var senderID: String
    public var senderSessionID: String
    public var contentType: ContentType
    public var body: String
    public var replyTo: Int64
    public var mention: [String]
    public var timestamp: Int64
    public var clientSeq: Int64
    public var convSeq: Int64
    public var status: MsgStatus

    public var id: Int64 { msgID }

    enum CodingKeys: String, CodingKey {
        case msgID = "msg_id"
        case convID = "conv_id"
        case senderID = "sender_id"
        case senderSessionID = "sender_session_id"
        case contentType = "content_type"
        case body
        case replyTo = "reply_to"
        case mention
        case timestamp
        case clientSeq = "client_seq"
        case convSeq = "conv_seq"
        case status
    }

    public init(msgID: Int64 = 0, convID: String, senderID: String,
                senderSessionID: String = "", contentType: ContentType = .text,
                body: String, replyTo: Int64 = 0, mention: [String] = [],
                timestamp: Int64 = 0, clientSeq: Int64 = 0,
                convSeq: Int64 = 0, status: MsgStatus = .sending) {
        self.msgID = msgID
        self.convID = convID
        self.senderID = senderID
        self.senderSessionID = senderSessionID
        self.contentType = contentType
        self.body = body
        self.replyTo = replyTo
        self.mention = mention
        self.timestamp = timestamp
        self.clientSeq = clientSeq
        self.convSeq = convSeq
        self.status = status
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        msgID = try container.decode(Int64.self, forKey: .msgID)
        convID = try container.decode(String.self, forKey: .convID)
        senderID = try container.decode(String.self, forKey: .senderID)
        senderSessionID = try container.decodeIfPresent(String.self, forKey: .senderSessionID) ?? ""
        contentType = try container.decodeIfPresent(ContentType.self, forKey: .contentType) ?? .text
        body = try container.decodeIfPresent(String.self, forKey: .body) ?? ""
        replyTo = try container.decodeIfPresent(Int64.self, forKey: .replyTo) ?? 0
        mention = try container.decodeIfPresent([String].self, forKey: .mention) ?? []
        timestamp = try container.decodeIfPresent(Int64.self, forKey: .timestamp) ?? 0
        clientSeq = try container.decodeIfPresent(Int64.self, forKey: .clientSeq) ?? 0
        convSeq = try container.decodeIfPresent(Int64.self, forKey: .convSeq) ?? 0
        status = try container.decodeIfPresent(MsgStatus.self, forKey: .status) ?? .sent
    }
}

public struct SyncMessage: Codable, Sendable, Identifiable {
    public let msgID: Int64
    public let senderID: String
    public let contentType: Int
    public let body: String
    public let timestamp: Int64
    public let convSeq: Int64

    public var id: Int64 { msgID }

    enum CodingKeys: String, CodingKey {
        case msgID = "msg_id"
        case senderID = "sender_id"
        case contentType = "content_type"
        case body, timestamp
        case convSeq = "conv_seq"
    }
}
