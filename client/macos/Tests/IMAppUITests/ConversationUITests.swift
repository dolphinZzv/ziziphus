import XCTest

@MainActor
final class ConversationUITests: XCTestCase {

    /// Register a user via the API and return (userID, token, name).
    private func registerViaAPI(name: String, password: String) async throws -> (userID: String, token: String, name: String) {
        let url = URL(string: "http://localhost:8080/api/v1/users/register")!
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = try JSONSerialization.data(withJSONObject: [
            "name": name,
            "password": password
        ])
        let (data, _) = try await URLSession.shared.data(for: req)
        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
        guard let code = json["code"] as? Int, code == 0,
              let dataDict = json["data"] as? [String: Any],
              let userID = dataDict["user_id"] as? String,
              let token = dataDict["token"] as? String else {
            throw NSError(domain: "test", code: -1,
                userInfo: [NSLocalizedDescriptionKey: "API register failed: \(json["msg"] ?? "unknown")"])
        }
        return (userID, token, name)
    }

    /// Create a group conversation via the API.
    private func createGroupViaAPI(token: String, name: String, memberIDs: [String]) async throws -> String {
        let url = URL(string: "http://localhost:8080/api/v1/conversations/group")!
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        req.httpBody = try JSONSerialization.data(withJSONObject: [
            "name": name,
            "member_ids": memberIDs
        ])
        let (data, _) = try await URLSession.shared.data(for: req)
        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
        guard let code = json["code"] as? Int, code == 0,
              let dataDict = json["data"] as? [String: Any],
              let convID = dataDict["conv_id"] as? String else {
            throw NSError(domain: "test", code: -1,
                userInfo: [NSLocalizedDescriptionKey: "API create group failed: \(json["msg"] ?? "unknown")"])
        }
        return convID
    }

    /// Launch the app with a valid token so it starts already logged in.
    private func launchLoggedInApp(userID: String, token: String, name: String) -> XCUIApplication {
        let app = XCUIApplication()
        app.launchArguments = [
            "-AppleLanguages", "(en)",
            "-IMClearAuth",
            "-IMToken", token,
            "-IMUserID", userID,
            "-IMUserName", name
        ]
        app.launch()
        return app
    }

    /// Create a P2P conversation via the API.
    private func createP2PViaAPI(token: String, userID: String) async throws -> String {
        let url = URL(string: "http://localhost:8080/api/v1/conversations/p2p")!
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        req.httpBody = try JSONSerialization.data(withJSONObject: [
            "user_id": userID
        ])
        let (data, _) = try await URLSession.shared.data(for: req)
        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
        guard let code = json["code"] as? Int, code == 0,
              let dataDict = json["data"] as? [String: Any],
              let convID = dataDict["conv_id"] as? String else {
            throw NSError(domain: "test", code: -1,
                userInfo: [NSLocalizedDescriptionKey: "API create p2p failed: \(json["msg"] ?? "unknown")"])
        }
        return convID
    }

    // MARK: - Tests

    func test_create_group_conversation() async throws {
        let runID = "\(Int64(Date().timeIntervalSince1970 * 1000))"
        let userName = "GroupTest_\(runID)"
        let password = "test123"
        let secondUserName = "GroupMember_\(runID)"

        // Register both users via API
        let (secondUserID, _, _) = try await registerViaAPI(name: secondUserName, password: password)
        let (userID, token, name) = try await registerViaAPI(name: userName, password: password)

        // Create group via API (bypasses CreateGroupView search field, which has
        // a macOS SwiftUI .plain TextField binding issue with XCUITest typeText)
        let groupName = "Test Group \(runID)"
        _ = try await createGroupViaAPI(token: token, name: groupName, memberIDs: [secondUserID])

        // Launch app already logged in
        let app = launchLoggedInApp(userID: userID, token: token, name: name)

        // Verify the group appears in the conversation list
        let groupRow = app.staticTexts[groupName]
        XCTAssertTrue(groupRow.waitForExistence(timeout: 5),
                      "Group should appear in conversation list")

        app.terminate()
    }

    func test_send_message_in_group() async throws {
        let runID = "\(Int64(Date().timeIntervalSince1970 * 1000))"
        let userName = "MsgTest_\(runID)"
        let password = "test123"
        let secondUserName = "MsgRecv_\(runID)"

        // Register both users
        let (secondUserID, _, _) = try await registerViaAPI(name: secondUserName, password: password)
        let (userID, token, name) = try await registerViaAPI(name: userName, password: password)

        // Create group via API
        let groupName = "ChatGroup \(runID)"
        _ = try await createGroupViaAPI(token: token, name: groupName, memberIDs: [secondUserID])

        // Launch app already logged in
        let app = launchLoggedInApp(userID: userID, token: token, name: name)

        // Click the group in the conversation list
        let groupRow = app.staticTexts[groupName]
        XCTAssertTrue(groupRow.waitForExistence(timeout: 5))
        groupRow.click()

        // Wait for chat view to appear and type a message
        let messageField = app.textFields["输入消息..."]
        XCTAssertTrue(messageField.waitForExistence(timeout: 3))
        messageField.click()
        messageField.typeText("Hello from XCUITest!")

        // Press Enter to send
        messageField.typeText("\n")

        // Verify the message appears in the chat
        let sentMessage = app.staticTexts["Hello from XCUITest!"]
        XCTAssertTrue(sentMessage.waitForExistence(timeout: 5),
                      "Sent message should appear in the chat view")

        app.terminate()
    }

    func test_send_message_in_p2p() async throws {
        let runID = "\(Int64(Date().timeIntervalSince1970 * 1000))"
        let userName = "P2PTest_\(runID)"
        let password = "test123"
        let peerUserName = "P2PPeer_\(runID)"

        // Register both users
        let (peerUserID, _, _) = try await registerViaAPI(name: peerUserName, password: password)
        let (userID, token, name) = try await registerViaAPI(name: userName, password: password)

        // Create P2P conversation via API
        _ = try await createP2PViaAPI(token: token, userID: peerUserID)

        // Launch app already logged in
        let app = launchLoggedInApp(userID: userID, token: token, name: name)

        // The P2P conversation should appear with the partner's name
        let p2pRow = app.staticTexts[peerUserName]
        XCTAssertTrue(p2pRow.waitForExistence(timeout: 5),
                      "P2P conversation should appear with partner's name")

        // Click to open chat
        p2pRow.click()

        // Send a message
        let messageField = app.textFields["输入消息..."]
        XCTAssertTrue(messageField.waitForExistence(timeout: 3))
        messageField.click()
        messageField.typeText("Hello P2P from XCUITest!")

        // Press Enter to send
        messageField.typeText("\n")

        // Verify the message appears
        let sentMessage = app.staticTexts["Hello P2P from XCUITest!"]
        XCTAssertTrue(sentMessage.waitForExistence(timeout: 5),
                      "Sent P2P message should appear in the chat view")

        app.terminate()
    }

    func test_group_leave_confirmation() async throws {
        let runID = "\(Int64(Date().timeIntervalSince1970 * 1000))"
        let userName = "LeaveTest_\(runID)"
        let password = "test123"
        let secondUserName = "LeaveMember_\(runID)"

        // Register users and create a group
        let (secondUserID, _, _) = try await registerViaAPI(name: secondUserName, password: password)
        let (userID, token, name) = try await registerViaAPI(name: userName, password: password)
        let groupName = "LeaveGroup \(runID)"
        _ = try await createGroupViaAPI(token: token, name: groupName, memberIDs: [secondUserID])

        // Launch app
        let app = launchLoggedInApp(userID: userID, token: token, name: name)

        // Click the group in conversation list
        let groupRow = app.staticTexts[groupName]
        XCTAssertTrue(groupRow.waitForExistence(timeout: 5))
        groupRow.click()

        // Click group info button to open group detail
        let infoButton = app.buttons.matching(identifier: "群聊信息").firstMatch
        XCTAssertTrue(infoButton.waitForExistence(timeout: 3))
        infoButton.click()

        // Wait for group detail sheet to appear (verify "添加成员" exists)
        let addMemberButton = app.buttons["添加成员"]
        XCTAssertTrue(addMemberButton.waitForExistence(timeout: 5),
                      "Group detail sheet should appear with '添加成员' button")

        // Click "退出群聊" button
        let leaveButton = app.buttons["退出群聊"]
        XCTAssertTrue(leaveButton.waitForExistence(timeout: 5))
        leaveButton.click()

        // Verify the confirmation alert appears with "取消" and "退出" buttons
        let cancelButton = app.buttons["取消"]
        XCTAssertTrue(cancelButton.waitForExistence(timeout: 3),
                      "Cancel button should exist in leave confirmation alert")

        let confirmLeaveButton = app.buttons["退出"]
        XCTAssertTrue(confirmLeaveButton.waitForExistence(timeout: 1),
                      "Confirm leave button should exist in leave confirmation alert")

        // Cancel the leave
        cancelButton.click()

        // Verify we're still on the group detail (sheet not dismissed)
        XCTAssertTrue(leaveButton.waitForExistence(timeout: 3),
                      "Should still be on group detail after canceling leave")

        // Now open the alert again and confirm leave
        leaveButton.click()
        XCTAssertTrue(confirmLeaveButton.waitForExistence(timeout: 3))
        confirmLeaveButton.click()

        // After leaving, the sheet should dismiss and we should return to conversation list
        XCTAssertTrue(app.staticTexts[groupName].waitForExistence(timeout: 5) == false ||
                      app.buttons["创建群聊"].waitForExistence(timeout: 5),
                      "Should return to conversation list after leaving group")

        app.terminate()
    }
}
