import SwiftUI
import IMCore

struct MessageBubble: View {
    let message: Message

    private var isMine: Bool {
        message.senderID == AuthManager.shared.currentUser?.userID
    }

    var body: some View {
        HStack {
            if isMine { Spacer(minLength: 60) }

            VStack(alignment: isMine ? .trailing : .leading, spacing: 2) {
                if !isMine, message.contentType == .text {
                    Text(message.senderID)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }

                HStack(alignment: .bottom, spacing: 6) {
                    if isMine {
                        statusIcon
                    }

                    Text(message.body)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 6)
                        .background(isMine ? Color.blue : Color(.displayP3, red: 0.9, green: 0.9, blue: 0.92))
                        .foregroundColor(isMine ? .white : .primary)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                }

                if message.timestamp > 0 {
                    Text(formatTime(message.timestamp))
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }

            if !isMine { Spacer(minLength: 60) }
        }
        .padding(.vertical, 3)
    }

    @ViewBuilder
    private var statusIcon: some View {
        switch message.status {
        case .sending:
            Image(systemName: "clock")
                .font(.caption2)
                .foregroundColor(.gray)
        case .sent, .delivered:
            Image(systemName: "checkmark")
                .font(.caption2)
                .foregroundColor(.gray)
        case .read:
            Image(systemName: "checkmark.circle.fill")
                .font(.caption2)
                .foregroundColor(.blue)
        }
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: date)
    }
}
