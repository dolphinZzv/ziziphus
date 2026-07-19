import React from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'

export default function TermsPage() {
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
          <h1>Terms of Service</h1>
          <p className="text-[var(--color-muted)]">Last updated: 2026-07-19</p>

          <h2>1. Acceptance of Terms</h2>
          <p>By accessing or using this service, you agree to be bound by these Terms of Service. If you do not agree, do not use the service.</p>

          <h2>2. Description of Service</h2>
          <p>This is a private messaging platform that allows users to communicate via text, files, and other media in one-on-one and group conversations.</p>

          <h2>3. User Responsibilities</h2>
          <ul>
            <li>You are responsible for maintaining the confidentiality of your account credentials.</li>
            <li>You may not use the service for any illegal or unauthorized purpose.</li>
            <li>You may not transmit malware, viruses, or any harmful code.</li>
            <li>You may not harass, abuse, or harm other users.</li>
          </ul>

          <h2>4. Account Termination</h2>
          <p>We reserve the right to suspend or terminate accounts for violations of these terms. You may delete your account at any time, which will remove your data from the service.</p>

          <h2>5. Disclaimer</h2>
          <p>This service is provided "as is" without warranty of any kind. We are not responsible for any damages arising from the use or inability to use the service.</p>

          <h2>6. Changes to Terms</h2>
          <p>We may update these terms at any time. Continued use of the service after changes constitutes acceptance of the new terms.</p>

          <h2>7. Governing Law</h2>
          <p>These terms shall be governed by applicable law. Any disputes shall be resolved in the competent courts.</p>
        </article>
      </div>
    </div>
  )
}
