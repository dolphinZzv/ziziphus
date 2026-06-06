import SwiftUI
import IMCore

struct ContactListView: View {
    @StateObject private var vm = ContactListViewModel()
    @State private var showAddContact = false

    var body: some View {
        List {
            if vm.isLoading && vm.contacts.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity)
                    .padding()
            } else if vm.contacts.isEmpty {
                Text("暂无联系人")
                    .foregroundColor(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding()
            } else {
                ForEach(vm.contacts) { contact in
                    HStack(spacing: 12) {
                        AvatarView(name: contact.name, url: contact.avatar, size: 44)
                        VStack(alignment: .leading, spacing: 2) {
                            HStack {
                                Text(contact.nickname.isEmpty ? contact.name : contact.nickname)
                                    .fontWeight(.medium)
                                OnlineStatusDot(status: contact.status)
                            }
                            Text(contact.userID)
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    .padding(.vertical, 2)
                }
                .onDelete { indexSet in
                    for idx in indexSet {
                        Task { try? await vm.removeContact(userID: vm.contacts[idx].userID) }
                    }
                }
            }
        }
        .listStyle(.plain)
        .refreshable { vm.refresh() }
        .navigationTitle("联系人")
        .toolbar {
            Button(action: { showAddContact = true }) {
                Image(systemName: "plus")
            }
        }
        .sheet(isPresented: $showAddContact) {
            AddContactView { userID, nickname in
                Task { try? await vm.addContact(userID: userID, nickname: nickname) }
                showAddContact = false
            }
        }
        .onAppear { vm.loadContacts() }
    }
}
