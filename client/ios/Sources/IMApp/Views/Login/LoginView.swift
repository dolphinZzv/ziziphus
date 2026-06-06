import SwiftUI
import IMCore

struct LoginView: View {
    @EnvironmentObject private var loginVM: LoginViewModel

    var body: some View {
        VStack(spacing: 20) {
            Spacer()

            Image(systemName: "message.fill")
                .font(.system(size: 60))
                .foregroundColor(.blue)

            Text("登录")
                .font(.largeTitle)
                .fontWeight(.bold)

            VStack(spacing: 16) {
                TextField("用户ID", text: $loginVM.userID)
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

            Button(action: loginVM.login) {
                if loginVM.isLoading {
                    ProgressView()
                        .progressViewStyle(.circular)
                        .tint(.white)
                } else {
                    Text("登录")
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .padding(.horizontal, 40)
            .disabled(loginVM.isLoading)

            Button("没有账号？点击注册") {
                loginVM.switchMode()
            }
            .foregroundColor(.blue)

            Spacer()
        }
        .padding()
    }
}
