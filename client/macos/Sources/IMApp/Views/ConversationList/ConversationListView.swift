import SwiftUI
import IMCore

struct ConversationListView: View {
    @StateObject private var vm = ConversationListViewModel()
    @State private var showCreateGroup = false
    @Binding var selectedConvID: String?
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
                    Text(vm.connectionStatus == .connecting ? "连接中..." : "连接已断开")
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
                Button(action: { showCreateGroup = true }) {
                    Label("创建群聊", systemImage: "person.3.fill")
                }
            }
        }
        .sheet(isPresented: $showCreateGroup) {
            CreateGroupView { conv in
                showCreateGroup = false
                vm.refresh()
            } onCancel: {
                showCreateGroup = false
            }
        }
        .onAppear {
            vm.loadConversations()
            vm.connectWebSocket()
        }
    }
}
