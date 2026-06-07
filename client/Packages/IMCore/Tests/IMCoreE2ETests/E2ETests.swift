import XCTest
import IMCore

// MARK: - E2E Configuration

@MainActor
final class IMCoreE2ETests: XCTestCase {

    /// Unique suffix per test method so each test creates fresh accounts.
    private var ts: String { "\(Int64(Date().timeIntervalSince1970 * 1000))" }

    private var userAID = ""
    private var userBID = ""

    nonisolated override func setUp() {
        super.setUp()
    }

    nonisolated override func tearDown() {
        super.tearDown()
    }

    /// Ensure clean auth state before each test.
    private func cleanupAuth() {
        WebSocketClient.shared.disconnect()
        AuthManager.shared.logout()
    }

    /// Register and immediately login, then return the User.
    @discardableResult
    private func registerAndLogin(name: String, account: String, password: String) async throws -> User {
        let user = try await AuthService.shared.register(account: account, name: name, password: password)
        try await AuthService.shared.login(account: account, password: password)
        return user
    }

    /// Create two unique user tuples for the calling test.
    private func makeUsers() -> (a: (name: String, account: String, password: String),
                                 b: (name: String, account: String, password: String)) {
        let s = ts
        return (
            (name: "E2E_A_\(s)", account: "acc_a_\(s)", password: "test123"),
            (name: "E2E_B_\(s)", account: "acc_b_\(s)", password: "test123")
        )
    }

    // MARK: - Registration & Login

    func test_register_and_login() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        // Register (no auto-login)
        let regUser = try await AuthService.shared.register(account: userA.account, name: userA.name, password: userA.password)
        XCTAssertEqual(regUser.name, userA.name)
        XCTAssertEqual(regUser.account, userA.account, "Register response should include account")
        XCTAssertFalse(AuthManager.shared.isLoggedIn, "Register should NOT auto-login")

        // Login explicitly
        try await AuthService.shared.login(account: userA.account, password: userA.password)
        XCTAssertTrue(AuthManager.shared.isLoggedIn)
        let uid = AuthManager.shared.currentUser!.userID
        XCTAssertEqual(uid, regUser.userID)

        // Logout
        AuthManager.shared.logout()
        XCTAssertFalse(AuthManager.shared.isLoggedIn)
        XCTAssertNil(AuthManager.shared.currentUser)

        // Login again
        try await AuthService.shared.login(account: userA.account, password: userA.password)
        XCTAssertTrue(AuthManager.shared.isLoggedIn)
        XCTAssertEqual(AuthManager.shared.currentUser?.userID, uid)
    }

    func test_register_returns_account() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        let regUser = try await AuthService.shared.register(account: userA.account, name: userA.name, password: userA.password)
        XCTAssertEqual(regUser.account, userA.account, "Register response should include account")
        XCTAssertEqual(regUser.name, userA.name)
        XCTAssertFalse(regUser.userID.isEmpty)

        try await AuthService.shared.login(account: userA.account, password: userA.password)
        XCTAssertEqual(AuthManager.shared.currentUser?.account, userA.account, "Login should set account on currentUser")
    }

    func test_register_with_custom_account() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        let user = try await AuthService.shared.register(account: userA.account, name: userA.name, password: userA.password)
        XCTAssertEqual(user.account, userA.account)
        XCTAssertFalse(user.userID.isEmpty)
        XCTAssertNotEqual(user.userID, userA.account, "userID should be auto-generated, not equal to account")
    }

    func test_register_duplicate_account_fails() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        // Register first time
        try await AuthService.shared.register(account: userA.account, name: userA.name, password: userA.password)

        // Register again with same account
        do {
            try await AuthService.shared.register(account: userA.account, name: "AnotherName", password: userA.password)
            XCTFail("Register with duplicate account should fail")
        } catch {
            // Expected — server should reject duplicate account
        }
    }

    func test_duplicate_register_fails() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        let dupAccount = "dup_\(ts)"

        // Register first time with a specific account
        _ = try await AuthService.shared.register(account: dupAccount, name: userA.name, password: userA.password)

        // Register again with same account — server should reject it
        do {
            _ = try await AuthService.shared.register(account: dupAccount, name: userA.name, password: userA.password)
            XCTFail("Register with duplicate account should fail")
        } catch {
            // Expected
        }
    }

    // MARK: - Identity

    func test_get_me_returns_account() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)

        let me = try await AuthService.shared.getMe()
        XCTAssertEqual(me.account, userA.account, "getMe should return account field")
        XCTAssertEqual(me.userID, AuthManager.shared.currentUser?.userID)
    }

    func test_identity_consistency() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        let regUser = try await AuthService.shared.register(account: userA.account, name: userA.name, password: userA.password)
        try await AuthService.shared.login(account: userA.account, password: userA.password)

        let me = try await AuthService.shared.getMe()

        XCTAssertEqual(me.userID, regUser.userID, "userID must be consistent across register and getMe")
        XCTAssertEqual(me.account, regUser.account, "account must be consistent across register and getMe")
        XCTAssertEqual(me.name, regUser.name, "name must be consistent across register and getMe")
    }

    // MARK: - Login failure

    func test_login_wrong_account_fails() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        try await AuthService.shared.register(account: userA.account, name: userA.name, password: userA.password)

        do {
            try await AuthService.shared.login(account: "nonexistent_\(ts)", password: userA.password)
            XCTFail("Login with non-existent account should fail")
        } catch {
            // Expected
        }
    }

    func test_login_wrong_password_fails() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        try await AuthService.shared.register(account: userA.account, name: userA.name, password: userA.password)

        do {
            try await AuthService.shared.login(account: userA.account, password: "wrong_password")
            XCTFail("Login with wrong password should fail")
        } catch {
            // Expected
        }
    }

    // MARK: - WebSocket

    func test_websocket_connect_disconnect() async throws {
        let (userA, _) = makeUsers()
        cleanupAuth()

        try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)

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
        let (userA, userB) = makeUsers()

        // Create userB
        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        userBID = userBResult.userID
        AuthManager.shared.logout()

        // Login as A
        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        userAID = userAResult.userID

        // Search for userB
        let results = try await ContactService.shared.searchUsers(query: userB.name)
        XCTAssertTrue(results.contains(where: { $0.userID == userBID }),
                      "UserB should appear in search results")
    }

    func test_search_returns_account() async throws {
        let (userA, userB) = makeUsers()

        // Register two users
        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        userBID = userBResult.userID
        AuthManager.shared.logout()

        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        userAID = userAResult.userID

        // Search for userB by name
        let results = try await ContactService.shared.searchUsers(query: userB.name)

        let found = results.first(where: { $0.userID == userBID })
        XCTAssertNotNil(found, "UserB should appear in search results")
        XCTAssertEqual(found?.account, userB.account, "Search results should include account field")
    }

    func test_add_list_remove_contact() async throws {
        let (userA, userB) = makeUsers()

        // Create userB
        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        userBID = userBResult.userID
        AuthManager.shared.logout()

        // Login as A
        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        userAID = userAResult.userID

        // Add B as A's contact
        try await ContactService.shared.addContact(userID: userBID)

        // List contacts of A — should include B
        var contactsA = try await ContactService.shared.listContacts()
        XCTAssertTrue(contactsA.contains(where: { $0.userID == userBID }),
                      "B should be in A's contacts")

        // Remove contact
        try await ContactService.shared.removeContact(userID: userBID)
        contactsA = try await ContactService.shared.listContacts()
        XCTAssertFalse(contactsA.contains(where: { $0.userID == userBID }),
                       "B should be removed from A's contacts")
    }

    // MARK: - Group

    func test_create_group_and_query_detail() async throws {
        let (userA, userB) = makeUsers()

        // Create userB
        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        userBID = userBResult.userID
        AuthManager.shared.logout()

        // Login as A
        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        userAID = userAResult.userID

        // Create group
        let group = try await ConversationService.shared.createGroup(
            name: "E2E Group \(ts)",
            memberIDs: [userAID, userBID]
        )
        XCTAssertEqual(group.type, .group)
        XCTAssertFalse(group.convID.isEmpty)

        // Get detail
        let detail = try await ConversationService.shared.getConversationDetail(convID: group.convID)
        XCTAssertEqual(detail.members.count, 2)
        XCTAssertTrue(detail.members.contains(where: { $0.userID == userAID }))
        XCTAssertTrue(detail.members.contains(where: { $0.userID == userBID }))

        // List conversations
        let convs = try await ConversationService.shared.listConversations()
        XCTAssertTrue(convs.contains(where: { $0.convID == group.convID }))
    }

    func test_group_add_remove_member() async throws {
        let (userA, userB) = makeUsers()

        // Create userB
        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        let uidB = userBResult.userID
        AuthManager.shared.logout()

        // Login as A
        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        let uidA = userAResult.userID

        // Create group with just A
        let group = try await ConversationService.shared.createGroup(
            name: "MemberTest \(ts)",
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

    func test_group_leave() async throws {
        let (userA, userB) = makeUsers()

        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        let uidB = userBResult.userID
        AuthManager.shared.logout()

        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        let uidA = userAResult.userID

        // Create group with both users
        let group = try await ConversationService.shared.createGroup(
            name: "LeaveTest \(ts)",
            memberIDs: [uidA, uidB]
        )
        XCTAssertEqual(group.type, .group)

        // B is a member
        var detail = try await ConversationService.shared.getConversationDetail(convID: group.convID)
        XCTAssertTrue(detail.members.contains(where: { $0.userID == uidB }))

        // Login as B and leave
        AuthManager.shared.logout()
        try await AuthService.shared.login(account: userB.account, password: userB.password)

        try await ConversationService.shared.leaveGroup(convID: group.convID)

        // B should no longer be a member
        AuthManager.shared.logout()
        try await AuthService.shared.login(account: userA.account, password: userA.password)

        detail = try await ConversationService.shared.getConversationDetail(convID: group.convID)
        XCTAssertFalse(detail.members.contains(where: { $0.userID == uidB }),
                       "B should have left the group")
    }

    // MARK: - Message

    func test_send_message_and_get_history() async throws {
        let (userA, userB) = makeUsers()

        // Setup two users
        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        let uidB = userBResult.userID
        AuthManager.shared.logout()

        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        let uidA = userAResult.userID

        // Create group
        let group = try await ConversationService.shared.createGroup(
            name: "MsgGroup \(ts)",
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
        let msgBody = "Hello E2E \(ts)"
        let ack = try await MessageService.shared.sendMessage(
            convID: group.convID,
            body: msgBody,
            clientSeq: clientSeq
        )
        XCTAssertGreaterThan(ack.msgID, 0, "Should get valid msgID")
        XCTAssertEqual(ack.clientSeq, clientSeq)

        // Verify via getHistory (retry loop for eventual consistency)
        var history: [Message] = []
        for _ in 0..<8 {
            try await Task.sleep(nanoseconds: 500_000_000)
            history = try await MessageService.shared.getHistory(convID: group.convID, limit: 10)
            if history.contains(where: { $0.body == msgBody }) { break }
        }
        XCTAssertTrue(history.contains(where: { $0.body == msgBody }),
                      "Message '\(msgBody)' not found in history (\(history.count) messages)")
    }

    // MARK: - P2P Chat

    func test_create_p2p_chat() async throws {
        let (userA, userB) = makeUsers()

        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        userBID = userBResult.userID
        AuthManager.shared.logout()

        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        userAID = userAResult.userID

        let (convID, _) = try await ConversationService.shared.createP2P(userID: userBID)
        XCTAssertFalse(convID.isEmpty)
        XCTAssertTrue(convID.contains(userAID) && convID.contains(userBID),
                      "P2P convID should contain both user IDs")

        // Verify it appears in conversation list
        let convs = try await ConversationService.shared.listConversations()
        XCTAssertTrue(convs.contains(where: { $0.convID == convID }),
                      "P2P conversation should appear in list")
    }

    // MARK: - Batch Get Users

    func test_batch_get_users() async throws {
        let (userA, userB) = makeUsers()

        let userBResult = try await registerAndLogin(name: userB.name, account: userB.account, password: userB.password)
        let uidB = userBResult.userID
        AuthManager.shared.logout()

        let userAResult = try await registerAndLogin(name: userA.name, account: userA.account, password: userA.password)
        let uidA = userAResult.userID

        let users = try await ContactService.shared.batchGetUsers(userIDs: [uidA, uidB])
        XCTAssertEqual(users.count, 2)
        XCTAssertNotNil(users[uidA])
        XCTAssertNotNil(users[uidB])
        XCTAssertEqual(users[uidA]?.name, userA.name)
        XCTAssertEqual(users[uidA]?.account, userA.account, "BatchGet should return account field")
        XCTAssertEqual(users[uidB]?.account, userB.account, "BatchGet should return account field")
    }
}
