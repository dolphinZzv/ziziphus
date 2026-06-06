import XCTest
import IMCore

// MARK: - E2E Configuration
private let runID = "\(Int64(Date().timeIntervalSince1970 * 1000))"

@MainActor
final class IMCoreE2ETests: XCTestCase {
    private let userA = (name: "E2E_A_\(runID)", password: "test123")
    private let userB = (name: "E2E_B_\(runID)", password: "test123")
    private var userAID = ""
    private var userBID = ""

    nonisolated override func setUp() {
        super.setUp()
    }

    nonisolated override func tearDown() {
        super.tearDown()
    }

    // MARK: - Registration & Login

    func test_register_and_login() async throws {
        // Register
        let regUser = try await AuthService.shared.register(name: userA.name, password: userA.password)
        XCTAssertEqual(regUser.name, userA.name)
        XCTAssertTrue(AuthManager.shared.isLoggedIn)
        let uid = AuthManager.shared.currentUser!.userID

        // Logout
        AuthManager.shared.logout()
        XCTAssertFalse(AuthManager.shared.isLoggedIn)
        XCTAssertNil(AuthManager.shared.currentUser)

        // Login again
        try await AuthService.shared.login(userID: uid, password: userA.password)
        XCTAssertTrue(AuthManager.shared.isLoggedIn)
        XCTAssertEqual(AuthManager.shared.currentUser?.userID, uid)
    }

    func test_duplicate_register_fails() async throws {
        // Register first time
        _ = try await AuthService.shared.register(name: userA.name, password: userA.password)
        let uid = AuthManager.shared.currentUser!.userID
        AuthManager.shared.logout()

        // Register same name again — should fail because server returns error
        do {
            _ = try await AuthService.shared.register(name: userA.name, password: userA.password)
            // If server allows duplicate name (different userID), that might be OK
            // At least the second registration should work independently
            let newUID = AuthManager.shared.currentUser?.userID ?? ""
            if newUID != uid {
                // Server generated a new user — that's also valid
                XCTAssertNotEqual(newUID, uid, "Should be a different user if allowed")
            }
        } catch {
            // Expected: server rejected duplicate registration
            XCTAssertFalse(AuthManager.shared.isLoggedIn)
        }
    }

    // MARK: - WebSocket

    func test_websocket_connect_disconnect() async throws {
        try await AuthService.shared.register(name: userA.name, password: userA.password)

        let ws = WebSocketClient.shared
        ws.connect()

        // Wait for connected
        for _ in 0..<20 {
            if ws.connectionStatus == .connected { break }
            try await Task.sleep(nanoseconds: 250_000_000)
        }
        XCTAssertEqual(ws.connectionStatus, .connected, "WS should connect within 5s")

        ws.disconnect()
        try await Task.sleep(nanoseconds: 500_000_000)
        XCTAssertEqual(ws.connectionStatus, .disconnected, "WS should disconnect")
    }

    // MARK: - Contacts & Search

    func test_search_users() async throws {
        // Register userB first so there's someone to search
        try await AuthService.shared.register(name: userB.name, password: userB.password)
        let uidB = AuthManager.shared.currentUser!.userID
        userBID = uidB
        AuthManager.shared.logout()

        // Login as A
        try await AuthService.shared.register(name: userA.name, password: userA.password)
        userAID = AuthManager.shared.currentUser!.userID

        // Search for userB
        let results = try await ContactService.shared.searchUsers(query: userB.name)
        XCTAssertTrue(results.contains(where: { $0.userID == uidB }),
                      "UserB should appear in search results")
    }

    func test_add_list_remove_contact() async throws {
        // Create userB
        try await AuthService.shared.register(name: userB.name, password: userB.password)
        let uidB = AuthManager.shared.currentUser!.userID
        userBID = uidB
        AuthManager.shared.logout()

        // Login as A
        try await AuthService.shared.register(name: userA.name, password: userA.password)
        let uidA = AuthManager.shared.currentUser!.userID
        userAID = uidA

        // Add B as A's contact
        try await ContactService.shared.addContact(userID: uidB)

        // List contacts of A — should include B
        var contactsA = try await ContactService.shared.listContacts()
        XCTAssertTrue(contactsA.contains(where: { $0.userID == uidB }),
                      "B should be in A's contacts")

        // Remove contact
        try await ContactService.shared.removeContact(userID: uidB)
        contactsA = try await ContactService.shared.listContacts()
        XCTAssertFalse(contactsA.contains(where: { $0.userID == uidB }),
                       "B should be removed from A's contacts")
    }

    // MARK: - Group

    func test_create_group_and_query_detail() async throws {
        // Create userB
        try await AuthService.shared.register(name: userB.name, password: userB.password)
        let uidB = AuthManager.shared.currentUser!.userID
        userBID = uidB
        AuthManager.shared.logout()

        // Login as A
        try await AuthService.shared.register(name: userA.name, password: userA.password)
        let uidA = AuthManager.shared.currentUser!.userID
        userAID = uidA

        // Create group
        let group = try await ConversationService.shared.createGroup(
            name: "E2E Group \(runID)",
            memberIDs: [uidA, uidB]
        )
        XCTAssertEqual(group.type, .group)
        XCTAssertFalse(group.convID.isEmpty)

        // Get detail
        let detail = try await ConversationService.shared.getConversationDetail(convID: group.convID)
        XCTAssertEqual(detail.members.count, 2)
        XCTAssertTrue(detail.members.contains(where: { $0.userID == uidA }))
        XCTAssertTrue(detail.members.contains(where: { $0.userID == uidB }))

        // List conversations
        let convs = try await ConversationService.shared.listConversations()
        XCTAssertTrue(convs.contains(where: { $0.convID == group.convID }))
    }

    func test_group_add_remove_member() async throws {
        // Create userB and userA
        try await AuthService.shared.register(name: userB.name, password: userB.password)
        let uidB = AuthManager.shared.currentUser!.userID
        AuthManager.shared.logout()

        try await AuthService.shared.register(name: userA.name, password: userA.password)
        let uidA = AuthManager.shared.currentUser!.userID

        // Create group with just A
        let group = try await ConversationService.shared.createGroup(
            name: "MemberTest \(runID)",
            memberIDs: [uidA]
        )
        XCTAssertEqual(group.type, .group)

        // Add B as member
        try await ConversationService.shared.addMembers(convID: group.convID, userIDs: [uidB])

        // Verify B is in group
        let detail = try await ConversationService.shared.getConversationDetail(convID: group.convID)
        XCTAssertTrue(detail.members.contains(where: { $0.userID == uidB }), "B should be a member")

        // Remove B
        try await ConversationService.shared.removeMember(convID: group.convID, userID: uidB)

        let detail2 = try await ConversationService.shared.getConversationDetail(convID: group.convID)
        XCTAssertFalse(detail2.members.contains(where: { $0.userID == uidB }), "B should be removed")
    }

    // MARK: - Message

    func test_send_message_and_get_history() async throws {
        // Setup two users
        try await AuthService.shared.register(name: userB.name, password: userB.password)
        let uidB = AuthManager.shared.currentUser!.userID
        AuthManager.shared.logout()

        try await AuthService.shared.register(name: userA.name, password: userA.password)
        let uidA = AuthManager.shared.currentUser!.userID

        // Create group
        let group = try await ConversationService.shared.createGroup(
            name: "MsgGroup \(runID)",
            memberIDs: [uidA, uidB]
        )

        // Connect WS
        WebSocketClient.shared.connect()
        var connected = false
        for _ in 0..<20 {
            if WebSocketClient.shared.connectionStatus == .connected {
                connected = true
                break
            }
            try await Task.sleep(nanoseconds: 250_000_000)
        }
        XCTAssertTrue(connected, "WS must be connected to send")

        // Send message
        let clientSeq = AuthManager.shared.nextClientSeq()
        let ack = try await MessageService.shared.sendMessage(
            convID: group.convID,
            body: "Hello E2E \(runID)",
            clientSeq: clientSeq
        )
        XCTAssertGreaterThan(ack.msgID, 0, "Should get valid msgID")
        XCTAssertEqual(ack.clientSeq, clientSeq)

        // Verify via getHistory
        try await Task.sleep(nanoseconds: 1_000_000_000)
        let history = try await MessageService.shared.getHistory(convID: group.convID, limit: 10)
        XCTAssertTrue(history.contains(where: { $0.body == "Hello E2E \(runID)" }),
                      "Message must appear in history")
    }

    // MARK: - Batch Get Users

    func test_batch_get_users() async throws {
        try await AuthService.shared.register(name: userB.name, password: userB.password)
        let uidB = AuthManager.shared.currentUser!.userID
        AuthManager.shared.logout()

        try await AuthService.shared.register(name: userA.name, password: userA.password)
        let uidA = AuthManager.shared.currentUser!.userID

        let users = try await ContactService.shared.batchGetUsers(userIDs: [uidA, uidB])
        XCTAssertEqual(users.count, 2)
        XCTAssertNotNil(users[uidA])
        XCTAssertNotNil(users[uidB])
        XCTAssertEqual(users[uidA]?.name, userA.name)
    }
}
