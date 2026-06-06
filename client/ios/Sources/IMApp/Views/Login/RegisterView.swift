import SwiftUI
import IMCore

struct RegisterView: View {
    @EnvironmentObject private var loginVM: LoginViewModel

    var body: some View {
        VStack(spacing: 20) {
            Spacer()

            Image(systemName: "person.crop.circle.badge.plus")
                .font(.system(size: 60))
                .foregroundColor(.blue)

            Text("注册")
                .font(.largeTitle)
                .fontWeight(.bold)

            VStack(spacing: 16) {
                TextField("名称", text: $loginVM.name)
                    .textFieldStyle(.roundedBorder)
                    .autocapitalization(.none)
                    .disableAutocorrection(true)

                SecureField("密码", text: $loginVM.password)
                    .textFieldStyle(.roundedBorder)
            }
            .padding(.horizontal, 40)

            if let error = loginVM.errorMessage {
                Text(error)
                    .foregroundColor(.red)
                    .font(.callout)
            }

            Button(action: loginVM.register) {
                if loginVM.isLoading {
                    ProgressView()
                        .progressViewStyle(.circular)
                        .tint(.white)
                } else {
                    Text("注册")
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .padding(.horizontal, 40)
            .disabled(loginVM.isLoading)

            Button("已有账号？点击登录") {
                loginVM.switchMode()
            }
            .foregroundColor(.blue)

            Spacer()
        }
        .padding()
    }
}
