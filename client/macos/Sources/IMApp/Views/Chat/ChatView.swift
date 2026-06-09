import SwiftUI
import IMCore

struct ChatView: View {
    let convID: String
    let convName: String
    let convType: ConvType
    @StateObject private var vm: ChatViewModel
    @State private var showGroupDetail = false
    @State private var showP2PDetail = false
    @EnvironmentObject private var localizationManager: LocalizationManager

    init(convID: String, convName: String, convType: ConvType = .p2p) {
        self.convID = convID
        self.convName = convName
        self.convType = convType
        _vm = StateObject(wrappedValue: ChatViewModel(convID: convID))
    }

    var body: some View {
        VStack(spacing: 0) {
            // Messages
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 0) {
                        if vm.isLoadingHistory {
                            ProgressView()
                                .padding()
                        }

                        ForEach(vm.messages) { msg in
                            MessageBubble(message: msg, convType: convType, senderInfo: vm.senderInfo)
                                .id(msg.id)
                        }

                        if vm.isTyping {
                            TypingIndicator()
                                .padding(.leading, 12)
                        }
                    }
                    .padding(.horizontal)
                }
                .onAppear {
                    if let last = vm.messages.last {
                        proxy.scrollTo(last.id, anchor: .bottom)
                    }
                }
                .onChange(of: vm.messages.count) { _, _ in
                    if let last = vm.messages.last {
                        proxy.scrollTo(last.id, anchor: .bottom)
                    }
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
            })
        }
        .background(Color(nsColor: .windowBackgroundColor))
        .toolbar {
            ToolbarItem(placement: .principal) {
                Button(action: {
                    if convType == .group {
                        showGroupDetail = true
                    } else {
                        showP2PDetail = true
                    }
                }) {
                    Text(String(format: loc("chat.session_title"), convName))
                        .font(.appleBodySemibold)
                }
                .buttonStyle(.plain)
                .help(loc("chat.detail"))
            }

            ToolbarItem(placement: .primaryAction) {
                Button(action: {
                    if convType == .group {
                        showGroupDetail = true
                    } else {
                        showP2PDetail = true
                    }
                }) {
                    Image(systemName: "ellipsis.circle")
                }
                .accessibilityLabel(loc("chat.detail"))
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
        .onAppear {
            vm.loadInitialMessages()
            vm.markAsReadIfActive()
        }
    }
}
