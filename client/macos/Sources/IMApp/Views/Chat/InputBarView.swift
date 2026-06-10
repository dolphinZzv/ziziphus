import SwiftUI
import IMCore

struct InputBarView: View {
    @Binding var text: String
    @EnvironmentObject private var localizationManager: LocalizationManager
    let onSend: () -> Void
    let onTyping: () -> Void

    var body: some View {
        HStack(spacing: 0) {
            ZStack(alignment: .bottomTrailing) {
                ChatTextView(
                    text: $text,
                    placeholder: loc("chat.placeholder"),
                    onTyping: onTyping,
                    onSend: onSend
                )
                .frame(minHeight: 40, maxHeight: 120)
                .padding(.trailing, 36)
                .background(AppleDesign.Colors.pearl)
                .clipShape(RoundedRectangle(cornerRadius: 18))

                Button(action: onSend) {
                    Image(systemName: "arrow.up.circle.fill")
                        .font(.title2)
                        .foregroundColor(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                            ? AppleDesign.Colors.inkMuted
                            : AppleDesign.Colors.actionBlue)
                }
                .buttonStyle(.plain)
                .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                .offset(x: -6, y: -6)
            }
        }
        .padding(12)
        .background(AppleDesign.Colors.parchment)
    }
}
