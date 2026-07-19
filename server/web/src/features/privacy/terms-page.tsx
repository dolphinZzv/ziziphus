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
          <h1>{t('termsPage.title', 'Terms of Service')}</h1>
          <p className="text-[var(--color-muted)]">{t('termsPage.lastUpdated', 'Last updated: 2026-07-19')}</p>

          <h2>{t('termsPage.section1Title', '1. Acceptance of Terms')}</h2>
          <p>{t('termsPage.section1Body', 'By accessing or using this service...')}</p>

          <h2>{t('termsPage.section2Title', '2. Description of Service')}</h2>
          <p>{t('termsPage.section2Body', 'This is a private messaging platform...')}</p>

          <h2>{t('termsPage.section3Title', '3. User Responsibilities')}</h2>
          <ul>
            <li>{t('termsPage.section3Item1', 'Account confidentiality')}</li>
            <li>{t('termsPage.section3Item2', 'No illegal use')}</li>
            <li>{t('termsPage.section3Item3', 'No malware')}</li>
            <li>{t('termsPage.section3Item4', 'No harassment')}</li>
          </ul>

          <h2>{t('termsPage.section4Title', '4. Account Termination')}</h2>
          <p>{t('termsPage.section4Body', 'We reserve the right to suspend or terminate accounts...')}</p>

          <h2>{t('termsPage.section5Title', '5. Disclaimer')}</h2>
          <p>{t('termsPage.section5Body', 'This service is provided as is...')}</p>

          <h2>{t('termsPage.section6Title', '6. Changes to Terms')}</h2>
          <p>{t('termsPage.section6Body', 'We may update these terms at any time...')}</p>

          <h2>{t('termsPage.section7Title', '7. Governing Law')}</h2>
          <p>{t('termsPage.section7Body', 'These terms shall be governed by applicable law...')}</p>
        </article>
      </div>
    </div>
  )
}
