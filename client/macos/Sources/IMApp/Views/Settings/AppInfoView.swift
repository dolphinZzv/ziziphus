import SwiftUI
import IMCore

struct AppInfoView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var serverInfo: ConversationService.ServerVersionInfo?
    @State private var isLoading = true

    private let convService = ConversationService.shared

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(loc("settings.app_info"))
                    .font(.appleBodySemibold)
                    .foregroundColor(AppleDesign.Colors.ink)
                Spacer()
                Button(loc("common.done")) { dismiss() }
                    .font(.appleBody)
                    .foregroundColor(AppleDesign.Colors.actionBlue)
            }
            .padding(AppleDesign.Spacing.lg)

            Divider()
                .foregroundColor(AppleDesign.Colors.hairline)

            VStack(spacing: 12) {
                HStack {
                    Text(loc("settings.client_version"))
                        .font(.appleBody)
                    Spacer()
                    Text("\(convService.clientVersion) (\(convService.clientBuild))")
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                }

                Divider()

                HStack {
                    Text(loc("settings.build"))
                        .font(.appleBody)
                    Spacer()
                    Text(convService.clientGitHash)
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                }

                Divider()

                HStack {
                    Text(loc("settings.server_version"))
                        .font(.appleBody)
                    Spacer()
                    if isLoading {
                        ProgressView()
                            .scaleEffect(0.8)
                    } else {
                        Text(serverInfo?.version ?? loc("common.unknown"))
                            .font(.appleBody)
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }
                }

                Divider()

                HStack {
                    Text(loc("settings.server_build"))
                        .font(.appleBody)
                    Spacer()
                    if isLoading {
                        ProgressView()
                            .scaleEffect(0.8)
                    } else {
                        Text(serverInfo?.gitCommit ?? loc("common.unknown"))
                            .font(.appleBody)
                            .foregroundColor(AppleDesign.Colors.inkMuted)
                    }
                }
            }
            .padding(AppleDesign.Spacing.lg)

            Spacer()
        }
        .frame(width: 420, height: 260)
        .background(Color(nsColor: .windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 18))
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
