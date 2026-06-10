import SwiftUI
import IMCore

struct JoinGroupView: View {
    @State private var joinGroupQuery = ""
    @State private var joinGroupResults: [GroupSearchItem] = []
    @State private var isSearchingGroups = false
    @State private var joiningGroupID: String?
    @State private var errorMessage: String?

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
                Text(loc("group.join_request"))
                    .font(.appleBodySemibold)
                Spacer()
                // Empty spacer for alignment
                Rectangle()
                    .fill(.clear)
                    .frame(width: 40)
            }
            .padding()

            Divider()

            // Search
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField(loc("search.group_placeholder"), text: $joinGroupQuery)
                    .textFieldStyle(.plain)
                if isSearchingGroups {
                    ProgressView()
                        .scaleEffect(0.5)
                }
            }
            .padding(8)
            .background(Color(.windowBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 8))
            .padding()
            .onChange(of: joinGroupQuery) { _, newValue in
                searchGroups(query: newValue)
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
                                    .font(.appleBodySemibold)
                                Text(String(format: loc("group.member_count"), group.memberCount))
                                    .font(.appleCaption)
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
        .frame(width: 400, height: 500)
    }

    private func searchGroups(query: String) {
        guard !query.trimmingCharacters(in: .whitespaces).isEmpty else {
            joinGroupResults = []
            return
        }
        isSearchingGroups = true
        errorMessage = nil
        Task {
            do {
                let results = try await ConversationService.shared.searchGroups(query: query)
                joinGroupResults = results
            } catch {
                errorMessage = error.localizedDescription
            }
            isSearchingGroups = false
        }
    }

    private func requestJoinGroup(convID: String, name: String) {
        joiningGroupID = convID
        errorMessage = nil
        Task {
            do {
                try await ConversationService.shared.requestJoin(convID: convID)
                joiningGroupID = nil
                onCreated(convID, name, .group)
            } catch {
                errorMessage = error.localizedDescription
                joiningGroupID = nil
            }
        }
    }
}
