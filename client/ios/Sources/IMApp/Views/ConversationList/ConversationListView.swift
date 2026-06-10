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
    @State private var showCreateGroup = false
    @State private var showJoinGroup = false
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

            // Error banner
            if let err = vm.errorMessage {
                HStack(spacing: 6) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.caption)
                        .foregroundColor(.red)
                    Text(err)
                        .font(.caption)
                        .foregroundColor(.red)
                        .lineLimit(1)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 16)
                .padding(.vertical, 6)
                .background(.red.opacity(0.08))
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
                Menu {
                    Button(action: { showNewChat = true }) {
                        Label(loc("conv.new_chat"), systemImage: "plus.message")
                    }
                    Button(action: { showCreateGroup = true }) {
                        Label(loc("conv.new_group"), systemImage: "person.3")
                    }
                    Button(action: { showJoinGroup = true }) {
                        Label(loc("group.join_request"), systemImage: "person.badge.plus")
                    }
                } label: {
                    Image(systemName: "plus")
                        .font(.body)
                }
            }
        }
        .sheet(isPresented: $showNewChat) {
            NewChatView { convID, name, convType in
                showNewChat = false
                navigateToChat = ChatDestination(convID: convID, name: name, type: convType)
            }
        }
        .sheet(isPresented: $showCreateGroup) {
            CreateGroupView { convID, name, convType in
                showCreateGroup = false
                vm.refresh()
                navigateToChat = ChatDestination(convID: convID, name: name, type: convType)
            }
        }
        .sheet(isPresented: $showJoinGroup) {
            JoinGroupView { convID, name, convType in
                showJoinGroup = false
                navigateToChat = ChatDestination(convID: convID, name: name, type: convType)
            }
        }
        .navigationDestination(item: $navigateToChat) { dest in
            ChatView(convID: dest.convID, convName: dest.name, convType: dest.type)
        }
        .sheet(isPresented: $showProfile) {
            NavigationStack {
                ProfileView()
            }
        }
        .onAppear {
            vm.loadConversations()
            vm.connectWebSocket()
        }
    }
}
