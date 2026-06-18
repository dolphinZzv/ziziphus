import SwiftUI
import IMCore
import UniformTypeIdentifiers

struct ChatView: View {
    let convID: String
    let convName: String
    let convType: ConvType
    @StateObject private var vm: ChatViewModel
    @State private var showGroupDetail = false
    @State private var showP2PDetail = false
    @State private var showHistory = false
    @State private var showSearch = false
    @State private var searchText = ""
    @State private var searchResults: [Message] = []
    @State private var isSearching = false
    @State private var isInitialScrollDone = false
    @EnvironmentObject private var localizationManager: LocalizationManager

    init(convID: String, convName: String, convType: ConvType = .p2p) {
        self.convID = convID
        self.convName = convName
        self.convType = convType
        _vm = StateObject(wrappedValue: ChatViewModel(convID: convID))
    }

    var body: some View {
        VStack(spacing: 0) {
            // Messages area
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 0) {
                        if vm.isLoadingHistory {
                            ProgressView()
                                .padding()
                        }

                        ForEach(Array(vm.chatItems.enumerated()), id: \.element.id) { idx, item in
                            switch item {
                            case .dateSeparator(let date):
                                DateSeparatorView(date: date)
                                    .id(item.id)

                            case .message(let msg, let isFirst, let isLast):
                                let repliedMsg = vm.messages.first(where: { $0.msgID == msg.replyTo })
                                MessageBubble(
                                    message: msg,
                                    convType: convType,
                                    senderInfo: vm.senderInfo,
                                    onRetry: { vm.retryMessage(clientSeq: msg.clientSeq) },
                                    onReply: { vm.replyingToMsg = msg },
                                    repliedMessage: repliedMsg,
                                    isFirstInGroup: isFirst,
                                    isLastInGroup: isLast,
                                    uploadProgress: vm.uploadProgress[msg.clientSeq]
                                )
                                .id(msg.stableId)
                                .onAppear {
                                    if idx == 0 && isInitialScrollDone && !vm.allHistoryLoaded && !vm.isLoadingHistory {
                                        vm.loadMoreHistory()
                                    }
                                }
                            }
                        }

                        if vm.isTyping {
                            TypingIndicator()
                                .padding(.leading, 12)
                        }

                        Color.clear
                            .frame(height: 1)
                            .id("bottomAnchor")
                    }
                    .padding(.horizontal)
                }
                .onAppear {
                    guard vm.messages.last != nil else { return }
                    isInitialScrollDone = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                        proxy.scrollTo("bottomAnchor", anchor: .bottom)
                    }
                }
                .onChange(of: vm.messages.count) { _, _ in
                    guard vm.messages.last != nil else { return }
                    isInitialScrollDone = true
                    proxy.scrollTo("bottomAnchor", anchor: .bottom)
                }

                // Search overlay
                if showSearch {
                    VStack(spacing: 0) {
                        HStack {
                            Image(systemName: "magnifyingglass")
                                .foregroundColor(.secondary)
                            TextField(loc("search.chat_placeholder"), text: $searchText)
                                .textFieldStyle(.plain)
                                .onSubmit { performSearch() }
                            if !searchText.isEmpty {
                                Button(action: { searchText = "" }) {
                                    Image(systemName: "xmark.circle.fill")
                                        .foregroundColor(.secondary)
                                }
                                .buttonStyle(.plain)
                            }
                            Button(loc("common.cancel")) {
                                showSearch = false
                                searchText = ""
                                searchResults = []
                            }
                            .buttonStyle(.plain)
                            .foregroundColor(AppleDesign.Colors.actionBlue)
                        }
                        .padding(10)
                        .background(Color(nsColor: .controlBackgroundColor))
                        .cornerRadius(10)
                        .padding(.horizontal)
                        .padding(.top, 8)

                        if isSearching {
                            ProgressView()
                                .padding()
                            Spacer()
                        } else if !searchResults.isEmpty {
                            ScrollView {
                                LazyVStack(spacing: 0) {
                                    ForEach(searchResults) { msg in
                                        VStack(alignment: .leading, spacing: 4) {
                                            InlineText(markdown: msg.body, baseURL: URL(string: AppSettings.shared.serverURL))
                                                .font(.appleBody)
                                                .textual.textSelection(.enabled)
                                            Text(formatTime(msg.timestamp))
                                                .font(.system(size: AppleDesign.Typography.finePrintSize))
                                                .foregroundColor(AppleDesign.Colors.inkMuted)
                                        }
                                        .padding(.horizontal, AppleDesign.Spacing.lg)
                                        .padding(.vertical, 8)
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                        .contentShape(Rectangle())
                                        .onTapGesture {
                                            showSearch = false
                                            searchText = ""
                                            searchResults = []
                                            vm.loadContextAround(msgID: msg.msgID)
                                            DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                                                withAnimation {
                                                    proxy.scrollTo(msg.stableId, anchor: .center)
                                                }
                                            }
                                        }
                                        Divider()
                                    }
                                }
                            }
                        } else if !searchText.isEmpty {
                            Text(loc("search.no_results"))
                                .foregroundColor(.secondary)
                                .padding()
                            Spacer()
                        } else {
                            Spacer()
                        }
                    }
                    .frame(width: 360, height: 300)
                    .background(Color(nsColor: .windowBackgroundColor))
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                    .shadow(radius: 8)
                    .padding(.top, 40)
                }
            }

            // Error banner
            if let err = vm.sendErrorMessage {
                Text(err)
                    .font(.system(size: AppleDesign.Typography.finePrintSize))
                    .foregroundColor(.red)
                    .padding(.horizontal, 16)
                    .padding(.vertical, 4)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }

            // Input bar
            InputBarView(text: $vm.inputText, onSend: {
                vm.sendMessage()
            }, onTyping: {
                vm.userDidStartTyping()
            }, onPickImage: {
                pickFile(fileTypes: ["public.image"])
            }, onPickFile: {
                pickFile(fileTypes: ["public.data"])
            }, replyingToMsg: vm.replyingToMsg, replyingToSender: vm.replyingToMsg.map { msg in
                msg.senderID == AuthManager.shared.currentUser?.userID
                    ? loc("chat.you")
                    : vm.senderInfo[msg.senderID]?.name ?? msg.senderID
            }, onCancelReply: {
                vm.replyingToMsg = nil
            })
            .environmentObject(localizationManager)
        }
        .background(Color(nsColor: .windowBackgroundColor))
        .toolbar {
            ToolbarItem(placement: .principal) {
                Text(convType == .p2p ? String(format: loc("chat.session_title"), convName) : convName)
                    .font(.appleBodySemibold)
            }

            ToolbarItem(placement: .primaryAction) {
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
                        EmptyView()
                    }
                }
            }
        }
        .sheet(isPresented: $showGroupDetail) {
            GroupDetailView(convID: convID, convName: convName)
        }
        .sheet(isPresented: $showP2PDetail) {
            P2PDetailView(convID: convID, convName: convName, onCancel: {
                showP2PDetail = false
            })
        }
        .sheet(isPresented: $showHistory) {
            HistoryView(convID: convID, convName: convName, convType: convType)
        }
        .onAppear {
            vm.loadInitialMessages()
            vm.markAsReadIfActive()
            vm.inputText = UserDefaults.standard.string(forKey: "draft_\(convID)") ?? ""
        }
        .onChange(of: vm.inputText) { _, newText in
            UserDefaults.standard.set(newText, forKey: "draft_\(convID)")
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

    // MARK: - File Picker
    private func pickFile(fileTypes: [String]) {
        let panel = NSOpenPanel()
        panel.allowsMultipleSelection = false
        panel.canChooseDirectories = false
        panel.allowedContentTypes = fileTypes.compactMap { UTType($0) }
        panel.begin { response in
            guard response == .OK, let url = panel.url else { return }
            guard let data = try? Data(contentsOf: url) else { return }
            let isImage = ["jpg", "jpeg", "png", "gif", "webp", "heic"].contains(url.pathExtension.lowercased())
            if isImage {
                vm.sendImage(fileData: data, fileName: url.lastPathComponent)
            } else {
                vm.sendFile(fileData: data, fileName: url.lastPathComponent)
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
                .font(.system(size: AppleDesign.Typography.finePrintSize))
                .foregroundColor(.secondary)
                .padding(.horizontal, 12)
                .padding(.vertical, 4)
                .background(Color(nsColor: .controlBackgroundColor))
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
