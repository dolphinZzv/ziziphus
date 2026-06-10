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
    public var primaryColor: String
    public var secondaryColor: String
    public var createdAt: Int64?

    public var id: String { userID }

    enum CodingKeys: String, CodingKey {
        case userID = "user_id"
        case account
        case name, avatar, type, status
        case primaryColor = "primary_color"
        case secondaryColor = "secondary_color"
        case createdAt = "created_at"
    }

    public init(userID: String, account: String = "", name: String, avatar: String = "",
                type: UserType = .human, status: UserStatus = .offline,
                primaryColor: String = "", secondaryColor: String = "",
                createdAt: Int64? = nil) {
        self.userID = userID
        self.account = account
        self.name = name
        self.avatar = avatar
        self.type = type
        self.status = status
        self.primaryColor = primaryColor
        self.secondaryColor = secondaryColor
        self.createdAt = createdAt
    }
}
