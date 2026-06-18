import SwiftUI
import IMCore

struct InputBarView: View {
    @Binding var text: String
    let onSend: () -> Void
    let onTyping: () -> Void
    let onPickImage: () -> Void
    let onPickFile: () -> Void
    var replyingToMsg: Message?
    var replyingToSender: String?
    var onCancelReply: (() -> Void)?
    var members: [ConvMember] = []
    var senderInfo: [String: User] = [:]
    @FocusState private var isInputFocused
    @State private var showMentionSheet = false
    @State private var selectedMemberIDs: Set<String> = []

    var body: some View {
        VStack(spacing: 0) {
            // Reply preview bar
            if let replyingToMsg {
                HStack(spacing: 8) {
                    Rectangle()
                        .fill(Color.blue)
                        .frame(width: 3)
                        .cornerRadius(1.5)
                    VStack(alignment: .leading, spacing: 1) {
                        Text(String(format: loc("chat.replying"), replyingToSender ?? replyingToMsg.senderID))
                            .font(.caption2)
                            .foregroundColor(.blue)
                        Text(replyingToMsg.body)
                            .font(.caption2)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                    }
                    Spacer()
                    Button(action: { onCancelReply?() }) {
                        Image(systemName: "xmark")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 6)
                .background(Color(.systemGray5))
            }

            HStack(spacing: 0) {
                ZStack(alignment: .bottomTrailing) {
                    TextField(loc("chat.placeholder"), text: $text, axis: .vertical)
                        .textFieldStyle(.plain)
                        .font(.body)
                        .lineLimit(1...5)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .padding(.trailing, 60)
                        .background(Color(.systemBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 20))
                        .overlay(
                            RoundedRectangle(cornerRadius: 20)
                                .stroke(Color(.separator), lineWidth: 0.5)
                        )
                        .focused($isInputFocused)
                        .onChange(of: text) { _, _ in
                            onTyping()
                        }
                        .toolbar {
                            ToolbarItemGroup(placement: .keyboard) {
                                Spacer()
                                Button(loc("common.done")) {
                                    isInputFocused = false
                                }
                            }
                        }

                    HStack(spacing: 2) {
                        Button(action: { showMentionSheet = true }) {
                            Text("@")
                                .font(.title3)
                                .fontWeight(.semibold)
                                .foregroundColor(.blue)
                                .frame(width: 28, height: 28)
                        }
                        .disabled(members.isEmpty)

                        Menu {
                            Button(action: onPickImage) {
                                Label(loc("chat.image"), systemImage: "photo")
                            }
                            Button(action: onPickFile) {
                                Label(loc("chat.file"), systemImage: "doc")
                            }
                        } label: {
                            Image(systemName: "plus.circle.fill")
                                .font(.title2)
                                .foregroundColor(.blue)
                        }

                        Button(action: onSend) {
                            Image(systemName: "arrow.up.circle.fill")
                                .font(.title2)
                                .foregroundColor(.blue)
                        }
                        .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    }
                    .offset(x: -6, y: -6)
                }
            }
            .padding(12)
            .background(Color(.systemGray6))
            .sheet(isPresented: $showMentionSheet) {
                MentionPickerView(
                    members: members,
                    senderInfo: senderInfo,
                    selectedIDs: $selectedMemberIDs,
                    onDone: {
                        insertMentions()
                        showMentionSheet = false
                    }
                )
            }
        }
    }

    private func insertMentions() {
        guard !selectedMemberIDs.isEmpty else { return }
        var mentionTexts: [String] = []
        for id in selectedMemberIDs {
            let name = members.first(where: { $0.userID == id })?.nickname
                ?? senderInfo[id]?.name
                ?? id
            mentionTexts.append("@\(name)")
        }
        let mentionStr = mentionTexts.joined(separator: " ") + " "
        if text.isEmpty {
            text = mentionStr
        } else {
            text = text + " " + mentionStr
        }
        selectedMemberIDs = []
    }
}

// MARK: - Mention Picker

private struct MentionPickerView: View {
    let members: [ConvMember]
    let senderInfo: [String: User]
    @Binding var selectedIDs: Set<String>
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                ForEach(members, id: \.userID) { member in
                    let name = member.nickname ?? senderInfo[member.userID]?.name ?? member.userID
                    Button(action: {
                        if selectedIDs.contains(member.userID) {
                            selectedIDs.remove(member.userID)
                        } else {
                            selectedIDs.insert(member.userID)
                        }
                    }) {
                        HStack {
                            AvatarView(name: name, url: "", size: 36)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(name)
                                    .font(.body)
                                    .foregroundColor(.primary)
                                Text(member.userID)
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                            }
                            Spacer()
                            if selectedIDs.contains(member.userID) {
                                Image(systemName: "checkmark.circle.fill")
                                    .foregroundColor(.blue)
                                    .font(.title3)
                            } else {
                                Image(systemName: "circle")
                                    .foregroundColor(.secondary)
                                    .font(.title3)
                            }
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
            .listStyle(.plain)
            .navigationTitle("@ " + loc("group.members"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button(loc("common.cancel")) {
                        selectedIDs = []
                        dismiss()
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button(loc("common.done")) {
                        onDone()
                    }
                    .disabled(selectedIDs.isEmpty)
                }
            }
        }
    }
}
