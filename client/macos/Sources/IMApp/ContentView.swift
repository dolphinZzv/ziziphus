import SwiftUI
import IMCore

struct ContentView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @EnvironmentObject private var localizationManager: LocalizationManager

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
        .background(VisualEffectView().ignoresSafeArea())
        .task {
            await loginVM.checkExistingSession()
        }
        .onAppear {
            configureWindowForLogin(loginVM.isLoggedIn)
        }
        .onChange(of: loginVM.isLoggedIn) { _, loggedIn in
            configureWindowForLogin(loggedIn)
        }
    }

    private func configureWindowForLogin(_ loggedIn: Bool) {
        guard let window = NSApp.windows.first(where: { $0.isVisible }) else { return }
        if loggedIn {
            window.styleMask.insert(.titled)
            window.styleMask.remove(.fullSizeContentView)
            window.isOpaque = true
            window.backgroundColor = .windowBackgroundColor
            window.titlebarAppearsTransparent = false
            window.titleVisibility = .visible
            window.titlebarSeparatorStyle = .automatic
            window.isMovableByWindowBackground = false
            window.title = ""
        } else {
            window.styleMask.insert(.fullSizeContentView)
            window.isOpaque = false
            window.backgroundColor = .clear
            window.titlebarAppearsTransparent = true
            window.titleVisibility = .hidden
            window.titlebarSeparatorStyle = .none
            window.isMovableByWindowBackground = true
        }
    }
}

struct MainTabView: View {
    @State private var selectedConvID: String?
    @State private var selectedConvName = ""
    @State private var selectedConvType: ConvType = .p2p

    var body: some View {
        NavigationSplitView {
            ConversationListView(selectedConvID: $selectedConvID, onSelectConv: { conv in
                selectedConvID = conv.convID
                selectedConvName = conv.name
                selectedConvType = conv.type
            })
                .frame(minWidth: 240)
        } detail: {
            if let convID = selectedConvID {
                ChatView(convID: convID, convName: selectedConvName, convType: selectedConvType)
                    .id(convID)
                    .background(Color(nsColor: .windowBackgroundColor))
            } else {
                Text(loc("conv.no_conversations"))
                    .foregroundColor(.secondary)
            }
        }
    }
}

#if DEBUG
struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView()
            .environmentObject(LoginViewModel())
    }
}
#endif
