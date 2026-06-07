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
    @EnvironmentObject private var localizationManager: LocalizationManager

    let onCreated: (String, String, ConvType) -> Void
    let onCancel: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Button(loc("common.cancel")) { onCancel() }
                Spacer()
                Text(loc("conv.new_chat"))
                    .fontWeight(.semibold)
                Spacer()
                if selectedTab == .group {
                    Button(loc("group.create_button")) { createGroup() }
                        .disabled(groupName.isEmpty || selectedUsers.isEmpty || isCreating)
                }
            }
            .padding()

            Divider()

            // Tab picker
            Picker("", selection: $selectedTab) {
                ForEach(Tab.allCases, id: \.self) { tab in
                    Text(tab.label).tag(tab)
                }
            }
            .pickerStyle(.segmented)
            .padding(.horizontal)
            .padding(.vertical, 8)

            if selectedTab == .p2p {
                p2pContent
            } else {
                groupContent
            }
        }
        .frame(width: 400, height: 500)
    }

    // MARK: - P2P Content

    @ViewBuilder
    private var p2pContent: some View {
        // Search
        HStack {
            Image(systemName: "magnifyingglass")
                .foregroundColor(.secondary)
            TextField(loc("search.placeholder"), text: $searchVM.query)
                .textFieldStyle(.plain)
            if searchVM.isSearching {
                ProgressView()
                    .scaleEffect(0.5)
            }
        }
        .padding(8)
        .background(Color(.windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 8))
        .padding()

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
                    .buttonStyle(.plain)
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
            TextField(loc("group.add_member_search"), text: $groupSearchVM.query)
                .textFieldStyle(.plain)
            if groupSearchVM.isSearching {
                ProgressView()
                    .scaleEffect(0.5)
            }
        }
        .padding(8)
        .background(Color(.windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 8))
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

    // MARK: - Actions

    private func startP2PChat(user: User) {
        isCreating = true
        errorMessage = nil
        Task {
            do {
                let (convID, name) = try await ConversationService.shared.createP2P(userID: user.userID)
                isCreating = false
                onCreated(convID, name.isEmpty ? user.name : name, .p2p)
            } catch {
                errorMessage = error.localizedDescription
                isCreating = false
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
                onCreated(conv.convID, conv.name, .group)
            } catch {
                errorMessage = error.localizedDescription
            }
            isCreating = false
        }
    }
}
