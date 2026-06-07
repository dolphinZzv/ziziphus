import SwiftUI
import IMCore

struct SearchBarView: View {
    @Binding var text: String
    @EnvironmentObject private var localizationManager: LocalizationManager
    var onCommit: (() -> Void)?

    var body: some View {
        HStack {
            Image(systemName: "magnifyingglass")
                .foregroundColor(.secondary)
            TextField(loc("search.placeholder"), text: $text)
                .textFieldStyle(.plain)
                .onSubmit { onCommit?() }
            if !text.isEmpty {
                Button(action: { text = "" }) {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(8)
        .background(Color(.windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}
