import Foundation

public enum ConvRole: Int, Codable, Sendable {
    case member = 0
    case admin = 1
    case owner = 2
}

public struct ConvMember: Codable, Sendable, Identifiable, Hashable {
    public let convID: String
    public let userID: String
    public var role: ConvRole
    public var nickname: String?
    public var mute: Bool
    public var joinedAt: Int64

    public var id: String { "\(convID):\(userID)" }

    enum CodingKeys: String, CodingKey {
        case convID = "conv_id"
        case userID = "user_id"
        case role, nickname, mute
        case joinedAt = "joined_at"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        convID = try container.decode(String.self, forKey: .convID)
        userID = try container.decode(String.self, forKey: .userID)
        role = try container.decodeIfPresent(ConvRole.self, forKey: .role) ?? .member
        nickname = try container.decodeIfPresent(String.self, forKey: .nickname)
        mute = try container.decodeIfPresent(Bool.self, forKey: .mute) ?? false
        joinedAt = try container.decodeIfPresent(Int64.self, forKey: .joinedAt) ?? 0
    }

    public init(convID: String, userID: String, role: ConvRole = .member, nickname: String? = nil, mute: Bool = false, joinedAt: Int64 = 0) {
        self.convID = convID
        self.userID = userID
        self.role = role
        self.nickname = nickname
        self.mute = mute
        self.joinedAt = joinedAt
    }
}
