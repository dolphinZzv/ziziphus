import SwiftUI
import IMCore

struct ChatView: View {
    let convID: String
    let convName: String
    let convType: ConvType
    @StateObject private var vm: ChatViewModel
    @State private var showGroupDetail = false
    @State private var showP2PDetail = false
    @State private var showHistory = false
    @State private var isInitialScrollDone = false
    @State private var showSearch = false
    @State private var searchText = ""
    @State private var searchResults: [Message] = []
    @State private var isSearching = false

    init(convID: String, convName: String, convType: ConvType = .p2p) {
        self.convID = convID
        self.convName = convName
        self.convType = convType
        _vm = StateObject(wrappedValue: ChatViewModel(convID: convID))
    }

    var body: some View {
        ScrollViewReader { proxy in
            ZStack(alignment: .top) {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        if vm.isLoadingHistory {
                            HStack {
                                Spacer()
                                ProgressView()
                                Spacer()
                            }
                            .padding()
                        }

                        ForEach(Array(vm.chatItems.enumerated()), id: \.element.id) { idx, item in
                            switch item {
                            case .dateSeparator(let date):
                                DateSeparatorView(date: date)
                                    .id(item.id)

                            case .message(let msg, let isFirst, let isLast):
                                MessageBubble(
                                    message: msg,
                                    convType: convType,
                                    senderInfo: vm.senderInfo,
                                    onRetry: { vm.retryMessage(clientSeq: msg.clientSeq) },
                                    isFirstInGroup: isFirst,
                                    isLastInGroup: isLast
                                )
                                .id(msg.stableId)
                                .onAppear {
                                    // Scroll-to-top load more
                                    if idx == 0 && !vm.allHistoryLoaded && !vm.isLoadingHistory {
                                        vm.loadMoreHistory()
                                    }
                                    // Show sender name for group first-in-group messages
                                }
                            }
                        }

                        if vm.isTyping {
                            TypingIndicator()
                                .padding(.leading, 12)
                        }
                    }
                    .padding(.horizontal)
                }
                .scrollDismissesKeyboard(.interactively)
                .safeAreaInset(edge: .bottom, spacing: 0) {
                    VStack(spacing: 0) {
                        if let err = vm.sendErrorMessage {
                            HStack(spacing: 6) {
                                Image(systemName: "exclamationmark.triangle.fill")
                                    .font(.caption)
                                    .foregroundColor(.red)
                                Text(err)
                                    .font(.caption)
                                    .foregroundColor(.red)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 6)
                            .background(.red.opacity(0.08))
                        }
                        InputBarView(text: $vm.inputText, onSend: {
                            vm.sendMessage()
                        }, onTyping: {
                            vm.userDidStartTyping()
                        })
                    }
                }
                .onChange(of: vm.messages.count) { _, _ in
                    if !isInitialScrollDone, let last = vm.messages.last {
                        isInitialScrollDone = true
                        proxy.scrollTo(last.stableId, anchor: .bottom)
                    } else if isInitialScrollDone, let last = vm.messages.last {
                        withAnimation {
                            proxy.scrollTo(last.stableId, anchor: .bottom)
                        }
                    }
                }
                .onReceive(NotificationCenter.default.publisher(for: UIResponder.keyboardWillShowNotification)) { _ in
                    guard isInitialScrollDone else { return }
                    scrollToBottom(proxy)
                }

                // Search bar overlay
                if showSearch {
                    VStack(spacing: 0) {
                        HStack {
                            Image(systemName: "magnifyingglass")
                                .foregroundColor(.secondary)
                            TextField(loc("search.chat_placeholder"), text: $searchText)
                                .autocapitalization(.none)
                                .disableAutocorrection(true)
                                .onSubmit { performSearch() }
                            if !searchText.isEmpty {
                                Button(action: { searchText = "" }) {
                                    Image(systemName: "xmark.circle.fill")
                                        .foregroundColor(.secondary)
                                }
                            }
                            Button(loc("common.cancel")) {
                                showSearch = false
                                searchText = ""
                                searchResults = []
                            }
                        }
                        .padding(10)
                        .background(Color(.systemGray6))
                        .cornerRadius(10)
                        .padding(.horizontal)
                        .padding(.top, 8)

                        if isSearching {
                            ProgressView()
                                .padding()
                            Spacer()
                        } else if !searchResults.isEmpty {
                            List {
                                ForEach(searchResults) { msg in
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text(msg.body)
                                            .font(.body)
                                        Text(formatTime(msg.timestamp))
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                    .padding(.vertical, 4)
                                    .onTapGesture {
                                        showSearch = false
                                        searchText = ""
                                        searchResults = []
                                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                                            withAnimation {
                                                proxy.scrollTo(msg.stableId, anchor: .center)
                                            }
                                        }
                                    }
                                }
                            }
                            .listStyle(.plain)
                        } else if !searchText.isEmpty {
                            Text(loc("search.no_results"))
                                .foregroundColor(.secondary)
                                .padding()
                            Spacer()
                        } else {
                            Spacer()
                        }
                    }
                    .background(.regularMaterial)
                }
            }
        }
        .navigationTitle(convName)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.hidden, for: .tabBar)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                HStack(spacing: 4) {
                    Button(action: { showSearch.toggle() }) {
                        Image(systemName: "magnifyingglass")
                    }
                    Menu {
                        Button(action: {
                            if convType == .group {
                                showGroupDetail = true
                            } else {
                                showP2PDetail = true
                            }
                        }) {
                            Label(loc("history.conversation_info"), systemImage: "info.circle")
                        }
                        Button(action: { showHistory = true }) {
                            Label(loc("history.view_history"), systemImage: "clock")
                        }
                    } label: {
                        Image(systemName: "chevron.down")
                    }
                }
            }
        }
        .sheet(isPresented: $showGroupDetail) {
            NavigationStack {
                GroupDetailView(convID: convID, convName: convName)
            }
        }
        .sheet(isPresented: $showP2PDetail) {
            P2PDetailView(convID: convID, convName: convName)
        }
        .sheet(isPresented: $showHistory) {
            HistoryView(convID: convID, convName: convName, convType: convType)
        }
        .onAppear {
            vm.loadInitialMessages()
            vm.markAsReadIfActive()
        }
    }

    private func performSearch() {
        guard !searchText.isEmpty else { return }
        isSearching = true
        Task {
            do {
                let msgs = try await ConversationService.shared.getHistory(
                    convID: convID, limit: 50, keyword: searchText
                )
                searchResults = msgs.reversed()
            } catch {
                searchResults = []
            }
            isSearching = false
        }
    }

    private func scrollToBottom(_ proxy: ScrollViewProxy) {
        if let last = vm.messages.last {
            withAnimation {
                proxy.scrollTo(last.stableId, anchor: .bottom)
            }
        }
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy/MM/dd HH:mm"
        return formatter.string(from: date)
    }
}

private struct DateSeparatorView: View {
    let date: Date

    var body: some View {
        HStack {
            Spacer()
            Text(formattedDate)
                .font(.caption2)
                .foregroundColor(.secondary)
                .padding(.horizontal, 12)
                .padding(.vertical, 4)
                .background(Color(.systemGray5))
                .clipShape(Capsule())
            Spacer()
        }
        .padding(.vertical, 8)
    }

    private var formattedDate: String {
        let calendar = Calendar.current
        if calendar.isDateInToday(date) {
            return loc("chat.today")
        } else if calendar.isDateInYesterday(date) {
            return loc("chat.yesterday")
        } else {
            let formatter = DateFormatter()
            formatter.dateFormat = "yyyy/MM/dd"
            return formatter.string(from: date)
        }
    }
}
