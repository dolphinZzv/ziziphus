import SwiftUI
import IMCore

struct LoginView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @State private var showPassword = false

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
                if !loginVM.rememberedAccounts.isEmpty {
                    HStack(spacing: 12) {
                        ForEach(loginVM.rememberedAccounts, id: \.self) { acct in
                            AvatarCircle(account: acct, isSelected: acct == loginVM.account)
                                .onTapGesture { loginVM.selectAccount(acct) }
                                .contextMenu {
                                    Button(loc("common.delete"), role: .destructive) {
                                        loginVM.removeRememberedAccount(acct)
                                    }
                                }
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .center)
                }

                TextField(loc("login.account_placeholder"), text: $loginVM.account)
                    .textFieldStyle(.roundedBorder)
                    .autocapitalization(.none)
                    .disableAutocorrection(true)

                ZStack(alignment: .trailing) {
                    if showPassword {
                        TextField(loc("login.password_placeholder"), text: $loginVM.password)
                            .textFieldStyle(.roundedBorder)
                            .autocapitalization(.none)
                            .disableAutocorrection(true)
                    } else {
                        SecureField(loc("login.password_placeholder"), text: $loginVM.password)
                            .textFieldStyle(.roundedBorder)
                    }

                    Button(action: { showPassword.toggle() }) {
                        Image(systemName: showPassword ? "eye" : "eye.slash")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .padding(.trailing, 8)
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
    }
}

#Preview {
    LoginView()
        .environmentObject(LoginViewModel())
}

// MARK: - Account avatar circle

struct AvatarCircle: View {
    let account: String
    var isSelected: Bool = false

    private static let avatarColors: [Color] = [
        .red, .orange, .yellow, .green, .mint, .teal, .cyan,
        .blue, .indigo, .purple, .pink, .brown
    ]

    private var color: Color {
        let index = abs(account.hash) % Self.avatarColors.count
        return Self.avatarColors[index]
    }

    private var initial: String {
        String(account.prefix(1)).uppercased()
    }

    var body: some View {
        ZStack {
            Circle()
                .fill(color)
                .frame(width: 40, height: 40)

            Text(initial)
                .font(.system(size: 16, weight: .semibold))
                .foregroundColor(.white)

            if isSelected {
                Circle()
                    .stroke(Color.blue, lineWidth: 2.5)
                    .frame(width: 40, height: 40)
            }
        }
        .frame(width: 44, height: 44)
    }
}
