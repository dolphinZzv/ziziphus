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
                    AgentManageView()
                } label: {
                    Label(loc("agent.manage"), systemImage: "brain")
                }

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

// MARK: - Agent Manage View

struct AgentManageView: View {
    @State private var agents: [User] = []
    @State private var isLoading = true
    @State private var showCreateSheet = false
    @State private var editingAgent: User?

    var body: some View {
        List {
            if isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity)
            }

            ForEach(agents) { agent in
                Button {
                    editingAgent = agent
                } label: {
                    HStack(spacing: 12) {
                        AvatarView(name: agent.name, url: agent.avatar, size: 40)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(agent.name)
                                .fontWeight(.medium)
                            Text(loc("agent.type_label"))
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                        if !agent.apiKey.isEmpty {
                            Image(systemName: "key.fill")
                                .font(.caption2)
                                .foregroundColor(.green)
                        }
                        Image(systemName: "chevron.right")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .padding(.vertical, 4)
                }
            }
            .onDelete { indexSet in
                for idx in indexSet { deleteAgent(agents[idx]) }
            }

            if agents.count < 10 && !isLoading {
                Section {
                    Button {
                        showCreateSheet = true
                    } label: {
                        Label(loc("agent.create"), systemImage: "plus")
                    }
                }
            } else if agents.count >= 10 {
                Section {
                    HStack {
                        Spacer()
                        Text(loc("agent.limit_reached"))
                            .font(.footnote)
                            .foregroundColor(.secondary)
                        Spacer()
                    }
                }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle(loc("agent.manage"))
        .toolbar {
            if agents.count < 10 && !isLoading {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button { showCreateSheet = true } label: {
                        Image(systemName: "plus")
                    }
                }
            }
        }
        .sheet(isPresented: $showCreateSheet) {
            AgentEditView(onSave: { name, avatar, wakeMode in
                showCreateSheet = false
                createAgent(name: name, avatar: avatar, wakeMode: wakeMode)
            })
        }
        .sheet(item: $editingAgent) { agent in
            AgentEditView(
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
                        let newKey = try await AuthService.shared.regenerateAgentKey(agentID: agent.userID)
                        // reload agent to get updated apiKey
                        editingAgent = nil
                        loadAgents()
                    } catch {}
                }
            )
        }
        .onAppear { loadAgents() }
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

// MARK: - Agent Edit View

private struct AgentEditView: View {
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
        NavigationStack {
            List {
                Section {
                    HStack(spacing: 12) {
                        AvatarView(name: name, url: avatar, size: 60)
                        VStack {
                            TextField(loc("agent.name_placeholder"), text: $name)
                                .textFieldStyle(.roundedBorder)
                        }
                    }
                    .padding(.vertical, 4)
                }

                Section(loc("agent.wake_mode")) {
                    Picker(loc("agent.wake_mode"), selection: $wakeMode) {
                        Text(loc("agent.wake_all")).tag(0)
                        Text(loc("agent.wake_mention")).tag(1)
                    }
                    .pickerStyle(.segmented)
                }

                if !apiKey.isEmpty {
                    Section(loc("agent.api_key")) {
                        HStack {
                            Text(apiKey)
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .textSelection(.enabled)
                            Spacer()
                            Button {
                                UIPasteboard.general.string = apiKey
                            } label: {
                                Image(systemName: "doc.on.doc")
                                    .font(.caption)
                            }
                        }
                        Button(role: .destructive) {
                            showRegenerateAlert = true
                        } label: {
                            Label(loc("agent.api_key_regenerate"), systemImage: "arrow.triangle.2.circlepath")
                                .font(.caption)
                        }
                    }
                }
            }
            .listStyle(.insetGrouped)
            .navigationTitle(name.isEmpty ? loc("agent.create") : loc("agent.edit"))
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(loc("common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(loc("common.save")) {
                        dismiss()
                        onSave(name, avatar, wakeMode)
                    }
                    .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
        }
        .presentationDetents([.medium])
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
