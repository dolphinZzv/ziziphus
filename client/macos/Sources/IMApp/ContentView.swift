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
        .task {
            await loginVM.checkExistingSession()
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
                .frame(minWidth: 280)
        } detail: {
            if let convID = selectedConvID {
                ChatView(convID: convID, convName: selectedConvName, convType: selectedConvType)
                    .id(convID)
                    .background(Color.white)
            } else {
                Text(loc("conv.no_conversations"))
                    .foregroundColor(.secondary)
            }
        }
    }
}

#Preview {
    ContentView()
        .environmentObject(LoginViewModel())
}
