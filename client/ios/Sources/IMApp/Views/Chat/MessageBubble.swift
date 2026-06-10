import SwiftUI
import IMCore
import MarkdownUI

struct MessageBubble: View {
    let message: Message
    let convType: ConvType
    let senderInfo: [String: User]
    var onRetry: (() -> Void)?
    var isFirstInGroup = true
    var isLastInGroup = true

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
                    if message.status == .failed {
                        Button {
                            onRetry?()
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: "exclamationmark.circle.fill")
                                    .font(.caption)
                                    .foregroundColor(.red)
                                Markdown( message.body)
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 8)
                                    .background(Color.red.opacity(0.08))
                                    .foregroundColor(.red)
                                    .clipShape(bubbleShape)
                                    .overlay(
                                        bubbleShape
                                            .stroke(Color.red.opacity(0.4), lineWidth: 1)
                                    )
                                Text(loc("chat.retry"))
                                    .font(.caption)
                                    .foregroundColor(.red)
                            }
                        }
                        .buttonStyle(.plain)
                    } else {
                        Markdown(message.body)
                            .markdownTextStyle {
                                ForegroundColor(isMine ? .white : .primary)
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                            .background(isMine ? Color.blue : Color(.systemGray5))
                            .clipShape(bubbleShape)
                            .textSelection(.enabled)
                            .contextMenu {
                                Button(loc("common.copy")) {
                                    UIPasteboard.general.string = message.body
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
                                    .font(.caption2)
                            }
                        }
                        .foregroundColor(message.status == .failed ? .red : .secondary)
                    }
                }
            }

            if !isMine { Spacer(minLength: 60) }
        }
        .padding(.vertical, 3)
    }

    private var bubbleShape: UnevenRoundedRectangle {
        let r: CGFloat = 14
        let flat: CGFloat = 4
        return UnevenRoundedRectangle(
            topLeadingRadius: isFirstInGroup ? r : flat,
            bottomLeadingRadius: isLastInGroup ? r : flat,
            bottomTrailingRadius: isLastInGroup ? r : flat,
            topTrailingRadius: isFirstInGroup ? r : flat
        )
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
