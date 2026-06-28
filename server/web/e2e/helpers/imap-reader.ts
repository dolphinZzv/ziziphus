export async function readVerificationCode(): Promise<string | null> {
  return null // UI tests handle verification via API interceptor, not IMAP
}
