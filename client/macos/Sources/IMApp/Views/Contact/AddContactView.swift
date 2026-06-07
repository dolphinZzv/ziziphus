import SwiftUI
import IMCore

struct AddContactView: View {
    @StateObject private var searchVM = SearchViewModel()
    @State private var errorMessage: String?
    @State private var addingUserID: String?
    @EnvironmentObject private var localizationManager: LocalizationManager
    let onAdd: (String, String?) -> Void

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text(loc("contact.add"))
                    .font(.headline)
                Spacer()
                Button(loc("common.cancel")) { onAdd("", nil) }
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
                        Button(action: { addContact(user: user) }) {
                            HStack {
                                AvatarView(name: user.name, url: user.avatar, size: 32)
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
                        .buttonStyle(.plain)
                        .disabled(addingUserID != nil)
                    }
                }
            }
            .listStyle(.plain)
        }
        .frame(width: 320, height: 400)
    }

    private func addContact(user: User) {
        addingUserID = user.userID
        errorMessage = nil
        Task {
            do {
                try await ContactService.shared.addContact(userID: user.userID, nickname: nil)
                onAdd(user.userID, nil)
            } catch {
                errorMessage = error.localizedDescription
                addingUserID = nil
            }
        }
    }
}
