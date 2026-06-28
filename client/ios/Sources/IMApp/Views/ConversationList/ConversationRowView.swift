import SwiftUI
import IMCore

struct ConversationRowView: View {
    let conv: ConvListItem

    @State private var lastImageURL: URL?

    var body: some View {
        HStack(spacing: 12) {
            // Avatar
            AvatarView(name: conv.name, url: conv.avatar, size: 48)
                .overlay(alignment: .bottomTrailing) {
                    if conv.type == .p2p && conv.partnerType == 1 {
                        Image(systemName: "cpu.fill")
                            .font(.system(size: 10))
                            .foregroundColor(.white)
                            .padding(3)
                            .background(Color.purple)
                            .clipShape(Circle())
                            .overlay(Circle().stroke(Color(.systemBackground), lineWidth: 1.5))
                            .offset(x: 3, y: 3)
                    }
                }

            VStack(alignment: .leading, spacing: 3) {
                HStack {
                    HStack(spacing: 6) {
                        Text(conv.type == .system ? "系统消息" : conv.name)
                            .fontWeight(.semibold)
                            .lineLimit(1)

                        if conv.mute {
                            Image(systemName: "bell.slash.fill")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                    }

                    Spacer()

                    if let timestamp = conv.lastMessage?.timestamp, timestamp > 0 {
                        Text(DateFormatterCache.string(from: timestamp))
                            .font(.caption)
                            .foregroundColor(.secondary)
                    } else if conv.lastMsgAt > 0 {
                        Text(DateFormatterCache.string(from: conv.lastMsgAt))
                            .font(.caption)
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
                        if last.contentType == 1, let url = lastImageURL {
                            LastImageThumbnailView(url: url)
                        } else if last.contentType == 9 {
                            Text(agentPreviewBody(last.body))
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                        } else if last.contentType == 10 {
                            Text(formPreviewBody(last.body))
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                        } else if last.contentType == 11 {
                            Text(formResponsePreviewBody(last.body))
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
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
                        Text(conv.unreadCount > 99 ? "99+" : "\(conv.unreadCount)")
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
        .padding(.vertical, 10)
        .padding(.horizontal, 16)
        .overlay(alignment: .bottom) {
            Rectangle()
                .fill(Color(.separator))
                .frame(height: 0.5)
                .padding(.leading, 76)  // 16 (leading) + 48 (avatar) + 12 (spacing)
        }
        .task {
            lastImageURL = lastImageFileURL
        }
    }

    private var lastImageFileURL: URL? {
        guard let last = conv.lastMessage, last.contentType == 1,
              let data = last.body.data(using: .utf8),
              let fileBody = try? Self.jsonDecoder.decode(FileMessageBody.self, from: data) else {
            return nil
        }
        return URL(string: AppSettings.serverBaseURL() + fileBody.url)
    }

    private static let jsonDecoder = JSONDecoder()

    private func agentPreviewBody(_ body: String) -> String {
        guard let data = body.data(using: .utf8),
              let timeline = try? Self.jsonDecoder.decode(AgentTimelineBody.self, from: data) else {
            return loc("agent.preview")
        }
        if let title = timeline.title {
            return title
        }
        // Append message without title — try to resolve from parent
        if timeline.parentMsgID > 0 {
            let msgs = MessageCache.shared.getMessages(convID: conv.convID)
            if let parent = msgs.first(where: { $0.msgID == timeline.parentMsgID }),
               let parentData = parent.body.data(using: .utf8),
               let parentTimeline = try? Self.jsonDecoder.decode(AgentTimelineBody.self, from: parentData),
               let title = parentTimeline.title {
                return title
            }
        }
        return loc("agent.preview")
    }
}

// MARK: - DateFormatter cache

private enum DateFormatterCache {
    static func string(from timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        if Calendar.current.isDateInToday(date) {
            return todayFormatter.string(from: date)
        } else if Calendar.current.isDateInYesterday(date) {
            return yesterdayFormatter.string(from: date)
        } else {
            return otherFormatter.string(from: date)
        }
    }

    private static let todayFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "HH:mm"
        return f
    }()

    private static let yesterdayFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "'昨天' HH:mm"
        return f
    }()

    private static let otherFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "MM/dd"
        return f
    }()
}

// MARK: - Last Image Thumbnail

private struct LastImageThumbnailView: View {
    let url: URL

    var body: some View {
        CachedAsyncImage(url: url) { img in
            img
                .resizable()
                .aspectRatio(contentMode: .fill)
                .frame(width: 36, height: 36)
                .clipShape(RoundedRectangle(cornerRadius: 6))
        } placeholder: {
            Image(systemName: "photo")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .frame(width: 36, height: 36)
    }
}
