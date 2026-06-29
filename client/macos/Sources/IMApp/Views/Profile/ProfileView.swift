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
    @State private var showAgentManage = false
    @State private var showMFA = false
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
                Button(action: { showAgentManage = true }) {
                    Label(loc("agent.manage"), systemImage: "brain")
                        .font(.appleBody)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.plain)
                .padding(.horizontal)

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

                Button(action: { showMFA = true }) {
                    Label("双重验证", systemImage: "lock.shield")
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
        .sheet(isPresented: $showAgentManage) {
            AgentManageView()
                .environmentObject(localizationManager)
        }
        .sheet(isPresented: $showMFA) {
            MFASettingsView()
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

// MARK: - Agent Manage View

struct AgentManageView: View {
    @EnvironmentObject private var localizationManager: LocalizationManager
    @Environment(\.dismiss) private var dismiss

    @State private var agents: [User] = []
    @State private var isLoading = true
    @State private var showCreateSheet = false
    @State private var editingAgent: User?
    @State private var deleteAlertAgent: User?

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(loc("agent.manage"))
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

            if isLoading {
                Spacer()
                ProgressView()
                Spacer()
            } else {
                ScrollView {
                    VStack(spacing: 0) {
                        ForEach(agents) { agent in
                            HStack(spacing: 12) {
                                AvatarView(name: agent.name, url: agent.avatar, size: 36)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(agent.name)
                                        .font(.appleBody)
                                        .foregroundColor(AppleDesign.Colors.ink)
                                    Text(loc("agent.type_label"))
                                        .font(.appleCaption)
                                        .foregroundColor(AppleDesign.Colors.inkMuted)
                                }
                                if !agent.apiKey.isEmpty {
                                    Image(systemName: "key.fill")
                                        .font(.caption2)
                                        .foregroundColor(.green)
                                }
                                Spacer()

                                Button(action: { editingAgent = agent }) {
                                    Image(systemName: "pencil")
                                        .font(.caption)
                                        .foregroundColor(AppleDesign.Colors.actionBlue)
                                }
                                .buttonStyle(.plain)

                                Button(action: { deleteAlertAgent = agent }) {
                                    Image(systemName: "trash")
                                        .font(.caption)
                                        .foregroundColor(.red)
                                }
                                .buttonStyle(.plain)
                                .padding(.leading, 4)
                            }
                            .padding(.horizontal, AppleDesign.Spacing.lg)
                            .padding(.vertical, 8)

                            Divider()
                                .foregroundColor(AppleDesign.Colors.hairline)
                        }

                        if agents.count < 10 {
                            Button(action: { showCreateSheet = true }) {
                                Label(loc("agent.create"), systemImage: "plus")
                                    .font(.appleBody)
                                    .frame(maxWidth: .infinity)
                            }
                            .buttonStyle(.plain)
                            .padding(.vertical, 12)
                        } else {
                            Text(loc("agent.limit_reached"))
                                .font(.appleCaption)
                                .foregroundColor(AppleDesign.Colors.inkMuted)
                                .padding(.vertical, 12)
                        }
                    }
                }
            }
        }
        .frame(width: 360, height: 420)
        .background(Color(nsColor: .windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .sheet(isPresented: $showCreateSheet) {
            AgentEditSheet(onSave: { name, avatar, wakeMode in
                showCreateSheet = false
                createAgent(name: name, avatar: avatar, wakeMode: wakeMode)
            })
        }
        .sheet(item: $editingAgent) { agent in
            AgentEditSheet(
                initialName: agent.name,
                initialAvatar: agent.avatar,
                initialWakeMode: agent.wakeMode,
                initialApiKey: agent.apiKey,
                onSave: { name, avatar, wakeMode in
                    editingAgent = nil
                    updateAgent(agent, name: name, avatar: avatar, wakeMode: wakeMode)
                },
                onRegenerate: { [agent] in
                    do {
                        _ = try await AuthService.shared.regenerateAgentKey(agentID: agent.userID)
                        editingAgent = nil
                        loadAgents()
                    } catch {}
                }
            )
        }
        .onAppear { loadAgents() }
        .alert(loc("agent.delete"), isPresented: .constant(deleteAlertAgent != nil)) {
            Button(loc("common.cancel"), role: .cancel) { deleteAlertAgent = nil }
            Button(loc("agent.delete"), role: .destructive) {
                if let agent = deleteAlertAgent {
                    deleteAgent(agent)
                    deleteAlertAgent = nil
                }
            }
        } message: {
            Text(loc("agent.delete_confirm"))
        }
    }

    private func loadAgents() {
        isLoading = true
        Task {
            do { agents = try await AuthService.shared.listAgents() } catch { agents = [] }
            isLoading = false
        }
    }

    private func createAgent(name: String, avatar: String, wakeMode: Int) {
        Task {
            do {
                _ = try await AuthService.shared.createAgent(name: name, avatar: avatar, wakeMode: wakeMode)
                loadAgents()
            } catch {}
        }
    }

    private func updateAgent(_ agent: User, name: String, avatar: String, wakeMode: Int) {
        Task {
            do {
                try await AuthService.shared.updateAgent(agentID: agent.userID, name: name, avatar: avatar, wakeMode: wakeMode)
                loadAgents()
            } catch {}
        }
    }

    private func deleteAgent(_ agent: User) {
        Task {
            do {
                try await AuthService.shared.deleteAgent(agentID: agent.userID)
                loadAgents()
            } catch {}
        }
    }
}

// MARK: - Agent Edit Sheet

private struct AgentEditSheet: View {
    @EnvironmentObject private var localizationManager: LocalizationManager
    @Environment(\.dismiss) private var dismiss

    @State private var name: String
    @State private var avatar: String
    @State private var wakeMode: Int
    @State private var apiKey: String
    @State private var showRegenerateAlert = false

    var onSave: (String, String, Int) -> Void
    var onRegenerate: (() async -> Void)?

    init(initialName: String = "", initialAvatar: String = "", initialWakeMode: Int = 0,
         initialApiKey: String = "",
         onSave: @escaping (String, String, Int) -> Void,
         onRegenerate: (() async -> Void)? = nil) {
        _name = State(initialValue: initialName)
        _avatar = State(initialValue: initialAvatar)
        _wakeMode = State(initialValue: initialWakeMode)
        _apiKey = State(initialValue: initialApiKey)
        self.onSave = onSave
        self.onRegenerate = onRegenerate
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(name.isEmpty ? loc("agent.create") : loc("agent.edit"))
                    .font(.appleBodySemibold)
                    .foregroundColor(AppleDesign.Colors.ink)
                Spacer()
                Button(loc("common.cancel")) { dismiss() }
                    .font(.appleBody)
                    .foregroundColor(AppleDesign.Colors.actionBlue)
            }
            .padding(AppleDesign.Spacing.lg)

            Divider()
                .foregroundColor(AppleDesign.Colors.hairline)

            VStack(spacing: 16) {
                AvatarView(name: name, url: avatar, size: 56)

                TextField(loc("agent.name_placeholder"), text: $name)
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 200)

                VStack(alignment: .leading, spacing: 4) {
                    Text(loc("agent.wake_mode"))
                        .font(.appleCaption)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    Picker(loc("agent.wake_mode"), selection: $wakeMode) {
                        Text(loc("agent.wake_all")).tag(0)
                        Text(loc("agent.wake_mention")).tag(1)
                    }
                    .pickerStyle(.segmented)
                }

                if !apiKey.isEmpty {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(loc("agent.api_key"))
                            .font(.appleCaption)
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                        HStack {
                            Text(apiKey)
                                .font(.appleCaption)
                                .foregroundColor(.secondary)
                                .textSelection(.enabled)
                                .lineLimit(1)
                            Button(action: {
                                NSPasteboard.general.clearContents()
                                NSPasteboard.general.setString(apiKey, forType: .string)
                            }) {
                                Image(systemName: "doc.on.doc")
                                    .font(.caption)
                            }
                            .buttonStyle(.plain)
                        }
                        Button(action: { showRegenerateAlert = true }) {
                            Label(loc("agent.api_key_regenerate"), systemImage: "arrow.triangle.2.circlepath")
                                .font(.appleCaption)
                        }
                        .buttonStyle(.plain)
                        .foregroundColor(.red)
                    }
                }

                Button(action: {
                    dismiss()
                    onSave(name, avatar, wakeMode)
                }) {
                    Text(loc("common.save"))
                        .font(.appleBody)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            .padding(AppleDesign.Spacing.lg)
        }
        .frame(width: 300, height: apiKey.isEmpty ? 300 : 380)
        .background(Color(nsColor: .windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .alert(loc("agent.api_key_regenerate"), isPresented: $showRegenerateAlert) {
            Button(loc("common.cancel"), role: .cancel) {}
            Button(loc("agent.api_key_regenerate"), role: .destructive) {
                Task { await onRegenerate?() }
            }
        } message: {
            Text(loc("agent.api_key_regenerate_confirm"))
        }
    }
}
