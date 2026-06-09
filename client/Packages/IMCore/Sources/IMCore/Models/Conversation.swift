import Foundation

public enum ConvType: Int, Codable, Sendable {
    case p2p = 1
    case group = 2
}

public struct Conversation: Codable, Sendable, Identifiable, Hashable {
    public let convID: String
    public var type: ConvType
    public var name: String
    public var ownerID: String
    public var avatar: String
    public var lastMsgAt: Int64
    public var createdAt: Int64

    public var id: String { convID }

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case type, name
        case ownerID = "owner_id"
        case avatar
        case lastMsgAt = "last_msg_at"
        case createdAt = "created_at"
    }

    public init(convID: String, type: ConvType = .group, name: String, ownerID: String = "",
                avatar: String = "", lastMsgAt: Int64 = 0, createdAt: Int64 = 0) {
        self.convID = convID
        self.type = type
        self.name = name
        self.ownerID = ownerID
        self.avatar = avatar
        self.lastMsgAt = lastMsgAt
        self.createdAt = createdAt
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        convID = try container.decode(String.self, forKey: .convID)
        type = try container.decodeIfPresent(ConvType.self, forKey: .type) ?? .group
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        ownerID = try container.decodeIfPresent(String.self, forKey: .ownerID) ?? ""
        avatar = try container.decodeIfPresent(String.self, forKey: .avatar) ?? ""
        lastMsgAt = try container.decodeIfPresent(Int64.self, forKey: .lastMsgAt) ?? 0
        createdAt = try container.decodeIfPresent(Int64.self, forKey: .createdAt) ?? 0
    }
}

public struct LastMessage: Codable, Sendable, Hashable {
    public var msgID: Int64
    public var senderID: String
    public var senderName: String
    public var body: String
    public var contentType: Int
    public var timestamp: Int64

    enum CodingKeys: String, CodingKey {
        case msgID = "msg_id"
        case senderID = "sender_id"
        case senderName = "sender_name"
        case body
        case contentType = "content_type"
        case timestamp
    }
}

public struct ConvListItem: Codable, Sendable, Identifiable, Hashable {
    public let convID: String
    public let type: ConvType
    public var name: String
    public var avatar: String
    public var unreadCount: Int
    public var lastMessage: LastMessage?
    public var lastMsgAt: Int64
    public var role: Int
    public var mute: Bool
    public var mentionMe: Bool

    public var id: String { convID }

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case type, name, avatar
        case unreadCount = "unread_count"
        case lastMessage = "last_message"
        case lastMsgAt = "last_msg_at"
        case role, mute
        case mentionMe = "mention_me"
    }

    public init(convID: String, type: ConvType, name: String, avatar: String = "",
                unreadCount: Int = 0, lastMessage: LastMessage? = nil, lastMsgAt: Int64 = 0,
                role: Int = 0, mute: Bool = false, mentionMe: Bool = false) {
        self.convID = convID
        self.type = type
        self.name = name
        self.avatar = avatar
        self.unreadCount = unreadCount
        self.lastMessage = lastMessage
        self.lastMsgAt = lastMsgAt
        self.role = role
        self.mute = mute
        self.mentionMe = mentionMe
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        convID = try container.decode(String.self, forKey: .convID)
        type = try container.decode(ConvType.self, forKey: .type)
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        avatar = try container.decodeIfPresent(String.self, forKey: .avatar) ?? ""
        unreadCount = try container.decodeIfPresent(Int.self, forKey: .unreadCount) ?? 0
        lastMessage = try container.decodeIfPresent(LastMessage.self, forKey: .lastMessage)
        lastMsgAt = try container.decodeIfPresent(Int64.self, forKey: .lastMsgAt) ?? 0
        role = try container.decodeIfPresent(Int.self, forKey: .role) ?? 0
        mute = try container.decodeIfPresent(Bool.self, forKey: .mute) ?? false
        mentionMe = try container.decodeIfPresent(Bool.self, forKey: .mentionMe) ?? false
    }
}

public struct ConversationDetail: Codable, Sendable {
    public let convID: String
    public let type: ConvType
    public var name: String
    public var ownerID: String
    public var avatar: String
    public var members: [ConvMember]
    public var unreadCount: Int

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case type, name
        case ownerID = "owner_id"
        case avatar
        case members
        case unreadCount = "unread_count"
    }
}

public struct JoinRequest: Codable, Sendable, Identifiable, Hashable {
    public let convID: String
    public let userID: String
    public var status: JoinRequestStatus
    public var createdAt: Int64
    public var updatedAt: Int64

    public var id: String { "\(convID):\(userID)" }

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case userID = "user_id"
        case status
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

public struct GroupSearchItem: Codable, Sendable, Identifiable, Hashable {
    public let convID: String
    public let name: String
    public let avatar: String
    public let ownerID: String
    public let memberCount: Int
    public let createdAt: Int64

    public var id: String { convID }

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case name, avatar
        case ownerID = "owner_id"
        case memberCount = "member_count"
        case createdAt = "created_at"
    }
}

public enum JoinRequestStatus: Int, Codable, Sendable {
    case pending = 0
    case approved = 1
    case rejected = 2
}
