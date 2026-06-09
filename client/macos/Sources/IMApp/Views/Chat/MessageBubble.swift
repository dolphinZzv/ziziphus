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

            VStack(alignment: isMine ? .trailing : .leading, spacing: 3) {
                if convType == .group {
                    Text(senderDisplayName)
                        .font(.system(size: AppleDesign.Typography.finePrintSize))
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                }

                VStack(alignment: isMine ? .leading : .trailing, spacing: 3) {
                    Text(message.body)
                        .font(.system(size: AppleDesign.Typography.bodySize))
                        .foregroundColor(isMine ? .white : AppleDesign.Colors.ink)
                        .padding(.horizontal, 14)
                        .padding(.vertical, 8)
                        .background(isMine ? AppleDesign.Colors.actionBlue : AppleDesign.Colors.chatGray)
                        .clipShape(RoundedRectangle(cornerRadius: 18))
                        .textSelection(.enabled)
                        .contextMenu {
                            Button(loc("common.copy")) {
                                NSPasteboard.general.clearContents()
                                NSPasteboard.general.setString(message.body, forType: .string)
                            }
                        }

                    if message.timestamp > 0 {
                        HStack(spacing: 3) {
                            if isMine {
                                Image(systemName: statusIconName)
                                    .font(.system(size: 9))
                            }
                            Text(formatTime(message.timestamp))
                                .font(.system(size: 11))
                        }
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    }
                }
            }

            if !isMine { Spacer(minLength: 60) }
        }
        .padding(.vertical, 2)
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
