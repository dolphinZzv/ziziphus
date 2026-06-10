import Foundation

public struct DeviceSession: Codable, Sendable, Identifiable, Hashable {
    public let sessionID: String
    public let userID: String
    public let device: Int
    public let deviceName: String
    public let deviceID: String?
    public let clientIP: String?
    public let connID: String?
    public let status: Int
    public let loginAt: Int64
    public let lastActive: Int64

    public var id: String { sessionID }

    public var isOnline: Bool { connID != nil && !connID!.isEmpty && status == 0 }

    public var deviceDisplayName: String {
        switch device {
        case 0: return "iPhone"
        case 1: return "Mac"
        case 3: return "iPad"
        default: return "Unknown"
        }
    }

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case userID = "user_id"
        case device, deviceName = "device_name"
        case deviceID = "device_id"
        case clientIP = "client_ip"
        case connID = "conn_id"
        case status
        case loginAt = "login_at"
        case lastActive = "last_active"
    }
}
