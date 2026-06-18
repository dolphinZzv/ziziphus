import SwiftUI
import IMCore

struct AccountSelectView: View {
    @EnvironmentObject private var loginVM: LoginViewModel
    @State private var navigateToLogin = false

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Spacer()
                    .frame(height: 40)

                Image(systemName: "message.fill")
                    .font(.system(size: 60))
                    .foregroundColor(.blue)

                Text("PandaAI")
                    .font(.largeTitle)
                    .fontWeight(.bold)

                Text(loc("login.select_account"))
                    .font(.subheadline)
                    .foregroundColor(.secondary)

                List {
                    ForEach(loginVM.rememberedAccounts, id: \.self) { acct in
                        Button {
                            loginVM.selectAccount(acct)
                            navigateToLogin = true
                        } label: {
                            HStack(spacing: 14) {
                                AvatarCircle(account: acct, size: 44)
                                Text(acct)
                                    .font(.body)
                                    .foregroundColor(.primary)
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                            .padding(.vertical, 6)
                        }
                    }
                    .onDelete { indexSet in
                        for idx in indexSet {
                            let acct = loginVM.rememberedAccounts[idx]
                            loginVM.removeRememberedAccount(acct)
                        }
                    }
                }
                .listStyle(.plain)
                .frame(maxHeight: CGFloat(loginVM.rememberedAccounts.count * 62 + 20))

                Button {
                    loginVM.account = ""
                    loginVM.password = ""
                    navigateToLogin = true
                } label: {
                    Text(loc("login.other_account"))
                        .font(.body)
                }
                .buttonStyle(.bordered)

                Spacer()
            }
            .padding()
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .navigationDestination(isPresented: $navigateToLogin) {
                LoginView()
            }
            .onChange(of: loginVM.rememberedAccounts.count) { _, newCount in
                if newCount == 0 {
                    loginVM.account = ""
                    loginVM.password = ""
                    navigateToLogin = true
                }
            }
        }
    }
}

// MARK: - Account avatar circle

private struct AvatarCircle: View {
    let account: String
    var size: CGFloat = 40

    private static let avatarColors: [Color] = [
        .red, .orange, .yellow, .green, .mint, .teal, .cyan,
        .blue, .indigo, .purple, .pink, .brown
    ]

    private var color: Color {
        let index = abs(account.hash) % Self.avatarColors.count
        return Self.avatarColors[index]
    }

    private var initial: String {
        String(account.prefix(1)).uppercased()
    }

    var body: some View {
        ZStack {
            Circle()
                .fill(color)
                .frame(width: size, height: size)
            Text(initial)
                .font(.system(size: size * 0.4, weight: .semibold))
                .foregroundColor(.white)
        }
    }
}
