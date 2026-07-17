import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'

interface Props { onClose: () => void }

const SHORTCUTS = [
  { keys: ['⌘', 'N'], desc: 'conversation.newChat' },
  { keys: ['⌘', 'K'], desc: 'search.placeholder' },
  { keys: ['Esc'], desc: 'common.close' },
  { keys: ['↑', '↓'], desc: 'shortcuts.navigateConvs' },
  { keys: ['Enter'], desc: 'shortcuts.openConv' },
  { keys: ['Shift', 'Enter'], desc: 'shortcuts.newLine' },
  { keys: ['Tab'], desc: 'shortcuts.indent' },
  { keys: ['?'], desc: 'shortcuts.showHelp' },
]

export default function ShortcutsView({ onClose }: Props) {
  const { t } = useTranslation()
  const mod = navigator.platform.includes('Mac') ? '⌘' : 'Ctrl'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[380px] h-full sm:h-auto bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">{t('shortcuts.title')}</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>
        <div className="space-y-1">
          {SHORTCUTS.map(({ keys, desc }) => (
            <div key={desc} className="flex items-center justify-between py-2 px-2 rounded-xl hover:bg-[var(--color-surface-soft)]">
              <span className="text-sm text-[var(--color-body)]">{t(desc)}</span>
              <div className="flex items-center gap-1">
                {keys.map((k, i) => (
                  <span key={i}>
                    <kbd className="px-1.5 py-0.5 rounded bg-[var(--color-surface-soft)] border border-[var(--color-hairline)] text-[11px] font-mono text-[var(--color-muted)]">
                      {k === '⌘' ? mod : k}
                    </kbd>
                    {i < keys.length - 1 && <span className="text-[var(--color-muted-soft)] mx-0.5 text-[10px]">+</span>}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
        <div className="text-[11px] text-[var(--color-muted)] text-center mt-4 pt-3 border-t border-[var(--color-hairline)]">
          {t('shortcuts.tip')}
        </div>
      </div>
    </div>
  )
}
