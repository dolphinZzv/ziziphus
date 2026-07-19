import React from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'

export default function PrivacyPage() {
  const { t } = useTranslation()

  return (
    <div className="min-h-screen bg-[var(--color-canvas)]">
      <div className="max-w-2xl mx-auto px-4 py-8">
        <nav className="mb-6">
          <a href="javascript:history.back()"
            className="inline-flex items-center gap-1 text-sm text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors">
            <ArrowLeft size={16} /> {t('common.back', 'Back')}
          </a>
        </nav>
        <article className="prose prose-sm dark:prose-invert max-w-none">
          <h1>Privacy Policy</h1>
          <p className="text-[var(--color-muted)]">Last updated: 2026-07-19</p>

          <h2>1. Information We Collect</h2>
          <p>We collect information you provide when creating an account (name, email, avatar) and messages, files, and other content you send through the service. We also collect basic connection metadata (IP address, device type, session timestamps) for operational purposes.</p>

          <h2>2. How We Use Your Data</h2>
          <p>Your data is used solely to provide the messaging service: delivering messages, storing conversation history, managing contacts, and maintaining your account. We do not sell, rent, or share your personal data with third parties for their own purposes.</p>

          <h2>3. Data Storage and Security</h2>
          <p>All data is encrypted in transit (TLS) and stored with industry-standard security measures. Passwords are hashed using bcrypt. You can enable two-factor authentication (TOTP or email OTP) for additional account security.</p>

          <h2>4. Your Rights</h2>
          <p>You have the right to:</p>
          <ul>
            <li><strong>Access</strong> your data — view your profile, messages, and contacts at any time</li>
            <li><strong>Export</strong> your data — download all your information in a portable format</li>
            <li><strong>Delete</strong> your account — permanently remove your data from the service</li>
            <li><strong>Correct</strong> your information — update your profile details</li>
          </ul>

          <h2>5. Data Retention</h2>
          <p>We retain your data for as long as your account is active. When you delete your account, all associated data is permanently removed within a reasonable period.</p>

          <h2>6. Third-Party Services</h2>
          <p>We use OAuth providers (GitHub, Google) solely for authentication. No message content or personal data is shared with these providers. You can disable or unlink these connections at any time.</p>

          <h2>7. Contact</h2>
          <p>For privacy-related inquiries, please contact the server administrator.</p>
        </article>
      </div>
    </div>
  )
}
