import SwiftUI
import IMCore

struct CreateGroupView: View {
    @StateObject private var searchVM = SearchViewModel()
    @State private var groupName = ""
    @State private var selectedUsers: [User] = []
    @State private var errorMessage: String?
    @State private var isCreating = false
    @State private var showError = false

    @Environment(\.dismiss) private var dismiss
    let onCreated: (String, String, ConvType) -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Group name
                TextField(loc("group.name_placeholder"), text: $groupName)
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
                    TextField(loc("search.placeholder"), text: $searchVM.query)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                    if searchVM.isSearching {
                        ProgressView()
                            .scaleEffect(0.5)
                    }
                }
                .padding(10)
                .background(Color(.systemGray6))
                .clipShape(RoundedRectangle(cornerRadius: 10))
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
            .navigationTitle(loc("conv.new_group"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(loc("common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(loc("group.create_button")) { createGroup() }
                        .disabled(groupName.isEmpty || selectedUsers.isEmpty || isCreating)
                }
            }
            .alert(loc("group.error_title"), isPresented: $showError) {
                Button(loc("common.confirm"), role: .cancel) {}
            } message: {
                Text(errorMessage ?? "")
            }
        }
    }

    private func createGroup() {
        isCreating = true
        errorMessage = nil
        let memberIDs = [AuthManager.shared.currentUser?.userID ?? ""] + selectedUsers.map(\.userID)
        Task {
            do {
                let groupVM = GroupManagementViewModel()
                let conv = try await groupVM.createGroup(name: groupName, memberIDs: memberIDs)
                onCreated(conv.convID, conv.name, .group)
                dismiss()
            } catch {
                errorMessage = error.localizedDescription
                showError = true
            }
            isCreating = false
        }
    }
}
