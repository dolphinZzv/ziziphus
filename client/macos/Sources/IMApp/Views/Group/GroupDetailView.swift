import SwiftUI
import IMCore

struct GroupDetailView: View {
    let convID: String
    let convName: String
    @StateObject private var vm = GroupManagementViewModel()
    @State private var showAddMember = false
    @State private var showLeaveAlert = false
    @Environment(\.dismiss) private var dismiss

    private let currentUserID = AuthManager.shared.currentUser?.userID ?? ""

    var body: some View {
        List {
            // Info section
            Section {
                HStack {
                    Circle()
                        .fill(Color.blue.opacity(0.2))
                        .frame(width: 50, height: 50)
                        .overlay {
                            Text(String(convName.prefix(1)))
                                .font(.title)
                                .fontWeight(.semibold)
                                .foregroundColor(.blue)
                        }

                    VStack(alignment: .leading) {
                        Text(convName)
                            .fontWeight(.medium)
                        Text("\(vm.members.count) 位成员")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
                .padding(.vertical, 4)
            } header: {
                Text("群信息")
            }

            // Members
            Section {
                if vm.isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .padding()
                } else {
                    ForEach(vm.members) { member in
                        HStack(spacing: 10) {
                            let user = vm.membersInfo[member.userID]
                            AvatarView(name: user?.name ?? member.userID, url: user?.avatar ?? "", size: 36)
                            VStack(alignment: .leading) {
                                Text(user?.name ?? member.userID)
                                    .fontWeight(.medium)
                                if let nickname = member.nickname, !nickname.isEmpty {
                                    Text(nickname)
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                                Text(roleName(member.role))
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                            }
                            Spacer()
                            if canRemove(member) {
                                Button(action: {
                                    Task { try? await vm.removeMember(convID: convID, userID: member.userID) }
                                }) {
                                    Image(systemName: "xmark.circle")
                                        .foregroundColor(.red)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            } header: {
                Text("群成员")
            }

            // Actions
            Section {
                Button(action: { showAddMember = true }) {
                    Label("添加成员", systemImage: "person.badge.plus")
                }

                Button(action: { showLeaveAlert = true }) {
                    Label("退出群聊", systemImage: "arrow.right.square")
                        .foregroundColor(.red)
                }
            }
        }
        .listStyle(.inset)
        .frame(width: 360, height: 450)
        .sheet(isPresented: $showAddMember) {
            AddMemberView(convID: convID, onAdd: { userID in
                Task { try? await vm.addMember(convID: convID, userID: userID) }
                showAddMember = false
            })
        }
        .alert("退出群聊", isPresented: $showLeaveAlert) {
            Button("取消", role: .cancel) {}
            Button("退出", role: .destructive) {
                Task {
                    try? await vm.leaveGroup(convID: convID)
                    dismiss()
                }
            }
        } message: {
            Text("确定要退出群聊「\(convName)」吗？")
        }
        .onAppear { vm.loadDetail(convID: convID) }
    }

    private func roleName(_ role: ConvRole) -> String {
        switch role {
        case .owner: return "群主"
        case .admin: return "管理员"
        case .member: return "成员"
        }
    }

    private func canRemove(_ member: ConvMember) -> Bool {
        let currentRole = vm.members.first(where: { $0.userID == currentUserID })?.role ?? .member
        if member.userID == currentUserID { return false }
        switch currentRole {
        case .owner: return true
        case .admin: return member.role == .member
        case .member: return false
        }
    }
}

// MARK: - Add Member Sheet
private struct AddMemberView: View {
    let convID: String
    let onAdd: (String) -> Void
    @StateObject private var vm = SearchViewModel()

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField("搜索用户...", text: $vm.query)
                    .textFieldStyle(.plain)
            }
            .padding(8)
            .background(Color(.windowBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 8))
            .padding()

            List {
                if vm.results.isEmpty && !vm.query.isEmpty && !vm.isSearching {
                    Text("未找到用户")
                        .foregroundColor(.secondary)
                } else {
                    ForEach(vm.results) { user in
                        Button(action: { onAdd(user.userID) }) {
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
                            }
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .listStyle(.plain)
        }
        .frame(width: 320, height: 400)
    }
}
