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

            Text(loc("login.register_title"))
                .font(.largeTitle)
                .fontWeight(.bold)

            VStack(spacing: 16) {
                TextField(loc("login.account_placeholder"), text: $loginVM.account)
                    .textFieldStyle(.roundedBorder)
                    .autocapitalization(.none)
                    .disableAutocorrection(true)

                TextField(loc("login.name_placeholder"), text: $loginVM.name)
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

            Button(action: loginVM.register) {
                if loginVM.isLoading {
                    ProgressView()
                        .progressViewStyle(.circular)
                        .tint(.white)
                } else {
                    Text(loc("login.register_button"))
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .padding(.horizontal, 40)
            .disabled(loginVM.isLoading)

            Button(loc("login.switch_to_login")) {
                loginVM.switchMode()
            }
            .foregroundColor(.blue)

            Spacer()
        }
        .padding()
    }
}
