import SwiftUI
import IMCore

struct LoginView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var localizationManager: LocalizationManager

    var body: some View {
        VStack(spacing: 20) {
            Spacer()

            Image(systemName: "message.fill")
                .font(.system(size: 60))
                .foregroundColor(.blue)

            Text(loc("login.title"))
                .font(.largeTitle)
                .fontWeight(.bold)

            VStack(spacing: 16) {
                TextField(loc("login.account_placeholder"), text: $loginVM.account)
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 300)

                SecureField(loc("login.password_placeholder"), text: $loginVM.password)
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
                    Text(loc("login.login_button"))
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .frame(maxWidth: 300)
            .disabled(loginVM.isLoading)

            Button(loc("login.switch_to_register")) {
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

#Preview {
    LoginView()
        .environmentObject(LoginViewModel())
}
