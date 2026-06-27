import SwiftUI
import IMCore

struct CreateGroupView: View {
    @StateObject private var searchVM = SearchViewModel()
    @State private var groupName = ""
    @State private var selectedUsers: [User] = []
    @State private var errorMessage: String?
    @State private var isCreating = false

    @EnvironmentObject private var localizationManager: LocalizationManager
    let onCreated: (String, String, ConvType) -> Void
    let onCancel: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Button(loc("common.cancel")) { onCancel() }
                    .buttonStyle(.plain)
                Spacer()
                Text(loc("conv.new_group"))
                    .font(.appleBodySemibold)
                Spacer()
                Button(loc("group.create_button")) { createGroup() }
                    .disabled(groupName.isEmpty || selectedUsers.isEmpty || isCreating)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)

            Divider()

            // Group name
            TextField(loc("group.name_placeholder"), text: $groupName)
                .textFieldStyle(.roundedBorder)
                .padding(.horizontal)
                .padding(.vertical, 8)

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
                TextField(loc("group.add_member_search"), text: $searchVM.query)
                    .textFieldStyle(.plain)
                if searchVM.isSearching {
                    ProgressView()
                        .scaleEffect(0.5)
                }
            }
            .padding(8)
            .background(Color(.windowBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 8))
            .padding(.horizontal)
            .padding(.bottom, 8)

            if let error = errorMessage {
                Text(error)
                    .foregroundColor(.red)
                    .font(.callout)
                    .padding(.horizontal)
            }

            List {
                if searchVM.results.isEmpty && !searchVM.query.isEmpty && !searchVM.isSearching {
                    Text(loc("search.no_results"))
                        .foregroundColor(.secondary)
                } else {
                    ForEach(searchVM.results) { user in
                        Button(action: {
                            if !selectedUsers.contains(where: { $0.id == user.id }) {
                                selectedUsers.append(user)
                            }
                        }) {
                            HStack {
                                AvatarView(name: user.name, url: user.avatar, size: 32)
                                VStack(alignment: .leading) {
                                    Text(user.name)
                                        .font(.appleBodySemibold)
                                    Text(user.userID)
                                        .font(.appleCaption)
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
                onCreated(conv.convID, conv.name, .group)
            } catch {
                errorMessage = error.localizedDescription
            }
            isCreating = false
        }
    }
}
