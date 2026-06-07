import SwiftUI
import IMCore

struct ContactListView: View {
    @StateObject private var vm = ContactListViewModel()
    @State private var showAddContact = false
    @State private var confirmDelete: IndexSet?
    @State private var showDeleteAlert = false
    @State private var isNavigating = false
    @State private var navigateToChat: ChatDestination?

    private func avatarColor(for userID: String) -> Color {
        let colors: [Color] = [.blue, .green, .orange, .purple, .pink, .teal, .indigo, .mint]
        let hash = abs(userID.hashValue)
        return colors[hash % colors.count]
    }

    var body: some View {
        ZStack {
            if vm.isLoading && vm.contacts.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if vm.contacts.isEmpty {
                VStack {
                    Spacer()
                    Text(loc("contact.no_contacts"))
                        .foregroundColor(.secondary)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    ForEach(vm.contacts) { contact in
                        Button(action: { startChat(contact: contact) }) {
                            HStack(spacing: 12) {
                                // Avatar
                                let color = avatarColor(for: contact.userID)
                                Circle()
                                    .fill(color.opacity(0.2))
                                    .frame(width: 48, height: 48)
                                    .overlay {
                                        Text(String((contact.nickname.isEmpty ? contact.name : contact.nickname).prefix(1)))
                                            .font(.title3)
                                            .fontWeight(.semibold)
                                            .foregroundColor(color)
                                    }

                                VStack(alignment: .leading, spacing: 3) {
                                    HStack(spacing: 6) {
                                        Text(contact.nickname.isEmpty ? contact.name : contact.nickname)
                                            .fontWeight(.medium)
                                            .lineLimit(1)
                                        OnlineStatusDot(status: contact.status)
                                    }
                                    Text(contact.userID)
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }

                                Spacer()

                                Image(systemName: "chevron.right")
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                            }
                        }
                        .buttonStyle(.plain)
                        .padding(.vertical, 4)
                        .listRowSeparator(.hidden)
                        .overlay(alignment: .bottom) {
                            Rectangle()
                                .fill(Color(.separator))
                                .frame(height: 0.5)
                        }
                    }
                    .onDelete { indexSet in
                        confirmDelete = indexSet
                        showDeleteAlert = true
                    }
                }
                .listStyle(.plain)
                .refreshable { vm.refresh() }
            }
        }
        .navigationTitle(loc("contact.title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            Button(action: { showAddContact = true }) {
                Image(systemName: "plus")
                    .font(.body)
            }
        }
        .sheet(isPresented: $showAddContact) {
            AddContactView { _, _ in
                showAddContact = false
                vm.refresh()
            }
        }
        .navigationDestination(item: $navigateToChat) { dest in
            ChatView(convID: dest.convID, convName: dest.name, convType: dest.type)
        }
        .alert(loc("contact.delete_title"), isPresented: $showDeleteAlert) {
            Button(loc("common.cancel"), role: .cancel) {
                confirmDelete = nil
            }
            Button(loc("contact.delete_button"), role: .destructive) {
                if let indexSet = confirmDelete {
                    for idx in indexSet {
                        Task { try? await vm.removeContact(userID: vm.contacts[idx].userID) }
                    }
                }
                confirmDelete = nil
            }
        } message: {
            if let indexSet = confirmDelete, let idx = indexSet.first {
                let name = vm.contacts[idx].nickname.isEmpty ? vm.contacts[idx].name : vm.contacts[idx].nickname
                Text(String(format: loc("contact.delete_confirm_message"), name))
            } else {
                Text(loc("contact.delete_confirm_default"))
            }
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
                navigateToChat = ChatDestination(
                    convID: convID,
                    name: name.isEmpty ? contact.name : name,
                    type: .p2p
                )
            } catch {
                isNavigating = false
            }
        }
    }
}
