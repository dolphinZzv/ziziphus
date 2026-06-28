import SwiftUI
import IMCore

/// Renders a form message bubble.
/// For contact_request type: shows sender avatar + name + message + approve/reject buttons.
/// For other form types: shows title + description + generic action buttons.
struct FormBubbleView: View {
    let body: FormDefinitionBody
    let msgID: Int64
    let convID: String
    let isMine: Bool
    var onAction: ((String) -> Void)?

    @State private var localStatus: String = "active"
    @State private var localActionResult: String?
    @State private var isSubmitting = false

    private var isContactRequest: Bool { form.type == "contact_request" }
    private var isResolved: Bool { localStatus == "closed" }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if isContactRequest {
                contactRequestCard
            } else {
                genericFormContent
            }
        }
        .onAppear { localStatus = form.status }
    }

    // MARK: - Contact request card

    private var contactRequestCard: some View {
        VStack(alignment: .leading, spacing: 6) {
            // Sender info (only for recipient)
            if !isMine {
                HStack(spacing: 8) {
                    avatarView
                    VStack(alignment: .leading, spacing: 1) {
                        Text(form.fromUserName ?? "")
                            .font(.system(size: AppleDesign.Typography.bodySize, weight: .medium))
                        Text(form.title)
                            .font(.system(size: AppleDesign.Typography.finePrintSize))
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }
                }
            }

            // Message text
            if let msg = form.message, !msg.isEmpty {
                Text(msg)
                    .font(.system(size: AppleDesign.Typography.finePrintSize))
                    .padding(.horizontal, 8).padding(.vertical, 4)
                    .background(AppleDesign.Colors.chatGray)
                    .cornerRadius(6)
            }

            // Result badge
            if isResolved {
                HStack(spacing: 4) {
                    Image(systemName: localActionResult == "approve" ? "checkmark.circle.fill" : "xmark.circle.fill")
                    Text(localActionResult == "approve" ? "已通过" : "已拒绝")
                }
                .font(.system(size: 12))
                .foregroundColor(localActionResult == "approve" ? .green : .red)
            }

            // Action buttons (only for recipient, pending)
            if !isMine && !isResolved {
                HStack(spacing: 10) {
                    ForEach(form.actions, id: \.action) { action in
                        Button {
                            handleAction(action)
                        } label: {
                            HStack(spacing: 4) {
                                if isSubmitting { ProgressView().scaleEffect(0.6) }
                                Text(action.label)
                            }
                            .font(.system(size: 12, weight: .medium))
                            .padding(.horizontal, 14).padding(.vertical, 6)
                            .background(action.style == "primary"
                                ? Color.accentColor : Color.red.opacity(0.1))
                            .foregroundColor(action.style == "primary" ? .white : .red)
                            .cornerRadius(6)
                        }
                        .disabled(isSubmitting)
                        .buttonStyle(.plain)
                    }
                }
            }

            // Sender: waiting message
            if isMine && !isResolved {
                Text("等待对方回复")
                    .font(.system(size: 11))
                    .foregroundColor(AppleDesign.Colors.inkMuted)
            }
        }
    }

    private func handleAction(_ action: FormAction) {
        isSubmitting = true
        localStatus = "closed"
        localActionResult = action.action
        onAction?(action.action)
    }

    func submitFailed() {
        isSubmitting = false
        localStatus = "active"
        localActionResult = nil
    }

    // MARK: - Generic (non-contact_request) form

    private var genericFormContent: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(form.title)
                .font(.system(size: AppleDesign.Typography.bodySize, weight: .semibold))
            if let desc = form.description { Text(desc).font(.caption).foregroundColor(.secondary) }
            if !isResolved {
                HStack {
                    ForEach(form.actions, id: \.action) { a in
                        Button(a.label) { handleAction(a) }
                            .disabled(isSubmitting)
                            .buttonStyle(.borderedProminent)
                            .controlSize(.small)
                    }
                }
            }
        }
    }

    private var avatarView: some View {
        Group {
            if let avatar = form.fromUserAvatar, !avatar.isEmpty,
               let url = URL(string: AppSettings.shared.serverURL + "/files/" + avatar) {
                AsyncImage(url: url) { phase in
                    if let img = phase.image { img.resizable().aspectRatio(contentMode: .fill) }
                    else { defaultAvatar }
                }
                .frame(width: 32, height: 32).clipShape(Circle())
            } else { defaultAvatar }
        }
    }

    private var defaultAvatar: some View {
        Circle()
            .fill(Color.accentColor.opacity(0.3))
            .frame(width: 32, height: 32)
            .overlay(Text(String((form.fromUserName ?? "?").prefix(1))).font(.caption))
    }
}
