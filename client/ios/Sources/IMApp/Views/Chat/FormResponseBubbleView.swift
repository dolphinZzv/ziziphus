import SwiftUI
import IMCore

/// Renders a FormResponse bubble (ContentType=11).
struct FormResponseBubbleView: View {
    let body: FormResponseBody

    private var isApproved: Bool { form.action == "approve" }

    var body: some View {
        HStack(spacing: 4) {
            Image(systemName: isApproved ? "checkmark.circle.fill" : "xmark.circle.fill")
                .font(.system(size: 12))
            Text("\(form.responderName) \(isApproved ? "已通过" : "已拒绝")")
                .font(.system(size: AppleDesign.Typography.finePrintSize))
        }
        .foregroundColor(isApproved ? .green : .red)
        .padding(.horizontal, 6)
    }
}
