import SwiftUI
import IMCore

struct ProfileView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @State private var showLogoutAlert = false

    var body: some View {
        List {
            // User info
            Section {
                HStack(spacing: 16) {
                    AvatarView(
                        name: AuthManager.shared.currentUser?.name ?? "",
                        url: AuthManager.shared.currentUser?.avatar ?? "",
                        size: 60
                    )

                    VStack(alignment: .leading, spacing: 4) {
                        Text(AuthManager.shared.currentUser?.name ?? "")
                            .font(.title2)
                            .fontWeight(.semibold)
                        HStack(spacing: 4) {
                            Text(loc("profile.account_label"))
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                            Text(AuthManager.shared.currentUser?.account ?? "")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                                .textSelection(.enabled)
                        }
                        HStack(spacing: 4) {
                            Text(loc("profile.id_label"))
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                            Text(AuthManager.shared.currentUser?.userID ?? "")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                                .textSelection(.enabled)
                        }
                    }
                }
                .padding(.vertical, 8)
            }

            // Settings
            Section {
                NavigationLink {
                    AppSettingsView()
                        .environmentObject(appSettings)
                        .environmentObject(themeManager)
                        .environmentObject(localizationManager)
                } label: {
                    Label(loc("settings.title"), systemImage: "gearshape")
                }
            }

            // Actions
            Section {
                Button(role: .destructive, action: { showLogoutAlert = true }) {
                    Label(loc("login.logout"), systemImage: "rectangle.portrait.and.arrow.right")
                }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle(loc("profile.title"))
        .navigationBarTitleDisplayMode(.inline)
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
