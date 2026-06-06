import SwiftUI

struct AvatarView: View {
    let name: String
    let url: String
    var size: CGFloat = 40

    var body: some View {
        Circle()
            .fill(Color.blue.opacity(0.2))
            .frame(width: size, height: size)
            .overlay {
                Text(String(name.prefix(1)).uppercased())
                    .font(.system(size: size * 0.4))
                    .fontWeight(.semibold)
                    .foregroundColor(.blue)
            }
    }
}
