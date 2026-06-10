import SwiftUI
import IMCore

struct ConversationRowView: View {
    let conv: ConvListItem

    var body: some View {
        HStack(spacing: 12) {
            // Avatar
            AvatarView(name: conv.name, url: conv.avatar, size: 48)

            VStack(alignment: .leading, spacing: 3) {
                HStack {
                    HStack(spacing: 6) {
                        Text(conv.type == .p2p ? String(format: loc("chat.session_title"), conv.name) : conv.name)
                            .fontWeight(.medium)
                            .lineLimit(1)

                        if conv.mute {
                            Image(systemName: "bell.slash.fill")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                    }

                    Spacer()

                    if let timestamp = conv.lastMessage?.timestamp, timestamp > 0 {
                        Text(formatTime(timestamp))
                    } else if conv.lastMsgAt > 0 {
                        Text(formatTime(conv.lastMsgAt))
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }

                HStack(spacing: 4) {
                    if let last = conv.lastMessage {
                        if conv.type == .group {
                            Text(last.senderName.isEmpty ? last.senderID : last.senderName)
                                .font(.caption2)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                            Text(":")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                        if last.contentType == 1, let url = last.imageFileURL {
                            LastImageThumbnailView(url: url)
                        } else {
                            Text(last.body)
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                        }
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

// MARK: - Last Image Thumbnail

private struct LastImageThumbnailView: View {
    let url: URL

    var body: some View {
        AsyncImage(url: url) { phase in
            switch phase {
            case .success(let img):
                img
                    .resizable()
                    .aspectRatio(contentMode: .fill)
                    .frame(width: 36, height: 36)
                    .clipShape(RoundedRectangle(cornerRadius: 6))
            default:
                Image(systemName: "photo")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .frame(width: 36, height: 36)
    }
}

private extension LastMessage {
    var imageFileURL: URL? {
        guard contentType == 1,
              let data = body.data(using: .utf8),
              let fileBody = try? JSONDecoder().decode(FileMessageBody.self, from: data) else {
            return nil
        }
        return URL(string: AppSettings.serverBaseURL() + fileBody.url)
    }
}
