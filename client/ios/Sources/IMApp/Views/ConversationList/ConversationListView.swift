import SwiftUI
import IMCore

struct ChatDestination: Identifiable, Hashable {
    let convID: String
    let name: String
    let type: ConvType
    var id: String { convID }
}

struct ConversationListView: View {
    @StateObject private var vm = ConversationListViewModel()
    @State private var showNewChat = false
    @State private var showProfile = false
    @State private var navigateToChat: ChatDestination?

    var body: some View {
        VStack(spacing: 0) {
            // Connection status
            if vm.connectionStatus == .disconnected || vm.connectionStatus == .connecting {
                HStack {
                    Circle()
                        .fill(vm.connectionStatus == .connecting ? Color.orange : Color.red)
                        .frame(width: 6, height: 6)
                    Text(vm.connectionStatus == .connecting ? loc("common.loading") : loc("conv.disconnected"))
                        .font(.caption)
                }
                .padding(.vertical, 4)
                .frame(maxWidth: .infinity)
                .background(Color(.systemGroupedBackground))
            }

            if vm.isLoading && vm.conversations.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if vm.conversations.isEmpty {
                VStack {
                    Spacer()
                    Text(loc("conv.no_conversations"))
                        .foregroundColor(.secondary)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    ForEach(vm.conversations) { conv in
                        NavigationLink {
                            ChatView(convID: conv.convID, convName: conv.name, convType: conv.type)
                        } label: {
                            ConversationRowView(conv: conv)
                        }
                        .listRowSeparator(.hidden)
                        .overlay(alignment: .bottom) {
                            Rectangle()
                                .fill(Color(.separator))
                                .frame(height: 0.5)
                        }
                    }
                }
                .listStyle(.plain)
                .refreshable {
                    vm.refresh()
                }
            }
        }
        .navigationTitle(loc("conv.title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItemGroup(placement: .navigationBarLeading) {
                Button(action: { showProfile = true }) {
                    Image(systemName: "person.circle")
                        .font(.body)
                }
            }
            ToolbarItemGroup(placement: .navigationBarTrailing) {
                Button(action: { showNewChat = true }) {
                    Image(systemName: "plus")
                        .font(.body)
                }
            }
        }
        .sheet(isPresented: $showNewChat) {
            NewConversationView { convID, name, convType in
                showNewChat = false
                navigateToChat = ChatDestination(convID: convID, name: name, type: convType)
            }
        }
        .navigationDestination(item: $navigateToChat) { dest in
            ChatView(convID: dest.convID, convName: dest.name, convType: dest.type)
        }
        .sheet(isPresented: $showProfile) {
            ProfileView()
        }
        .onAppear {
            vm.loadConversations()
            vm.connectWebSocket()
        }
    }
}
