import SwiftUI

struct AddContactView: View {
    @State private var userID = ""
    @State private var nickname = ""
    let onAdd: (String, String?) -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section("用户信息") {
                    TextField("用户ID", text: $userID)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                    TextField("备注（可选）", text: $nickname)
                }
            }
            .navigationTitle("添加联系人")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { /* dismiss handled by parent */ }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("添加") { onAdd(userID, nickname.isEmpty ? nil : nickname) }
                        .disabled(userID.isEmpty)
                }
            }
        }
    }
}
