import SwiftUI
import IMCore

struct LoginView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @State private var showErrorAlert = false
    @State private var showPassword = false
    @State private var showSettings = false

    var body: some View {
        VStack(spacing: AppleDesign.Spacing.xl) {
            HStack {
                Spacer()
                Button(action: { showSettings = true }) {
                    Image(systemName: "gearshape.fill")
                        .font(.body)
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal)

            Spacer()

            Image(systemName: "message.fill")
                .font(.system(size: 60))
                .foregroundColor(.blue)

            Text("DolphinZ")
                .font(.largeTitle)
                .fontWeight(.bold)

            VStack(spacing: AppleDesign.Spacing.sm) {
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

                AppleTextField(placeholder: loc("login.account_placeholder"), text: $loginVM.account)

                ZStack(alignment: .trailing) {
                    if showPassword {
                        TextField(loc("login.password_placeholder"), text: $loginVM.password)
                            .textFieldStyle(.plain)
                            .font(.system(size: AppleDesign.Typography.bodySize))
                            .padding(.horizontal, 16)
                            .padding(.vertical, 12)
                            .background(Color(nsColor: .controlBackgroundColor))
                            .clipShape(Capsule())
                            .onSubmit { loginVM.login() }
                    } else {
                        AppleSecureField(placeholder: loc("login.password_placeholder"), text: $loginVM.password, onSubmit: loginVM.login)
                    }

                    Button(action: { showPassword.toggle() }) {
                        Image(systemName: showPassword ? "eye" : "eye.slash")
                            .font(.caption)
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }
                    .buttonStyle(.plain)
                    .padding(.trailing, 14)
                }

                Toggle(isOn: $loginVM.rememberAccount) {
                    Text(loc("login.remember_account"))
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                }
                .toggleStyle(.checkbox)
            }
            .frame(width: 320)

            Button(action: loginVM.login) {
                if loginVM.isLoading {
                    ProgressView()
                        .progressViewStyle(.circular)
                        .scaleEffect(0.8)
                        .tint(.white)
                } else {
                    Text(loc("login.login_button"))
                        .font(.system(size: AppleDesign.Typography.bodySize))
                }
            }
            .buttonStyle(ApplePrimaryButtonStyle())
            .frame(width: 320)
            .disabled(loginVM.isLoading)

            Button(loc("login.switch_to_register")) {
                loginVM.switchMode()
            }
            .buttonStyle(AppleSecondaryPillStyle())

            Spacer()
        }
        .padding(AppleDesign.Spacing.lg)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .onChange(of: loginVM.errorMessage) { _, msg in
            if msg != nil { showErrorAlert = true }
        }
        .onChange(of: loginVM.account) { _, _ in
            loginVM.showAccountPicker = false
        }
        .alert(loc("common.error"), isPresented: $showErrorAlert) {
            Button(loc("common.confirm"), role: .cancel) {
                loginVM.errorMessage = nil
            }
        } message: {
            Text(loginVM.errorMessage ?? "")
        }
        .background(
            Color.clear
                .contentShape(Rectangle())
                .onTapGesture { }
        )
        .sheet(isPresented: $showSettings) {
            AppSettingsView()
                .environmentObject(appSettings)
                .environmentObject(themeManager)
                .environmentObject(localizationManager)
        }
    }
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

// MARK: - No-border Apple-style text fields

struct AppleTextField: View {
    let placeholder: String
    @Binding var text: String

    var body: some View {
        TextField(placeholder, text: $text)
            .textFieldStyle(.plain)
            .font(.system(size: AppleDesign.Typography.bodySize))
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color(nsColor: .controlBackgroundColor))
            .clipShape(Capsule())
    }
}

struct AppleSecureField: View {
    let placeholder: String
    @Binding var text: String
    var onSubmit: (() -> Void)?

    var body: some View {
        SecureField(placeholder, text: $text)
            .textFieldStyle(.plain)
            .font(.system(size: AppleDesign.Typography.bodySize))
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color(nsColor: .controlBackgroundColor))
            .clipShape(Capsule())
            .onSubmit { onSubmit?() }
    }
}

#if DEBUG
struct LoginView_Previews: PreviewProvider {
    static var previews: some View {
        LoginView()
            .environmentObject(LoginViewModel())
    }
}
#endif
