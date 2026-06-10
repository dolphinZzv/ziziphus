import SwiftUI
import IMCore

struct NewChatView: View {
    @StateObject private var searchVM = SearchViewModel()
    @State private var errorMessage: String?
    @State private var isCreating = false

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
                Text(loc("conv.new_chat"))
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
                        Button(action: { startP2PChat(user: user) }) {
                            HStack {
                                AvatarView(name: user.name, url: user.avatar, size: 36)
                                VStack(alignment: .leading) {
                                    Text(user.name)
                                        .font(.appleBodySemibold)
                                    Text(user.userID)
                                        .font(.appleCaption)
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
        .frame(width: 400, height: 500)
    }

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
}
