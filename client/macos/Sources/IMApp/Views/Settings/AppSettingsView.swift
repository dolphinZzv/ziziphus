import SwiftUI
import IMCore

struct AppSettingsView: View {
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @State private var showAppInfo = false
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(loc("settings.title"))
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

            ScrollView {
                VStack(spacing: 0) {
                    // Server
                    GroupBox(loc("settings.server")) {
                        TextField(loc("settings.server_url"), text: $appSettings.serverURL)
                            .font(.appleBody)
                            .textFieldStyle(.roundedBorder)
                    }
                    .padding(.horizontal)
                    .padding(.top, AppleDesign.Spacing.md)

                    // Display
                    GroupBox(loc("settings.display")) {
                        VStack(spacing: 12) {
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
                        .padding(.vertical, 4)
                    }
                    .padding(.horizontal)
                    .padding(.top, AppleDesign.Spacing.md)

                    // Bubble color
                    GroupBox("聊天气泡") {
                        BubbleColorPicker(selectedHex: $appSettings.bubbleColorHex)
                            .padding(.vertical, 4)
                    }
                    .padding(.horizontal)
                    .padding(.top, AppleDesign.Spacing.md)

                    // Device
                    GroupBox(loc("settings.device")) {
                        VStack(spacing: 8) {
                            HStack {
                                Text(loc("settings.session_id"))
                                    .font(.appleBody)
                                    .foregroundColor(AppleDesign.Colors.ink)
                                Spacer()
                                Text(AuthManager.shared.sessionID ?? "")
                                    .font(.appleBody)
                                    .foregroundColor(AppleDesign.Colors.inkMuted)
                                    .textSelection(.enabled)
                            }
                            HStack {
                                Text(loc("settings.device_id"))
                                    .font(.appleBody)
                                    .foregroundColor(AppleDesign.Colors.ink)
                                Spacer()
                                Text(DeviceManager.shared.deviceID)
                                    .font(.appleBody)
                                    .foregroundColor(AppleDesign.Colors.inkMuted)
                                    .textSelection(.enabled)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                    .padding(.horizontal)
                    .padding(.top, AppleDesign.Spacing.md)

                    // App Info
                    GroupBox(loc("settings.app_info")) {
                        Button(action: { showAppInfo = true }) {
                            Label(loc("settings.app_info"), systemImage: "info.circle")
                                .font(.appleBody)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        .buttonStyle(.plain)
                        .padding(.vertical, 4)
                    }
                    .padding(.horizontal)
                    .padding(.top, AppleDesign.Spacing.md)

                }
            }

            Spacer()
        }
        .frame(width: 380, height: 400)
        .background(Color(nsColor: .windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .sheet(isPresented: $showAppInfo) {
            AppInfoView()
        }
    }
}
