import SwiftUI
import IMCore

struct UserCardView: View {
    let user: User
    var onStartChat: ((String, String) -> Void)?

    @EnvironmentObject private var themeManager: ThemeManager
    @Environment(\.dismiss) private var dismiss

    private var isSelf: Bool {
        user.userID == AuthManager.shared.currentUser?.userID
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 20) {
                Spacer().frame(height: 12)

                AvatarView(name: user.name, url: user.avatar, size: 80)

                VStack(spacing: 4) {
                    Text(user.name)
                        .font(.title3)
                        .fontWeight(.semibold)

                    Text("@\(user.account)")
                        .font(.subheadline)
                        .foregroundColor(.secondary)

                    HStack(spacing: 6) {
                        Circle()
                            .fill(statusColor)
                            .frame(width: 8, height: 8)
                        Text(statusText)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                if !isSelf {
                    Button {
                        dismiss()
                        onStartChat?(user.userID, user.name)
                    } label: {
                        Label(loc("conv.new_chat"), systemImage: "message.fill")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .padding(.horizontal, 40)
                }

                Spacer()
            }
            .padding()
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(loc("common.done")) { dismiss() }
                }
            }
        }
        .presentationDetents([.medium])
    }

    private var statusColor: Color {
        switch user.status {
        case .online: return .green
        case .busy: return .orange
        case .offline: return .gray
        }
    }

    private var statusText: String {
        switch user.status {
        case .online: return loc("common.online")
        case .busy: return loc("common.busy")
        case .offline: return loc("common.offline")
        }
    }
}
