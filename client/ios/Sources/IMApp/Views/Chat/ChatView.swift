import SwiftUI
import IMCore

struct ChatView: View {
    let convID: String
    let convName: String
    let convType: ConvType
    @StateObject private var vm: ChatViewModel
    @State private var showGroupDetail = false
    @State private var showP2PDetail = false
    @State private var isInitialScrollDone = false

    init(convID: String, convName: String, convType: ConvType = .p2p) {
        self.convID = convID
        self.convName = convName
        self.convType = convType
        _vm = StateObject(wrappedValue: ChatViewModel(convID: convID))
    }

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(spacing: 0) {
                    if vm.isLoadingHistory {
                        ProgressView()
                            .padding()
                    }

                    ForEach(vm.messages) { msg in
                        MessageBubble(message: msg, convType: convType, senderInfo: vm.senderInfo, onRetry: {
                            vm.retryMessage(clientSeq: msg.clientSeq)
                        })
                        .id(msg.id)
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
                guard !vm.isLoadingHistory else { return }
                scrollToBottom(proxy)
            }
            .onChange(of: vm.isLoadingHistory) { _, loading in
                if !loading, !isInitialScrollDone {
                    isInitialScrollDone = true
                    scrollToBottom(proxy)
                }
            }
            .onReceive(NotificationCenter.default.publisher(for: UIResponder.keyboardWillShowNotification)) { _ in
                scrollToBottom(proxy)
            }
        }
        .navigationTitle(convName)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.hidden, for: .tabBar)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button(action: {
                    if convType == .group {
                        showGroupDetail = true
                    } else {
                        showP2PDetail = true
                    }
                }) {
                    Image(systemName: "ellipsis")
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
        .onAppear {
            vm.loadInitialMessages()
            vm.markAsReadIfActive()
        }
    }

    private func scrollToBottom(_ proxy: ScrollViewProxy) {
        if let last = vm.messages.last {
            withAnimation {
                proxy.scrollTo(last.id, anchor: .bottom)
            }
        }
    }
}
