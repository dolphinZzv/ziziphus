import SwiftUI
import IMCore

@main
struct IMApp: App {
    @StateObject private var loginVM = LoginViewModel()
    @StateObject private var themeManager = ThemeManager.shared
    @StateObject private var localizationManager = LocalizationManager.shared

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(loginVM)
                .environmentObject(themeManager)
                .environmentObject(localizationManager)
                .preferredColorScheme(themeManager.resolvedColorScheme)
                .id(localizationManager.currentLanguage)
        }
    }
}

#Preview {
    ContentView()
        .environmentObject(LoginViewModel())
}
