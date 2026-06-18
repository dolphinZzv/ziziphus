import SwiftUI
import IMCore
import Textual

struct AgentTimelineView: View {
    let timeline: AgentTimelineBody
    let isMine: Bool

    @State private var expandedEntries: Set<String> = []
    @State private var imageViewerImages: [URL] = []
    @State private var imageViewerIndex: Int = 0

    private var serverBaseURL: URL? { URL(string: AppSettings.shared.serverURL) }

    private var textColor: Color { isMine ? .primary : .ink }
    private var mutedColor: Color { .secondary }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            headerView
            if !timeline.entries.isEmpty {
                Divider()
                    .padding(.vertical, 6)
            }
            ForEach(timeline.entries) { entry in
                entryRow(entry)
            }
        }
        .textual.textSelection(.enabled)
        .textual.imageAttachmentLoader(.tappableImage(relativeTo: serverBaseURL))
        .fullScreenCover(isPresented: Binding(
            get: { !imageViewerImages.isEmpty },
            set: { if !$0 { imageViewerImages = [] } }
        )) {
            ImageViewer(images: imageViewerImages, initialIndex: imageViewerIndex)
        }
        .onReceive(NotificationCenter.default.publisher(for: .textualImageTapped)) { notif in
            guard let url = notif.object as? URL else { return }
            let resolved = URL(string: url.absoluteString, relativeTo: serverBaseURL)?.absoluteURL ?? url
            let urls = timeline.entries.flatMap { $0.content.extractImageURLs(baseURL: serverBaseURL) }
            if urls.isEmpty {
                imageViewerImages = [resolved]
                imageViewerIndex = 0
            } else {
                imageViewerImages = urls
                imageViewerIndex = urls.firstIndex(of: resolved) ?? 0
            }
        }
    }

    // MARK: - Header

    private var headerView: some View {
        HStack(spacing: 6) {
            headerIcon
            Text(timeline.title ?? loc("agent.preview"))
                .font(.subheadline)
                .fontWeight(.medium)
                .foregroundColor(textColor)
            Spacer()
            statusBadge
        }
    }

    @ViewBuilder
    private var headerIcon: some View {
        switch timeline.status {
        case "running":
            ProgressView()
                .scaleEffect(0.65)
                .frame(width: 14, height: 14)
        case "completed":
            Image(systemName: "checkmark.circle.fill")
                .font(.caption2)
                .foregroundColor(.green)
        case "error":
            Image(systemName: "xmark.circle.fill")
                .font(.caption2)
                .foregroundColor(.red)
        default:
            EmptyView()
        }
    }

    @ViewBuilder
    private var statusBadge: some View {
        let (label, color): (String, Color) = {
            switch timeline.status {
            case "running":   return (loc("agent.running"), .orange)
            case "completed": return (loc("agent.completed"), .green)
            case "error":     return (loc("agent.error"), .red)
            default:          return ("", .secondary)
            }
        }()
        Text(label)
            .font(.system(size: 10))
            .foregroundColor(color)
            .padding(.horizontal, 5)
            .padding(.vertical, 2)
            .background(color.opacity(0.12))
            .clipShape(Capsule())
    }

    // MARK: - Entry Row

    @ViewBuilder
    private func entryRow(_ entry: AgentTimelineBody.Entry) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            entryHeader(entry)
            if expandedEntries.contains(entry.id) {
                entryContent(entry)
                    .padding(.top, 4)
            }
        }
        .padding(.vertical, 4)
        .padding(.horizontal, 2)
    }

    @ViewBuilder
    private func entryHeader(_ entry: AgentTimelineBody.Entry) -> some View {
        switch entry.type {
        case .thinking:
            collapsibleRow(
                id: entry.id,
                icon: "brain",
                iconColor: .purple,
                label: loc("agent.thinking"),
                labelItalic: true
            )
        case .toolCall:
            collapsibleRow(
                id: entry.id,
                icon: "wrench",
                iconColor: .orange,
                label: entry.toolName ?? loc("agent.tool_call"),
                status: entry.status
            )
        case .toolResult:
            collapsibleRow(
                id: entry.id,
                icon: "doc.text",
                iconColor: entry.status == "error" ? .red : .green,
                label: entry.toolName.map { "\(loc("agent.tool_result")): \($0)" } ?? loc("agent.tool_result"),
                status: entry.status
            )
        case .response:
            InlineText(markdown:entry.content, baseURL: serverBaseURL)
                .foregroundColor(textColor)
                .font(.body)
        }
    }

    private func collapsibleRow(id: String, icon: String, iconColor: Color, label: String, labelItalic: Bool = false, status: String? = nil) -> some View {
        let expanded = expandedEntries.contains(id)
        return Button {
            withAnimation(.easeInOut(duration: 0.15)) {
                if expanded { expandedEntries.remove(id) }
                else { expandedEntries.insert(id) }
            }
        } label: {
            HStack(spacing: 5) {
                Image(systemName: expanded ? "\(icon).fill" : icon)
                    .font(.caption2)
                    .foregroundColor(iconColor)
                    .frame(width: 14)
                Text(label)
                    .font(.caption)
                    .foregroundColor(mutedColor)
                    .italic(labelItalic)
                    .lineLimit(1)
                if let status {
                    Circle()
                        .fill(status == "error" ? Color.red : Color.green)
                        .frame(width: 4, height: 4)
                }
                Spacer()
                Image(systemName: expanded ? "chevron.up" : "chevron.down")
                    .font(.system(size: 8, weight: .medium))
                    .foregroundColor(.secondary.opacity(0.5))
            }
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private func entryContent(_ entry: AgentTimelineBody.Entry) -> some View {
        switch entry.type {
        case .thinking:
            InlineText(markdown:entry.content, baseURL: serverBaseURL)
                .font(.caption)
                .foregroundColor(mutedColor)
                .padding(.leading, 19)

        case .toolCall:
            if let input = entry.toolInput, !input.isEmpty {
                InlineText(markdown:"```json\n\(input)\n```", baseURL: serverBaseURL)
                    .font(.caption2)
                    .padding(.leading, 19)
            }

        case .toolResult:
            InlineText(markdown:entry.content, baseURL: serverBaseURL)
                .font(.caption)
                .foregroundColor(mutedColor)
                .padding(.leading, 19)

        case .response:
            EmptyView()
        }
    }
}
