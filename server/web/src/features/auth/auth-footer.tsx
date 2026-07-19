import { useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { uiStore } from '@/stores/ui-store'
import { Sun, Moon, Monitor, ChevronDown } from 'lucide-react'

type Theme = 'light' | 'dark' | 'auto'
type Language = 'zh' | 'en' | 'ja' | 'fr' | 'de' | 'es' | 'ko' | 'ru' | 'auto'

const themes: { key: Theme; icon: typeof Sun }[] = [
  { key: 'auto', icon: Monitor },
  { key: 'light', icon: Sun },
  { key: 'dark', icon: Moon },
]

const languages: { key: Language; label: string }[] = [
  { key: 'auto', label: 'Auto' },
  { key: 'zh', label: '中文' },
  { key: 'en', label: 'English' },
  { key: 'ja', label: '日本語' },
  { key: 'fr', label: 'Français' },
  { key: 'de', label: 'Deutsch' },
  { key: 'es', label: 'Español' },
  { key: 'ko', label: '한국어' },
  { key: 'ru', label: 'Русский' },
]

export default function AuthFooter() {
  const { t } = useTranslation()
  const language = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.language)
  const theme = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.theme)

  return (
    <div className="absolute bottom-8 left-0 right-0 flex flex-col items-center gap-4 text-xs text-[var(--color-muted)]">
      {/* Privacy / Terms */}
      <div className="flex items-center gap-3 text-[var(--color-muted-soft)]">
        <a href="/privacy" target="_blank" className="hover:text-[var(--color-primary)] transition-colors">{t('auth.privacy', 'Privacy')}</a>
        <span className="text-[var(--color-hairline)]">·</span>
        <a href="/terms" target="_blank" className="hover:text-[var(--color-primary)] transition-colors">{t('auth.terms', 'Terms')}</a>
      </div>
      {/* Theme + Language */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-0.5">
          {themes.map(({ key, icon: Icon }) => (
            <button
              key={key}
              onClick={() => uiStore.setTheme(key)}
              className={`p-1.5 rounded transition-colors ${theme === key ? 'text-[var(--color-primary)] bg-[var(--color-primary)]/10' : 'hover:text-[var(--color-ink)] hover:bg-[var(--color-hairline)]'}`}
            >
              <Icon size={14} />
            </button>
          ))}
        </div>
        <span className="opacity-30">|</span>
        <div className="relative">
          <select
            value={language}
            onChange={e => uiStore.setLanguage(e.target.value as Language)}
            className="appearance-none bg-transparent border border-[var(--color-hairline)] rounded pl-2 pr-6 py-1 text-xs text-[var(--color-muted)] hover:text-[var(--color-ink)] cursor-pointer outline-none focus:border-[var(--color-primary)]"
          >
            {languages.map(({ key, label }) => (
              <option key={key} value={key} className="bg-[var(--color-surface-card)] text-[var(--color-ink)]">
                {label}</option>
            ))}
          </select>
          <ChevronDown size={12} className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 text-[var(--color-muted-soft)]" />
        </div>
      </div>
    </div>
  )
}
