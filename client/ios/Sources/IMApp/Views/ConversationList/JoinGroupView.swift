import SwiftUI
import IMCore

struct JoinGroupView: View {
    @State private var joinGroupQuery = ""
    @State private var joinGroupResults: [GroupSearchItem] = []
    @State private var isSearchingGroups = false
    @State private var joiningGroupID: String?
    @State private var errorMessage: String?
    @State private var showError = false
    @State private var searchGroupsTask: Task<Void, Never>?

    @Environment(\.dismiss) private var dismiss
    let onCreated: (String, String, ConvType) -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Search
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField(loc("search.group_placeholder"), text: $joinGroupQuery)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                    if isSearchingGroups {
                        ProgressView()
                            .scaleEffect(0.5)
                    }
                }
                .padding(10)
                .background(Color(.systemGray6))
                .clipShape(RoundedRectangle(cornerRadius: 10))
                .padding()
                .onChange(of: joinGroupQuery) { _, newValue in
                    searchGroupsTask?.cancel()
                    let q = newValue
                    searchGroupsTask = Task {
                        try? await Task.sleep(nanoseconds: 300_000_000)
                        guard !Task.isCancelled else { return }
                        await searchGroups(query: q)
                    }
                }

                if let error = errorMessage {
                    Text(error)
                        .foregroundColor(.red)
                        .font(.callout)
                        .padding(.horizontal)
                }

                List {
                    if joinGroupResults.isEmpty && !joinGroupQuery.isEmpty && !isSearchingGroups {
                        Text(loc("search.no_results"))
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(joinGroupResults) { group in
                            HStack {
                                AvatarView(name: group.name, url: group.avatar, size: 36)
                                VStack(alignment: .leading) {
                                    Text(group.name)
                                        .fontWeight(.medium)
                                    Text(String(format: loc("group.member_count"), group.memberCount))
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                if joiningGroupID == group.convID {
                                    ProgressView()
                                        .scaleEffect(0.7)
                                } else {
                                    Button(loc("group.join_request")) {
                                        requestJoinGroup(convID: group.convID, name: group.name)
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .controlSize(.small)
                                }
                            }
                        }
                    }
                }
                .listStyle(.plain)
            }
            .navigationTitle(loc("group.join_request"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(loc("common.cancel")) { dismiss() }
                }
            }
            .alert(loc("group.error_title"), isPresented: $showError) {
                Button(loc("common.confirm"), role: .cancel) {}
            } message: {
                Text(errorMessage ?? "")
            }
        }
    }

    private func searchGroups(query: String) async {
        guard !query.trimmingCharacters(in: .whitespaces).isEmpty else {
            joinGroupResults = []
            return
        }
        isSearchingGroups = true
        errorMessage = nil
        do {
            let results = try await ConversationService.shared.searchGroups(query: query)
            joinGroupResults = results
        } catch {
            guard !Task.isCancelled else { return }
            errorMessage = error.localizedDescription
            showError = true
        }
        isSearchingGroups = false
    }

    private func requestJoinGroup(convID: String, name: String) {
        joiningGroupID = convID
        errorMessage = nil
        Task {
            do {
                try await ConversationService.shared.requestJoin(convID: convID)
                joiningGroupID = nil
                onCreated(convID, name, .group)
                dismiss()
            } catch {
                errorMessage = error.localizedDescription
                showError = true
                joiningGroupID = nil
            }
        }
    }
}
