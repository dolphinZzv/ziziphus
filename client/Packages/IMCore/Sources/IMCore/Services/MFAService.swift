import Foundation

@MainActor
public class MFAService {
    public static let shared = MFAService()
    private let api = APIClient.shared

    private init() {}

    // MARK: - Get MFA Status
    public func getStatus() async throws -> MFAStatus {
        struct StatusResp: Codable, Sendable {
            let enabled: Bool
            let mfaType: Int?
            let maskedEmail: String?
            enum CodingKeys: String, CodingKey {
                case enabled
                case mfaType = "mfa_type"
                case maskedEmail = "masked_email"
            }
        }
        let resp: StatusResp = try await api.request("/api/v1/users/me/mfa")
        return MFAStatus(enabled: resp.enabled, mfaType: resp.mfaType ?? 0, maskedEmail: resp.maskedEmail ?? "")
    }

    // MARK: - Setup MFA
    public func setup(mfaType: Int) async throws -> MFASetupResult {
        struct SetupReq: Codable, Sendable {
            let mfaType: Int
            enum CodingKeys: String, CodingKey { case mfaType = "mfa_type" }
        }
        struct SetupResp: Codable, Sendable {
            let secret: String?
            let uri: String?
            let maskedEmail: String?
            enum CodingKeys: String, CodingKey {
                case secret, uri
                case maskedEmail = "masked_email"
            }
        }
        let resp: SetupResp = try await api.request("/api/v1/users/me/mfa/setup", method: .post, body: SetupReq(mfaType: mfaType))
        return MFASetupResult(secret: resp.secret ?? "", qrCodeURI: resp.uri ?? "", maskedEmail: resp.maskedEmail ?? "")
    }

    // MARK: - Verify & Enable MFA
    public func verify(code: String) async throws {
        struct VerifyReq: Codable, Sendable { let code: String }
        let _: [String: String] = try await api.request("/api/v1/users/me/mfa/verify", method: .post, body: VerifyReq(code: code))
    }

    // MARK: - Disable MFA
    public func disable() async throws {
        let _: [String: String] = try await api.request("/api/v1/users/me/mfa/disable", method: .post)
    }
}

// MARK: - Data types
public struct MFAStatus: Codable, Sendable {
    public let enabled: Bool
    public let mfaType: Int
    public let maskedEmail: String
}

public struct MFASetupResult: Codable, Sendable {
    public let secret: String
    public let qrCodeURI: String
    public let maskedEmail: String
}
