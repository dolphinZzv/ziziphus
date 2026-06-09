import SwiftUI
import IMCore

struct SearchResultsView: View {
    let results: [User]
    let onSelect: (User) -> Void

    var body: some View {
        List(results) { user in
            Button(action: { onSelect(user) }) {
                HStack {
                    AvatarView(name: user.name, url: user.avatar, size: 32)
                    VStack(alignment: .leading) {
                        Text(user.name)
                            .font(.appleBodySemibold)
                        Text(user.userID)
                            .font(.appleCaption)
                            .foregroundColor(.secondary)
                    }
                }
            }
            .buttonStyle(.plain)
        }
        .listStyle(.plain)
    }
}
