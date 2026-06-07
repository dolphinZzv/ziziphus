import SwiftUI
import IMCore

struct P2PDetailView: View {
    let convID: String
    let convName: String
    @State private var otherUser: User?
    @State private var isLoading = true
    @EnvironmentObject private var localizationManager: LocalizationManager
    let onCancel: () -> Void

    private var otherUserID: String {
        let parts = convID.split(separator: ":").map(String.init)
        let currentID = AuthManager.shared.currentUser?.userID ?? ""
        return parts.first { $0 != currentID } ?? convName
    }

    var body: some View {
        VStack(spacing: 24) {
            // Header
            HStack {
                Button(loc("common.close")) { onCancel() }
                Spacer()
                Text(loc("chat.p2p_detail"))
                    .fontWeight(.semibold)
                Spacer()
            }
            .padding()

            Divider()

            Spacer()

            if isLoading {
                ProgressView()
            } else {
                AvatarView(
                    name: otherUser?.name ?? convName,
                    url: otherUser?.avatar ?? "",
                    size: 72
                )

                Text(otherUser?.name ?? convName)
                    .font(.title3)
                    .fontWeight(.semibold)

                HStack(spacing: 4) {
                    Text(loc("profile.account_label"))
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(otherUserID)
                        .font(.caption)
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
        .frame(width: 320, height: 300)
        .task {
            await loadUser()
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
