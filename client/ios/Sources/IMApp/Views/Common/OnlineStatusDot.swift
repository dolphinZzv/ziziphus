import SwiftUI
import IMCore

struct OnlineStatusDot: View {
    let status: UserStatus

    var body: some View {
        Circle()
            .fill(status == .online ? Color.green : Color.gray)
            .frame(width: 8, height: 8)
    }
}
