import SwiftUI
import IMCore
import UniformTypeIdentifiers

struct ProfileView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @State private var showLogoutAlert = false
    @State private var showSettings = false
    @State private var showSessionManage = false
    @State private var isEditing = false
    @State private var editName = ""
    @State private var editAvatar = ""
    @State private var editPrimaryColor = Color.blue
    @State private var editSecondaryColor = Color.blue
    @State private var isSaving = false
    @State private var showImagePicker = false
    @Environment(\.dismiss) private var dismiss

    private var user: User? { AuthManager.shared.currentUser }

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text(loc("profile.title"))
                    .font(.appleBodySemibold)
                    .foregroundColor(AppleDesign.Colors.ink)
                Spacer()
                if isEditing {
                    Button(loc("common.save")) { saveProfile() }
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.actionBlue)
                        .disabled(isSaving)
                } else {
                    Button(loc("common.edit")) { startEditing() }
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.actionBlue)
                    Button(loc("common.done")) { dismiss() }
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.actionBlue)
                }
            }
            .padding(AppleDesign.Spacing.lg)

            Divider()
                .foregroundColor(AppleDesign.Colors.hairline)

            ScrollView {
                // User info card
                VStack(spacing: AppleDesign.Spacing.sm) {
                AvatarView(
                    name: isEditing ? editName : (user?.name ?? ""),
                    url: isEditing ? editAvatar : (user?.avatar ?? ""),
                    size: 56,
                    primaryColor: hexString(from: editPrimaryColor) ?? "",
                    secondaryColor: hexString(from: editSecondaryColor) ?? ""
                )

                if isEditing {
                    TextField(loc("profile.name_placeholder"), text: $editName)
                        .font(.appleTitle)
                        .textFieldStyle(.roundedBorder)
                        .frame(maxWidth: 200)
                        .multilineTextAlignment(.center)
                } else {
                    Text(user?.name ?? "")
                        .font(.appleTitle)
                        .foregroundColor(AppleDesign.Colors.ink)
                }

                HStack(spacing: 4) {
                    Text(loc("profile.account_label"))
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    Text(user?.account ?? "")
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                        .textSelection(.enabled)
                }
                HStack(spacing: 4) {
                    Text("ID:")
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    Text(user?.userID ?? "")
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                        .textSelection(.enabled)
                }
            }
            .padding(.vertical, 24)

            if isEditing {
                VStack(spacing: 8) {
                    Button(action: { showImagePicker = true }) {
                        HStack {
                            Image(systemName: "photo")
                                .foregroundColor(AppleDesign.Colors.actionBlue)
                            Text(loc("profile.change_avatar"))
                                .font(.appleBody)
                        }
                    }
                    .buttonStyle(.plain)

                    ColorPicker(loc("profile.primary_color"), selection: $editPrimaryColor, supportsOpacity: false)
                    ColorPicker(loc("profile.secondary_color"), selection: $editSecondaryColor, supportsOpacity: false)
                }
                .padding(.horizontal, AppleDesign.Spacing.lg)
                .padding(.vertical, AppleDesign.Spacing.sm)

                Divider()
                    .foregroundColor(AppleDesign.Colors.hairline)
            }

            // Settings section
            VStack(spacing: 8) {
                Button(action: { showSettings = true }) {
                    Label(loc("settings.title"), systemImage: "gearshape")
                        .font(.appleBody)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.plain)
                .padding(.horizontal)

                Button(action: { showSessionManage = true }) {
                    Label(loc("settings.device_management"), systemImage: "ipad.and.iphone")
                        .font(.appleBody)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.plain)
                .padding(.horizontal)
            }
            .padding(.vertical, AppleDesign.Spacing.md)
            }
            .padding(.vertical, 24)

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
        .overlay {
            if isSaving {
                ProgressView()
                    .padding()
                    .background(.ultraThinMaterial)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
            }
        }
        .sheet(isPresented: $showSettings) {
            AppSettingsView()
                .environmentObject(appSettings)
                .environmentObject(themeManager)
                .environmentObject(localizationManager)
        }
        .sheet(isPresented: $showSessionManage) {
            SessionManageView()
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
        .fileImporter(isPresented: $showImagePicker, allowedContentTypes: [.image]) { result in
            switch result {
            case .success(let url):
                guard url.startAccessingSecurityScopedResource() else { return }
                defer { url.stopAccessingSecurityScopedResource() }
                guard let data = try? Data(contentsOf: url) else { return }
                Task { await uploadAvatar(data: data) }
            case .failure:
                break
            }
        }
    }

    private func startEditing() {
        editName = user?.name ?? ""
        editAvatar = user?.avatar ?? ""
        editPrimaryColor = color(from: user?.primaryColor ?? "") ?? .blue
        editSecondaryColor = color(from: user?.secondaryColor ?? "") ?? .blue
        withAnimation { isEditing = true }
    }

    private func saveProfile() {
        isSaving = true
        Task {
            do {
                let pc = hexString(from: editPrimaryColor) ?? ""
                let sc = hexString(from: editSecondaryColor) ?? ""
                _ = try await AuthService.shared.updateProfile(
                    name: editName,
                    avatar: editAvatar,
                    primaryColor: pc,
                    secondaryColor: sc
                )
                await MainActor.run {
                    withAnimation { isEditing = false }
                    isSaving = false
                }
            } catch {
                await MainActor.run { isSaving = false }
            }
        }
    }

    private func uploadAvatar(data: Data) async {
        await MainActor.run { isSaving = true }
        do {
            let finfo = try await APIClient.shared.uploadFile(
                fileData: data,
                fileName: "avatar.jpg",
                fileType: 0
            )
            await MainActor.run {
                editAvatar = finfo.url
            }
        } catch {
            print("[Profile] avatar upload error: \(error)")
        }
        await MainActor.run { isSaving = false }
    }

    private func color(from hex: String) -> Color? {
        guard !hex.isEmpty else { return nil }
        return Color(hex: hex)
    }

    private func hexString(from color: Color) -> String? {
        let nsColor = NSColor(color)
        var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
        guard let rgb = nsColor.usingColorSpace(.sRGB) else { return nil }
        rgb.getRed(&r, green: &g, blue: &b, alpha: &a)
        return String(format: "#%02X%02X%02X", Int(r * 255), Int(g * 255), Int(b * 255))
    }
}
