import SwiftUI
import IMCore

struct NewChatView: View {
    @StateObject private var searchVM = SearchViewModel()
    @State private var errorMessage: String?
    @State private var isCreating = false

    @Environment(\.dismiss) private var dismiss
    let onCreated: (String, String, ConvType) -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
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
            .navigationTitle(loc("conv.new_chat"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(loc("common.cancel")) { dismiss() }
                }
            }
        }
    }

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
}
