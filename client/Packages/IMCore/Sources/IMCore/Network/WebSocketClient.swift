import Foundation
import Combine
#if os(iOS)
import UIKit
#else
import AppKit
#endif

public enum ConnectionStatus: String, Sendable {
    case disconnected
    case connecting
    case connected
    case recovering
}

@MainActor
public class WebSocketClient: ObservableObject {
    public static let shared = WebSocketClient()

    @Published public private(set) var connectionStatus: ConnectionStatus = .disconnected

    private var webSocketTask: URLSessionWebSocketTask?
    private var session: URLSession!
    private var pingTimer: Timer?
    private var reconnectWork: Task<Void, Never>?
    private var reconnectDelay: TimeInterval = 1
    private var isActive = false
    private var handlers: [Int: [(WSFrame) -> Void]] = [:]
    private var ackContinuations: [String: CheckedContinuation<WSFrame, Error>] = [:]
    private let ackLock = NSLock()
    private var hasConnectedOnce = false

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        session = URLSession(configuration: config)
        observeLifecycle()
    }

    private func observeLifecycle() {
        #if os(iOS)
        NotificationCenter.default.addObserver(
            forName: UIApplication.willEnterForegroundNotification,
            object: nil, queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                self?.retryImmediately()
            }
        }
        #else
        NotificationCenter.default.addObserver(
            forName: NSApplication.willBecomeActiveNotification,
            object: nil, queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                self?.retryImmediately()
            }
        }
        #endif
    }

    private func retryImmediately() {
        guard connectionStatus != .connected, isActive else { return }
        reconnectDelay = 1
        reconnectWork?.cancel()
        connectionStatus = .connecting
        connect()
    }

    // MARK: - Connection
    public func connect() {
        if connectionStatus == .connected { return }

        guard let token = AuthManager.shared.readToken() else {
            connectionStatus = .disconnected
            return
        }

        isActive = true
        connectionStatus = .connecting

        guard let encodedToken = token.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) else {
            connectionStatus = .disconnected
            return
        }
        let wsBase = AppSettings.shared.serverURL
            .replacingOccurrences(of: "http://", with: "ws://")
            .replacingOccurrences(of: "https://", with: "wss://")
        #if os(iOS)
        let platform: String = {
            if UIDevice.current.userInterfaceIdiom == .pad {
                return "ipados"
            }
            return "ios"
        }()
        #else
        let platform = "macos"
        #endif
        let deviceID = DeviceManager.shared.deviceID
        guard let url = URL(string: "\(wsBase)/ws?token=\(encodedToken)&platform=\(platform)&device_id=\(deviceID)") else {
            connectionStatus = .disconnected
            return
        }

        webSocketTask?.cancel()
        let task = session.webSocketTask(with: url)
        webSocketTask = task
        task.resume()
        hasConnectedOnce = false
        // Don't set .connected here — wait for the first incoming message
        // in startReadLoop to confirm the handshake actually succeeded.
        startReadLoop()

        if let sessionID = AuthManager.shared.sessionID {
            let payload = SessionRecoverPayload(sessionID: sessionID)
            if let data = try? JSONEncoder().encode(payload) {
                send(frame: WSFrame(type: .sessionRecover, id: UUID().uuidString, payload: data))
            }
        }
    }

    public func disconnect() {
        isActive = false
        reconnectWork?.cancel()
        reconnectWork = nil
        pingTimer?.invalidate()
        pingTimer = nil
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        connectionStatus = .disconnected
        sendQueue.removeAll()
        ackLock.lock()
        for (_, cont) in ackContinuations {
            cont.resume(throwing: APIError.disconnected)
        }
        ackContinuations.removeAll()
        ackLock.unlock()
    }

    private var sendQueue: [WSFrame] = []

    // MARK: - Send
    public func send(frame: WSFrame) {
        guard let task = webSocketTask else {
            sendQueue.append(frame)
            return
        }
        do {
            let data = try frame.toRawJSONData()
            task.send(.data(data)) { [weak self] error in
                if let error {
                    Task { @MainActor [weak self] in
                        self?.handleError(error)
                    }
                }
            }
        } catch {
            print("WebSocket send error: \(error)")
        }
    }

    public func sendWithAck(frame: WSFrame, timeout: TimeInterval = 5) async throws -> WSFrame {
        let ackID = frame.id.isEmpty ? UUID().uuidString : frame.id
        let sendFrame = WSFrame(type: frame.type, id: ackID, payload: frame.payload)

        return try await withCheckedThrowingContinuation { continuation in
            ackContinuations[ackID] = continuation
            send(frame: sendFrame)

            Task { [weak self] in
                try? await Task.sleep(nanoseconds: UInt64(timeout * 1_000_000_000))
                await MainActor.run { [weak self] in
                    if let cont = self?.ackContinuations.removeValue(forKey: ackID) {
                        cont.resume(throwing: APIError.timeout)
                    }
                    // Remove from sendQueue to prevent stale delivery on reconnect
                    self?.sendQueue.removeAll { $0.id == ackID }
                }
            }
        }
    }

    // MARK: - Handlers
    public func on(_ type: MessageType, handler: @escaping (WSFrame) -> Void) {
        handlers[type.rawValue, default: []].append(handler)
    }

    public func off(_ type: MessageType) {
        handlers.removeValue(forKey: type.rawValue)
    }

    // MARK: - Read Loop
    private func startReadLoop() {
        webSocketTask?.receive { [weak self] result in
            guard let self else { return }
            Task { @MainActor in
                switch result {
                case .success(let message):
                    if !self.hasConnectedOnce {
                        self.hasConnectedOnce = true
                        self.connectionStatus = .connected
                        self.startPing()
                        self.resetReconnectDelay()
                        self.flushSendQueue()
                    }
                    self.handleMessage(message)
                    if self.isActive {
                        self.startReadLoop()
                    }
                case .failure(let error):
                    self.handleError(error)
                }
            }
        }
    }

    private func handleMessage(_ message: URLSessionWebSocketTask.Message) {
        let data: Data
        switch message {
        case .data(let d): data = d
        case .string(let s):
            guard let d = s.data(using: .utf8) else { return }
            data = d
        @unknown default: return
        }

        guard let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let typeRaw = dict["type"] as? Int,
              let messageType = MessageType(rawValue: typeRaw) else { return }

        let frameID = dict["id"] as? String ?? ""
        var payloadData = Data()
        if let payloadString = dict["payload"] as? String {
            payloadData = Data(payloadString.utf8)
        } else if let payloadDict = dict["payload"] as? [String: Any],
                  let pd = try? JSONSerialization.data(withJSONObject: payloadDict) {
            payloadData = pd
        }

        let frame = WSFrame(type: messageType, id: frameID, payload: payloadData)

        if !frameID.isEmpty, let cont = ackContinuations.removeValue(forKey: frameID) {
            cont.resume(returning: frame)
            return
        }

        if messageType == .error,
           let errPayload = try? JSONDecoder().decode(WSErrorPayload.self, from: payloadData),
           errPayload.code == 4001 {
            // kicked by server — stop reconnecting and trigger logout
            isActive = false
            disconnect()
            AuthManager.shared.logout()
            NotificationCenter.default.post(name: .init("kicked"), object: nil)
            return
        }

        handlers[typeRaw]?.forEach { $0(frame) }

        if messageType == .sessionRecoverAck,
           let ack = try? JSONDecoder().decode(SessionRecoverAckPayload.self, from: payloadData) {
            AuthManager.shared.sessionID = ack.sessionID
        }
    }

    // MARK: - Heartbeat
    private func startPing() {
        pingTimer?.invalidate()
        pingTimer = Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.send(frame: WSFrame(type: .ping))
            }
        }
    }

    // MARK: - Reconnect
    private func handleError(_ error: Error) {
        guard isActive else { return }
        connectionStatus = .disconnected
        webSocketTask = nil
        pingTimer?.invalidate()

        reconnectWork?.cancel()
        reconnectWork = Task { [weak self] in
            guard let self else { return }
            try? await Task.sleep(nanoseconds: UInt64(reconnectDelay * 1_000_000_000))
            guard !Task.isCancelled else { return }
            connectionStatus = .connecting
            connect()
            reconnectDelay = min(reconnectDelay * 2, 30)
        }
    }

    private func flushSendQueue() {
        let frames = sendQueue
        sendQueue.removeAll()
        for frame in frames {
            send(frame: frame)
        }
    }

    private func resetReconnectDelay() {
        reconnectDelay = 1
    }
}
