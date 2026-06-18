import SwiftUI
import IMCore

struct ContentView: View {
    @EnvironmentObject private var loginVM: LoginViewModel

    var body: some View {
        Group {
            if loginVM.isCheckingSession {
                Color(.systemBackground)
                    .ignoresSafeArea()
            } else if loginVM.isLoggedIn {
                MainTabView()
            } else if loginVM.isRegistering {
                RegisterView()
            } else if !loginVM.rememberedAccounts.isEmpty {
                AccountSelectView()
            } else {
                LoginView()
            }
        }
        .task {
            await loginVM.checkExistingSession()
        }
    }
}

#Preview {
    ContentView()
        .environmentObject(LoginViewModel())
}

struct MainTabView: View {
    var body: some View {
        TabView {
            NavigationStack {
                ConversationListView()
            }
            .tabItem {
                Label(loc("conv.title"), systemImage: "message.fill")
            }

            NavigationStack {
                ContactListView()
            }
            .tabItem {
                Label(loc("contact.title"), systemImage: "person.crop.circle")
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
