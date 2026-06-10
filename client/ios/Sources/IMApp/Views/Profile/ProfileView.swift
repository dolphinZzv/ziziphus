import SwiftUI
import IMCore
import PhotosUI

struct ProfileView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @State private var showLogoutAlert = false
    @State private var isEditing = false
    @State private var editName = ""
    @State private var editAvatar = ""
    @State private var editPrimaryColor = Color.blue
    @State private var editSecondaryColor = Color.blue
    @State private var showImagePicker = false
    @State private var selectedPhotoItem: PhotosPickerItem?
    @State private var isSaving = false

    private var user: User? { AuthManager.shared.currentUser }

    var body: some View {
        List {
            // User info
            Section {
                HStack(spacing: 16) {
                    ZStack(alignment: .bottomTrailing) {
                        AvatarView(
                            name: isEditing ? editName : (user?.name ?? ""),
                            url: isEditing ? editAvatar : (user?.avatar ?? ""),
                            size: 60,
                            primaryColor: hexString(from: editPrimaryColor) ?? "",
                            secondaryColor: hexString(from: editSecondaryColor) ?? ""
                        )
                        .overlay(
                            Circle()
                                .stroke(Color(.separator), lineWidth: 0.5)
                        )
                    }

                    VStack(alignment: .leading, spacing: 4) {
                        if isEditing {
                            TextField(loc("profile.name_placeholder"), text: $editName)
                                .font(.title2)
                                .textFieldStyle(.roundedBorder)
                        } else {
                            Text(user?.name ?? "")
                                .font(.title2)
                                .fontWeight(.semibold)
                        }
                        HStack(spacing: 4) {
                            Text(loc("profile.account_label"))
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                            Text(user?.account ?? "")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                                .textSelection(.enabled)
                        }
                        HStack(spacing: 4) {
                            Text(loc("profile.id_label"))
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                            Text(user?.userID ?? "")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                                .textSelection(.enabled)
                        }
                    }
                }
                .padding(.vertical, 8)
            }

            if isEditing {
                // Avatar picker
                Section {
                    Button(action: { showImagePicker = true }) {
                        HStack {
                            Image(systemName: "photo")
                                .foregroundColor(.blue)
                            Text(loc("profile.change_avatar"))
                                .foregroundColor(.primary)
                        }
                    }
                }

                // Colors
                Section {
                    ColorPicker(loc("profile.primary_color"), selection: $editPrimaryColor, supportsOpacity: false)
                    ColorPicker(loc("profile.secondary_color"), selection: $editSecondaryColor, supportsOpacity: false)
                }
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

                NavigationLink {
                    SessionManageView()
                } label: {
                    Label(loc("settings.device_management"), systemImage: "ipad.and.iphone")
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
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                if isEditing {
                    Button(loc("common.save")) { saveProfile() }
                        .disabled(isSaving)
                } else {
                    Button(loc("common.edit")) { startEditing() }
                }
            }
        }
        .overlay {
            if isSaving {
                ProgressView()
                    .padding()
                    .background(.ultraThinMaterial)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
            }
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
        .photosPicker(isPresented: $showImagePicker, selection: $selectedPhotoItem, matching: .images)
        .onChange(of: selectedPhotoItem) { _, newItem in
            guard let item = newItem else { return }
            Task {
                guard let data = try? await item.loadTransferable(type: Data.self) else { return }
                await uploadAvatar(data: data)
                selectedPhotoItem = nil
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
        let uiColor = UIColor(color)
        var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
        uiColor.getRed(&r, green: &g, blue: &b, alpha: &a)
        return String(format: "#%02X%02X%02X", Int(r * 255), Int(g * 255), Int(b * 255))
    }
}
