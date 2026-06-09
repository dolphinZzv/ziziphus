import SwiftUI
import IMCore

struct ConversationListView: View {
    @StateObject private var vm = ConversationListViewModel()
    @State private var showNewChat = false
    @State private var showProfile = false
    @Binding var selectedConvID: String?
    @EnvironmentObject private var localizationManager: LocalizationManager
    let onSelectConv: ((ConvListItem) -> Void)?

    init(selectedConvID: Binding<String?>, onSelectConv: ((ConvListItem) -> Void)? = nil) {
        _selectedConvID = selectedConvID
        self.onSelectConv = onSelectConv
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header: user name + new chat button
            HStack {
                Button(action: { showProfile = true }) {
                    Text(AuthManager.shared.currentUser?.name ?? "")
                        .font(.appleBodySemibold)
                        .foregroundColor(AppleDesign.Colors.ink)
                }
                .buttonStyle(.plain)
                .help(loc("profile.title"))

                Spacer()

                Button(action: { showNewChat = true }) {
                    Image(systemName: "plus")
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.actionBlue)
                }
                .buttonStyle(.plain)
                .help(loc("conv.new_chat"))
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)

            // Connection status
            if vm.connectionStatus == .disconnected || vm.connectionStatus == .connecting {
                HStack {
                    Circle()
                        .fill(vm.connectionStatus == .connecting ? Color.orange : Color.red)
                        .frame(width: 6, height: 6)
                    Text(vm.connectionStatus == .connecting ? loc("common.loading") : loc("conv.disconnected"))
                        .font(.system(size: AppleDesign.Typography.finePrintSize))
                }
                .padding(.vertical, 4)
                .frame(maxWidth: .infinity)
                .background(AppleDesign.Colors.parchment)
            }

            List {
                ForEach(vm.conversations, id: \.convID) { conv in
                    ConversationRowView(conv: conv)
                        .listRowInsets(EdgeInsets(top: 0, leading: 16, bottom: 0, trailing: 16))
                        .listRowSeparator(.hidden)
                        .listRowBackground(
                            selectedConvID == conv.convID
                                ? AppleDesign.Colors.actionBlue.opacity(0.1)
                                : Color.clear
                        )
                        .contentShape(Rectangle())
                        .onTapGesture {
                            selectedConvID = conv.convID
                            onSelectConv?(conv)
                        }
                }
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)
            .refreshable {
                vm.refresh()
            }
        }
        .background(AppleDesign.Colors.parchment.ignoresSafeArea())
        .sheet(isPresented: $showProfile) {
            ProfileView()
                .frame(width: 340, height: 460)
        }
        .sheet(isPresented: $showNewChat) {
            NewConversationView { convID, name, convType in
                showNewChat = false
                if convType == .p2p {
                    selectedConvID = convID
                    vm.selectConversation(convID: convID, name: name, onSelectConv: onSelectConv)
                } else {
                    vm.refresh()
                }
            } onCancel: {
                showNewChat = false
            }
        }
        .onAppear {
            vm.loadConversations()
            vm.connectWebSocket()
        }
    }
}

// MARK: - ViewModel Helper
extension ConversationListViewModel {
    func selectConversation(convID: String, name: String, onSelectConv: ((ConvListItem) -> Void)?) {
        refresh()
        let item = ConvListItem(convID: convID, type: .p2p, name: name)
        onSelectConv?(item)
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) { [weak self] in
            self?.refresh()
        }
    }
}
