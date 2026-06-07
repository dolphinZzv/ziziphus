import SwiftUI
import IMCore

struct ConversationRowView: View {
    @EnvironmentObject private var localizationManager: LocalizationManager

    let conv: ConvListItem

    private var avatarColor: Color {
        let colors: [Color] = [.blue, .green, .orange, .purple, .pink, .teal, .indigo, .mint]
        let hash = abs(conv.convID.hashValue)
        return colors[hash % colors.count]
    }

    var body: some View {
        HStack(spacing: 12) {
            // Avatar
            if conv.type == .group {
                Circle()
                    .fill(avatarColor.opacity(0.2))
                    .frame(width: 48, height: 48)
                    .overlay {
                        Image(systemName: "person.3.fill")
                            .font(.caption)
                            .foregroundColor(avatarColor)
                    }
            } else {
                Circle()
                    .fill(avatarColor.opacity(0.2))
                    .frame(width: 48, height: 48)
                    .overlay {
                        Text(String(conv.name.prefix(1)))
                            .font(.title3)
                            .fontWeight(.semibold)
                            .foregroundColor(avatarColor)
                    }
            }

            VStack(alignment: .leading, spacing: 3) {
                HStack {
                    HStack(spacing: 6) {
                        Text(conv.name)
                            .fontWeight(.medium)
                            .lineLimit(1)

                        if conv.mute {
                            Image(systemName: "bell.slash.fill")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                    }

                    Spacer()

                    if conv.lastMsgAt > 0 {
                        Text(formatTime(conv.lastMsgAt))
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }

                HStack(spacing: 4) {
                    if let last = conv.lastMessage {
                        if conv.type == .group {
                            Text(last.senderID)
                                .font(.caption2)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                            Text(":")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                        Text(last.body)
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                    } else {
                        Text(loc("chat.no_messages"))
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    Spacer()

                    if conv.mentionMe {
                        Text(loc("conv.mention"))
                            .font(.caption2)
                            .foregroundColor(.white)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.orange)
                            .clipShape(Capsule())
                    } else if conv.unreadCount > 0 {
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
        .padding(.vertical, 6)
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
