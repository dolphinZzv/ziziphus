import SwiftUI
import IMCore

struct P2PDetailView: View {
    let convID: String
    let convName: String
    @State private var otherUser: User?
    @State private var isLoading = true
    @Environment(\.dismiss) private var dismiss

    private var otherUserID: String {
        let currentID = AuthManager.shared.currentUser?.userID ?? ""
        // convID format: "userA:userB" for P2P conversations
        let parts = convID.split(separator: ":", maxSplits: 1).map(String.init)
        if parts.count == 2 {
            return parts.first { $0 != currentID } ?? parts[0]
        }
        // Fallback: if convID is not in expected format, use convName
        return convName
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Spacer()

                if isLoading {
                    ProgressView()
                } else {
                    AvatarView(
                        name: otherUser?.name ?? convName,
                        url: otherUser?.avatar ?? "",
                        size: 80
                    )

                    Text(otherUser?.name ?? convName)
                        .font(.title2)
                        .fontWeight(.semibold)

                    HStack(spacing: 4) {
                        Text(loc("profile.account_label"))
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                        Text(otherUserID)
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                            .textSelection(.enabled)
                    }

                    HStack(spacing: 4) {
                        Text(loc("profile.id_label"))
                            .font(.caption)
                            .foregroundColor(.secondary)
                        Text(convID)
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .textSelection(.enabled)
                    }
                }

                Spacer()
            }
            .navigationTitle(loc("chat.p2p_detail"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(loc("common.close")) { dismiss() }
                }
            }
            .task {
                await loadUser()
            }
        }
    }

    private func loadUser() async {
        do {
            let users = try await ContactService.shared.batchGetUsers(userIDs: [otherUserID])
            otherUser = users[otherUserID]
        } catch {
            otherUser = nil
        }
        isLoading = false
    }
}
