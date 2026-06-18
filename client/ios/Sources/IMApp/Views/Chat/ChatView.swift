import SwiftUI
import IMCore
import Textual
import PhotosUI
import UniformTypeIdentifiers

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
    @State private var showImagePicker = false
    @State private var showFilePicker = false
    @State private var selectedPhotoItem: PhotosPickerItem?
    @State private var isSearching = false
    @State private var cardUser: User?
    @State private var navigateToChat: ChatDestination?
    @State private var highlightedMsgID: Int64?
    @State private var isNearBottom = true
    @State private var newMessageCount = 0
    @State private var scrollOffset: CGFloat = 0
    @State private var scrollContentHeight: CGFloat = 0
    @State private var scrollViewHeight: CGFloat = 0
    @State private var userHasScrolledUp = false

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
                                let repliedMsg = vm.messages.first(where: { $0.msgID == msg.replyTo })
                                MessageBubble(
                                    message: msg,
                                    convType: convType,
                                    senderInfo: vm.senderInfo,
                                    onRetry: { vm.retryMessage(clientSeq: msg.clientSeq) },
                                    onReply: { vm.replyingToMsg = msg },
                                    onTapSender: {
                                        cardUser = vm.senderInfo[msg.senderID]
                                    },
                                    repliedMessage: repliedMsg,
                                    isFirstInGroup: isFirst,
                                    isLastInGroup: isLast,
                                    uploadProgress: vm.uploadProgress[msg.clientSeq],
                                    isHighlighted: highlightedMsgID == msg.stableId
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

                        // Anchor to detect when bottom is visible
                        Color.clear
                            .frame(height: 1)
                            .id("bottomAnchor")
                    }
                    .padding(.horizontal)
                    .background(GeometryReader { geo in
                        Color.clear.preference(
                            key: ScrollOffsetKey.self,
                            value: ScrollInfo(
                                contentHeight: geo.size.height,
                                offset: -geo.frame(in: .named("scroll")).minY,
                                viewHeight: UIScreen.main.bounds.height
                            )
                        )
                    })
                }
                .coordinateSpace(name: "scroll")
                .scrollDismissesKeyboard(.interactively)
                .onTapGesture {
                    UIApplication.shared.sendAction(#selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
                }
                .onPreferenceChange(ScrollOffsetKey.self) { info in
                    let bottomInset: CGFloat = 340 // approximate keyboard + input bar
                    let isNear = (info.offset + info.viewHeight) >= (info.contentHeight - bottomInset - 20)
                    if !isNear && isNearBottom {
                        // User scrolled away — only if we were previously at bottom
                        userHasScrolledUp = true
                    }
                    if isNear {
                        userHasScrolledUp = false
                        isNearBottom = true
                        newMessageCount = 0
                    }
                }
                .overlay(alignment: .bottom) {
                    if userHasScrolledUp && newMessageCount > 0 {
                        Button {
                            userHasScrolledUp = false
                            scrollToBottom(proxy)
                            newMessageCount = 0
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: "chevron.down")
                                    .font(.caption)
                                Text("\(newMessageCount)")
                                    .font(.caption2)
                            }
                            .foregroundColor(.white)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(Capsule().fill(Color.blue).shadow(radius: 4))
                        }
                        .padding(.bottom, 4)
                    }
                }
                .overlay(alignment: .bottomTrailing) {
                    if userHasScrolledUp {
                        Button {
                            userHasScrolledUp = false
                            scrollToBottom(proxy)
                            newMessageCount = 0
                        } label: {
                            Image(systemName: "arrow.down")
                                .font(.system(size: 14, weight: .semibold))
                                .foregroundColor(.white)
                                .frame(width: 32, height: 32)
                                .background(Circle().fill(Color(.systemGray2)).shadow(radius: 3))
                        }
                        .padding(.trailing, 8)
                        .padding(.bottom, 8)
                    }
                }
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
                        }, onPickImage: {
                            showImagePicker = true
                        }, onPickFile: {
                            showFilePicker = true
                        }, replyingToMsg: vm.replyingToMsg, replyingToSender: vm.replyingToMsg.map { msg in
                            msg.senderID == AuthManager.shared.currentUser?.userID
                                ? loc("chat.you")
                                : vm.senderInfo[msg.senderID]?.name ?? msg.senderID
                        }, onCancelReply: {
                            vm.replyingToMsg = nil
                        }, members: vm.members, senderInfo: vm.senderInfo)
                    }
                }
                .onReceive(vm.$chatVersion) { _ in
                    guard vm.messages.last != nil else { return }
                    if !isInitialScrollDone {
                        isInitialScrollDone = true
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                            proxy.scrollTo("bottomAnchor", anchor: .bottom)
                        }
                    } else if !userHasScrolledUp {
                        withAnimation {
                            proxy.scrollTo("bottomAnchor", anchor: .bottom)
                        }
                    } else {
                        newMessageCount += 1
                    }
                }
                .onReceive(NotificationCenter.default.publisher(for: UIResponder.keyboardWillShowNotification)) { _ in
                    guard isInitialScrollDone else { return }
                    scrollToBottom(proxy)
                }

                // Search results
                if showSearch && !searchText.isEmpty {
                    VStack(spacing: 0) {
                        if isSearching {
                            ProgressView()
                                .padding()
                            Spacer()
                        } else if !searchResults.isEmpty {
                            List {
                                ForEach(searchResults) { msg in
                                    VStack(alignment: .leading, spacing: 4) {
                                        InlineText(markdown: msg.body, baseURL: URL(string: AppSettings.shared.serverURL))
                                            .font(.body)
                                            .textual.textSelection(.enabled)
                                        Text(formatTime(msg.timestamp))
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                    .padding(.vertical, 4)
                                    .onTapGesture {
                                        showSearch = false
                                        searchText = ""
                                        searchResults = []
                                        vm.loadContextAround(msgID: msg.msgID)
                                        let targetID = msg.stableId
                                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                                            withAnimation {
                                                proxy.scrollTo(targetID, anchor: .center)
                                            }
                                            highlightedMsgID = targetID
                                            DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) {
                                                withAnimation { highlightedMsgID = nil }
                                            }
                                        }
                                    }
                                }
                            }
                            .listStyle(.plain)
                        } else {
                            Text(loc("search.no_results"))
                                .foregroundColor(.secondary)
                                .padding()
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
        .searchable(text: $searchText, isPresented: $showSearch, placement: .navigationBarDrawer(displayMode: .automatic))
        .onSubmit(of: .search) { performSearch() }
        .onChange(of: showSearch) { _, visible in
            if !visible {
                searchText = ""
                searchResults = []
                isSearching = false
            }
        }
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
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
        .sheet(item: $cardUser) { user in
            UserCardView(user: user, onStartChat: { userID, name in
                cardUser = nil
                Task {
                    do {
                        let result = try await ConversationService.shared.createP2P(userID: userID)
                        navigateToChat = ChatDestination(convID: result.convID, name: result.name, type: .p2p)
                    } catch {
                        // navigation will not occur on failure
                    }
                }
            })
        }
        .navigationDestination(item: $navigateToChat) { dest in
            ChatView(convID: dest.convID, convName: dest.name, convType: dest.type)
        }
        .photosPicker(isPresented: $showImagePicker, selection: $selectedPhotoItem, matching: .images)
        .onChange(of: selectedPhotoItem) { _, newItem in
            guard let item = newItem else { return }
            Task {
                guard let data = try? await item.loadTransferable(type: Data.self) else { return }
                vm.sendImage(fileData: data, fileName: "image.jpg")
                selectedPhotoItem = nil
            }
        }
        .sheet(isPresented: $showFilePicker) {
            DocumentPicker { url in
                guard let data = try? Data(contentsOf: url) else { return }
                vm.sendFile(fileData: data, fileName: url.lastPathComponent)
            }
        }
        .onAppear {
            vm.loadInitialMessages()
            vm.loadMembers()
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

    private func scrollToBottom(_ proxy: ScrollViewProxy) {
        guard vm.messages.last != nil else { return }
        isInitialScrollDone = true
        withAnimation {
            proxy.scrollTo("bottomAnchor", anchor: .bottom)
        }
    }

    private func formatTime(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy/MM/dd HH:mm"
        return formatter.string(from: date)
    }
}

// MARK: - Scroll Position Tracking

private struct ScrollInfo: Equatable {
    let contentHeight: CGFloat
    let offset: CGFloat
    let viewHeight: CGFloat
}

private struct ScrollOffsetKey: PreferenceKey {
    static let defaultValue = ScrollInfo(contentHeight: 0, offset: 0, viewHeight: 0)
    static func reduce(value: inout ScrollInfo, nextValue: () -> ScrollInfo) {
        value = nextValue()
    }
}

// MARK: - Document Picker
struct DocumentPicker: UIViewControllerRepresentable {
    let onPick: (URL) -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(onPick: onPick)
    }

    func makeUIViewController(context: Context) -> UIDocumentPickerViewController {
        let picker = UIDocumentPickerViewController(forOpeningContentTypes: [.data])
        picker.delegate = context.coordinator
        return picker
    }

    func updateUIViewController(_ uiViewController: UIDocumentPickerViewController, context: Context) {}

    class Coordinator: NSObject, UIDocumentPickerDelegate {
        let onPick: (URL) -> Void
        init(onPick: @escaping (URL) -> Void) { self.onPick = onPick }

        func documentPicker(_ controller: UIDocumentPickerViewController, didPickDocumentsAt urls: [URL]) {
            guard let url = urls.first else { return }
            let didAccess = url.startAccessingSecurityScopedResource()
            onPick(url)
            if didAccess {
                url.stopAccessingSecurityScopedResource()
            }
        }
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
