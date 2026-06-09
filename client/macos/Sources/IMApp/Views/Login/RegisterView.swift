import SwiftUI
import IMCore

struct RegisterView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @State private var showErrorAlert = false

    var body: some View {
        VStack(spacing: AppleDesign.Spacing.xl) {
            Spacer()

            Text(loc("login.register_title"))
                .font(.appleDisplay)
                .foregroundColor(AppleDesign.Colors.ink)
                .kerning(-0.374)

            VStack(spacing: AppleDesign.Spacing.sm) {
                AppleTextField(placeholder: loc("login.account_placeholder"), text: $loginVM.account)

                AppleTextField(placeholder: loc("login.name_placeholder"), text: $loginVM.name)

                AppleSecureField(placeholder: loc("login.password_placeholder"), text: $loginVM.password, onSubmit: loginVM.register)
            }
            .frame(width: 320)

            Button(action: loginVM.register) {
                if loginVM.isLoading {
                    ProgressView()
                        .progressViewStyle(.circular)
                        .scaleEffect(0.8)
                        .tint(.white)
                } else {
                    Text(loc("login.register_button"))
                        .font(.system(size: AppleDesign.Typography.bodySize))
                }
            }
            .buttonStyle(ApplePrimaryButtonStyle())
            .frame(width: 320)
            .disabled(loginVM.isLoading)

            Button(loc("login.switch_to_login")) {
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
        .alert(loc("common.error"), isPresented: $showErrorAlert) {
            Button(loc("common.confirm"), role: .cancel) {
                loginVM.errorMessage = nil
            }
        } message: {
            Text(loginVM.errorMessage ?? "")
        }
    }
}

#if DEBUG
struct RegisterView_Previews: PreviewProvider {
    static var previews: some View {
        RegisterView()
            .environmentObject(LoginViewModel())
    }
}
#endif
