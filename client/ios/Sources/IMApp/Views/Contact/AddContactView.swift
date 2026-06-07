import SwiftUI
import IMCore

struct AddContactView: View {
    @StateObject private var searchVM = SearchViewModel()
    @State private var errorMessage: String?
    @State private var addingUserID: String?
    @Environment(\.dismiss) private var dismiss
    let onAdd: (String, String?) -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Search bar
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
                            Button(action: { addContact(user: user) }) {
                                HStack {
                                    AvatarView(name: user.name, url: user.avatar, size: 40)
                                    VStack(alignment: .leading) {
                                        Text(user.name)
                                            .fontWeight(.medium)
                                        Text(user.userID)
                                            .font(.caption)
                                            .foregroundColor(.secondary)
                                    }
                                    Spacer()
                                    if addingUserID == user.userID {
                                        ProgressView()
                                            .scaleEffect(0.7)
                                    }
                                }
                            }
                            .disabled(addingUserID != nil)
                        }
                    }
                }
                .listStyle(.plain)
            }
            .navigationTitle(loc("contact.add"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(loc("common.cancel")) { dismiss() }
                }
            }
        }
    }

    private func addContact(user: User) {
        addingUserID = user.userID
        errorMessage = nil
        Task {
            do {
                try await ContactService.shared.addContact(userID: user.userID, nickname: nil)
                onAdd(user.userID, nil)
                dismiss()
            } catch {
                errorMessage = error.localizedDescription
                addingUserID = nil
            }
        }
    }
}
