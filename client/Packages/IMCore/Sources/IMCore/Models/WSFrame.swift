import Foundation

// MARK: - Message Types
public enum MessageType: Int, Codable, Sendable {
    case msgSend = 1
    case msgSendAck = 2
    case msgPush = 11
    case msgReceived = 12
    case syncReq = 21
    case syncRes = 22
    case msgReadNotify = 32
    case sessionOnline = 41
    case sessionOffline = 42
    case sessionRecover = 43
    case sessionRecoverAck = 44
    case typing = 51
    case ping = 61
    case pong = 62
    case error = 71
}

// MARK: - Generic Frame
public struct WSFrame: Codable, Sendable {
    public let type: MessageType
    public let id: String
    public let payload: Data

    enum CodingKeys: String, CodingKey {
        case type, id, payload
    }

    public init(type: MessageType, id: String = "", payload: Data = Data()) {
        self.type = type
        self.id = id
        self.payload = payload
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        type = try container.decode(MessageType.self, forKey: .type)
        id = try container.decodeIfPresent(String.self, forKey: .id) ?? ""
        let payloadString = try container.decodeIfPresent(String.self, forKey: .payload) ?? "{}"
        guard let data = payloadString.data(using: .utf8) else {
            throw DecodingError.dataCorruptedError(forKey: .payload, in: container, debugDescription: "payload not a valid JSON string")
        }
        payload = data
    }

    public func toRawJSONData() throws -> Data {
        var dict: [String: Any] = [
            "type": type.rawValue,
            "id": id,
        ]
        if let payloadObj = try JSONSerialization.jsonObject(with: payload) as? [String: Any], !payloadObj.isEmpty {
            dict["payload"] = payloadObj
        } else {
            dict["payload"] = NSNull()
        }
        return try JSONSerialization.data(withJSONObject: dict)
    }
}

// MARK: - Payloads
public struct MsgSendPayload: Codable, Sendable {
    public let convID: String
    public let contentType: Int
    public let body: String
    public let replyTo: Int64
    public let mention: [String]
    public let clientSeq: Int64

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case contentType = "content_type"
        case body
        case replyTo = "reply_to"
        case mention
        case clientSeq = "client_seq"
    }

    public init(convID: String, contentType: Int = 0, body: String,
                replyTo: Int64 = 0, mention: [String] = [], clientSeq: Int64) {
        self.convID = convID
        self.contentType = contentType
        self.body = body
        self.replyTo = replyTo
        self.mention = mention
        self.clientSeq = clientSeq
    }
}

public struct MsgSendAckPayload: Codable, Sendable {
    public let msgID: Int64
    public let timestamp: Int64
    public let clientSeq: Int64
    public let status: Int

    enum CodingKeys: String, CodingKey {
        case msgID = "msg_id"
        case timestamp
        case clientSeq = "client_seq"
        case status
    }
}

public struct MsgPushPayload: Codable, Sendable {
    public let msgID: Int64
    public let convID: String
    public let senderID: String
    public let contentType: Int
    public let body: String
    public let replyTo: Int64?
    public let mention: [String]?
    public let timestamp: Int64
    public let convSeq: Int64

    enum CodingKeys: String, CodingKey {
        case msgID = "msg_id"
        case convID = "conv_id"
        case senderID = "sender_id"
        case contentType = "content_type"
        case body
        case replyTo = "reply_to"
        case mention
        case timestamp
        case convSeq = "conv_seq"
    }
}

public struct SyncReqPayload: Codable, Sendable {
    public let convID: String
    public let lastConvSeq: Int64
    public let limit: Int

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case lastConvSeq = "last_conv_seq"
        case limit
    }

    public init(convID: String, lastConvSeq: Int64, limit: Int = 50) {
        self.convID = convID
        self.lastConvSeq = lastConvSeq
        self.limit = limit
    }
}

public struct SyncResPayload: Codable, Sendable {
    public let convID: String
    public let messages: [SyncResMessage]
    public let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case messages
        case hasMore = "has_more"
    }
}

public struct SyncResMessage: Codable, Sendable, Identifiable {
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

public struct MsgReadNotifyPayload: Codable, Sendable {
    public let convID: String
    public let userID: String
    public let msgID: Int64
    public let timestamp: Int64

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case userID = "user_id"
        case msgID = "msg_id"
        case timestamp
    }
}

public struct SessionEventPayload: Codable, Sendable {
    public let userID: String
    public let sessionID: String
    public let device: Int

    enum CodingKeys: String, CodingKey {
        case userID = "user_id"
        case sessionID = "session_id"
        case device
    }

    public init(userID: String, sessionID: String, device: Int = 1) {
        self.userID = userID
        self.sessionID = sessionID
        self.device = device
    }
}

public struct SessionRecoverPayload: Codable, Sendable {
    public let sessionID: String

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
    }

    public init(sessionID: String) {
        self.sessionID = sessionID
    }
}

public struct SessionRecoverAckPayload: Codable, Sendable {
    public let sessionID: String
    public let userID: String
    public let timestamp: Int64

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case userID = "user_id"
        case timestamp
    }
}

public struct TypingPayload: Codable, Sendable {
    public let convID: String
    public let userID: String

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case userID = "user_id"
    }
}

public struct WSErrorPayload: Codable, Sendable {
    public let code: Int
    public let message: String
}
