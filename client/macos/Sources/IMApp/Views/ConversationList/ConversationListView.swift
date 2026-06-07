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
            // Connection status
            if vm.connectionStatus == .disconnected || vm.connectionStatus == .connecting {
                HStack {
                    Circle()
                        .fill(vm.connectionStatus == .connecting ? Color.orange : Color.red)
                        .frame(width: 8, height: 8)
                    Text(vm.connectionStatus == .connecting ? loc("common.loading") : loc("conv.disconnected"))
                        .font(.caption)
                }
                .padding(.vertical, 4)
                .frame(maxWidth: .infinity)
                .background(Color(.windowBackgroundColor))
            }

            List(vm.conversations, selection: $selectedConvID) { conv in
                ConversationRowView(conv: conv)
            }
            .listStyle(.plain)
            .refreshable {
                vm.refresh()
            }
            .onChange(of: selectedConvID) { _, newID in
                guard let id = newID, let conv = vm.conversations.first(where: { $0.convID == id }) else { return }
                onSelectConv?(conv)
            }
        }
        .toolbar {
            ToolbarItemGroup {
                Button(action: { showNewChat = true }) {
                    Label(loc("conv.new_chat"), systemImage: "plus.bubble")
                }
            }
            ToolbarItemGroup(placement: .automatic) {
                Button(action: { showProfile = true }) {
                    Label(loc("profile.title"), systemImage: "person.circle")
                }
            }
        }
        .sheet(isPresented: $showProfile) {
            ProfileView()
                .frame(width: 320, height: 280)
        }
        .sheet(isPresented: $showNewChat) {
            NewConversationView { convID, name, convType in
                showNewChat = false
                if convType == .p2p {
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
        // Refresh list then select
        refresh()
        // Create a temporary ConvListItem for navigation
        let item = ConvListItem(convID: convID, type: .p2p, name: name)
        onSelectConv?(item)
        // Re-select by setting the convID (the list may have the real item after refresh)
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) { [weak self] in
            self?.refresh()
        }
    }
}
