import SwiftUI
import IMCore

struct InputBarView: View {
    @Binding var text: String
    @EnvironmentObject private var localizationManager: LocalizationManager
    let onSend: () -> Void
    let onTyping: () -> Void
    let onPickImage: () -> Void
    let onPickFile: () -> Void
    var replyingToMsg: Message?
    var replyingToSender: String?
    var onCancelReply: (() -> Void)?

    var body: some View {
        VStack(spacing: 0) {
            // Reply preview bar
            if let replyingToMsg {
                HStack(spacing: 8) {
                    Rectangle()
                        .fill(Color.blue)
                        .frame(width: 3)
                        .cornerRadius(1.5)
                    VStack(alignment: .leading, spacing: 1) {
                        Text(String(format: loc("chat.replying"), replyingToSender ?? replyingToMsg.senderID))
                            .font(.system(size: AppleDesign.Typography.finePrintSize))
                            .foregroundColor(.blue)
                        Text(replyingToMsg.body)
                            .font(.system(size: AppleDesign.Typography.finePrintSize))
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                            .lineLimit(1)
                    }
                    Spacer()
                    Button(action: { onCancelReply?() }) {
                        Image(systemName: "xmark")
                            .font(.caption2)
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }
                    .buttonStyle(.plain)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 4)
                .background(AppleDesign.Colors.parchment)
            }

            HStack(spacing: 0) {
            ZStack(alignment: .bottomTrailing) {
                ChatTextView(
                    text: $text,
                    placeholder: loc("chat.placeholder"),
                    onTyping: onTyping,
                    onSend: onSend
                )
                .frame(minHeight: 40, maxHeight: 120)
                .padding(.trailing, 60)
                .background(AppleDesign.Colors.pearl)
                .clipShape(RoundedRectangle(cornerRadius: 18))

                HStack(spacing: 2) {
                    Menu {
                        Button(action: onPickImage) {
                            Label("图片", systemImage: "photo")
                        }
                        Button(action: onPickFile) {
                            Label("文件", systemImage: "doc")
                        }
                    } label: {
                        Image(systemName: "plus.circle.fill")
                            .font(.title2)
                            .foregroundColor(AppleDesign.Colors.actionBlue)
                    }
                    .buttonStyle(.plain)

                    Button(action: onSend) {
                        Image(systemName: "arrow.up.circle.fill")
                            .font(.title2)
                            .foregroundColor(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                                ? AppleDesign.Colors.inkMuted
                                : AppleDesign.Colors.actionBlue)
                    }
                    .buttonStyle(.plain)
                    .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
                .offset(x: -6, y: -6)
            }
        }
        .padding(12)
        .background(AppleDesign.Colors.parchment)
    }
}
}
