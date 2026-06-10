import SwiftUI
import IMCore
import MarkdownUI

struct HistoryView: View {
    let convID: String
    let convName: String
    let convType: ConvType
    @State private var messages: [Message] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var loadedAll = false
    @State private var loadingMore = false
    @State private var searchText = ""
    @State private var selectedYear: Int?
    @State private var selectedMonth: Int?
    @State private var selectedDay: Int?
    @State private var showDatePicker = false

    private let pageSize = 50

    private var availableYears: [Int] {
        let year = Calendar.current.component(.year, from: Date())
        return Array((year - 5)...year).reversed()
    }

    private var availableMonths: [Int] { Array(1...12) }

    private var availableDays: [Int] {
        guard let year = selectedYear, let month = selectedMonth else { return [] }
        let dc = DateComponents(year: year, month: month)
        let calendar = Calendar.current
        if let date = calendar.date(from: dc), let range = calendar.range(of: .day, in: .month, for: date) {
            return Array(range)
        }
        return []
    }

    private var hasActiveFilters: Bool {
        !searchText.isEmpty || selectedYear != nil || selectedMonth != nil || selectedDay != nil
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text(String(format: loc("history.title"), convName))
                    .font(.appleBodySemibold)
                Spacer()
            }
            .padding(AppleDesign.Spacing.lg)

            // Search bar
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField(loc("search.placeholder"), text: $searchText)
                    .textFieldStyle(.plain)
                    .font(.system(size: AppleDesign.Typography.bodySize))
                    .onSubmit { applyFilters() }
                if !searchText.isEmpty {
                    Button(action: { searchText = "" }) {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundColor(.secondary)
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(8)
            .background(Color(nsColor: .controlBackgroundColor))
            .clipShape(Capsule())
            .padding(.horizontal, AppleDesign.Spacing.lg)

            // Date filter row
            HStack(spacing: 6) {
                DateFilterButton(title: selectedYear.map { "\($0)" } ?? loc("history.filter_year"),
                                 isActive: selectedYear != nil) { showDatePicker.toggle() }

                if selectedYear != nil {
                    DateFilterButton(title: selectedMonth.map { String(format: "%02d", $0) } ?? loc("history.filter_month"),
                                     isActive: selectedMonth != nil) {
                        if selectedMonth != nil { selectedMonth = nil; applyFilters() }
                        else { showDatePicker.toggle() }
                    }
                }

                if selectedMonth != nil {
                    DateFilterButton(title: selectedDay.map { String(format: "%02d", $0) } ?? loc("history.filter_day"),
                                     isActive: selectedDay != nil) {
                        if selectedDay != nil { selectedDay = nil; applyFilters() }
                    }
                }

                if hasActiveFilters {
                    Button(loc("history.filter_clear")) { clearFilters() }
                        .font(.system(size: AppleDesign.Typography.finePrintSize))
                        .foregroundColor(AppleDesign.Colors.actionBlue)
                        .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, AppleDesign.Spacing.lg)
            .padding(.vertical, 6)

            // Date picker popover
            if showDatePicker {
                VStack(spacing: 8) {
                    HStack {
                        Picker(loc("history.filter_year"), selection: $selectedYear) {
                            Text(loc("history.filter_year")).tag(nil as Int?)
                            ForEach(availableYears, id: \.self) { year in
                                Text("\(year)").tag(year as Int?)
                            }
                        }
                        if selectedYear != nil {
                            Picker(loc("history.filter_month"), selection: $selectedMonth) {
                                Text(loc("history.filter_month")).tag(nil as Int?)
                                ForEach(availableMonths, id: \.self) { m in
                                    Text(String(format: "%02d", m)).tag(m as Int?)
                                }
                            }
                        }
                        if selectedMonth != nil {
                            Picker(loc("history.filter_day"), selection: $selectedDay) {
                                Text(loc("history.filter_day")).tag(nil as Int?)
                                ForEach(availableDays, id: \.self) { d in
                                    Text(String(format: "%02d", d)).tag(d as Int?)
                                }
                            }
                        }
                    }

                    Button(loc("common.confirm")) {
                        showDatePicker = false
                        applyFilters()
                    }
                    .buttonStyle(ApplePrimaryButtonStyle())
                }
                .padding()
                .background(Color(nsColor: .controlBackgroundColor))
                .cornerRadius(8)
                .padding(.horizontal, AppleDesign.Spacing.lg)
            }

            Divider()
                .padding(.top, 4)

            // Content
            if isLoading {
                Spacer()
                ProgressView()
                Spacer()
            } else if let err = errorMessage {
                Spacer()
                VStack(spacing: 8) {
                    Text(err)
                        .foregroundColor(.secondary)
                    Button(loc("common.retry")) { loadHistory() }
                        .buttonStyle(.plain)
                        .foregroundColor(AppleDesign.Colors.actionBlue)
                }
                Spacer()
            } else if messages.isEmpty {
                Spacer()
                Text(loc("chat.no_messages"))
                    .foregroundColor(.secondary)
                Spacer()
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(messages) { msg in
                            HistoryMessageRow(message: msg, convType: convType)
                                .padding(.horizontal, AppleDesign.Spacing.lg)
                                .padding(.vertical, 6)
                                .onAppear {
                                    if msg.msgID == messages.last?.msgID && !loadedAll && !loadingMore {
                                        loadMore()
                                    }
                                }

                            Divider()
                                .padding(.leading, AppleDesign.Spacing.lg)
                        }

                        if loadingMore {
                            ProgressView()
                                .padding()
                        }

                        if loadedAll && !messages.isEmpty {
                            Text(loc("history.all_loaded"))
                                .font(.system(size: AppleDesign.Typography.finePrintSize))
                                .foregroundColor(.secondary)
                                .frame(maxWidth: .infinity)
                                .padding()
                        }
                    }
                }
            }
        }
        .frame(width: 400, height: 450)
        .background(Color(nsColor: .windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .onAppear { loadHistory() }
    }

    private func clearFilters() {
        searchText = ""
        selectedYear = nil
        selectedMonth = nil
        selectedDay = nil
        applyFilters()
    }

    private func applyFilters() {
        isLoading = true
        errorMessage = nil
        messages = []
        loadedAll = false
        Task {
            do {
                let (startDate, endDate) = computeDateRange()
                let msgs = try await ConversationService.shared.getHistory(
                    convID: convID, limit: pageSize,
                    keyword: searchText,
                    startDate: startDate,
                    endDate: endDate
                )
                messages = msgs.reversed()
                loadedAll = msgs.count < pageSize
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    private func loadHistory() {
        isLoading = true
        errorMessage = nil
        messages = []
        loadedAll = false
        Task {
            do {
                let msgs = try await ConversationService.shared.getHistory(convID: convID, limit: pageSize, keyword: searchText)
                messages = msgs.reversed()
                loadedAll = msgs.count < pageSize
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    private func loadMore() {
        guard let lastMsgID = messages.last?.msgID, lastMsgID > 0 else { return }
        loadingMore = true
        Task {
            do {
                let msgs = try await ConversationService.shared.getHistory(convID: convID, beforeMsgID: lastMsgID, limit: pageSize, keyword: searchText)
                if msgs.isEmpty {
                    loadedAll = true
                } else {
                    messages += msgs.reversed()
                    loadedAll = msgs.count < pageSize
                }
            } catch {
                // Silently fail on load-more errors
            }
            loadingMore = false
        }
    }

    private func computeDateRange() -> (Int64, Int64) {
        guard let year = selectedYear else { return (0, 0) }
        let calendar = Calendar.current
        var startDC = DateComponents()
        startDC.year = year
        startDC.month = selectedMonth ?? 1
        startDC.day = selectedDay ?? 1
        guard let start = calendar.date(from: startDC) else { return (0, 0) }
        let startMs = Int64(start.timeIntervalSince1970 * 1000)

        var endDC = DateComponents()
        if let day = selectedDay, let month = selectedMonth {
            endDC.year = year
            endDC.month = month
            endDC.day = day + 1
        } else if let month = selectedMonth {
            endDC.year = year
            endDC.month = month + 1
            endDC.day = 1
        } else {
            endDC.year = year + 1
            endDC.month = 1
            endDC.day = 1
        }
        guard let end = calendar.date(from: endDC) else { return (startMs, 0) }
        return (startMs, Int64(end.timeIntervalSince1970 * 1000))
    }
}

private struct DateFilterButton: View {
    let title: String
    let isActive: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 4) {
                Text(title)
                    .font(.system(size: AppleDesign.Typography.finePrintSize))
                if isActive {
                    Image(systemName: "xmark")
                        .font(.system(size: 8))
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 4)
            .background(isActive ? AppleDesign.Colors.actionBlue.opacity(0.15) : Color(nsColor: .controlBackgroundColor))
            .foregroundColor(isActive ? AppleDesign.Colors.actionBlue : .primary)
            .clipShape(Capsule())
        }
        .buttonStyle(.plain)
    }
}

private struct HistoryMessageRow: View {
    let message: Message
    let convType: ConvType

    private var isMine: Bool {
        message.senderID == AuthManager.shared.currentUser?.userID
    }

    private var senderDisplayName: String {
        if isMine {
            return AuthManager.shared.currentUser?.name ?? message.senderID
        }
        return message.senderName.isEmpty ? message.senderID : message.senderName
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            if convType == .group {
                Text(senderDisplayName)
                    .font(.system(size: AppleDesign.Typography.captionSize))
                    .foregroundColor(AppleDesign.Colors.inkMuted)
            }

            Markdown(message.body)
                .font(.appleBody)
                .frame(maxWidth: .infinity, alignment: .leading)

            Text(formatTime(message.timestamp))
                .font(.system(size: AppleDesign.Typography.finePrintSize))
                .foregroundColor(AppleDesign.Colors.inkMuted)
        }
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy/MM/dd HH:mm"
        return formatter.string(from: date)
    }
}
