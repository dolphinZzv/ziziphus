import SwiftUI
import IMCore

struct MessageBubble: View {
    let message: Message
    let convType: ConvType
    let senderInfo: [String: User]
    var onRetry: (() -> Void)?

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
                    if message.status == .failed {
                        Button {
                            onRetry?()
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: "exclamationmark.circle.fill")
                                    .font(.system(size: 11))
                                    .foregroundColor(.red)
                                Text(message.body)
                                    .font(.system(size: AppleDesign.Typography.bodySize))
                                    .foregroundColor(.red)
                                    .padding(.horizontal, 14)
                                    .padding(.vertical, 8)
                                    .background(.red.opacity(0.08))
                                    .clipShape(RoundedRectangle(cornerRadius: 18))
                                    .overlay(
                                        RoundedRectangle(cornerRadius: 18)
                                            .stroke(.red.opacity(0.4), lineWidth: 1)
                                    )
                                Text(loc("chat.retry"))
                                    .font(.system(size: 11))
                                    .foregroundColor(.red)
                            }
                        }
                        .buttonStyle(.plain)
                    } else {
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
                    }

                    if message.timestamp > 0 || message.status == .sending || message.status == .failed {
                        HStack(spacing: 3) {
                            if isMine {
                                Image(systemName: statusIconName)
                                    .font(.system(size: 9))
                            }
                            if message.timestamp > 0 {
                                Text(formatTime(message.timestamp))
                                    .font(.system(size: 11))
                            }
                        }
                        .foregroundColor(message.status == .failed ? .red : AppleDesign.Colors.inkMuted)
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
        case .failed: return "exclamationmark.circle.fill"
        }
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: date)
    }
}
