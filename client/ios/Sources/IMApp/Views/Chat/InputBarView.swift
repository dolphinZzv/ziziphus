import SwiftUI
import IMCore

struct InputBarView: View {
    @Binding var text: String
    let onSend: () -> Void
    let onTyping: () -> Void
    let onPickImage: () -> Void
    let onPickFile: () -> Void
    var replyingToMsg: Message?
    var replyingToSender: String?
    var onCancelReply: (() -> Void)?
    @State private var showEmojiPicker = false

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
                            .font(.caption2)
                            .foregroundColor(.blue)
                        Text(replyingToMsg.body)
                            .font(.caption2)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                    }
                    Spacer()
                    Button(action: { onCancelReply?() }) {
                        Image(systemName: "xmark")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 6)
                .background(Color(.systemGray5))
            }

            HStack(spacing: 0) {
            ZStack(alignment: .bottomTrailing) {
                TextField(loc("chat.placeholder"), text: $text, axis: .vertical)
                    .textFieldStyle(.plain)
                    .font(.body)
                    .lineLimit(1...5)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 8)
                    .padding(.trailing, 60)
                    .background(Color(.systemBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 20))
                    .overlay(
                        RoundedRectangle(cornerRadius: 20)
                            .stroke(Color(.separator), lineWidth: 0.5)
                    )
                    .onChange(of: text) { _, _ in
                        onTyping()
                    }

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
                            .foregroundColor(.blue)
                    }

                    Button(action: {
                        UIApplication.shared.sendAction(#selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
                        showEmojiPicker = true
                    }) {
                        Image(systemName: "face.smiling.fill")
                            .font(.title2)
                            .foregroundColor(.blue)
                    }
                    .popover(isPresented: $showEmojiPicker) {
                        EmojiPickerView { emoji in
                            text.append(emoji)
                            showEmojiPicker = false
                        }
                        .presentationCompactAdaptation(.popover)
                    }

                    Button(action: onSend) {
                        Image(systemName: "arrow.up.circle.fill")
                            .font(.title2)
                            .foregroundColor(.blue)
                    }
                    .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
                .offset(x: -6, y: -6)
            }
        }
        .padding(12)
        .background(Color(.systemGray6))
    }
}
}

// MARK: - Emoji Picker
private struct EmojiPickerView: View {
    let onSelect: (String) -> Void
    @Environment(\.dismiss) private var dismiss

    private let emojiRows: [[String]] = [
        ["😊", "😂", "🥰", "😍", "😘", "😜", "😎", "🤗"],
        ["😢", "😡", "🥺", "😴", "🤔", "🙄", "😏", "😅"],
        ["👍", "👎", "👌", "✌️", "🤞", "💪", "🤝", "👏"],
        ["🎉", "🎊", "🎈", "🔥", "⭐", "💯", "✅", "❌"],
        ["❤️", "💔", "💖", "💕", "💗", "💙", "💚", "💜"],
    ]

    var body: some View {
        VStack(spacing: 8) {
            ForEach(emojiRows, id: \.self) { row in
                HStack(spacing: 12) {
                    ForEach(row, id: \.self) { emoji in
                        Button(action: { onSelect(emoji) }) {
                            Text(emoji).font(.system(size: 28))
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
        .padding()
    }
}
