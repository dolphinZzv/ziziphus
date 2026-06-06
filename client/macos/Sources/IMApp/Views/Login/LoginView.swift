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
                    .frame(maxWidth: 300)

                SecureField("密码", text: $loginVM.password)
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 300)
            }

            if let error = loginVM.errorMessage {
                Text(error)
                    .foregroundColor(.red)
                    .font(.callout)
            }

            Button(action: loginVM.login) {
                if loginVM.isLoading {
                    ProgressView()
                        .progressViewStyle(.circular)
                        .scaleEffect(0.8)
                } else {
                    Text("登录")
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .frame(maxWidth: 300)
            .disabled(loginVM.isLoading)

            Button("没有账号？点击注册") {
                loginVM.switchMode()
            }
            .buttonStyle(.plain)
            .foregroundColor(.blue)

            Spacer()
        }
        .padding()
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
