import SwiftUI
import IMCore

struct RegisterView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var localizationManager: LocalizationManager

    var body: some View {
        VStack(spacing: 20) {
            Spacer()

            Image(systemName: "person.crop.circle.badge.plus")
                .font(.system(size: 60))
                .foregroundColor(.blue)

            Text(loc("login.register_title"))
                .font(.largeTitle)
                .fontWeight(.bold)

            VStack(spacing: 16) {
                TextField(loc("login.account_placeholder"), text: $loginVM.account)
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 300)

                TextField(loc("login.name_placeholder"), text: $loginVM.name)
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

            Button(action: loginVM.register) {
                if loginVM.isLoading {
                    ProgressView()
                        .progressViewStyle(.circular)
                        .scaleEffect(0.8)
                } else {
                    Text(loc("login.register_button"))
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .frame(maxWidth: 300)
            .disabled(loginVM.isLoading)

            Button(loc("login.switch_to_login")) {
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
    RegisterView()
        .environmentObject(LoginViewModel())
}
