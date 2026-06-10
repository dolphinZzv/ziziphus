import SwiftUI
import IMCore

struct SessionManageView: View {
    @State private var sessions: [DeviceSession] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var kickConfirmSession: DeviceSession?

    private let sessionService = SessionService.shared
    private var currentSessionID: String? { AuthManager.shared.sessionID }

    var body: some View {
        Group {
            if isLoading {
                ProgressView(loc("common.loading"))
            } else if let error = errorMessage {
                VStack(spacing: 12) {
                    Text(error)
                        .foregroundColor(.secondary)
                    Button(loc("common.confirm")) {
                        loadSessions()
                    }
                }
            } else {
                List {
                    Section {
                        ForEach(sessions) { session in
                            HStack(spacing: 12) {
                                Image(systemName: deviceIcon(session.device))
                                    .font(.title2)
                                    .foregroundColor(session.isOnline ? .green : .secondary)
                                    .frame(width: 28)

                                VStack(alignment: .leading, spacing: 2) {
                                    Text(deviceLabel(session))
                                        .fontWeight(.medium)
                                    Text(formatDate(session.loginAt))
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                    if let ip = session.clientIP, !ip.isEmpty {
                                        Text(ip)
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                    if let devID = session.deviceID, !devID.isEmpty {
                                        Text(devID)
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                }

                                Spacer()

                                if session.isOnline {
                                    Circle()
                                        .fill(Color.green)
                                        .frame(width: 8, height: 8)
                                }

                                if session.sessionID == currentSessionID {
                                    Text(loc("settings.current_device"))
                                        .font(.caption)
                                        .foregroundColor(.blue)
                                }

                                if session.sessionID != currentSessionID {
                                    Button(action: { kickConfirmSession = session }) {
                                        Image(systemName: "xmark.circle")
                                            .foregroundColor(.red)
                                            .font(.title3)
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                            .swipeActions(edge: .trailing) {
                                if session.sessionID != currentSessionID {
                                    Button(role: .destructive) {
                                        kickConfirmSession = session
                                    } label: {
                                        Label(loc("settings.kick_device"), systemImage: "xmark.circle")
                                    }
                                }
                            }
                        }
                    } header: {
                        Text(String(format: loc("group.member_count"), sessions.count))
                    }
                }
                .listStyle(.insetGrouped)
                .refreshable { loadSessions() }
            }
        }
        .navigationTitle(loc("settings.device_management"))
        .navigationBarTitleDisplayMode(.inline)
        .onAppear { loadSessions() }
        .alert(loc("settings.kick_device"), isPresented: .init(
            get: { kickConfirmSession != nil },
            set: { if !$0 { kickConfirmSession = nil } }
        )) {
            Button(loc("common.cancel"), role: .cancel) { kickConfirmSession = nil }
            Button(loc("settings.kick_device"), role: .destructive) {
                if let session = kickConfirmSession {
                    kickSession(session)
                }
            }
        } message: {
            Text(kickConfirmSession.map { "\(loc("settings.kick_device_confirm")) (\(deviceLabel($0)))" } ?? "")
        }
    }

    private func loadSessions() {
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let list = try await sessionService.listSessions()
                await MainActor.run {
                    sessions = list
                    isLoading = false
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    isLoading = false
                }
            }
        }
    }

    private func kickSession(_ session: DeviceSession) {
        Task {
            do {
                try await sessionService.deleteSession(session.sessionID)
                await MainActor.run {
                    sessions.removeAll { $0.sessionID == session.sessionID }
                    kickConfirmSession = nil
                }
            } catch {
                await MainActor.run {
                    kickConfirmSession = nil
                }
            }
        }
    }

    private func deviceIcon(_ device: Int) -> String {
        switch device {
        case 0: return "iphone"
        case 1: return "macbook"
        case 3: return "ipad"
        default: return "questionmark"
        }
    }

    private func deviceLabel(_ session: DeviceSession) -> String {
        var name = session.deviceDisplayName
        if !session.deviceName.isEmpty {
            name += " (\(session.deviceName))"
        }
        return name
    }

    private func formatDate(_ timestamp: Int64) -> String {
        let date = Date(timeIntervalSince1970: Double(timestamp) / 1000)
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd HH:mm"
        return formatter.string(from: date)
    }
}
