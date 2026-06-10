import Foundation

public struct DeviceManager: @unchecked Sendable {
    public static let shared = DeviceManager()

    private let defaults = UserDefaults.standard
    private let deviceIDKey = "com.dolphinz.device_id"

    public var deviceID: String {
        if let stored = defaults.string(forKey: deviceIDKey) {
            return stored
        }
        let newID = "dev_" + UUID().uuidString.prefix(8).lowercased()
        defaults.set(newID, forKey: deviceIDKey)
        return newID
    }
}
