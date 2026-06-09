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
        VStack(spacing: 0) {
            // Header
            HStack {
                Text(loc("profile.title"))
                    .font(.appleBodySemibold)
                    .foregroundColor(AppleDesign.Colors.ink)
                Spacer()
                Button(loc("common.close")) { onCancel() }
                    .font(.appleBody)
                    .foregroundColor(AppleDesign.Colors.actionBlue)
            }
            .padding(AppleDesign.Spacing.lg)

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
                    .font(.appleBodySemibold)
                    .padding(.top, AppleDesign.Spacing.sm)

                HStack(spacing: 4) {
                    Text(loc("profile.account_label"))
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    Text(otherUserID)
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                        .textSelection(.enabled)
                }
                .padding(.top, 8)

                HStack(spacing: 4) {
                    Text(loc("profile.id_label"))
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    Text(convID)
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                        .textSelection(.enabled)
                }
            }

            Spacer()
        }
        .frame(width: 400, height: 500)
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
