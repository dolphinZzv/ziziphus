import XCTest

@MainActor
final class LoginUITests: XCTestCase {

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

    /// Test full register flow through the UI.
    /// This verifies that text fields, buttons, and the register API flow work correctly.
    func test_user_registration_via_ui() {
        let runID = "\(Int64(Date().timeIntervalSince1970 * 1000))"
        let userName = "UITest_\(runID)"
        let password = "test123"

        let app = XCUIApplication()
        app.launchArguments = ["-AppleLanguages", "(en)", "-IMClearAuth"]
        app.launch()

        // Should be on Login screen initially
        let loginButton = app.buttons["登录"]
        XCTAssertTrue(loginButton.waitForExistence(timeout: 5), "Login button should exist on launch")

        // Switch to Register
        app.buttons["没有账号？点击注册"].click()

        // Fill in account
        let accountField = app.textFields["账户"]
        XCTAssertTrue(accountField.waitForExistence(timeout: 2))
        accountField.click()
        accountField.typeText(userName)

        // Fill in nickname
        let nameField = app.textFields["昵称"]
        XCTAssertTrue(nameField.waitForExistence(timeout: 2))
        nameField.click()
        nameField.typeText(userName)

        // Fill in password
        let passwordField = app.secureTextFields["密码"]
        XCTAssertTrue(passwordField.waitForExistence(timeout: 2))
        passwordField.click()
        passwordField.typeText(password)

        // Click register
        app.buttons["注册"].click()

        // Wait for transition to main view
        let createGroupButton = app.buttons["创建群聊"]
        XCTAssertTrue(createGroupButton.waitForExistence(timeout: 10),
                      "Main view should appear after registration")

        app.terminate()
    }

    /// Test that an existing session with a valid token restores the logged-in state.
    /// Uses direct token injection to bypass the macOS SwiftUI TextField typing issues.
    func test_login_with_existing_account() async throws {
        let runID = "\(Int64(Date().timeIntervalSince1970 * 1000))"
        let userName = "LoginTest_\(runID)"
        let password = "test123"

        // Pre-register a user via API to get userID and token
        let (userID, token, name) = try await registerViaAPI(name: userName, password: password)
        print("Pre-registered user: \(userID) token: \(token.prefix(20))...")

        // Launch app with token injection (bypass UI login)
        let app = XCUIApplication()
        app.launchArguments = [
            "-AppleLanguages", "(en)",
            "-IMClearAuth",
            "-IMToken", token,
            "-IMUserID", userID,
            "-IMUserName", name
        ]
        app.launch()

        // App should detect the token and auto-authenticate via getMe()
        let createGroupButton = app.buttons["创建群聊"]
        XCTAssertTrue(createGroupButton.waitForExistence(timeout: 10),
                      "Main view should appear after token-based login")

        app.terminate()
    }
}
