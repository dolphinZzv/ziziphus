import XCTest
@testable import IMCore

final class AgentTimelineBodyTests: XCTestCase {

    // MARK: - EntryType

    func test_entryType_rawValues() {
        XCTAssertEqual(AgentTimelineBody.EntryType.thinking.rawValue, "thinking")
        XCTAssertEqual(AgentTimelineBody.EntryType.toolCall.rawValue, "toolCall")
        XCTAssertEqual(AgentTimelineBody.EntryType.toolResult.rawValue, "toolResult")
        XCTAssertEqual(AgentTimelineBody.EntryType.response.rawValue, "response")
    }

    func test_entryType_roundtrip() throws {
        for type: AgentTimelineBody.EntryType in [.thinking, .toolCall, .toolResult, .response] {
            let data = try JSONEncoder().encode(type)
            let decoded = try JSONDecoder().decode(AgentTimelineBody.EntryType.self, from: data)
            XCTAssertEqual(decoded, type)
        }
    }

    // MARK: - Entry

    func test_entry_encodeDecode() throws {
        let entry = AgentTimelineBody.Entry(
            id: "abc-123",
            type: .toolCall,
            content: "Calling search_web",
            toolName: "search_web",
            toolInput: #"{"query":"Swift 6"}"#,
            status: "success",
            timestamp: 1718123456000
        )
        let data = try JSONEncoder().encode(entry)
        let decoded = try JSONDecoder().decode(AgentTimelineBody.Entry.self, from: data)

        XCTAssertEqual(decoded.id, "abc-123")
        XCTAssertEqual(decoded.type, .toolCall)
        XCTAssertEqual(decoded.content, "Calling search_web")
        XCTAssertEqual(decoded.toolName, "search_web")
        XCTAssertEqual(decoded.toolInput, #"{"query":"Swift 6"}"#)
        XCTAssertEqual(decoded.status, "success")
        XCTAssertEqual(decoded.timestamp, 1718123456000)
    }

    func test_entry_defaultValues() throws {
        let entry = AgentTimelineBody.Entry(type: .thinking, content: "reasoning...")
        XCTAssertEqual(entry.type, .thinking)
        XCTAssertEqual(entry.content, "reasoning...")
        XCTAssertNil(entry.toolName)
        XCTAssertNil(entry.toolInput)
        XCTAssertEqual(entry.timestamp, 0)
        // id is auto-generated UUID, should be non-empty
        XCTAssertFalse(entry.id.isEmpty)
    }

    // MARK: - AgentTimelineBody

    func test_body_encodeDecode() throws {
        let entries = [
            AgentTimelineBody.Entry(type: .thinking, content: "Let me search..."),
            AgentTimelineBody.Entry(
                type: .toolCall, content: "search_web called",
                toolName: "search_web", toolInput: #"{"q":"test"}"#,
                status: "running", timestamp: 1
            ),
            AgentTimelineBody.Entry(type: .response, content: "Here are the results..."),
        ]
        let body = AgentTimelineBody(
            title: "Researching query",
            entries: entries,
            status: "completed"
        )

        let data = try JSONEncoder().encode(body)
        let decoded = try JSONDecoder().decode(AgentTimelineBody.self, from: data)

        XCTAssertEqual(decoded.title, "Researching query")
        XCTAssertEqual(decoded.status, "completed")
        XCTAssertEqual(decoded.parentMsgID, 0)
        XCTAssertEqual(decoded.entries.count, 3)
        XCTAssertEqual(decoded.entries[0].type, .thinking)
        XCTAssertEqual(decoded.entries[1].type, .toolCall)
        XCTAssertEqual(decoded.entries[2].type, .response)
    }

    func test_body_withParentMsgID() throws {
        let body = AgentTimelineBody(
            entries: [AgentTimelineBody.Entry(type: .toolCall, content: "search")],
            status: "running",
            parentMsgID: 100
        )
        let data = try JSONEncoder().encode(body)
        let decoded = try JSONDecoder().decode(AgentTimelineBody.self, from: data)

        XCTAssertEqual(decoded.parentMsgID, 100)
    }

    func test_body_parentMsgID_defaultsToZero() throws {
        let body = AgentTimelineBody(
            entries: [AgentTimelineBody.Entry(type: .thinking, content: "thinking")],
            status: "running"
        )
        let data = try JSONEncoder().encode(body)
        let decoded = try JSONDecoder().decode(AgentTimelineBody.self, from: data)

        XCTAssertEqual(decoded.parentMsgID, 0)
    }

    func test_body_jsonStructure() throws {
        let entry = AgentTimelineBody.Entry(
            id: "e1",
            type: .thinking,
            content: "reasoning",
            timestamp: 123
        )
        let body = AgentTimelineBody(
            title: "Task",
            entries: [entry],
            status: "running",
            parentMsgID: 42
        )
        let data = try JSONEncoder().encode(body)
        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any]

        XCTAssertEqual(json?["title"] as? String, "Task")
        XCTAssertEqual(json?["status"] as? String, "running")
        XCTAssertEqual(json?["parentMsgID"] as? Int, 42)
        let entriesJSON = json?["entries"] as? [[String: Any]]
        XCTAssertEqual(entriesJSON?.count, 1)
        XCTAssertEqual(entriesJSON?[0]["id"] as? String, "e1")
        XCTAssertEqual(entriesJSON?[0]["type"] as? String, "thinking")
    }

    // MARK: - Append (dedup) logic

    func test_appendEntries_mergesAndDedups() throws {
        // Simulate existing parent message body
        let existing = AgentTimelineBody(
            title: "Research",
            entries: [
                AgentTimelineBody.Entry(id: "1", type: .thinking, content: "thinking..."),
                AgentTimelineBody.Entry(id: "2", type: .toolCall, content: "search", toolName: "web"),
            ],
            status: "running",
            parentMsgID: 0
        )

        // Simulate an append delta
        let delta = AgentTimelineBody(
            entries: [
                AgentTimelineBody.Entry(id: "2", type: .toolCall, content: "search", toolName: "web"), // duplicate
                AgentTimelineBody.Entry(id: "3", type: .toolResult, content: "found 5 results", toolName: "web"),
                AgentTimelineBody.Entry(id: "4", type: .response, content: "Here's what I found..."),
            ],
            status: "completed",
            parentMsgID: 100
        )

        // Dedup logic (same as appendAgentTimelineEntries)
        let existingIDs = Set(existing.entries.map(\.id))
        let newEntries = delta.entries.filter { !existingIDs.contains($0.id) }
        var merged = existing
        merged.entries.append(contentsOf: newEntries)
        merged.status = delta.status

        XCTAssertEqual(merged.entries.count, 4)
        XCTAssertEqual(merged.entries[0].id, "1")
        XCTAssertEqual(merged.entries[1].id, "2")
        XCTAssertEqual(merged.entries[2].id, "3")
        XCTAssertEqual(merged.entries[2].type, .toolResult)
        XCTAssertEqual(merged.entries[3].id, "4")
        XCTAssertEqual(merged.entries[3].type, .response)
        XCTAssertEqual(merged.status, "completed")
    }

    func test_appendEntries_noDuplicates_straightAppend() throws {
        let existing = AgentTimelineBody(
            entries: [AgentTimelineBody.Entry(id: "a", type: .thinking, content: "hmm")],
            status: "running"
        )
        let delta = AgentTimelineBody(
            entries: [AgentTimelineBody.Entry(id: "b", type: .response, content: "ok")],
            status: "completed",
            parentMsgID: 1
        )

        let existingIDs = Set(existing.entries.map(\.id))
        let newEntries = delta.entries.filter { !existingIDs.contains($0.id) }
        var merged = existing
        merged.entries.append(contentsOf: newEntries)
        merged.status = delta.status

        XCTAssertEqual(merged.entries.count, 2)
        XCTAssertEqual(merged.status, "completed")
    }

    // MARK: - ContentType

    func test_contentType_agentTimeline_rawValue() {
        XCTAssertEqual(ContentType.agentTimeline.rawValue, 9)
    }

    func test_contentType_decode_agentTimeline() throws {
        // Verify the raw value 9 maps back to agentTimeline
        let raw = 9
        let decoded = ContentType(rawValue: raw)
        XCTAssertNotNil(decoded)
        XCTAssertEqual(decoded, .agentTimeline)
    }

    // MARK: - Message with agentTimeline

    func test_message_withAgentTimelineBody() throws {
        let body = AgentTimelineBody(
            title: "Test",
            entries: [AgentTimelineBody.Entry(type: .response, content: "done")],
            status: "completed"
        )
        let bodyData = try JSONEncoder().encode(body)
        let bodyStr = String(data: bodyData, encoding: .utf8)!

        let msg = Message(
            msgID: 1,
            convID: "c1",
            senderID: "u1",
            contentType: .agentTimeline,
            body: bodyStr
        )

        XCTAssertEqual(msg.contentType, .agentTimeline)
        XCTAssertEqual(msg.contentType.rawValue, 9)

        let decoded = try JSONDecoder().decode(AgentTimelineBody.self, from: msg.body.data(using: .utf8)!)
        XCTAssertEqual(decoded.title, "Test")
        XCTAssertEqual(decoded.status, "completed")
    }
}
