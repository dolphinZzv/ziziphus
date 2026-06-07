import SwiftUI
import IMCore

struct ProfileView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @State private var showLogoutAlert = false
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text(loc("profile.title"))
                    .font(.headline)
                Spacer()
                Button(loc("common.done")) { dismiss() }
            }
            .padding()

            Divider()

            // User info
            VStack(spacing: 12) {
                AvatarView(
                    name: AuthManager.shared.currentUser?.name ?? "",
                    url: AuthManager.shared.currentUser?.avatar ?? "",
                    size: 56
                )

                Text(AuthManager.shared.currentUser?.name ?? "")
                    .font(.title3)
                    .fontWeight(.semibold)

                HStack(spacing: 4) {
                    Text(loc("profile.account_label"))
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(AuthManager.shared.currentUser?.account ?? "")
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .textSelection(.enabled)
                }
                HStack(spacing: 4) {
                    Text("ID:")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(AuthManager.shared.currentUser?.userID ?? "")
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .textSelection(.enabled)
                }
            }
            .padding(.vertical, 24)

            // Settings section
            VStack(spacing: 8) {
                Picker(loc("settings.theme"), selection: $themeManager.currentTheme) {
                    ForEach(AppTheme.allCases, id: \.self) { theme in
                        Text(theme.displayName).tag(theme)
                    }
                }
                .pickerStyle(.segmented)

                Picker(loc("settings.language"), selection: $localizationManager.currentLanguage) {
                    ForEach(Language.allCases, id: \.self) { lang in
                        Text(lang.displayName).tag(lang)
                    }
                }
                .pickerStyle(.segmented)
            }
            .padding(.horizontal)
            .padding(.bottom, 16)

            Spacer()

            Divider()

            // Logout button
            Button(role: .destructive, action: { showLogoutAlert = true }) {
                Label(loc("login.logout"), systemImage: "rectangle.portrait.and.arrow.right")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .tint(.red)
            .controlSize(.large)
            .padding()
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
