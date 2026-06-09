import SwiftUI
import IMCore

struct ProfileView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @State private var showLogoutAlert = false
    @State private var showSettings = false
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text(loc("profile.title"))
                    .font(.appleBodySemibold)
                    .foregroundColor(AppleDesign.Colors.ink)
                Spacer()
                Button(loc("common.done")) { dismiss() }
                    .font(.appleBody)
                    .foregroundColor(AppleDesign.Colors.actionBlue)
            }
            .padding(AppleDesign.Spacing.lg)

            Divider()
                .foregroundColor(AppleDesign.Colors.hairline)

            // User info card
            VStack(spacing: AppleDesign.Spacing.sm) {
                AvatarView(
                    name: AuthManager.shared.currentUser?.name ?? "",
                    url: AuthManager.shared.currentUser?.avatar ?? "",
                    size: 56
                )

                Text(AuthManager.shared.currentUser?.name ?? "")
                    .font(.appleTitle)
                    .foregroundColor(AppleDesign.Colors.ink)

                HStack(spacing: 4) {
                    Text(loc("profile.account_label"))
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    Text(AuthManager.shared.currentUser?.account ?? "")
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                        .textSelection(.enabled)
                }
                HStack(spacing: 4) {
                    Text("ID:")
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    Text(AuthManager.shared.currentUser?.userID ?? "")
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                        .textSelection(.enabled)
                }
            }
            .padding(.vertical, 24)

            Divider()
                .foregroundColor(AppleDesign.Colors.hairline)

            // Settings section
            VStack(spacing: 8) {
                Button(action: { showSettings = true }) {
                    Label(loc("settings.title"), systemImage: "gearshape")
                        .font(.appleBody)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.plain)
                .padding(.horizontal)
            }
            .padding(.vertical, AppleDesign.Spacing.md)

            Spacer()

            Divider()
                .foregroundColor(AppleDesign.Colors.hairline)

            // Logout
            Button(action: { showLogoutAlert = true }) {
                Label(loc("login.logout"), systemImage: "rectangle.portrait.and.arrow.right")
                    .font(.appleBody)
                    .frame(maxWidth: .infinity)
                    .foregroundColor(.red)
            }
            .buttonStyle(.plain)
            .padding(AppleDesign.Spacing.lg)
        }
        .background(Color(nsColor: .windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .sheet(isPresented: $showSettings) {
            AppSettingsView()
                .environmentObject(appSettings)
                .environmentObject(themeManager)
                .environmentObject(localizationManager)
        }
        .alert(loc("login.logout"), isPresented: $showLogoutAlert) {
            Button(loc("common.cancel"), role: .cancel) {}
            Button(loc("login.logout"), role: .destructive) {
                WebSocketClient.shared.disconnect()
                AuthManager.shared.logout()
                loginVM.isLoggedIn = false
            }
        } message: {
            Text(loc("login.logout_confirm"))
        }
    }
}
