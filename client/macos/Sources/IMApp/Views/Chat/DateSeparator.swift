import SwiftUI

struct DateSeparator: View {
    let date: Date

    var body: some View {
        Text(formatDate(date))
            .font(.caption2)
            .foregroundColor(.secondary)
            .padding(.horizontal, 12)
            .padding(.vertical, 4)
            .background(Color(.windowBackgroundColor).opacity(0.8))
            .clipShape(Capsule())
            .padding(.vertical, 8)
    }

    private func formatDate(_ date: Date) -> String {
        let formatter = DateFormatter()
        if Calendar.current.isDateInToday(date) {
            formatter.dateFormat = "HH:mm"
        } else {
            formatter.dateFormat = "yyyy-MM-dd"
        }
        return formatter.string(from: date)
    }
}
