import SwiftUI
import IMCore

struct InputBarView: View {
    @Binding var text: String
    @EnvironmentObject private var localizationManager: LocalizationManager
    let onSend: () -> Void
    let onTyping: () -> Void
    let onPickImage: () -> Void
    let onPickFile: () -> Void
    var replyingToMsg: Message?
    var replyingToSender: String?
    var onCancelReply: (() -> Void)?
    var members: [ConvMember] = []
    var senderInfo: [String: User] = [:]
    @State private var showMentionSheet = false
    @State private var selectedMemberIDs: Set<String> = []
    @State private var mentionQuery = ""
    @State private var mentionAtPos: Int = -1

    private var showMentionPopup: Bool { mentionAtPos >= 0 && !members.isEmpty }

    private var filteredMentionMembers: [ConvMember] {
        if mentionQuery.isEmpty { return Array(members.prefix(8)) }
        let q = mentionQuery.lowercased()
        return members.filter { member in
            let name = (member.nickname ?? senderInfo[member.userID]?.name ?? member.userID).lowercased()
            return name.contains(q) || member.userID.lowercased().contains(q)
        }
    }

    private func selectMentionMember(_ member: ConvMember) {
        guard mentionAtPos >= 0 else { return }
        let name = member.nickname ?? senderInfo[member.userID]?.name ?? member.userID
        let nsText = text as NSString
        let replacement = "@\(name) "
        let mentionLen = min(mentionQuery.count + 1, nsText.length - mentionAtPos)
        let range = NSRange(location: mentionAtPos, length: max(0, mentionLen))
        text = nsText.replacingCharacters(in: range, with: replacement)
        mentionAtPos = -1
        mentionQuery = ""
    }

    var body: some View {
        VStack(spacing: 0) {
            // Inline mention autocomplete popup
            if showMentionPopup {
                MentionPopupView(
                    members: filteredMentionMembers,
                    senderInfo: senderInfo,
                    onSelect: { selectMentionMember($0) }
                )
                .transition(.move(edge: .bottom).combined(with: .opacity))
                .animation(.easeOut(duration: 0.15), value: showMentionPopup)
            }

            // Reply preview bar
            if let replyingToMsg {
                HStack(spacing: 8) {
                    Rectangle()
                        .fill(Color.blue)
                        .frame(width: 3)
                        .cornerRadius(1.5)
                    VStack(alignment: .leading, spacing: 1) {
                        Text(String(format: loc("chat.replying"), replyingToSender ?? replyingToMsg.senderID))
                            .font(.system(size: AppleDesign.Typography.finePrintSize))
                            .foregroundColor(.blue)
                        Text(replyingToMsg.body)
                            .font(.system(size: AppleDesign.Typography.finePrintSize))
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                            .lineLimit(1)
                    }
                    Spacer()
                    Button(action: { onCancelReply?() }) {
                        Image(systemName: "xmark")
                            .font(.caption2)
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }
                    .buttonStyle(.plain)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 4)
                .background(AppleDesign.Colors.parchment)
            }

            HStack(spacing: 0) {
            ZStack(alignment: .bottomTrailing) {
                ChatTextView(
                    text: $text,
                    placeholder: loc("chat.placeholder"),
                    onTyping: onTyping,
                    onSend: onSend,
                    onMentionChanged: { query, atPos in
                        if atPos >= 0, query.utf8.count <= 30 {
                            mentionQuery = query
                            mentionAtPos = atPos
                        } else {
                            mentionQuery = ""
                            mentionAtPos = -1
                        }
                    }
                )
                .frame(minHeight: 40, maxHeight: 120)
                .padding(.trailing, 90)
                .background(AppleDesign.Colors.pearl)
                .clipShape(RoundedRectangle(cornerRadius: 18))

                HStack(spacing: 2) {
                    Button(action: { showMentionSheet = true }) {
                        Text("@")
                            .font(.title3)
                            .fontWeight(.semibold)
                            .foregroundColor(AppleDesign.Colors.actionBlue)
                            .frame(width: 28, height: 28)
                    }
                    .buttonStyle(.plain)
                    .disabled(members.isEmpty)

                    Menu {
                        Button(action: onPickImage) {
                            Label("图片", systemImage: "photo")
                        }
                        Button(action: onPickFile) {
                            Label("文件", systemImage: "doc")
                        }
                    } label: {
                        Image(systemName: "plus.circle.fill")
                            .font(.title2)
                            .foregroundColor(AppleDesign.Colors.actionBlue)
                    }
                    .buttonStyle(.plain)

                    Button(action: onSend) {
                        Image(systemName: "arrow.up.circle.fill")
                            .font(.title2)
                            .foregroundColor(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                                ? AppleDesign.Colors.inkMuted
                                : AppleDesign.Colors.actionBlue)
                    }
                    .buttonStyle(.plain)
                    .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
                .offset(x: -6, y: -6)
            }
        }
        .padding(12)
        .background(AppleDesign.Colors.parchment)
    }
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
            .frame(width: 340, height: 440)
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

// MARK: - Inline Mention Autocomplete Popup

private struct MentionPopupView: View {
    let members: [ConvMember]
    let senderInfo: [String: User]
    let onSelect: (ConvMember) -> Void

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                ForEach(members, id: \.userID) { member in
                    let name = member.nickname ?? senderInfo[member.userID]?.name ?? member.userID
                    Button(action: { onSelect(member) }) {
                        HStack(spacing: 8) {
                            AvatarView(name: name, url: senderInfo[member.userID]?.avatar ?? "", size: 28)
                            Text(name)
                                .font(.appleBody)
                                .foregroundColor(.primary)
                            Spacer()
                            Text(member.userID)
                                .font(.appleCaption)
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    Divider()
                }
            }
        }
        .frame(maxWidth: 260, maxHeight: min(CGFloat(members.count) * 40 + 8, 200))
        .background(.regularMaterial)
        .clipShape(RoundedRectangle(cornerRadius: 8))
        .shadow(color: .black.opacity(0.12), radius: 8, y: 2)
        .padding(.horizontal, 12)
        .padding(.bottom, 2)
    }
}

// MARK: - Mention Picker Sheet

private struct MentionPickerView: View {
    let members: [ConvMember]
    let senderInfo: [String: User]
    @Binding var selectedIDs: Set<String>
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Button(loc("common.cancel")) {
                    selectedIDs = []
                    dismiss()
                }
                .buttonStyle(.plain)
                Spacer()
                Text("@ " + loc("group.members"))
                    .font(.appleBodySemibold)
                Spacer()
                Button(loc("common.done")) {
                    onDone()
                }
                .buttonStyle(.plain)
                .disabled(selectedIDs.isEmpty)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)

            Divider()

            if members.isEmpty {
                Spacer()
                Text(loc("search.no_results"))
                    .foregroundColor(.secondary)
                Spacer()
            } else {
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
                                AvatarView(name: name, url: senderInfo[member.userID]?.avatar ?? "", size: 36)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(name)
                                        .font(.appleBody)
                                        .foregroundColor(.primary)
                                    Text(member.userID)
                                        .font(.appleCaption)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                if selectedIDs.contains(member.userID) {
                                    Image(systemName: "checkmark.circle.fill")
                                        .foregroundColor(AppleDesign.Colors.actionBlue)
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
            }
        }
    }
}
