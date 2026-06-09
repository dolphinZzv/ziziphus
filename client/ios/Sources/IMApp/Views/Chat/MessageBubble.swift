import SwiftUI
import IMCore

struct MessageBubble: View {
    let message: Message
    let convType: ConvType
    let senderInfo: [String: User]

    private var isMine: Bool {
        message.senderID == AuthManager.shared.currentUser?.userID
    }

    private var senderDisplayName: String {
        if message.senderID == AuthManager.shared.currentUser?.userID {
            return AuthManager.shared.currentUser?.name ?? message.senderID
        }
        return senderInfo[message.senderID]?.name ?? message.senderID
    }

    var body: some View {
        HStack {
            if isMine { Spacer(minLength: 60) }

            VStack(alignment: isMine ? .trailing : .leading, spacing: 2) {
                if convType == .group {
                    Text(senderDisplayName)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }

                VStack(alignment: isMine ? .leading : .trailing, spacing: 2) {
                    Text(message.body)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .background(isMine ? Color.blue : Color(.systemGray5))
                        .foregroundColor(isMine ? .white : .primary)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                        .textSelection(.enabled)
                        .contextMenu {
                            Button(loc("common.copy")) {
                                UIPasteboard.general.string = message.body
                            }
                        }

                    if message.timestamp > 0 {
                        HStack(spacing: 3) {
                            if isMine {
                                Image(systemName: statusIconName)
                                    .font(.system(size: 9))
                            }
                            Text(formatTime(message.timestamp))
                                .font(.caption2)
                        }
                        .foregroundColor(.secondary)
                    }
                }
            }

            if !isMine { Spacer(minLength: 60) }
        }
        .padding(.vertical, 3)
    }

    private var statusIconName: String {
        switch message.status {
        case .sending: return "clock"
        case .sent, .delivered: return "checkmark"
        case .read: return "checkmark.circle.fill"
        }
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: date)
    }
}
