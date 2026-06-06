import SwiftUI
import IMCore

struct ConversationListView: View {
    @StateObject private var vm = ConversationListViewModel()
    @State private var showCreateGroup = false

    var body: some View {
        VStack(spacing: 0) {
            // Connection status
            if vm.connectionStatus == .disconnected || vm.connectionStatus == .connecting {
                HStack {
                    Circle()
                        .fill(vm.connectionStatus == .connecting ? Color.orange : Color.red)
                        .frame(width: 6, height: 6)
                    Text(vm.connectionStatus == .connecting ? "连接中..." : "连接已断开")
                        .font(.caption)
                }
                .padding(.vertical, 4)
                .frame(maxWidth: .infinity)
                .background(Color(.systemGroupedBackground))
            }

            List {
                if vm.isLoading && vm.conversations.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .padding()
                } else if vm.conversations.isEmpty {
                    Text("暂无会话")
                        .foregroundColor(.secondary)
                        .frame(maxWidth: .infinity)
                        .padding()
                } else {
                    ForEach(vm.conversations) { conv in
                        NavigationLink {
                            ChatView(convID: conv.convID, convName: conv.name, convType: conv.type)
                        } label: {
                            ConversationRowView(conv: conv)
                        }
                    }
                }
            }
            .listStyle(.plain)
            .refreshable {
                vm.refresh()
            }
        }
        .navigationTitle("会话")
        .toolbar {
            Button(action: { showCreateGroup = true }) {
                Image(systemName: "person.3.fill")
            }
        }
        .sheet(isPresented: $showCreateGroup) {
            CreateGroupView { conv in
                showCreateGroup = false
                vm.refresh()
            }
        }
        .onAppear {
            vm.loadConversations()
            vm.connectWebSocket()
        }
    }
}
