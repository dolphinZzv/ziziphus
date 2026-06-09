import SwiftUI
import IMCore

struct AppSettingsView: View {
    @EnvironmentObject private var appSettings: AppSettings
    @EnvironmentObject private var themeManager: ThemeManager
    @EnvironmentObject private var localizationManager: LocalizationManager
    @Environment(\.dismiss) private var dismiss

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

                Section {
                    NavigationLink {
                        AppInfoView()
                    } label: {
                        Label(loc("settings.app_info"), systemImage: "info.circle")
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
        }
    }
}
