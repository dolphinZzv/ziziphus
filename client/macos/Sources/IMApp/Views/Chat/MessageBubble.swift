import SwiftUI
import IMCore
import Textual
import UniformTypeIdentifiers

struct MessageBubble: View {
    let message: Message
    let convType: ConvType
    let senderInfo: [String: User]
    var onRetry: (() -> Void)?
    var onReply: (() -> Void)?
    var onFormAction: ((String, Int64) -> Void)?
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

    private var senderDisplayName: String {
        if message.senderID == AuthManager.shared.currentUser?.userID {
            return AuthManager.shared.currentUser?.name ?? message.senderID
        }
        return senderInfo[message.senderID]?.name ?? message.senderID
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
        HStack {
            if isMine { Spacer(minLength: 40) }

            VStack(alignment: isMine ? .trailing : .leading, spacing: 3) {
                if convType == .group && isFirstInGroup {
                    Text(senderDisplayName)
                        .font(.system(size: AppleDesign.Typography.finePrintSize))
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                }

                VStack(alignment: isMine ? .leading : .trailing, spacing: 3) {
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
                                    .font(.system(size: 11))
                            }
                        }
                        .foregroundColor(message.status == .failed ? .red : AppleDesign.Colors.inkMuted)
                    }
                }
            }

            if !isMine { Spacer() }
        }
        .padding(.vertical, 2)
        .background(
            RoundedRectangle(cornerRadius: 10)
                .fill(isHighlighted ? Color.yellow.opacity(0.3) : .clear)
                .animation(.easeOut(duration: 0.8), value: isHighlighted)
        )
        .environment(\.openURL, OpenURLAction { url in
            // only handle non-image URLs now; image taps come via notification
            guard url.absoluteString.isImageURL else {
                NSWorkspace.shared.open(url)
                return .handled
            }
            return .handled
        })
        .textual.imageAttachmentLoader(.tappableImage(relativeTo: serverBaseURL))
        .sheet(isPresented: Binding(
            get: { !imageViewerImages.isEmpty },
            set: { if !$0 { imageViewerImages = [] } }
        )) {
            ImageViewer(images: imageViewerImages, initialIndex: imageViewerIndex)
                .frame(minWidth: 600, minHeight: 400)
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

    // MARK: - Text Bubble
    private var textBubble: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let repliedMessage {
                replyQuoteView(for: repliedMessage)
            }
            InlineText(markdown: message.body, baseURL: serverBaseURL)
                .foregroundColor(isMine ? .primary : AppleDesign.Colors.ink)
        }
        .font(.system(size: AppleDesign.Typography.bodySize))
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
        .background(isMine ? bubbleMine : AppleDesign.Colors.chatGray)
        .clipShape(bubbleShape)
        .textSelection(.enabled)
        .textual.textSelection(.enabled)
        .contextMenu {
            Button(loc("common.copy")) {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(message.body, forType: .string)
            }
            Button(loc("chat.reply")) {
                onReply?()
            }
        }
    }

    // MARK: - Image Bubble
    private var imageBubble: some View {
        VStack(spacing: 0) {
            if let fileBody {
                if let repliedMessage {
                    replyQuoteView(for: repliedMessage)
                        .padding(.horizontal, 8)
                        .padding(.top, 4)
                }
                let base = AppSettings.shared.serverURL
                let imageURL = URL(string: "\(base)\(fileBody.url)")
                ImageDownloadView(url: imageURL)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                    .padding(4)
                    .background(isMine ? bubbleMine : AppleDesign.Colors.chatGray)
                    .clipShape(bubbleShape)
                    .contextMenu {
                        if let url = imageURL {
                            Button("保存到下载") {
                                saveImageToDownloads(from: url)
                            }
                        }
                        Button(loc("common.copy")) {
                            NSPasteboard.general.clearContents()
                            NSPasteboard.general.setString(message.body, forType: .string)
                        }
                        Button(loc("chat.reply")) {
                            onReply?()
                        }
                    }

                Text(fileBody.name)
                    .font(.caption)
                    .foregroundColor(isMine ? .primary : .secondary)
                    .padding(.horizontal, 8)
                    .padding(.bottom, 4)
            } else {
                textBubble
            }
        }
    }

    // MARK: - File Bubble
    private var fileBubble: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let repliedMessage {
                replyQuoteView(for: repliedMessage)
            }
            HStack(spacing: 10) {
                Image(systemName: "doc.fill")
                    .font(.title2)
                    .foregroundColor(isMine ? .primary : AppleDesign.Colors.actionBlue)
                VStack(alignment: .leading, spacing: 2) {
                    Text(fileBody?.name ?? loc("common.unknown_file"))
                        .font(.system(size: AppleDesign.Typography.bodySize))
                        .foregroundColor(isMine ? .primary : AppleDesign.Colors.ink)
                        .lineLimit(2)
                    if let size = fileBody?.size {
                        Text(formatFileSize(size))
                            .font(.caption)
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
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background(isMine ? bubbleMine : AppleDesign.Colors.chatGray)
        .clipShape(bubbleShape)
        .contextMenu {
            if let url = fileBody.flatMap({ URL(string: AppSettings.shared.serverURL + $0.url) }) {
                Button(loc("common.download")) {
                    NSWorkspace.shared.open(url)
                }
            }
            Button(loc("common.copy")) {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(message.body, forType: .string)
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
        .background(isMine ? bubbleMine : AppleDesign.Colors.chatGray)
        .clipShape(bubbleShape)
    }

    // MARK: - Failed Bubble
    private var failedBubble: some View {
        Button {
            onRetry?()
        } label: {
            HStack(spacing: 4) {
                Image(systemName: "exclamationmark.circle.fill")
                    .font(.system(size: 11))
                    .foregroundColor(.red)
                InlineText(markdown:message.body, baseURL: serverBaseURL)
                    .font(.system(size: AppleDesign.Typography.bodySize))
                    .foregroundColor(.red)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 8)
                    .background(.red.opacity(0.08))
                    .clipShape(bubbleShape)
                    .overlay(
                        bubbleShape
                            .stroke(.red.opacity(0.4), lineWidth: 1)
                    )
                Text(loc("chat.retry"))
                    .font(.system(size: 11))
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
                .foregroundColor(isMine ? .primary : AppleDesign.Colors.actionBlue)
            VStack(alignment: .leading, spacing: 4) {
                Text(message.body)
                    .font(.system(size: AppleDesign.Typography.bodySize))
                    .foregroundColor(isMine ? .primary : AppleDesign.Colors.ink)
                    .lineLimit(1)
                ProgressView(value: progress, total: 1.0)
                    .progressViewStyle(.linear)
                    .frame(width: 120)
                Text("\(Int(progress * 100))%")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background(isMine ? bubbleMine : AppleDesign.Colors.chatGray)
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
                    .font(.system(size: AppleDesign.Typography.finePrintSize))
                    .foregroundColor(.blue)
                Text(replyPreviewBody(for: msg))
                    .font(.system(size: AppleDesign.Typography.finePrintSize))
                    .foregroundColor(AppleDesign.Colors.inkMuted)
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

    private var bubbleMine: Color {
        Color(nsColor: NSColor(name: nil) { appearance in
            let isDark = appearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
            if isDark {
                return NSColor(red: 0x2c / 255, green: 0x2c / 255, blue: 0x2e / 255, alpha: 1)
            }
            let hex = AppSettings.shared.bubbleColorHex
            let h = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
            var int: UInt64 = 0
            Scanner(string: h).scanHexInt64(&int)
            return NSColor(
                srgbRed: Double((int >> 16) & 0xFF) / 255,
                green: Double((int >> 8) & 0xFF) / 255,
                blue: Double(int & 0xFF) / 255,
                alpha: 1
            )
        })
    }

    private func saveImageToDownloads(from url: URL) {
        let task = URLSession.shared.downloadTask(with: url) { tempURL, _, error in
            guard let tempURL, error == nil else { return }
            let downloads = FileManager.default.urls(for: .downloadsDirectory, in: .userDomainMask).first!
            let dest = downloads.appendingPathComponent(url.lastPathComponent)
            try? FileManager.default.removeItem(at: dest)
            do {
                try FileManager.default.moveItem(at: tempURL, to: dest)
                DispatchQueue.main.async {
                    NSWorkspace.shared.activateFileViewerSelecting([dest])
                }
            } catch {}
        }
        task.resume()
    }
}

// MARK: - Image Download View with Progress

private struct ImageDownloadView: View {
    let url: URL?

    @State private var phase: DownloadPhase = .empty
    @State private var progress: Double = 0

    private enum DownloadPhase {
        case empty, loading, success(NSImage), failure
    }

    var body: some View {
        Group {
            switch phase {
            case .success(let img):
                Image(nsImage: img)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxWidth: 240, maxHeight: 300)
                    .onTapGesture(count: 2) {
                        if let url { NSWorkspace.shared.open(url) }
                    }
            case .failure:
                VStack(spacing: 4) {
                    Image(systemName: "photo.badge.exclamationmark")
                        .font(.title2)
                    Text(loc("common.load_failed"))
                        .font(.caption)
                }
                .foregroundColor(.secondary)
                .frame(width: 120, height: 120)
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
                .frame(width: 120, height: 120)
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
                  let image = NSImage(data: data) else {
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
