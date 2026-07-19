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
          <h1>{t('privacyPage.title', 'Privacy Policy')}</h1>
          <p className="text-[var(--color-muted)]">{t('privacyPage.lastUpdated', 'Last updated: 2026-07-19')}</p>

          <h2>{t('privacyPage.section1Title', '1. Information We Collect')}</h2>
          <p>{t('privacyPage.section1Body', 'We collect information you provide when creating an account...')}</p>

          <h2>{t('privacyPage.section2Title', '2. How We Use Your Data')}</h2>
          <p>{t('privacyPage.section2Body', 'Your data is used solely to provide the messaging service...')}</p>

          <h2>{t('privacyPage.section3Title', '3. Data Storage and Security')}</h2>
          <p>{t('privacyPage.section3Body', 'All data is encrypted in transit (TLS)...')}</p>

          <h2>{t('privacyPage.section4Title', '4. Your Rights')}</h2>
          <p>{t('privacyPage.section4Intro', 'You have the right to:')}</p>
          <ul>
            <li>{t('privacyPage.section4Item1', 'Access your data')}</li>
            <li>{t('privacyPage.section4Item2', 'Export your data')}</li>
            <li>{t('privacyPage.section4Item3', 'Delete your account')}</li>
            <li>{t('privacyPage.section4Item4', 'Correct your information')}</li>
          </ul>

          <h2>{t('privacyPage.section5Title', '5. Data Retention')}</h2>
          <p>{t('privacyPage.section5Body', 'We retain your data for as long as your account is active...')}</p>

          <h2>{t('privacyPage.section6Title', '6. Third-Party Services')}</h2>
          <p>{t('privacyPage.section6Body', 'We use OAuth providers (GitHub, Google) solely for authentication...')}</p>

          <h2>{t('privacyPage.section7Title', '7. Contact')}</h2>
          <p>{t('privacyPage.section7Body', 'For privacy-related inquiries, please contact the server administrator.')}</p>
        </article>
      </div>
    </div>
  )
}
