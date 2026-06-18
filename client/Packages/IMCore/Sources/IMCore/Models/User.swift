import Foundation

public enum UserType: Int, Codable, Sendable {
    case human = 0
    case agent = 1
}

public enum UserStatus: Int, Codable, Sendable {
    case offline = 0
    case online = 1
    case busy = 2
}

public struct User: Codable, Sendable, Identifiable, Hashable {
    public let userID: String
    public let account: String
    public var name: String
    public var avatar: String
    public var type: UserType
    public var status: UserStatus
    public var uid: String
    public var primaryColor: String
    public var secondaryColor: String
    public var wakeMode: Int
    public var apiKey: String
    public var createdAt: Int64?

    public var id: String { userID }

    enum CodingKeys: String, CodingKey {
        case userID = "user_id"
        case account
        case name, avatar, type, status
        case uid
        case primaryColor = "primary_color"
        case secondaryColor = "secondary_color"
        case wakeMode = "wake_mode"
        case apiKey = "api_key"
        case createdAt = "created_at"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        userID = try container.decode(String.self, forKey: .userID)
        account = try container.decodeIfPresent(String.self, forKey: .account) ?? ""
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        avatar = try container.decodeIfPresent(String.self, forKey: .avatar) ?? ""
        type = try container.decodeIfPresent(UserType.self, forKey: .type) ?? .human
        status = try container.decodeIfPresent(UserStatus.self, forKey: .status) ?? .offline
        uid = try container.decodeIfPresent(String.self, forKey: .uid) ?? ""
        primaryColor = try container.decodeIfPresent(String.self, forKey: .primaryColor) ?? ""
        secondaryColor = try container.decodeIfPresent(String.self, forKey: .secondaryColor) ?? ""
        wakeMode = try container.decodeIfPresent(Int.self, forKey: .wakeMode) ?? 0
        apiKey = try container.decodeIfPresent(String.self, forKey: .apiKey) ?? ""
        createdAt = try container.decodeIfPresent(Int64.self, forKey: .createdAt)
    }

    public init(userID: String, account: String = "", name: String, avatar: String = "",
                type: UserType = .human, status: UserStatus = .offline,
                uid: String = "", primaryColor: String = "", secondaryColor: String = "",
                wakeMode: Int = 0, apiKey: String = "", createdAt: Int64? = nil) {
        self.userID = userID
        self.account = account
        self.name = name
        self.avatar = avatar
        self.type = type
        self.status = status
        self.uid = uid
        self.primaryColor = primaryColor
        self.secondaryColor = secondaryColor
        self.wakeMode = wakeMode
        self.apiKey = apiKey
        self.createdAt = createdAt
    }
}
