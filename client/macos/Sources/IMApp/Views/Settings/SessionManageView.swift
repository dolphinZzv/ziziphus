import SwiftUI
import IMCore

struct SessionManageView: View {
    @State private var sessions: [DeviceSession] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var kickConfirmSession: DeviceSession?
    @Environment(\.dismiss) private var dismiss

    private let sessionService = SessionService.shared

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(loc("settings.device_management"))
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

            if isLoading {
                Spacer()
                ProgressView(loc("common.loading"))
                Spacer()
            } else if let error = errorMessage {
                Spacer()
                VStack(spacing: 12) {
                    Text(error)
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                    Button(loc("common.confirm")) { loadSessions() }
                }
                Spacer()
            } else {
                ScrollView {
                    VStack(spacing: 0) {
                        ForEach(sessions) { session in
                            SessionRowView(
                                session: session,
                                isCurrent: session.sessionID == AuthManager.shared.sessionID,
                                onKick: {
                                    kickConfirmSession = session
                                }
                            )
                            if session != sessions.last {
                                Divider()
                                    .foregroundColor(AppleDesign.Colors.hairline)
                            }
                        }
                    }
                    .padding(.horizontal)
                    .padding(.top, AppleDesign.Spacing.sm)
                }
            }
        }
        .frame(width: 380, height: 360)
        .background(Color(nsColor: .windowBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .onAppear { loadSessions() }
        .alert(loc("settings.kick_device"), isPresented: .init(
            get: { kickConfirmSession != nil },
            set: { if !$0 { kickConfirmSession = nil } }
        )) {
            Button(loc("common.cancel"), role: .cancel) { kickConfirmSession = nil }
            Button(loc("settings.kick_device"), role: .destructive) {
                if let session = kickConfirmSession { kickSession(session) }
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
                await MainActor.run { kickConfirmSession = nil }
            }
        }
    }

    private func deviceLabel(_ session: DeviceSession) -> String {
        var name = session.deviceDisplayName
        if !session.deviceName.isEmpty {
            name += " (\(session.deviceName))"
        }
        return name
    }
}

private struct SessionRowView: View {
    let session: DeviceSession
    let isCurrent: Bool
    let onKick: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: deviceIcon)
                .font(.title2)
                .foregroundColor(session.isOnline ? .green : AppleDesign.Colors.inkMuted)
                .frame(width: 28)

            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    Text(deviceLabel)
                        .font(.appleBody)
                        .foregroundColor(AppleDesign.Colors.ink)
                    if isCurrent {
                        Text(loc("settings.current_device"))
                            .font(.system(size: 10))
                            .foregroundColor(.blue)
                    }
                }
                Text(formatDate(session.loginAt))
                    .font(.system(size: 11))
                    .foregroundColor(AppleDesign.Colors.inkMuted)
                if let ip = session.clientIP, !ip.isEmpty {
                    Text(ip)
                        .font(.system(size: 10))
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                }
                if let devID = session.deviceID, !devID.isEmpty {
                    Text(devID)
                        .font(.system(size: 10))
                        .foregroundColor(AppleDesign.Colors.inkMuted)
                }
            }

            Spacer()

            if session.isOnline {
                Circle()
                    .fill(Color.green)
                    .frame(width: 8, height: 8)
            }

            if !isCurrent {
                Button(action: onKick) {
                    Image(systemName: "xmark.circle")
                        .foregroundColor(.red)
                }
                .buttonStyle(.plain)
                .help(loc("settings.kick_device"))
            }
        }
        .padding(.vertical, 8)
    }

    private var deviceIcon: String {
        switch session.device {
        case 0: return "iphone"
        case 1: return "macbook"
        case 3: return "ipad"
        default: return "questionmark"
        }
    }

    private var deviceLabel: String {
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
