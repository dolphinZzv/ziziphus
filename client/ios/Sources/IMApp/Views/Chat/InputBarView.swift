import SwiftUI
import IMCore

struct InputBarView: View {
    @Binding var text: String
    let onSend: () -> Void
    let onTyping: () -> Void

    var body: some View {
        HStack(spacing: 8) {
            TextField(loc("chat.placeholder"), text: $text)
                .textFieldStyle(.roundedBorder)
                .onChange(of: text) { _, _ in
                    onTyping()
                }

            Button(action: onSend) {
                Image(systemName: "arrow.up.circle.fill")
                    .font(.title2)
                    .foregroundColor(.blue)
            }
            .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
        .padding(12)
        .background(Color(.systemGroupedBackground))
    }
}
