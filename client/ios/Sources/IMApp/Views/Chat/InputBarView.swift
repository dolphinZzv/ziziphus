import SwiftUI
import IMCore

struct InputBarView: View {
    @Binding var text: String
    let onSend: () -> Void
    let onTyping: () -> Void

    var body: some View {
        HStack(spacing: 0) {
            ZStack(alignment: .bottomTrailing) {
                TextField(loc("chat.placeholder"), text: $text, axis: .vertical)
                    .textFieldStyle(.plain)
                    .font(.body)
                    .lineLimit(1...5)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 8)
                    .padding(.trailing, 36)
                    .background(Color(.systemGray6))
                    .clipShape(RoundedRectangle(cornerRadius: 20))
                    .onChange(of: text) { _, _ in
                        onTyping()
                    }

                Button(action: onSend) {
                    Image(systemName: "arrow.up.circle.fill")
                        .font(.title2)
                        .foregroundColor(.blue)
                }
                .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                .offset(x: -6, y: -6)
            }
        }
        .padding(12)
        .background(Color(.systemGroupedBackground))
    }
}
