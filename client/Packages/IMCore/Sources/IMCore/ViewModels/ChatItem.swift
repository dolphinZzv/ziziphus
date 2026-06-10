import Foundation

public enum ChatItem: Identifiable {
    case dateSeparator(Date)
    case message(Message, isFirstInGroup: Bool, isLastInGroup: Bool)

    public var id: String {
        switch self {
        case .dateSeparator(let date):
            return "sep_\(Int(date.timeIntervalSince1970 / 86400))"
        case .message(let msg, _, _):
            return "msg_\(msg.stableId)"
        }
    }

    public var message: Message? {
        if case .message(let msg, _, _) = self { return msg }
        return nil
    }
}
