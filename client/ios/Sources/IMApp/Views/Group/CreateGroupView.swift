import SwiftUI
import IMCore

struct CreateGroupView: View {
    @StateObject private var vm = SearchViewModel()
    @State private var groupName = ""
    @State private var selectedUsers: [User] = []
    @State private var isCreating = false
    @State private var showError = false
    @State private var errorMessage = ""
    @Environment(\.dismiss) private var dismiss
    let onCreated: (Conversation) -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
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
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                    if vm.isSearching {
                        ProgressView()
                            .scaleEffect(0.5)
                    }
                }
                .padding(10)
                .background(Color(.systemGray6))
                .clipShape(RoundedRectangle(cornerRadius: 10))
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
                                    AvatarView(name: user.name, url: user.avatar, size: 36)
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
                        }
                    }
                }
                .listStyle(.plain)
            }
            .navigationTitle("创建群聊")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("创建") { createGroup() }
                        .disabled(groupName.isEmpty || selectedUsers.isEmpty || isCreating)
                }
            }
            .alert("创建失败", isPresented: $showError) {
                Button("确定", role: .cancel) {}
            } message: {
                Text(errorMessage)
            }
        }
    }

    private func createGroup() {
        isCreating = true
        let memberIDs = [AuthManager.shared.currentUser?.userID ?? ""] + selectedUsers.map(\.userID)
        Task {
            do {
                let groupVM = GroupManagementViewModel()
                let conv = try await groupVM.createGroup(name: groupName, memberIDs: memberIDs)
                onCreated(conv)
                dismiss()
            } catch {
                errorMessage = error.localizedDescription
                showError = true
            }
            isCreating = false
        }
    }
}
