import { useSyncExternalStore } from 'react'
import { uiStore } from '@/stores/ui-store'
import { X, Monitor, Sun, Moon } from 'lucide-react'
import { cn } from '@/lib/cn'

interface Props { onClose: () => void }
const BUBBLE_COLORS = ['#0F172A', '#059669', '#0EA5E9', '#22C55E', '#EAB308', '#EF4444', '#8B5CF6', '#64748B']

export default function SettingsView({ onClose }: Props) {
  const theme = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.theme)
  const language = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.language)
  const serverUrl = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.serverUrl)
  const bubbleColor = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.bubbleColor)
  const deviceId = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.deviceId)

  const inputClass = 'w-full h-[42px] px-3.5 rounded-lg bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'
  const segBtn = (active: boolean) => cn('flex-1 h-[34px] rounded text-xs font-medium transition-colors flex items-center justify-center gap-1.5',
    active ? 'bg-[var(--color-primary)] text-white' : 'bg-[var(--color-surface-soft)] text-[var(--color-body)] hover:bg-[var(--color-hairline)]')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[380px] max-h-[580px] overflow-y-auto bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg p-6"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-headline text-xl font-semibold text-[var(--color-ink)]">设置</h3>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)]"><X size={16} /></button>
        </div>

        <div className="space-y-6">
          {/* Theme */}
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">主题</label>
            <div className="flex gap-2">
              {[{ v: 'light' as const, i: <Sun size={14} />, l: '浅色' }, { v: 'dark' as const, i: <Moon size={14} />, l: '深色' }, { v: 'system' as const, i: <Monitor size={14} />, l: '跟随系统' }].map(item => (
                <button key={item.v} onClick={() => uiStore.setTheme(item.v)} className={segBtn(theme === item.v)}>{item.i} {item.l}</button>
              ))}
            </div>
          </div>

          {/* Language */}
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">语言</label>
            <div className="flex gap-2">
              {[{ v: 'zh' as const, l: '中文' }, { v: 'en' as const, l: 'English' }].map(item => (
                <button key={item.v} onClick={() => uiStore.setLanguage(item.v)} className={segBtn(language === item.v)}>{item.l}</button>
              ))}
            </div>
          </div>

          {/* Bubble color */}
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">气泡颜色</label>
            <div className="flex gap-2 flex-wrap">
              {BUBBLE_COLORS.map(color => (
                <button key={color} onClick={() => uiStore.setBubbleColor(color)}
                  className={cn('w-8 h-8 rounded-lg transition-transform', bubbleColor === color && 'ring-2 ring-offset-1 ring-[var(--color-primary)] scale-110')}
                  style={{ background: color }} />
              ))}
            </div>
          </div>

          {/* Device info */}
          <div className="text-[11px] text-[var(--color-muted)] space-y-1 pt-2 border-t border-[var(--color-hairline)]">
            <div>设备 ID: <span className="font-mono select-all text-[var(--color-muted-soft)]">{deviceId}</span></div>
          </div>
        </div>
      </div>
    </div>
  )
}
