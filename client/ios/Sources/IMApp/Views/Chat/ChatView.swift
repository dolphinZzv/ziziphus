import SwiftUI
import IMCore

struct ChatView: View {
    let convID: String
    let convName: String
    let convType: ConvType
    @StateObject private var vm: ChatViewModel
    @State private var showGroupDetail = false

    init(convID: String, convName: String, convType: ConvType = .p2p) {
        self.convID = convID
        self.convName = convName
        self.convType = convType
        _vm = StateObject(wrappedValue: ChatViewModel(convID: convID))
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 0) {
                        if vm.isLoadingHistory {
                            ProgressView()
                                .padding()
                        }

                        ForEach(vm.messages) { msg in
                            MessageBubble(message: msg)
                                .id(msg.id)
                        }

                        if vm.isTyping {
                            TypingIndicator()
                                .padding(.leading, 12)
                        }
                    }
                    .padding(.horizontal)
                }
                .onChange(of: vm.messages.count) { _, _ in
                    if let last = vm.messages.last {
                        proxy.scrollTo(last.id, anchor: .bottom)
                    }
                }
            }

            InputBarView(text: $vm.inputText, onSend: {
                vm.sendMessage()
            }, onTyping: {
                vm.userDidStartTyping()
            })
        }
        .navigationTitle(convName)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if convType == .group {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { showGroupDetail = true }) {
                        Image(systemName: "info.circle")
                    }
                }
            }
        }
        .sheet(isPresented: $showGroupDetail) {
            NavigationStack {
                GroupDetailView(convID: convID, convName: convName)
            }
        }
        .onAppear {
            vm.loadInitialMessages()
            vm.markAsReadIfActive()
        }
    }
}
