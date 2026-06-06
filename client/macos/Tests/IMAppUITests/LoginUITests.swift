import XCTest

final class LoginUITests: XCTestCase {
    private let app = XCUIApplication()
    private let runID = "\(Int64(Date().timeIntervalSince1970 * 1000))"

    override func setUp() {
        continueAfterFailure = false
        app.launchArguments = ["-AppleLanguages", "(en)"]
        app.launch()
    }

    override func tearDown() {
        app.terminate()
    }

    func test_user_registration_and_login() {
        let userName = "UITest_\(runID)"
        let password = "test123"

        // Should be on Login screen initially
        let loginButton = app.buttons["登录"]
        XCTAssertTrue(loginButton.exists, "Login button should exist on launch")

        // Switch to Register mode
        let registerLink = app.buttons["没有账号？去注册"]
        if registerLink.exists {
            registerLink.click()
        }

        // Fill in name
        let nameField = app.textFields["名称"]
        XCTAssertTrue(nameField.waitForExistence(timeout: 2))
        nameField.click()
        nameField.typeText(userName)

        // Fill in password
        let passwordField = app.secureTextFields["密码"]
        XCTAssertTrue(passwordField.waitForExistence(timeout: 2))
        passwordField.click()
        passwordField.typeText(password)

        // Click register
        let registerButton = app.buttons["注册"]
        XCTAssertTrue(registerButton.exists)
        registerButton.click()

        // Wait for login success — conversation list should appear
        let conversationList = app.collectionViews.firstMatch
        XCTAssertTrue(conversationList.waitForExistence(timeout: 5),
                      "Conversation list should appear after successful registration")

        // Verify connection status is not "disconnected"
        let disconnectedLabel = app.staticTexts["连接已断开"]
        let isDisconnected = disconnectedLabel.waitForExistence(timeout: 3)
        XCTAssertFalse(isDisconnected, "WebSocket should be connected after login")
    }

    func test_login_with_existing_account() {
        // First register a user
        let userName = "UITestLogin_\(runID)"
        registerUser(name: userName, password: "test123")

        // Logout via the app menu or by restarting
        app.terminate()
        app.launch()

        // Switch to login mode if on register
        let registerLink = app.buttons["没有账号？去注册"]
        if registerLink.exists {
            registerLink.click()
        }
        let loginLink = app.buttons["已有账号？去登录"]
        if loginLink.exists {
            loginLink.click()
        }

        // Fill in userID
        let userIDField = app.textFields["用户ID"]
        XCTAssertTrue(userIDField.waitForExistence(timeout: 2))
        userIDField.click()
        userIDField.typeText(userName)

        // Fill in password
        let passwordField = app.secureTextFields["密码"]
        XCTAssertTrue(passwordField.waitForExistence(timeout: 2))
        passwordField.click()
        passwordField.typeText("test123")

        // Click login
        let loginButton = app.buttons["登录"]
        XCTAssertTrue(loginButton.exists)
        loginButton.click()

        // Wait for conversation list
        let conversationList = app.collectionViews.firstMatch
        XCTAssertTrue(conversationList.waitForExistence(timeout: 5),
                      "Conversation list should appear after successful login")
    }

    // MARK: - Helpers

    private func registerUser(name: String, password: String) {
        let loginButton = app.buttons["登录"]
        if loginButton.exists {
            let registerLink = app.buttons["没有账号？去注册"]
            if registerLink.exists {
                registerLink.click()
            }
        }

        let nameField = app.textFields["名称"]
        if nameField.waitForExistence(timeout: 2) {
            nameField.click()
            nameField.typeText(name)
        }

        let passwordField = app.secureTextFields["密码"]
        if passwordField.waitForExistence(timeout: 2) {
            passwordField.click()
            passwordField.typeText(password)
        }

        app.buttons["注册"].click()

        let conversationList = app.collectionViews.firstMatch
        XCTAssertTrue(conversationList.waitForExistence(timeout: 5),
                      "Should reach conversation list after register")
    }
}
