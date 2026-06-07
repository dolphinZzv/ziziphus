import SwiftUI
import IMCore

@main
struct IMApp: App {
    @StateObject private var loginVM = LoginViewModel()
    @StateObject private var themeManager = ThemeManager.shared
    @StateObject private var localizationManager = LocalizationManager.shared

    init() {
        let args = ProcessInfo.processInfo.arguments
        if args.contains("-IMClearAuth") {
            AuthManager.shared.logout()
        }
        // -IMToken <token> -IMUserID <userID> -IMUserName <name>
        if let tokenIdx = args.firstIndex(of: "-IMToken"),
           tokenIdx + 1 < args.count {
            let token = args[tokenIdx + 1]
            AuthManager.shared.saveToken(token)
            let userID = args.firstIndex(of: "-IMUserID").flatMap { $0 + 1 < args.count ? args[$0 + 1] : nil } ?? ""
            let userName = args.firstIndex(of: "-IMUserName").flatMap { $0 + 1 < args.count ? args[$0 + 1] : nil } ?? ""
            AuthManager.shared.setLoggedIn(user: User(userID: userID, name: userName))
        }
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(loginVM)
                .environmentObject(themeManager)
                .environmentObject(localizationManager)
                .preferredColorScheme(themeManager.resolvedColorScheme)
                .frame(minWidth: 400, minHeight: 600)
        }
        .commands {
            CommandMenu(loc("profile.settings")) {
                Picker(loc("settings.language"), selection: $localizationManager.currentLanguage) {
                    ForEach(Language.allCases, id: \.self) { lang in
                        Text(lang.displayName).tag(lang)
                    }
                }
                Divider()
                Picker(loc("settings.theme"), selection: $themeManager.currentTheme) {
                    ForEach(AppTheme.allCases, id: \.self) { theme in
                        Text(theme.displayName).tag(theme)
                    }
                }
                Divider()
                Button(loc("login.logout")) {
                    WebSocketClient.shared.disconnect()
                    AuthManager.shared.logout()
                    loginVM.isLoggedIn = false
                }
                .keyboardShortcut("q", modifiers: [.command, .shift])
            }
        }
    }
}
