import SwiftUI
import IMCore

struct GroupDetailView: View {
    let convID: String
    let convName: String
    @StateObject private var vm = GroupManagementViewModel()
    @State private var showAddMember = false
    @State private var showLeaveAlert = false
    @State private var confirmRemoveMember: ConvMember?
    @State private var errorMessage = ""
    @State private var showError = false
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
                        Text(String(format: loc("group.member_count"), vm.members.count))
                            .font(.caption)
                            .foregroundColor(.secondary)
                        Text("\(loc("profile.id_label")) \(convID)")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                            .textSelection(.enabled)
                    }
                }
                .padding(.vertical, 4)
            } header: {
                Text(loc("group.info"))
            }

            // Members
            Section {
                if vm.isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .padding()
                } else {
                    ForEach(vm.members) { member in
                        HStack(spacing: 12) {
                            let user = vm.membersInfo[member.userID]
                            AvatarView(name: user?.name ?? member.userID, url: user?.avatar ?? "", size: 40)
                            VStack(alignment: .leading, spacing: 2) {
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
                                Button(action: { confirmRemoveMember = member }) {
                                    Image(systemName: "xmark.circle")
                                        .foregroundColor(.red)
                                }
                            }
                        }
                    }
                }
            } header: {
                Text(loc("group.members_title"))
            }

            // Actions
            Section {
                Button(action: { showAddMember = true }) {
                    Label(loc("group.add_member"), systemImage: "person.badge.plus")
                }

                Button(role: .destructive, action: { showLeaveAlert = true }) {
                    Label(loc("group.leave"), systemImage: "arrow.right.square")
                }
            }

            // Join Requests (admin only)
            if vm.isAdmin {
                Section {
                    if vm.isLoadingRequests {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                            .padding()
                    } else if vm.joinRequests.isEmpty {
                        Text(loc("group.no_join_requests"))
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(vm.joinRequests) { request in
                            HStack {
                                let user = vm.membersInfo[request.userID]
                                AvatarView(name: user?.name ?? request.userID, url: user?.avatar ?? "", size: 36)
                                VStack(alignment: .leading) {
                                    Text(user?.name ?? request.userID)
                                        .fontWeight(.medium)
                                    Text(request.userID)
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                Button(loc("group.approve")) {
                                    Task {
                                        do {
                                            try await vm.approveJoinRequest(convID: convID, userID: request.userID)
                                        } catch {
                                            errorMessage = error.localizedDescription
                                            showError = true
                                        }
                                    }
                                }
                                .buttonStyle(.borderedProminent)
                                .tint(.green)

                                Button(loc("group.reject")) {
                                    Task {
                                        do {
                                            try await vm.rejectJoinRequest(convID: convID, userID: request.userID)
                                        } catch {
                                            errorMessage = error.localizedDescription
                                            showError = true
                                        }
                                    }
                                }
                                .buttonStyle(.bordered)
                                .tint(.red)
                            }
                        }
                    }
                } header: {
                    Text(loc("group.join_requests_title"))
                }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle(loc("group.info"))
        .navigationBarTitleDisplayMode(.inline)
        .sheet(isPresented: $showAddMember) {
            NavigationStack {
                AddMemberView(convID: convID, onAdd: { userID in
                    Task {
                        do {
                            try await vm.addMember(convID: convID, userID: userID)
                            showAddMember = false
                        } catch {
                            errorMessage = error.localizedDescription
                            showError = true
                        }
                    }
                })
            }
        }
        .alert(loc("group.leave"), isPresented: $showLeaveAlert) {
            Button(loc("common.cancel"), role: .cancel) {}
            Button(loc("group.leave"), role: .destructive) {
                Task {
                    do {
                        try await vm.leaveGroup(convID: convID)
                        dismiss()
                    } catch {
                        errorMessage = error.localizedDescription
                        showError = true
                    }
                }
            }
        } message: {
            Text(String(format: loc("group.leave_confirm_message"), convName))
        }
        .alert(loc("group.remove_confirm_title"), isPresented: .init(
            get: { confirmRemoveMember != nil },
            set: { if !$0 { confirmRemoveMember = nil } }
        )) {
            Button(loc("common.cancel"), role: .cancel) { confirmRemoveMember = nil }
            Button(loc("group.remove_button"), role: .destructive) {
                if let member = confirmRemoveMember {
                    Task {
                        do {
                            try await vm.removeMember(convID: convID, userID: member.userID)
                        } catch {
                            errorMessage = error.localizedDescription
                            showError = true
                        }
                    }
                }
                confirmRemoveMember = nil
            }
        } message: {
            let name = confirmRemoveMember.flatMap { member in
                let user = vm.membersInfo[member.userID]
                return user?.name ?? member.userID
            } ?? ""
            Text(String(format: loc("group.remove_confirm_message"), name))
        }
        .alert(loc("group.error_title"), isPresented: $showError) {
            Button(loc("common.confirm"), role: .cancel) {}
        } message: {
            Text(errorMessage)
        }
        .onAppear {
            vm.loadDetail(convID: convID)
            vm.loadJoinRequests(convID: convID)
        }
    }


    private func roleName(_ role: ConvRole) -> String {
        switch role {
        case .owner: return loc("group.owner")
        case .admin: return loc("group.admin")
        case .member: return loc("group.member")
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
                TextField(loc("search.placeholder"), text: $vm.query)
                    .autocapitalization(.none)
                    .disableAutocorrection(true)
            }
            .padding(10)
            .background(Color(.systemGray6))
            .clipShape(RoundedRectangle(cornerRadius: 10))
            .padding()

            List {
                if vm.results.isEmpty && !vm.query.isEmpty && !vm.isSearching {
                    Text(loc("search.no_results"))
                        .foregroundColor(.secondary)
                } else {
                    ForEach(vm.results) { user in
                        Button(action: { onAdd(user.userID) }) {
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
                            }
                        }
                    }
                }
            }
            .listStyle(.plain)
        }
        .navigationTitle(loc("group.add_member"))
        .navigationBarTitleDisplayMode(.inline)
    }
}
