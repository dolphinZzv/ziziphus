import SwiftUI
import IMCore

struct LoginView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @State private var showPassword = false
    @State private var showSettings = false

    var body: some View {
        VStack(spacing: 24) {
            HStack {
                Spacer()
                Button(action: { showSettings = true }) {
                    Image(systemName: "gearshape.fill")
                        .font(.body)
                        .foregroundColor(.secondary)
                }
            }
            .padding(.horizontal)

            Spacer()
                .frame(height: 40)

            Image(systemName: "message.fill")
                .font(.system(size: 60))
                .foregroundColor(.blue)

            Text("PandaAI")
                .font(.largeTitle)
                .fontWeight(.bold)

            VStack(spacing: 16) {
                CapsuleTextField(placeholder: loc("login.account_placeholder"), text: $loginVM.account)

                ZStack(alignment: .trailing) {
                    if showPassword {
                        CapsuleTextField(placeholder: loc("login.password_placeholder"), text: $loginVM.password)
                            .onSubmit { loginVM.login() }
                    } else {
                        CapsuleSecureField(placeholder: loc("login.password_placeholder"), text: $loginVM.password, onSubmit: loginVM.login)
                    }

                    Button(action: { showPassword.toggle() }) {
                        Image(systemName: showPassword ? "eye" : "eye.slash")
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .frame(width: 44, height: 44)
                    }
                    .padding(.trailing, 4)
                }

                Toggle(loc("login.remember_account"), isOn: $loginVM.rememberAccount)
                    .font(.caption)
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
        .sheet(isPresented: $showSettings) {
            AppSettingsView()
                .environmentObject(appSettings)
                .environmentObject(themeManager)
                .environmentObject(localizationManager)
        }
    }
}

#Preview {
    LoginView()
        .environmentObject(LoginViewModel())
}

// MARK: - Capsule-style text fields matching macOS style

struct CapsuleTextField: View {
    let placeholder: String
    @Binding var text: String

    var body: some View {
        TextField(placeholder, text: $text)
            .textFieldStyle(.plain)
            .font(.body)
            .autocapitalization(.none)
            .disableAutocorrection(true)
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color(.systemGray6))
            .clipShape(Capsule())
    }
}

struct CapsuleSecureField: View {
    let placeholder: String
    @Binding var text: String
    var onSubmit: (() -> Void)?

    var body: some View {
        SecureField(placeholder, text: $text)
            .textFieldStyle(.plain)
            .font(.body)
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color(.systemGray6))
            .clipShape(Capsule())
            .onSubmit { onSubmit?() }
    }
}
