import SwiftUI
import Combine
import IMCore

struct ConversationRowView: View {
    @EnvironmentObject private var localizationManager: LocalizationManager

    let conv: ConvListItem
    private let timer = Timer.publish(every: 30, on: .main, in: .common).autoconnect()
    @State private var now = Date()

    var body: some View {
        HStack(spacing: AppleDesign.Spacing.xs) {
            // Avatar
            AvatarView(name: conv.name, url: conv.avatar, size: 36)
                .overlay(alignment: .bottomTrailing) {
                    if conv.type == .p2p && conv.partnerType == 1 {
                        Image(systemName: "cpu.fill")
                            .font(.system(size: 8))
                            .foregroundColor(.white)
                            .padding(2)
                            .background(Color.purple)
                            .clipShape(Circle())
                            .overlay(Circle().stroke(Color(.controlBackgroundColor), lineWidth: 1.5))
                            .offset(x: 2, y: 2)
                    }
                }

            VStack(alignment: .leading, spacing: 2) {
                HStack {
                    Text(conv.type == .system ? "系统消息" : conv.name)
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
                        if last.contentType == 1, let url = last.imageFileURL {
                            LastImageThumbnailView(url: url)
                        } else if last.contentType == 9 {
                            Text(agentPreviewBody(last.body))
                                .font(.system(size: AppleDesign.Typography.captionSize))
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                                .lineLimit(1)
                        } else if last.contentType == 10 {
                            Text(formPreviewBody(last.body))
                                .font(.system(size: AppleDesign.Typography.captionSize))
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                                .lineLimit(1)
                        } else if last.contentType == 11 {
                            Text(formResponsePreviewBody(last.body))
                                .font(.system(size: AppleDesign.Typography.captionSize))
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                                .lineLimit(1)
                        } else {
                            Text(last.body)
                                .font(.system(size: AppleDesign.Typography.captionSize))
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                                .lineLimit(1)
                        }
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
        .onReceive(timer) { _ in now = Date() }
    }

    private func agentPreviewBody(_ body: String) -> String {
        guard let data = body.data(using: .utf8),
              let timeline = try? JSONDecoder().decode(AgentTimelineBody.self, from: data) else {
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
               let parentTimeline = try? JSONDecoder().decode(AgentTimelineBody.self, from: parentData),
               let title = parentTimeline.title {
                return title
            }
        }
        return loc("agent.preview")
    }

    private func formPreviewBody(_ body: String) -> String {
        guard let data = body.data(using: .utf8),
              let form = try? JSONDecoder().decode(FormDefinitionBody.self, from: data) else { return "[表单]" }
        if form.type == "contact_request", let name = form.fromUserName {
            return "好友申请 · \(name)"
        }
        return form.title
    }

    private func formResponsePreviewBody(_ body: String) -> String {
        guard let data = body.data(using: .utf8),
              let resp = try? JSONDecoder().decode(FormResponseBody.self, from: data) else { return "[回复]" }
        let name = resp.responderName
        return resp.action == "approve" ? "已通过 · \(name)" : "已拒绝 · \(name)"
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let diff = now.timeIntervalSince(date)
        if diff < 60 {
            return "刚刚"
        }
        if diff < 3600 {
            return "\(Int(diff / 60))分钟前"
        }
        if Calendar.current.isDateInToday(date) {
            let f = DateFormatter()
            f.dateFormat = "HH:mm"
            return f.string(from: date)
        }
        if Calendar.current.isDateInYesterday(date) {
            let f = DateFormatter()
            f.dateFormat = "'昨天' HH:mm"
            return f.string(from: date)
        }
        let f = DateFormatter()
        f.dateFormat = "MM/dd"
        return f.string(from: date)
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
