import Foundation

public final class SessionService: @unchecked Sendable {
    public static let shared = SessionService()
    private let api = APIClient.shared

    private init() {}

    public func listSessions() async throws -> [DeviceSession] {
        try await api.request("/api/v1/sessions")
    }

    public func deleteSession(_ sessionID: String) async throws {
        let _: [String: String] = try await api.request(
            "/api/v1/sessions/\(sessionID)",
            method: .delete
        )
    }
}
