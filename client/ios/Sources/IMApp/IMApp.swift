import SwiftUI
import IMCore

@main
struct IMApp: App {
    @StateObject private var loginVM = LoginViewModel()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(loginVM)
        }
    }
}
