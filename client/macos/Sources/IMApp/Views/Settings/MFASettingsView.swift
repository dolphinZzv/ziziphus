import SwiftUI
import IMCore

struct MFASettingsView: View {
    @State private var mfaStatus: MFAStatus?
    @State private var setupResult: MFASetupResult?
    @State private var isLoading = false
    @State private var verifyCode = ""
    @State private var errorMsg: String?

    /// type picked before setup: 1=TOTP, 2=Email
    @State private var settingUpType: Int?
    @State private var emailInput = ""

    var body: some View {
        VStack(spacing: 0) {
            headerView

            if isLoading {
                ProgressView().padding(40)
            } else if let status = mfaStatus {
                if status.enabled {
                    enabledView(status)
                } else if let t = settingUpType {
                    setupStepView(t)
                } else {
                    pickerStepView
                }
            }
        }
        .frame(width: 360, height: 420)
        .task { await loadStatus() }
    }

    // MARK: - Header

    private var headerView: some View {
        Text("双重验证")
            .font(.headline)
            .padding()
    }

    // MARK: - Enabled state

    private func enabledView(_ s: MFAStatus) -> some View {
        VStack(spacing: 20) {
            Spacer().frame(height: 10)

            Image(systemName: "checkmark.shield.fill")
                .font(.system(size: 44))
                .foregroundColor(.green)

            Text("已开启")
                .font(.title3)
                .fontWeight(.semibold)

            Text(s.mfaType == 1
                 ? "通过 TOTP 验证器保护你的账户"
                 : "通过邮件 OTP 保护你的账户")
                .font(.body)
                .foregroundColor(.secondary)

            Button(role: .destructive, action: disableMFA) {
                Text("关闭双重验证")
            }
            .buttonStyle(.borderedProminent)
            .tint(.red)

            Spacer()
        }
    }

    // MARK: - Picker step

    private var pickerStepView: some View {
        VStack(spacing: 20) {
            Spacer().frame(height: 10)

            Image(systemName: "lock.shield")
                .font(.system(size: 44))
                .foregroundColor(.blue)

            Text("选择验证方式")
                .font(.title3)
                .fontWeight(.semibold)

            Button(action: { startSetup(1) }) {
                HStack {
                    Image(systemName: "key.fill")
                    VStack(alignment: .leading) {
                        Text("TOTP 验证器").fontWeight(.medium)
                        Text("使用 Google Authenticator 等").font(.caption).foregroundColor(.secondary)
                    }
                    Spacer()
                }
                .padding()
                .background(Color(nsColor: .controlBackgroundColor))
                .cornerRadius(8)
            }
            .buttonStyle(.plain)
            .frame(width: 280)

            Button(action: { startSetup(2) }) {
                HStack {
                    Image(systemName: "envelope.fill")
                    VStack(alignment: .leading) {
                        Text("邮件验证码").fontWeight(.medium)
                        Text("登录时发送验证码到邮箱").font(.caption).foregroundColor(.secondary)
                    }
                    Spacer()
                }
                .padding()
                .background(Color(nsColor: .controlBackgroundColor))
                .cornerRadius(8)
            }
            .buttonStyle(.plain)
            .frame(width: 280)

            Spacer()
        }
    }

    // MARK: - Setup step

    private func setupStepView(_ type: Int) -> some View {
        ScrollView(.vertical) {
            VStack(spacing: 16) {
                if let r = setupResult {
                    if type == 1 {
                        Text("扫描二维码或手动输入密钥").font(.headline)
                        Text(r.secret)
                            .font(.system(.caption, design: .monospaced))
                            .textSelection(.enabled)
                            .padding(8)
                            .background(Color(nsColor: .controlBackgroundColor))
                            .cornerRadius(6)
                        // QR code
                        if !r.qrCodeURI.isEmpty, let qrURL = URL(string: "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=\(r.qrCodeURI.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed)!)") {
                            AsyncImage(url: qrURL) { p in
                                if let img = p.image { img.resizable().aspectRatio(contentMode: .fit).frame(width: 180) }
                                else { ProgressView() }
                            }
                        }
                    } else {
                        Text("验证码已发送至 \(r.maskedEmail)").font(.headline)
                    }
                }

                TextField(type == 1 ? "6 位 TOTP 代码" : "6 位验证码", text: $verifyCode)
                    .font(.system(size: 20, design: .monospaced))
                    .multilineTextAlignment(.center)
                    .frame(width: 160)
                    .textFieldStyle(.roundedBorder)

                if let e = errorMsg { Text(e).foregroundColor(.red).font(.callout) }

                HStack(spacing: 16) {
                    Button("取消") { settingUpType = nil; verifyCode = ""; errorMsg = nil }
                    Button("验证") { Task { await doVerify() } }
                        .buttonStyle(.borderedProminent)
                        .disabled(verifyCode.count != 6)
                }
            }
            .padding()
        }
    }

    // MARK: - Actions

    private func loadStatus() async {
        isLoading = true
        do { mfaStatus = try await MFAService.shared.getStatus() } catch {}
        isLoading = false
    }

    private func startSetup(_ type: Int) {
        settingUpType = type
        verifyCode = ""
        errorMsg = nil
        Task {
            isLoading = true
            do { setupResult = try await MFAService.shared.setup(mfaType: type) } catch {}
            isLoading = false
        }
    }

    private func doVerify() async {
        isLoading = true
        errorMsg = nil
        do {
            try await MFAService.shared.verify(code: verifyCode)
            mfaStatus = try await MFAService.shared.getStatus()
            settingUpType = nil
            setupResult = nil
            verifyCode = ""
        } catch {
            errorMsg = "验证失败，请确认代码正确"
        }
        isLoading = false
    }

    private func disableMFA() {
        Task {
            isLoading = true
            do { try await MFAService.shared.disable(); mfaStatus = try await MFAService.shared.getStatus() } catch {}
            isLoading = false
        }
    }
}
