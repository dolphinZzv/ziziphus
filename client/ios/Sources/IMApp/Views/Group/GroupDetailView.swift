import SwiftUI
import IMCore
import PhotosUI

struct GroupDetailView: View {
    let convID: String
    let convName: String
    @StateObject private var vm = GroupManagementViewModel()
    @State private var showAddMember = false
    @State private var showLeaveAlert = false
    @State private var confirmRemoveMember: ConvMember?
    @State private var errorMessage = ""
    @State private var showError = false
    @State private var showImagePicker = false
    @State private var selectedPhotoItem: PhotosPickerItem?
    @State private var isUploadingAvatar = false
    @State private var isEditingName = false
    @State private var editedName = ""
    @State private var isSavingName = false
    @State private var displayName = ""
    @Environment(\.dismiss) private var dismiss

    private let currentUserID = AuthManager.shared.currentUser?.userID ?? ""

    var body: some View {
        List {
            // Info section
            Section {
                HStack {
                    ZStack(alignment: .bottomTrailing) {
                        AvatarView(name: displayName, url: vm.conversationAvatar, size: 50)

                        if vm.isAdmin {
                            Circle()
                                .fill(Color.blue)
                                .frame(width: 20, height: 20)
                                .overlay {
                                    Image(systemName: "camera.fill")
                                        .font(.system(size: 10))
                                        .foregroundColor(.white)
                                }
                        }
                    }
                    .onTapGesture {
                        if vm.isAdmin { showImagePicker = true }
                    }

                    VStack(alignment: .leading) {
                        HStack {
                            if isEditingName {
                                TextField(loc("group.name_placeholder"), text: $editedName)
                                    .textFieldStyle(.roundedBorder)
                                    .fontWeight(.medium)
                                    .onSubmit { saveGroupName() }
                            } else {
                                Text(displayName)
                                    .fontWeight(.medium)
                                if vm.isAdmin {
                                    Button(action: {
                                        editedName = displayName
                                        isEditingName = true
                                    }) {
                                        Image(systemName: "pencil")
                                            .font(.caption)
                                            .foregroundColor(.blue)
                                    }
                                }
                            }
                        }
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
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                if isEditingName {
                    Button(loc("common.save")) { saveGroupName() }
                        .disabled(editedName.trimmingCharacters(in: .whitespaces).isEmpty || isSavingName)
                }
            }
            ToolbarItem(placement: .navigationBarLeading) {
                if isEditingName {
                    Button(loc("common.cancel")) {
                        isEditingName = false
                        editedName = displayName
                    }
                }
            }
        }
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
            Text(String(format: loc("group.leave_confirm_message"), displayName))
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
        .photosPicker(isPresented: $showImagePicker, selection: $selectedPhotoItem, matching: .images)
        .onChange(of: selectedPhotoItem) { _, newItem in
            guard let item = newItem else { return }
            Task {
                guard let data = try? await item.loadTransferable(type: Data.self) else { return }
                await uploadGroupAvatar(data: data)
                selectedPhotoItem = nil
            }
        }
        .overlay {
            if isUploadingAvatar {
                ProgressView()
                    .padding()
                    .background(.ultraThinMaterial)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
            }
        }
        .onAppear {
            displayName = convName
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

    private func uploadGroupAvatar(data: Data) async {
        await MainActor.run { isUploadingAvatar = true }
        do {
            let finfo = try await APIClient.shared.uploadFile(fileData: data, fileName: "group_avatar.jpg", fileType: 0)
            try await ConversationService.shared.updateGroup(convID: convID, name: displayName, avatar: finfo.url)
            await MainActor.run {
                vm.conversationAvatar = finfo.url
            }
        } catch {
            await MainActor.run {
                errorMessage = error.localizedDescription
                showError = true
            }
        }
        await MainActor.run { isUploadingAvatar = false }
    }

    private func saveGroupName() {
        let newName = editedName.trimmingCharacters(in: .whitespaces)
        guard !newName.isEmpty else { return }
        isSavingName = true
        Task {
            do {
                try await ConversationService.shared.updateGroup(convID: convID, name: newName)
                await MainActor.run {
                    displayName = newName
                    isEditingName = false
                    isSavingName = false
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isSavingName = false
                }
            }
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
