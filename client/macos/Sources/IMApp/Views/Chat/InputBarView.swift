import SwiftUI

struct InputBarView: View {
    @Binding var text: String
    let onSend: () -> Void
    let onTyping: () -> Void

    var body: some View {
        HStack(spacing: 8) {
            TextField("输入消息...", text: $text)
                .textFieldStyle(.roundedBorder)
                .onChange(of: text) { _, _ in
                    onTyping()
                }
                .onSubmit {
                    onSend()
                }

            Button(action: onSend) {
                Image(systemName: "arrow.up.circle.fill")
                    .font(.title2)
            }
            .buttonStyle(.plain)
            .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
        .padding(12)
        .background(Color(.windowBackgroundColor))
    }
}
