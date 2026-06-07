import SwiftUI
import IMCore

struct LoginView: View {
    @EnvironmentObject private var loginVM: LoginViewModel

    var body: some View {
        VStack(spacing: 24) {
            Spacer()

            Image(systemName: "message.fill")
                .font(.system(size: 60))
                .foregroundColor(.blue)

            Text("DolphinZ")
                .font(.largeTitle)
                .fontWeight(.bold)

            VStack(spacing: 16) {
                TextField(loc("login.account_placeholder"), text: $loginVM.account)
                    .textFieldStyle(.roundedBorder)
                    .autocapitalization(.none)
                    .disableAutocorrection(true)

                SecureField(loc("login.password_placeholder"), text: $loginVM.password)
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
                    Text(loc("login.login_button"))
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .padding(.horizontal, 40)
            .disabled(loginVM.isLoading)

            Button(loc("login.switch_to_register")) {
                loginVM.switchMode()
            }
            .foregroundColor(.blue)

            Spacer()
        }
        .padding()
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

#Preview {
    LoginView()
        .environmentObject(LoginViewModel())
}
