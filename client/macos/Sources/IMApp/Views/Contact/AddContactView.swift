import SwiftUI

struct AddContactView: View {
    @State private var userID = ""
    @State private var nickname = ""
    let onAdd: (String, String?) -> Void

    var body: some View {
        VStack(spacing: 16) {
            Text("添加联系人")
                .font(.headline)

            TextField("用户ID", text: $userID)
                .textFieldStyle(.roundedBorder)

            TextField("备注（可选）", text: $nickname)
                .textFieldStyle(.roundedBorder)

            HStack {
                Button("取消") {
                    // dismiss handled by parent
                }
                Button("添加") {
                    onAdd(userID, nickname.isEmpty ? nil : nickname)
                }
                .buttonStyle(.borderedProminent)
                .disabled(userID.isEmpty)
            }
        }
        .padding()
        .frame(width: 300)
    }
}
