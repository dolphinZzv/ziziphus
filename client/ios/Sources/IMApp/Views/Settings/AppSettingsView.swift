import SwiftUI
import IMCore

struct AppSettingsView: View {
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @Environment(\.dismiss) private var dismiss

    @State private var serverInfo: ConversationService.ServerVersionInfo?
    @State private var isLoadingServerInfo = true

    private let convService = ConversationService.shared

    var body: some View {
        NavigationStack {
            Form {
                Section(loc("settings.server")) {
                    TextField(loc("settings.server_url"), text: $appSettings.serverURL)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .keyboardType(.URL)
                }

                Section(loc("settings.display")) {
                    Picker(loc("settings.theme"), selection: $themeManager.currentTheme) {
                        ForEach(AppTheme.allCases, id: \.self) { theme in
                            Text(theme.displayName).tag(theme)
                        }
                    }
                    .pickerStyle(.menu)

                    Picker(loc("settings.language"), selection: $localizationManager.currentLanguage) {
                        ForEach(Language.allCases, id: \.self) { lang in
                            Text(lang.displayName).tag(lang)
                        }
                    }
                    .pickerStyle(.menu)
                }

                Section("聊天气泡") {
                    BubbleColorPicker(selectedHex: $appSettings.bubbleColorHex)
                }

                Section(loc("settings.device")) {
                    HStack {
                        Text(loc("settings.session_id"))
                        Spacer()
                        Text(AuthManager.shared.sessionID ?? "")
                            .textSelection(.enabled)
                            .foregroundColor(.secondary)
                    }
                    HStack {
                        Text(loc("settings.device_id"))
                        Spacer()
                        Text(DeviceManager.shared.deviceID)
                            .textSelection(.enabled)
                            .foregroundColor(.secondary)
                    }
                }

                Section(loc("settings.app_info")) {
                    HStack {
                        Text(loc("settings.client_version"))
                        Spacer()
                        Text("\(convService.clientVersion) (\(convService.clientBuild))")
                            .foregroundColor(.secondary)
                    }
                    HStack {
                        Text(loc("settings.build"))
                        Spacer()
                        Text(convService.clientGitHash)
                            .foregroundColor(.secondary)
                    }
                    HStack {
                        Text(loc("settings.server_version"))
                        Spacer()
                        if isLoadingServerInfo {
                            ProgressView().scaleEffect(0.8)
                        } else {
                            Text(serverInfo?.version ?? loc("common.unknown"))
                                .foregroundColor(.secondary)
                        }
                    }
                    HStack {
                        Text(loc("settings.server_build"))
                        Spacer()
                        if isLoadingServerInfo {
                            ProgressView().scaleEffect(0.8)
                        } else {
                            Text(serverInfo?.gitCommit ?? loc("common.unknown"))
                                .foregroundColor(.secondary)
                        }
                    }
                }
            }
            .navigationTitle(loc("settings.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(loc("common.done")) { dismiss() }
                }
            }
            .task {
                do {
                    serverInfo = try await convService.fetchServerVersion()
                } catch {
                    serverInfo = nil
                }
                isLoadingServerInfo = false
            }
        }
    }
}
