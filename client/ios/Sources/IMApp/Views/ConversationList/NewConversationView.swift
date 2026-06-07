import SwiftUI
import IMCore

struct NewConversationView: View {
    enum Tab: String, CaseIterable {
        case p2p
        case group

        var label: String {
            switch self {
            case .p2p: return loc("conv.new_chat")
            case .group: return loc("conv.new_group")
            }
        }
    }

    @State private var selectedTab: Tab = .p2p
    @State private var errorMessage: String?
    @State private var isCreating = false

    // P2P
    @StateObject private var searchVM = SearchViewModel()

    // Group
    @StateObject private var groupSearchVM = SearchViewModel()
    @State private var groupName = ""
    @State private var selectedUsers: [User] = []

    @Environment(\.dismiss) private var dismiss
    let onCreated: (String, String, ConvType) -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Tab picker
                Picker("", selection: $selectedTab) {
                    ForEach(Tab.allCases, id: \.self) { tab in
                        Text(tab.label).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .padding()

                if selectedTab == .p2p {
                    p2pContent
                } else {
                    groupContent
                }
            }
            .navigationTitle(loc("conv.new_chat"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(loc("common.cancel")) { dismiss() }
                }
                if selectedTab == .group {
                    ToolbarItem(placement: .confirmationAction) {
                        Button(loc("group.create_button")) { createGroup() }
                            .disabled(groupName.isEmpty || selectedUsers.isEmpty || isCreating)
                    }
                }
            }
            .alert(loc("group.error_title"), isPresented: $showError) {
                Button(loc("common.confirm"), role: .cancel) {}
            } message: {
                Text(errorMessage ?? "")
            }
        }
    }

    @State private var showError = false

    // MARK: - P2P Content

    @ViewBuilder
    private var p2pContent: some View {
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

        // Error
        if let error = errorMessage {
            Text(error)
                .foregroundColor(.red)
                .font(.callout)
                .padding(.horizontal)
        }

        // Results
        List {
            if searchVM.results.isEmpty && !searchVM.query.isEmpty && !searchVM.isSearching {
                Text(loc("search.no_results"))
                    .foregroundColor(.secondary)
            } else {
                ForEach(searchVM.results) { user in
                    Button(action: { startP2PChat(user: user) }) {
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
                            if isCreating {
                                ProgressView()
                                    .scaleEffect(0.7)
                            }
                        }
                    }
                    .disabled(isCreating)
                }
            }
        }
        .listStyle(.plain)
    }

    // MARK: - Group Content

    @ViewBuilder
    private var groupContent: some View {
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
            TextField(loc("search.placeholder"), text: $groupSearchVM.query)
                .autocapitalization(.none)
                .disableAutocorrection(true)
            if groupSearchVM.isSearching {
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
            if groupSearchVM.results.isEmpty && !groupSearchVM.query.isEmpty && !groupSearchVM.isSearching {
                Text(loc("search.no_results"))
                    .foregroundColor(.secondary)
            } else {
                ForEach(groupSearchVM.results) { user in
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

    // MARK: - Actions

    private func startP2PChat(user: User) {
        isCreating = true
        errorMessage = nil
        Task {
            do {
                let (convID, name) = try await ConversationService.shared.createP2P(userID: user.userID)
                isCreating = false
                onCreated(convID, name.isEmpty ? user.name : name, .p2p)
                dismiss()
            } catch {
                errorMessage = error.localizedDescription
                isCreating = false
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
