import SwiftUI
import IMCore

struct CreateGroupView: View {
    @StateObject private var vm = SearchViewModel()
    @State private var groupName = ""
    @State private var selectedUsers: [User] = []
    @State private var isCreating = false
    @State private var showError = false
    @State private var errorMessage = ""
    let onCreated: (Conversation) -> Void
    let onCancel: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Button("取消") { onCancel() }
                Spacer()
                Text("创建群聊")
                    .fontWeight(.semibold)
                Spacer()
                Button("创建") {
                    createGroup()
                }
                .disabled(groupName.isEmpty || selectedUsers.isEmpty || isCreating)
            }
            .padding()

            Divider()

            // Group name
            TextField("群名称", text: $groupName)
                .textFieldStyle(.roundedBorder)
                .padding()

            // Selected users
            if !selectedUsers.isEmpty {
                ScrollView(.horizontal) {
                    HStack {
                        ForEach(selectedUsers) { user in
                            HStack(spacing: 4) {
                                Text(user.name)
                                    .font(.caption)
                                Button(action: { selectedUsers.removeAll { $0.id == user.id } }) {
                                    Image(systemName: "xmark.circle.fill")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                                .buttonStyle(.plain)
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(Color.blue.opacity(0.1))
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                    }
                    .padding(.horizontal)
                }
                .padding(.bottom, 8)
            }

            // Search
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField("搜索用户...", text: $vm.query)
                    .textFieldStyle(.plain)
                if vm.isSearching {
                    ProgressView()
                        .scaleEffect(0.5)
                }
            }
            .padding(8)
            .background(Color(.windowBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 8))
            .padding(.horizontal)
            .padding(.bottom, 8)

            // Results
            List {
                if vm.results.isEmpty && !vm.query.isEmpty && !vm.isSearching {
                    Text("未找到用户")
                        .foregroundColor(.secondary)
                } else {
                    ForEach(vm.results) { user in
                        Button(action: {
                            if !selectedUsers.contains(where: { $0.id == user.id }) {
                                selectedUsers.append(user)
                            }
                        }) {
                            HStack {
                                AvatarView(name: user.name, url: user.avatar, size: 32)
                                VStack(alignment: .leading) {
                                    Text(user.name)
                                        .fontWeight(.medium)
                                    Text(user.userID)
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                if selectedUsers.contains(where: { $0.id == user.id }) {
                                    Image(systemName: "checkmark")
                                        .foregroundColor(.blue)
                                }
                            }
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .listStyle(.plain)
        }
        .frame(width: 400, height: 500)
    }

    private func createGroup() {
        isCreating = true
        let memberIDs = [AuthManager.shared.currentUser?.userID ?? ""] + selectedUsers.map(\.userID)
        Task {
            do {
                let groupVM = GroupManagementViewModel()
                let conv = try await groupVM.createGroup(name: groupName, memberIDs: memberIDs)
                onCreated(conv)
            } catch {
                errorMessage = error.localizedDescription
                showError = true
            }
            isCreating = false
        }
    }
}
