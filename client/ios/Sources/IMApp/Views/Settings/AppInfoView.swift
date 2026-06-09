import SwiftUI
import IMCore

struct AppInfoView: View {
    @State private var serverInfo: ConversationService.ServerVersionInfo?
    @State private var isLoading = true

    private let convService = ConversationService.shared

    var body: some View {
        Form {
            Section(loc("settings.app_info")) {
                HStack {
                    Text(loc("settings.client_version"))
                    Spacer()
                    Text("\(convService.clientVersion) (\(convService.clientBuild))")
                        .foregroundColor(.secondary)
                }

                HStack {
                    Text(loc("settings.build"))
                    Spacer()
                    Text(convService.clientGitHash)
                        .foregroundColor(.secondary)
                }

                HStack {
                    Text(loc("settings.server_version"))
                    Spacer()
                    if isLoading {
                        ProgressView()
                            .scaleEffect(0.8)
                    } else {
                        Text(serverInfo?.version ?? loc("common.unknown"))
                            .foregroundColor(.secondary)
                    }
                }

                HStack {
                    Text(loc("settings.server_build"))
                    Spacer()
                    if isLoading {
                        ProgressView()
                            .scaleEffect(0.8)
                    } else {
                        Text(serverInfo?.gitCommit ?? loc("common.unknown"))
                            .foregroundColor(.secondary)
                    }
                }
            }
        }
        .navigationTitle(loc("settings.app_info"))
        .navigationBarTitleDisplayMode(.inline)
        .task {
            do {
                serverInfo = try await convService.fetchServerVersion()
            } catch {
                serverInfo = nil
            }
            isLoading = false
        }
    }
}
