import SwiftUI
import IMCore
import UniformTypeIdentifiers

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
    @State private var isUploadingAvatar = false
    @State private var isEditingName = false
    @State private var editedName = ""
    @State private var isSavingName = false
    @State private var displayName = ""
    @EnvironmentObject private var localizationManager: LocalizationManager
    @Environment(\.dismiss) private var dismiss

    private let currentUserID = AuthManager.shared.currentUser?.userID ?? ""

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text(loc("profile.title"))
                    .font(.appleBodySemibold)
                    .foregroundColor(AppleDesign.Colors.ink)
                Spacer()
                if isEditingName {
                    Button(loc("common.save")) { saveGroupName() }
                        .disabled(editedName.trimmingCharacters(in: .whitespaces).isEmpty || isSavingName)
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.actionBlue)
                    Button(loc("common.cancel")) {
                        isEditingName = false
                        editedName = displayName
                    }
                    .font(.appleBody)
                    .foregroundColor(AppleDesign.Colors.actionBlue)
                } else {
                    Button(loc("common.close")) { dismiss() }
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.actionBlue)
                }
            }
            .padding(AppleDesign.Spacing.lg)

            Divider()

            List {
                // Info section
                Section {
                    HStack {
                        ZStack(alignment: .bottomTrailing) {
                            AvatarView(name: displayName, url: vm.conversationAvatar, size: 50)

                            if vm.isAdmin {
                                Circle()
                                    .fill(Color.blue)
                                    .frame(width: 18, height: 18)
                                    .overlay {
                                        Image(systemName: "camera.fill")
                                            .font(.system(size: 9))
                                            .foregroundColor(.white)
                                    }
                            }
                        }
                        .onTapGesture {
                            if vm.isAdmin { showImagePicker = true }
                        }

                        VStack(alignment: .leading) {
                            if isEditingName {
                                TextField(loc("group.name_placeholder"), text: $editedName)
                                    .textFieldStyle(.roundedBorder)
                                    .font(.appleBodySemibold)
                                    .onSubmit { saveGroupName() }
                            } else {
                                HStack {
                                    Text(displayName)
                                        .font(.appleBodySemibold)
                                    if vm.isAdmin {
                                        Button(action: {
                                            editedName = displayName
                                            isEditingName = true
                                        }) {
                                            Image(systemName: "pencil")
                                                .font(.caption)
                                                .foregroundColor(AppleDesign.Colors.actionBlue)
                                        }
                                        .buttonStyle(.plain)
                                    }
                                }
                            }
                            Text(String(format: loc("group.member_count"), vm.members.count))
                                .font(.appleCaption)
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                            Text("\(loc("profile.id_label")) \(convID)")
                                .font(.appleFinePrint)
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                                .textSelection(.enabled)
                        }
                    }
                    .padding(.vertical, 4)
                } header: {
                    Text(loc("group.info"))
                        .font(.appleCaption)
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
                                        .font(.appleBodySemibold)
                                    if let nickname = member.nickname, !nickname.isEmpty {
                                        Text(nickname)
                                            .font(.appleCaption)
                                            .foregroundColor(AppleDesign.Colors.inkMuted)
                                    }
                                    Text(roleName(member.role))
                                        .font(.appleFinePrint)
                                        .foregroundColor(AppleDesign.Colors.inkMuted)
                                }
                                Spacer()
                                if canRemove(member) {
                                    Button(action: {
                                        confirmRemoveMember = member
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
                    Text(loc("group.members_title"))
                        .font(.appleCaption)
                }

                // Actions
                Section {
                    Button(action: { showAddMember = true }) {
                        Label(loc("group.add_member"), systemImage: "person.badge.plus")
                            .font(.appleBody)
                    }

                    Button(action: { showLeaveAlert = true }) {
                        Label(loc("group.leave"), systemImage: "arrow.right.square")
                            .font(.appleBody)
                            .foregroundColor(.red)
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
                                .font(.appleCaption)
                        } else {
                            ForEach(vm.joinRequests) { request in
                                HStack {
                                    let user = vm.membersInfo[request.userID]
                                    AvatarView(name: user?.name ?? request.userID, url: user?.avatar ?? "", size: 32)
                                    VStack(alignment: .leading) {
                                        Text(user?.name ?? request.userID)
                                            .font(.appleBodySemibold)
                                        Text(request.userID)
                                            .font(.appleFinePrint)
                                            .foregroundColor(AppleDesign.Colors.inkMuted)
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
                                    .controlSize(.small)
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
                                    .controlSize(.small)
                                    .tint(.red)
                                }
                            }
                        }
                    } header: {
                        Text(loc("group.join_requests_title"))
                            .font(.appleCaption)
                    }
                }
            }
            .listStyle(.inset)
        }
        .frame(width: 400, height: 580)
        .sheet(isPresented: $showAddMember) {
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
            Text(confirmRemoveMember.map { String(format: loc("group.remove_confirm_message"), $0.nickname ?? $0.userID) } ?? "")
        }
        .alert(loc("group.error_title"), isPresented: $showError) {
            Button(loc("common.confirm"), role: .cancel) {}
        } message: {
            Text(errorMessage)
        }
        .fileImporter(isPresented: $showImagePicker, allowedContentTypes: [.image]) { result in
            switch result {
            case .success(let url):
                guard url.startAccessingSecurityScopedResource() else { return }
                defer { url.stopAccessingSecurityScopedResource() }
                guard let data = try? Data(contentsOf: url) else { return }
                Task { await uploadGroupAvatar(data: data) }
            case .failure:
                break
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
                TextField(loc("group.add_member_search"), text: $vm.query)
                    .textFieldStyle(.plain)
            }
            .padding(8)
            .background(Color(.windowBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 8))
            .padding()

            List {
                if vm.results.isEmpty && !vm.query.isEmpty && !vm.isSearching {
                    Text(loc("group.no_results"))
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
