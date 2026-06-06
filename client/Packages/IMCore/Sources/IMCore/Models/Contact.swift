import Foundation

public struct Contact: Codable, Sendable, Identifiable, Hashable {
    public let userID: String
    public var nickname: String
    public var name: String
    public var avatar: String
    public var status: UserStatus
    public var addedAt: Int64

    public var id: String { userID }

    enum CodingKeys: String, CodingKey {
        case userID = "user_id"
        case nickname, name, avatar, status
        case addedAt = "added_at"
    }

    public init(userID: String, nickname: String = "", name: String = "",
                avatar: String = "", status: UserStatus = .offline, addedAt: Int64 = 0) {
        self.userID = userID
        self.nickname = nickname
        self.name = name
        self.avatar = avatar
        self.status = status
        self.addedAt = addedAt
    }
}
