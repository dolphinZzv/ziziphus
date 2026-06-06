import Foundation

/// Placeholder — CoreData requires .xcdatamodeld resource.
/// ConversationCache and MessageCache use JSON file backing.
public class CoreDataStack: @unchecked Sendable {
    public static let shared = CoreDataStack()
    private init() {}
}
