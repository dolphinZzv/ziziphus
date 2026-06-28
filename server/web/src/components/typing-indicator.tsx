import { useTranslation } from 'react-i18next'

interface Props { names: string[] }

export default function TypingIndicator({ names }: Props) {
  const { t } = useTranslation()
  if (names.length === 0) return null

  const text = names.length === 1
    ? t('chat.typingUser', { name: names[0] })
    : t('chat.typingMulti', { count: names.length })

  return (
    <div className="flex items-center gap-2 px-4 py-1 text-xs text-[var(--color-muted)]">
      <div className="flex gap-0.5">
        <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-muted)] animate-bounce [animation-delay:0ms]" />
        <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-muted)] animate-bounce [animation-delay:150ms]" />
        <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-muted)] animate-bounce [animation-delay:300ms]" />
      </div>
      <span>{text}</span>
    </div>
  )
}
