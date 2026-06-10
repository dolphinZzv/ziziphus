import SwiftUI
import IMCore

struct ConversationRowView: View {
    @EnvironmentObject private var localizationManager: LocalizationManager

    let conv: ConvListItem

    private var avatarColor: Color {
        let colors: [Color] = [AppleDesign.Colors.actionBlue, .green, .orange, .purple, .pink, .teal, .indigo, .mint]
        let hash = abs(conv.convID.hashValue)
        return colors[hash % colors.count]
    }

    var body: some View {
        HStack(spacing: AppleDesign.Spacing.xs) {
            // Avatar
            if conv.type == .group {
                Circle()
                    .fill(avatarColor.opacity(0.15))
                    .frame(width: 36, height: 36)
                    .overlay {
                        Image(systemName: "person.3.fill")
                            .font(.caption2)
                            .foregroundColor(avatarColor)
                    }
            } else {
                Circle()
                    .fill(avatarColor.opacity(0.15))
                    .frame(width: 36, height: 36)
                    .overlay {
                        Text(String(conv.name.prefix(1)))
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundColor(avatarColor)
                    }
            }

            VStack(alignment: .leading, spacing: 2) {
                HStack {
                    Text(conv.type == .p2p ? String(format: loc("chat.session_title"), conv.name) : conv.name)
                        .font(.system(size: AppleDesign.Typography.captionSize, weight: .semibold))
                        .foregroundColor(AppleDesign.Colors.ink)
                        .lineLimit(1)

                    if conv.mute {
                        Image(systemName: "bell.slash.fill")
                            .font(.caption2)
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }

                    Spacer()

                    if let timestamp = conv.lastMessage?.timestamp, timestamp > 0 {
                        Text(formatTime(timestamp))
                    } else if conv.lastMsgAt > 0 {
                        Text(formatTime(conv.lastMsgAt))
                            .font(.system(size: AppleDesign.Typography.finePrintSize))
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }
                }

                HStack(spacing: 4) {
                    if let last = conv.lastMessage {
                        if conv.type == .group {
                            Text(last.senderName.isEmpty ? last.senderID : last.senderName)
                                .font(.system(size: AppleDesign.Typography.captionSize))
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                                .lineLimit(1)
                            Text(":")
                                .font(.system(size: AppleDesign.Typography.captionSize))
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                        }
                        Text(last.body)
                            .font(.system(size: AppleDesign.Typography.captionSize))
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                            .lineLimit(1)
                    } else {
                        Text(loc("chat.no_messages"))
                            .font(.system(size: AppleDesign.Typography.captionSize))
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }

                    Spacer()

                    if conv.mentionMe {
                        Text(loc("conv.mention"))
                            .font(.system(size: AppleDesign.Typography.finePrintSize, weight: .semibold))
                            .foregroundColor(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2)
                            .background(Color.orange)
                            .clipShape(Capsule())
                    } else if conv.unreadCount > 0 {
                        Text("\(conv.unreadCount)")
                            .font(.system(size: AppleDesign.Typography.finePrintSize, weight: .semibold))
                            .foregroundColor(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2)
                            .background(AppleDesign.Colors.actionBlue)
                            .clipShape(Capsule())
                    }
                }
            }
        }
        .padding(.vertical, 8)
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
