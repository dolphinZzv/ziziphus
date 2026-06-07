import SwiftUI
import IMCore

struct TypingIndicator: View {
    @EnvironmentObject private var localizationManager: LocalizationManager

    var body: some View {
        HStack(spacing: 4) {
            Text(loc("chat.typing"))
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 4)
    }
}
