import SwiftUI
import IMCore

struct ContactListView: View {
    @StateObject private var vm = ContactListViewModel()
    @State private var showAddContact = false
    @State private var confirmDelete: Contact?
    @State private var isNavigating = false
    @EnvironmentObject private var localizationManager: LocalizationManager

    private func avatarColor(for userID: String) -> Color {
        let colors: [Color] = [.blue, .green, .orange, .purple, .pink, .teal, .indigo, .mint]
        let hash = abs(userID.hashValue)
        return colors[hash % colors.count]
    }

    var body: some View {
        List {
            if vm.isLoading && vm.contacts.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity)
                    .padding()
            } else if vm.contacts.isEmpty {
                Text(loc("contact.no_contacts"))
                    .foregroundColor(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding()
            } else {
                ForEach(vm.contacts) { contact in
                    Button(action: { startChat(contact: contact) }) {
                        HStack(spacing: 10) {
                            let color = avatarColor(for: contact.userID)
                            Circle()
                                .fill(color.opacity(0.2))
                                .frame(width: 40, height: 40)
                                .overlay {
                                    Text(String((contact.nickname.isEmpty ? contact.name : contact.nickname).prefix(1)))
                                        .fontWeight(.semibold)
                                        .foregroundColor(color)
                                }

                            VStack(alignment: .leading, spacing: 2) {
                                HStack(spacing: 6) {
                                    Text(contact.nickname.isEmpty ? contact.name : contact.nickname)
                                        .fontWeight(.medium)
                                    OnlineStatusDot(status: contact.status)
                                }
                                Text(contact.userID)
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }

                            Spacer()
                        }
                        .padding(.vertical, 4)
                    }
                    .buttonStyle(.plain)
                }
                .onDelete { indexSet in
                    if let idx = indexSet.first {
                        confirmDelete = vm.contacts[idx]
                    }
                }
            }
        }
        .listStyle(.plain)
        .refreshable { vm.refresh() }
        .toolbar {
            Button(action: { showAddContact = true }) {
                Image(systemName: "plus")
            }
        }
        .sheet(isPresented: $showAddContact) {
            AddContactView { _, _ in
                showAddContact = false
                vm.refresh()
            }
        }
        .alert(loc("contact.delete_title"), isPresented: .init(
            get: { confirmDelete != nil },
            set: { if !$0 { confirmDelete = nil } }
        )) {
            Button(loc("common.cancel"), role: .cancel) { confirmDelete = nil }
            Button(loc("contact.delete_button"), role: .destructive) {
                if let contact = confirmDelete {
                    Task { try? await vm.removeContact(userID: contact.userID) }
                }
                confirmDelete = nil
            }
        } message: {
            Text(confirmDelete.map { String(format: loc("contact.delete_confirm_message"), $0.nickname.isEmpty ? $0.name : $0.nickname) } ?? "")
        }
        .onAppear { vm.loadContacts() }
    }

    private func startChat(contact: Contact) {
        guard !isNavigating else { return }
        isNavigating = true
        Task {
            do {
                let (convID, name) = try await ConversationService.shared.createP2P(userID: contact.userID)
                isNavigating = false
                // Navigate via sidebar selection - post notification or set convID
                NotificationCenter.default.post(
                    name: NSNotification.Name("OpenConversation"),
                    object: nil,
                    userInfo: ["convID": convID, "name": name.isEmpty ? contact.name : name]
                )
            } catch {
                isNavigating = false
            }
        }
    }
}
