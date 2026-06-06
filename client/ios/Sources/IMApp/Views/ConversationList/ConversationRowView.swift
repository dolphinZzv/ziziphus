import SwiftUI
import IMCore

struct ConversationRowView: View {
    let conv: ConvListItem

    var body: some View {
        HStack(spacing: 12) {
            // Avatar
            Circle()
                .fill(Color.blue.opacity(0.2))
                .frame(width: 48, height: 48)
                .overlay {
                    Text(String(conv.name.prefix(1)))
                        .fontWeight(.semibold)
                        .foregroundColor(.blue)
                }

            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(conv.name)
                        .fontWeight(.medium)
                        .lineLimit(1)

                    Spacer()

                    if conv.lastMsgAt > 0 {
                        Text(formatTime(conv.lastMsgAt))
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }

                HStack {
                    if let last = conv.lastMessage {
                        Text(last.body)
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                    }

                    Spacer()

                    if conv.unreadCount > 0 {
                        Text("\(conv.unreadCount)")
                            .font(.caption2)
                            .foregroundColor(.white)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.red)
                            .clipShape(Capsule())
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        if Calendar.current.isDateInToday(date) {
            formatter.dateFormat = "HH:mm"
        } else {
            formatter.dateFormat = "MM/dd"
        }
        return formatter.string(from: date)
    }
}
