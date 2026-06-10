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
    @Environment(\.dismiss) private var dismiss

    private let pageSize = 50

    private var availableYears: [Int] {
        let year = Calendar.current.component(.year, from: Date())
        return Array((year - 5)...year).reversed()
    }

    private var availableMonths: [Int] {
        Array(1...12)
    }

    private var availableDays: [Int] {
        guard let year = selectedYear, let month = selectedMonth else { return [] }
        let dateComponents = DateComponents(year: year, month: month)
        let calendar = Calendar.current
        if let date = calendar.date(from: dateComponents),
           let range = calendar.range(of: .day, in: .month, for: date) {
            return Array(range)
        }
        return []
    }

    private var hasActiveFilters: Bool {
        !searchText.isEmpty || selectedYear != nil || selectedMonth != nil || selectedDay != nil
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Search bar
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField(loc("search.placeholder"), text: $searchText)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .onSubmit { applyFilters() }
                    if !searchText.isEmpty {
                        Button(action: { searchText = "" }) {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                    }
                }
                .padding(10)
                .background(Color(.systemGray6))
                .cornerRadius(10)
                .padding(.horizontal)
                .padding(.top, 8)

                // Date filter row
                HStack(spacing: 8) {
                    DateFilterButton(title: selectedYear.map { "\($0)" } ?? loc("history.filter_year"),
                                     isActive: selectedYear != nil) {
                        showDatePicker.toggle()
                    }

                    if selectedYear != nil {
                        DateFilterButton(title: selectedMonth.map { String(format: "%02d", $0) } ?? loc("history.filter_month"),
                                         isActive: selectedMonth != nil) {
                            if selectedMonth != nil {
                                selectedMonth = nil
                                applyFilters()
                            } else {
                                showDatePicker.toggle()
                            }
                        }
                    }

                    if selectedMonth != nil {
                        DateFilterButton(title: selectedDay.map { String(format: "%02d", $0) } ?? loc("history.filter_day"),
                                         isActive: selectedDay != nil) {
                            if selectedDay != nil {
                                selectedDay = nil
                                applyFilters()
                            }
                        }
                    }

                    if hasActiveFilters {
                        Button(loc("history.filter_clear")) {
                            clearFilters()
                        }
                        .font(.caption)
                        .foregroundColor(.blue)
                    }
                }
                .padding(.horizontal)
                .padding(.vertical, 6)

                // Date picker popover
                if showDatePicker {
                    VStack(spacing: 12) {
                        HStack {
                            Picker(loc("history.filter_year"), selection: $selectedYear) {
                                Text(loc("history.filter_year")).tag(nil as Int?)
                                ForEach(availableYears, id: \.self) { year in
                                    Text("\(year)").tag(year as Int?)
                                }
                            }
                            .pickerStyle(.menu)

                            if selectedYear != nil {
                                Picker(loc("history.filter_month"), selection: $selectedMonth) {
                                    Text(loc("history.filter_month")).tag(nil as Int?)
                                    ForEach(availableMonths, id: \.self) { month in
                                        Text(String(format: "%02d", month)).tag(month as Int?)
                                    }
                                }
                                .pickerStyle(.menu)
                            }

                            if selectedMonth != nil {
                                Picker(loc("history.filter_day"), selection: $selectedDay) {
                                    Text(loc("history.filter_day")).tag(nil as Int?)
                                    ForEach(availableDays, id: \.self) { day in
                                        Text(String(format: "%02d", day)).tag(day as Int?)
                                    }
                                }
                                .pickerStyle(.menu)
                            }
                        }

                        Button(loc("common.confirm")) {
                            showDatePicker = false
                            applyFilters()
                        }
                        .buttonStyle(.borderedProminent)
                        .frame(maxWidth: .infinity)
                    }
                    .padding()
                    .background(Color(.systemGray5))
                    .cornerRadius(12)
                    .padding(.horizontal)
                }

                Divider()
                    .padding(.top, 4)

                // Content
                Group {
                    if isLoading {
                        Spacer()
                        ProgressView(loc("common.loading"))
                        Spacer()
                    } else if let err = errorMessage {
                        Spacer()
                        VStack(spacing: 12) {
                            Image(systemName: "exclamationmark.triangle")
                                .font(.largeTitle)
                                .foregroundColor(.secondary)
                            Text(err)
                                .foregroundColor(.secondary)
                            Button(loc("common.retry")) { loadHistory() }
                        }
                        Spacer()
                    } else if messages.isEmpty {
                        Spacer()
                        VStack {
                            Image(systemName: "text.bubble")
                                .font(.system(size: 40))
                                .foregroundColor(.secondary)
                            Text(loc("chat.no_messages"))
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                    } else {
                        List {
                            ForEach(messages) { msg in
                                HistoryMessageRow(message: msg, convType: convType)
                                    .listRowSeparator(.hidden)
                                    .onAppear {
                                        if msg.msgID == messages.last?.msgID && !loadedAll && !loadingMore {
                                            loadMore()
                                        }
                                    }
                            }

                            if loadingMore {
                                HStack {
                                    Spacer()
                                    ProgressView()
                                    Spacer()
                                }
                                .listRowSeparator(.hidden)
                            }

                            if loadedAll && !messages.isEmpty {
                                Text(loc("history.all_loaded"))
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                    .frame(maxWidth: .infinity)
                                    .listRowSeparator(.hidden)
                            }
                        }
                        .listStyle(.plain)
                        .refreshable { loadHistory() }
                    }
                }
            }
            .navigationTitle(String(format: loc("history.title"), convName))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(loc("common.close")) { dismiss() }
                }
            }
        }
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
                var startDate: Int64 = 0
                var endDate: Int64 = 0
                if let year = selectedYear {
                    var dc = DateComponents()
                    dc.year = year
                    dc.month = selectedMonth ?? 1
                    dc.day = selectedDay ?? 1
                    if let start = Calendar.current.date(from: dc) {
                        startDate = Int64(start.timeIntervalSince1970 * 1000)
                    }
                    if let day = selectedDay, let month = selectedMonth {
                        dc.day = day + 1
                        if let end = Calendar.current.date(from: dc) {
                            endDate = Int64(end.timeIntervalSince1970 * 1000)
                        }
                    } else if let month = selectedMonth {
                        dc.month = month + 1
                        dc.day = 1
                        if let end = Calendar.current.date(from: dc) {
                            endDate = Int64(end.timeIntervalSince1970 * 1000)
                        }
                    } else {
                        dc.year = year + 1
                        dc.month = 1
                        dc.day = 1
                        if let end = Calendar.current.date(from: dc) {
                            endDate = Int64(end.timeIntervalSince1970 * 1000)
                        }
                    }
                }
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
                let msgs = try await ConversationService.shared.getHistory(
                    convID: convID, beforeMsgID: lastMsgID, limit: pageSize,
                    keyword: searchText,
                    startDate: 0,
                    endDate: 0
                )
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
}

private struct DateFilterButton: View {
    let title: String
    let isActive: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 4) {
                Text(title)
                    .font(.caption)
                if isActive {
                    Image(systemName: "xmark")
                        .font(.system(size: 8))
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(isActive ? Color.blue.opacity(0.15) : Color(.systemGray5))
            .foregroundColor(isActive ? .blue : .primary)
            .cornerRadius(8)
        }
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
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Markdown(message.body)
                .font(.body)
                .frame(maxWidth: .infinity, alignment: .leading)

            Text(formatTime(message.timestamp))
                .font(.caption2)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 4)
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy/MM/dd HH:mm"
        return formatter.string(from: date)
    }
}
