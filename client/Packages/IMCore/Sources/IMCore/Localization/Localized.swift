import Foundation

/// Global convenience function for localized strings.
/// Uses the shared `LocalizationManager` to look up the key.
/// - Parameters:
///   - key: The string catalog key.
///   - args: Optional format arguments.
/// - Returns: The localized string.
public func loc(_ key: String, _ args: CVarArg...) -> String {
    LocalizationManager.lookup(key, args)
}
