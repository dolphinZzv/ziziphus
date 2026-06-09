import SwiftUI
import IMCore

@main
struct IMApp: App {
    @StateObject private var loginVM = LoginViewModel()
    @StateObject private var appSettings = AppSettings.shared
    @StateObject private var themeManager = ThemeManager.shared
    @StateObject private var localizationManager = LocalizationManager.shared

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(loginVM)
                .environmentObject(appSettings)
                .environmentObject(themeManager)
                .environmentObject(localizationManager)
                .preferredColorScheme(themeManager.resolvedColorScheme)
                .id(localizationManager.refreshVersion)
        }
    }
}

#Preview {
    ContentView()
        .environmentObject(LoginViewModel())
}
