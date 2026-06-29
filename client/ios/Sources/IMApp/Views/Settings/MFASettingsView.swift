import SwiftUI
import IMCore

struct MFASettingsView: View {
    @State private var mfaStatus: MFAStatus?
    @State private var setupResult: MFASetupResult?
    @State private var isLoading = false
    @State private var verifyCode = ""
    @State private var errorMsg: String?

    @State private var settingUpType: Int?

    var body: some View {
        Group {
            if isLoading && mfaStatus == nil {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
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
        .navigationTitle("双重验证")
        .task { await loadStatus() }
    }

    private func enabledView(_ s: MFAStatus) -> some View {
        List {
            Section {
                HStack {
                    Spacer()
                    VStack(spacing: 12) {
                        Image(systemName: "checkmark.shield.fill")
                            .font(.system(size: 48)).foregroundColor(.green)
                        Text("已开启").font(.title2).fontWeight(.semibold)
                        Text(s.mfaType == 1 ? "TOTP 验证器" : "邮件 OTP")
                            .foregroundColor(.secondary)
                    }
                    Spacer()
                }
                .padding(.vertical)
                .listRowBackground(Color.clear)
            }

            Section {
                Button(role: .destructive) {
                    Task { await doDisable() }
                } label: {
                    HStack { Spacer(); Text("关闭双重验证"); Spacer() }
                }
            }
        }
    }

    private var pickerStepView: some View {
        List {
            Section {
                HStack {
                    Spacer()
                    VStack(spacing: 12) {
                        Image(systemName: "lock.shield").font(.system(size: 48)).foregroundColor(.blue)
                        Text("选择验证方式").font(.title2).fontWeight(.semibold)
                    }
                    Spacer()
                }
                .padding(.vertical)
                .listRowBackground(Color.clear)
            }

            Section {
                Button { startSetup(1) } label: {
                    Label {
                        VStack(alignment: .leading) {
                            Text("TOTP 验证器").fontWeight(.medium)
                            Text("Google Authenticator 等").font(.caption).foregroundColor(.secondary)
                        }
                    } icon: {
                        Image(systemName: "key.fill").foregroundColor(.blue)
                    }
                }
                Button { startSetup(2) } label: {
                    Label {
                        VStack(alignment: .leading) {
                            Text("邮件验证码").fontWeight(.medium)
                            Text("登录时发送验证码").font(.caption).foregroundColor(.secondary)
                        }
                    } icon: {
                        Image(systemName: "envelope.fill").foregroundColor(.orange)
                    }
                }
            }
        }
    }

    private func setupStepView(_ type: Int) -> some View {
        List {
            Section {
                if let r = setupResult {
                    if type == 1 {
                        Text("扫描二维码或手动输入密钥")
                            .font(.headline)
                        Text(r.secret)
                            .font(.system(.caption, design: .monospaced))
                            .padding(8)
                            .background(Color(.systemGray6))
                            .cornerRadius(6)
                        if !r.qrCodeURI.isEmpty,
                           let enc = r.qrCodeURI.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed),
                           let u = URL(string: "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=\(enc)") {
                            AsyncImage(url: u) { p in
                                if let img = p.image { img.resizable().aspectRatio(contentMode: .fit).frame(maxWidth: 200) }
                                else { ProgressView() }
                            }
                            .frame(height: 200)
                            .frame(maxWidth: .infinity)
                        }
                    } else {
                        Text("验证码已发送至 \(r.maskedEmail)")
                    }
                }
            }

            Section {
                HStack {
                    TextField(type == 1 ? "6 位 TOTP 代码" : "6 位验证码", text: $verifyCode)
                        .font(.system(size: 20, design: .monospaced))
                        .multilineTextAlignment(.center)
                        .keyboardType(.numberPad)
                        .onChange(of: verifyCode) { _, v in
                            if v.count > 6 { verifyCode = String(v.prefix(6)) }
                        }
                }

                if let e = errorMsg {
                    Text(e).foregroundColor(.red).font(.callout)
                }
            }

            Section {
                Button { Task { await doVerify() } } label: {
                    HStack { Spacer(); Text("验证并启用"); Spacer() }
                }
                .disabled(verifyCode.count != 6)

                Button("取消") { settingUpType = nil; verifyCode = ""; errorMsg = nil }
                    .foregroundColor(.red)
            }
        }
    }

    private func loadStatus() async {
        isLoading = true
        do { mfaStatus = try await MFAService.shared.getStatus() } catch {}
        isLoading = false
    }
    private func startSetup(_ type: Int) {
        settingUpType = type; verifyCode = ""; errorMsg = nil
        Task {
            isLoading = true
            do { setupResult = try await MFAService.shared.setup(mfaType: type) } catch {}
            isLoading = false
        }
    }
    private func doVerify() async {
        isLoading = true; errorMsg = nil
        do { try await MFAService.shared.verify(code: verifyCode); mfaStatus = try await MFAService.shared.getStatus(); settingUpType = nil; setupResult = nil; verifyCode = "" } catch { errorMsg = "验证失败" }
        isLoading = false
    }
    private func doDisable() async {
        isLoading = true
        do { try await MFAService.shared.disable(); mfaStatus = try await MFAService.shared.getStatus() } catch {}
        isLoading = false
    }
}
