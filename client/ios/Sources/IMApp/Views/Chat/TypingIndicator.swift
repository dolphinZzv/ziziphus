import SwiftUI

struct TypingIndicator: View {
    var body: some View {
        HStack(spacing: 4) {
            Text("正在输入...")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 4)
    }
}
