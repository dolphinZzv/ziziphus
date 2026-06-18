import Foundation

@MainActor
public class MessageService {
    public static let shared = MessageService()
    private let api = APIClient.shared
    private let ws = WebSocketClient.shared

    private init() {}

    // MARK: - Get History (HTTP)
    public func getHistory(convID: String, beforeMsgID: Int64? = nil, limit: Int = 50) async throws -> [Message] {
        var query: [String: String] = ["limit": "\(limit)"]
        if let before = beforeMsgID {
            query["before_msg_id"] = "\(before)"
        }
        let messages: [Message] = try await api.request(
            "/api/v1/conversations/\(convID)/messages",
            query: query
        )
        return messages
    }

    // MARK: - Send Message (WS)
    public func sendMessage(convID: String, body: String, clientSeq: Int64, contentType: Int = 0, replyTo: Int64 = 0, mention: [String] = []) async throws -> MsgSendAckPayload {
        let payload = MsgSendPayload(convID: convID, contentType: contentType, body: body, replyTo: replyTo, mention: mention, clientSeq: clientSeq)
        let data = try JSONEncoder().encode(payload)
        let frame = WSFrame(type: .msgSend, id: UUID().uuidString, payload: data)
        let response = try await ws.sendWithAck(frame: frame, timeout: 5)
        let ack = try JSONDecoder().decode(MsgSendAckPayload.self, from: response.payload)
        return ack
    }

    // MARK: - Handle Push
    public func handlePush(frame: WSFrame) -> Message? {
        guard let payload = try? JSONDecoder().decode(MsgPushPayload.self, from: frame.payload) else {
            return nil
        }
        return Message(
            msgID: payload.msgID,
            convID: payload.convID,
            senderID: payload.senderID,
            contentType: ContentType(rawValue: payload.contentType) ?? .text,
            body: payload.body,
            replyTo: payload.replyTo ?? 0,
            mention: payload.mention ?? [],
            timestamp: payload.timestamp,
            convSeq: payload.convSeq,
            status: .delivered
        )
    }

    // MARK: - Sync (WS)
    public func syncMessages(convID: String, lastConvSeq: Int64, limit: Int = 50) async throws -> SyncResPayload {
        let payload = SyncReqPayload(convID: convID, lastConvSeq: lastConvSeq, limit: limit)
        let data = try JSONEncoder().encode(payload)
        let frame = WSFrame(type: .syncReq, id: UUID().uuidString, payload: data)
        let response = try await ws.sendWithAck(frame: frame, timeout: 10)
        let res = try JSONDecoder().decode(SyncResPayload.self, from: response.payload)
        return res
    }
}
