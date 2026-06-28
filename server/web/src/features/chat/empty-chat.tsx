import { useTranslation } from 'react-i18next'
export default function EmptyChat() {
  const { t } = useTranslation()
  return (
    <div className="flex-1 flex items-center justify-center">
      <div className="text-center space-y-2">
        <div className="text-5xl mb-2 opacity-50">💬</div>
        <p className="text-sm text-[var(--color-muted)]">{t('conversation.selectChat')}</p>
      </div>
    </div>
  )
}
