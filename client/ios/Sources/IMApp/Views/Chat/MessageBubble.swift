import SwiftUI
import IMCore
import Textual

struct MessageBubble: View {
    let message: Message
    let convType: ConvType
    let senderInfo: [String: User]
    var onRetry: (() -> Void)?
    var onReply: (() -> Void)?
    var onFormAction: ((String, Int64) -> Void)?
    var onTapSender: (() -> Void)?
    var repliedMessage: Message?
    var isFirstInGroup = true
    var isLastInGroup = true
    var uploadProgress: Double?
    var isHighlighted = false

    private var serverBaseURL: URL? { URL(string: AppSettings.shared.serverURL) }

    @State private var imageViewerImages: [URL] = []
    @State private var imageViewerIndex: Int = 0

    private var isMine: Bool {
        message.senderID == AuthManager.shared.currentUser?.userID
    }

    private var senderUser: User? {
        senderInfo[message.senderID]
    }

    private var senderDisplayName: String {
        if message.senderID == AuthManager.shared.currentUser?.userID {
            return AuthManager.shared.currentUser?.name ?? message.senderID
        }
        return senderUser?.name ?? message.senderID
    }

    private var showAvatar: Bool {
        convType == .group && !isMine && isFirstInGroup
    }

    private var fileBody: FileMessageBody? {
        guard let data = message.body.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(FileMessageBody.self, from: data)
    }

    private var agentTimelineBody: AgentTimelineBody? {
        guard let data = message.body.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(AgentTimelineBody.self, from: data)
    }

    private var formBody: FormDefinitionBody? {
        guard let data = message.body.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(FormDefinitionBody.self, from: data)
    }

    private var formResponseBody: FormResponseBody? {
        guard let data = message.body.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(FormResponseBody.self, from: data)
    }

    var body: some View {
        HStack(alignment: .top, spacing: 6) {
            if isMine { Spacer(minLength: 40) }

            VStack(alignment: isMine ? .trailing : .leading, spacing: 2) {
                if showAvatar || (convType == .group && isFirstInGroup) {
                    HStack(spacing: 6) {
                        if showAvatar {
                            AvatarView(
                                name: senderDisplayName,
                                url: senderUser?.avatar ?? "",
                                size: 28,
                                primaryColor: senderUser?.primaryColor ?? "",
                                secondaryColor: senderUser?.secondaryColor ?? ""
                            )
                            .onTapGesture { onTapSender?() }
                        }
                        if convType == .group && isFirstInGroup && !isMine {
                            Button(action: { onTapSender?() }) {
                                Text(senderDisplayName)
                                    .font(.caption2)
                                    .foregroundColor(.blue)
                            }
                        } else if convType == .group && isFirstInGroup && isMine {
                            Text(senderDisplayName)
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                    }
                }

                VStack(alignment: isMine ? .leading : .trailing, spacing: 2) {
                    if message.status == .failed {
                        failedBubble
                    } else if message.status == .sending, let uploadProgress {
                        uploadingBubble(progress: uploadProgress)
                    } else if message.contentType == .image {
                        imageBubble
                    } else if message.contentType == .file {
                        fileBubble
                    } else if message.contentType == .agentTimeline {
                        agentTimelineBubble
                    } else if message.contentType == .form {
                        formBubble
                    } else if message.contentType == .formResponse {
                        formResponseBubble
                    } else {
                        textBubble
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

            if !isMine { Spacer() }
        }
        .padding(.vertical, 3)
        .background(
            RoundedRectangle(cornerRadius: 10)
                .fill(isHighlighted ? Color.yellow.opacity(0.3) : .clear)
                .animation(.easeOut(duration: 0.8), value: isHighlighted)
        )
        .environment(\.openURL, OpenURLAction { url in
            // only handle non-image URLs now; image taps come via notification
            guard url.absoluteString.isImageURL else {
                UIApplication.shared.open(url)
                return .handled
            }
            return .handled
        })
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
            let urls = message.body.extractImageURLs(baseURL: serverBaseURL)
            if urls.isEmpty {
                imageViewerImages = [resolved]
                imageViewerIndex = 0
            } else {
                imageViewerImages = urls
                imageViewerIndex = urls.firstIndex(of: resolved) ?? 0
            }
        }
    }

    private var textBubble: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let repliedMessage {
                replyQuoteView(for: repliedMessage)
            }
            InlineText(markdown: message.body, baseURL: serverBaseURL)
                .foregroundColor(isMine ? Color.primary : Color.ink)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(isMine ? bubbleMine : bubbleOther)
        .clipShape(bubbleShape)
        .textSelection(.enabled)
        .textual.textSelection(.enabled)
        .contextMenu {
            Button(loc("common.copy")) {
                UIPasteboard.general.string = message.body
            }
            Button(loc("chat.reply")) {
                onReply?()
            }
        }
    }

    private var imageBubble: some View {
        VStack(spacing: 0) {
            if let fileBody {
                if let repliedMessage {
                    replyQuoteView(for: repliedMessage)
                        .padding(.horizontal, 8)
                        .padding(.top, 4)
                }
                let imageURL = URL(string: "\(AppSettings.shared.serverURL)\(fileBody.url)")
                ImageDownloadView(url: imageURL)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                    .padding(4)
                    .background(isMine ? bubbleMine : bubbleOther)
                    .clipShape(bubbleShape)

                Text(fileBody.name)
                    .font(.caption2)
                    .foregroundColor(isMine ? .primary : .secondary)
                    .padding(.horizontal, 8)
                    .padding(.bottom, 4)
            } else {
                textBubble
            }
        }
        .contextMenu {
            Button(loc("common.copy")) {
                UIPasteboard.general.string = message.body
            }
            Button(loc("chat.reply")) {
                onReply?()
            }
        }
    }

    private var fileBubble: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let repliedMessage {
                replyQuoteView(for: repliedMessage)
            }
            HStack(spacing: 8) {
                Image(systemName: "doc.fill")
                    .font(.title2)
                    .foregroundColor(isMine ? .primary : .blue)
                VStack(alignment: .leading, spacing: 2) {
                    Text(fileBody?.name ?? loc("common.unknown_file"))
                        .font(.body)
                        .foregroundColor(isMine ? .primary : Color.ink)
                        .lineLimit(2)
                    if let size = fileBody?.size {
                        Text(formatFileSize(size))
                            .font(.caption2)
                            .foregroundColor(isMine ? .primary.opacity(0.7) : .secondary)
                    }
                }
            }
            // PDF inline preview
            if let body = fileBody, body.name.lowercased().hasSuffix(".pdf"),
               let url = URL(string: AppSettings.shared.serverURL + body.url) {
                PDFPreviewView(url: url, filename: body.name)
                    .padding(.top, 4)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(isMine ? bubbleMine : bubbleOther)
        .clipShape(bubbleShape)
        .contextMenu {
            if let url = fileBody.flatMap({ URL(string: AppSettings.shared.serverURL + $0.url) }) {
                Button(loc("common.download")) {
                    UIApplication.shared.open(url)
                }
            }
            Button(loc("common.copy")) {
                UIPasteboard.general.string = message.body
            }
            Button(loc("chat.reply")) {
                onReply?()
            }
        }
    }

    // MARK: - Form Bubble
    private var formBubble: some View {
        Group {
            if let form = formBody {
                FormBubbleView(
                    form: form,
                    msgID: message.msgID,
                    convID: message.convID,
                    isMine: isMine,
                    onAction: { action in
                        onFormAction?(action, message.msgID)
                    }
                )
            } else {
                textBubble
            }
        }
    }

    // MARK: - FormResponse Bubble
    private var formResponseBubble: some View {
        Group {
            if let resp = formResponseBody {
                FormResponseBubbleView(form: resp)
            } else {
                textBubble
            }
        }
    }

    // MARK: - Agent Timeline Bubble
    private var agentTimelineBubble: some View {
        Group {
            if let timeline = agentTimelineBody {
                AgentTimelineView(timeline: timeline, isMine: isMine, convID: message.convID)
            } else {
                textBubble
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(isMine ? bubbleMine : bubbleOther)
        .clipShape(bubbleShape)
    }

    private var failedBubble: some View {
        Button {
            onRetry?()
        } label: {
            HStack(spacing: 4) {
                Image(systemName: "exclamationmark.circle.fill")
                    .font(.caption)
                    .foregroundColor(.red)
                InlineText(markdown:message.body, baseURL: serverBaseURL)
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
    }

    // MARK: - Uploading Bubble
    private func uploadingBubble(progress: Double) -> some View {
        HStack(spacing: 10) {
            Image(systemName: "arrow.up.doc.fill")
                .font(.title2)
                .foregroundColor(isMine ? .primary : .blue)
            VStack(alignment: .leading, spacing: 4) {
                Text(message.body)
                    .font(.body)
                    .foregroundColor(isMine ? .primary : Color.ink)
                    .lineLimit(1)
                ProgressView(value: progress, total: 1.0)
                    .progressViewStyle(.linear)
                    .frame(width: 120)
                Text("\(Int(progress * 100))%")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(isMine ? bubbleMine : bubbleOther)
        .clipShape(bubbleShape)
    }

    // MARK: - Reply Quote
    private func replyQuoteView(for msg: Message) -> some View {
        let name: String = {
            if msg.senderID == AuthManager.shared.currentUser?.userID {
                return loc("chat.you")
            }
            return senderInfo[msg.senderID]?.name ?? msg.senderID
        }()
        return HStack(spacing: 6) {
            Rectangle()
                .fill(Color.blue)
                .frame(width: 3)
                .cornerRadius(1.5)
            VStack(alignment: .leading, spacing: 1) {
                Text(name)
                    .font(.caption2)
                    .foregroundColor(.blue)
                Text(replyPreviewBody(for: msg))
                    .font(.caption2)
                    .foregroundColor(.secondary)
                    .lineLimit(2)
            }
        }
    }

    private func replyPreviewBody(for msg: Message) -> String {
        if msg.contentType == .image || msg.contentType == .file,
           let data = msg.body.data(using: .utf8),
           let fileBody = try? JSONDecoder().decode(FileMessageBody.self, from: data) {
            return fileBody.name
        }
        if msg.contentType == .agentTimeline,
           let data = msg.body.data(using: .utf8),
           let timeline = try? JSONDecoder().decode(AgentTimelineBody.self, from: data) {
            return timeline.title ?? loc("agent.preview")
        }
        return msg.body
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

    private func formatFileSize(_ bytes: Int64) -> String {
        let formatter = ByteCountFormatter()
        formatter.countStyle = .file
        return formatter.string(fromByteCount: bytes)
    }

    // MARK: - Adaptive Colors

    private var bubbleMine: Color {
        let hex = AppSettings.shared.bubbleColorHex
        let h = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: h).scanHexInt64(&int)
        let r = Double((int >> 16) & 0xFF) / 255
        let g = Double((int >> 8) & 0xFF) / 255
        let b = Double(int & 0xFF) / 255
        return Color(UIColor { trait in
            trait.userInterfaceStyle == .dark
                ? UIColor(red: 0x2c / 255, green: 0x2c / 255, blue: 0x2e / 255, alpha: 1)
                : UIColor(red: r, green: g, blue: b, alpha: 1)
        })
    }

    private var bubbleOther: Color {
        Color(UIColor { trait in
            trait.userInterfaceStyle == .dark
                ? UIColor(red: 0x2c / 255, green: 0x2c / 255, blue: 0x2e / 255, alpha: 1)
                : UIColor(red: 0xf0 / 255, green: 0xf0 / 255, blue: 0xf5 / 255, alpha: 1)
        })
    }
}

extension Color {
    static let ink = Color(UIColor { trait in
        trait.userInterfaceStyle == .dark
            ? UIColor(red: 0xf5 / 255, green: 0xf5 / 255, blue: 0xf7 / 255, alpha: 1)
            : UIColor(red: 0x1d / 255, green: 0x1d / 255, blue: 0x1f / 255, alpha: 1)
    })
}

// MARK: - Image Download View with Progress

private struct ImageDownloadView: View {
    let url: URL?

    @State private var phase: DownloadPhase = .empty
    @State private var progress: Double = 0

    private enum DownloadPhase {
        case empty, loading, success(UIImage), failure
    }

    var body: some View {
        Group {
            switch phase {
            case .success(let img):
                Image(uiImage: img)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxWidth: 200, maxHeight: 250)
                    .onTapGesture(count: 2) {
                        if let url { UIApplication.shared.open(url) }
                    }
            case .failure:
                VStack(spacing: 4) {
                    Image(systemName: "photo.badge.exclamationmark")
                        .font(.title2)
                    Text(loc("common.load_failed"))
                        .font(.caption)
                }
                .foregroundColor(.secondary)
                .frame(width: 100, height: 100)
            case .loading, .empty:
                VStack(spacing: 6) {
                    ProgressView(value: progress, total: 1.0)
                        .progressViewStyle(.linear)
                        .frame(width: 80)
                    if progress > 0 {
                        Text("\(Int(progress * 100))%")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }
                .frame(width: 100, height: 100)
            }
        }
        .task { await load() }
    }

    private func load() async {
        guard let url else { phase = .failure; return }
        phase = .loading; progress = 0

        let delegate = DownloadProgressDelegate { p in
            Task { @MainActor in self.progress = p }
        }
        let session = URLSession(configuration: .default, delegate: delegate, delegateQueue: nil)
        defer { session.invalidateAndCancel() }

        do {
            let (tempURL, _) = try await session.download(from: url)
            guard let data = try? Data(contentsOf: tempURL),
                  let image = UIImage(data: data) else {
                await MainActor.run { phase = .failure }
                return
            }
            await MainActor.run { phase = .success(image) }
        } catch {
            await MainActor.run { phase = .failure }
        }
    }
}

private final class DownloadProgressDelegate: NSObject, URLSessionDownloadDelegate, @unchecked Sendable {
    let onProgress: (Double) -> Void

    init(onProgress: @escaping (Double) -> Void) {
        self.onProgress = onProgress
    }

    func urlSession(_ session: URLSession, downloadTask: URLSessionDownloadTask, didWriteData bytesWritten: Int64, totalBytesWritten: Int64, totalBytesExpectedToWrite: Int64) {
        guard totalBytesExpectedToWrite > 0 else { return }
        onProgress(Double(totalBytesWritten) / Double(totalBytesExpectedToWrite))
    }

    func urlSession(_ session: URLSession, downloadTask: URLSessionDownloadTask, didFinishDownloadingTo location: URL) {}
}
