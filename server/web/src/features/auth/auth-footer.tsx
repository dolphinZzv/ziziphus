import { useSyncExternalStore } from 'react'
import { uiStore } from '@/stores/ui-store'
import { Sun, Moon, Monitor } from 'lucide-react'

type Theme = 'light' | 'dark' | 'auto'
type Language = 'zh' | 'en' | 'ja' | 'fr' | 'de' | 'es' | 'ko' | 'auto'

const themes: { key: Theme; icon: typeof Sun }[] = [
  { key: 'auto', icon: Monitor },
  { key: 'light', icon: Sun },
  { key: 'dark', icon: Moon },
]

const languages: { key: Language; label: string }[] = [
  { key: 'auto', label: 'Auto' },
  { key: 'zh', label: '中文' },
  { key: 'en', label: 'EN' },
  { key: 'ja', label: '日本語' },
  { key: 'fr', label: 'Français' },
  { key: 'de', label: 'Deutsch' },
  { key: 'es', label: 'Español' },
  { key: 'ko', label: '한국어' },
]

export default function AuthFooter() {
  const language = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.language)
  const theme = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.theme)

  return (
    <div className="absolute bottom-6 left-0 right-0 flex items-center justify-center gap-3 text-xs text-[var(--color-muted)]">
      {/* Theme */}
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
      {/* Language */}
      <div className="flex items-center gap-0.5">
        {languages.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => uiStore.setLanguage(key)}
            className={`px-1.5 py-1 rounded transition-colors ${language === key ? 'text-[var(--color-primary)] font-medium' : 'hover:text-[var(--color-ink)]'}`}
          >
            {label}
          </button>
        ))}
      </div>
    </div>
  )
}
