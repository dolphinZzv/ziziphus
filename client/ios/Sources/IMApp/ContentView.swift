import SwiftUI
import IMCore

struct ContentView: View {
    @EnvironmentObject private var loginVM: LoginViewModel

    var body: some View {
        Group {
            if loginVM.isLoggedIn {
                MainTabView()
            } else {
                if loginVM.isRegistering {
                    RegisterView()
                } else {
                    LoginView()
                }
            }
        }
        .task {
            await loginVM.checkExistingSession()
        }
    }
}

struct MainTabView: View {
    var body: some View {
        TabView {
            NavigationStack {
                ConversationListView()
            }
            .tabItem {
                Label("会话", systemImage: "message.fill")
            }

            NavigationStack {
                ContactListView()
                    .navigationTitle("联系人")
            }
            .tabItem {
                Label("联系人", systemImage: "person.crop.circle")
            }
        }
    }
}
