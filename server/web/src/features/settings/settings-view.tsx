import { useSyncExternalStore } from 'react'
import { useNavigate } from 'react-router-dom'
import { uiStore } from '@/stores/ui-store'
import { authStore } from '@/stores/auth-store'
import { api } from '@/services/api-client'
import { X, Monitor, Sun, Moon, Trash2 } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useTranslation } from 'react-i18next'

interface Props { onClose: () => void }
const BUBBLE_COLORS = ['#0F172A', '#059669', '#0EA5E9', '#22C55E', '#EAB308', '#EF4444', '#8B5CF6', '#64748B']

export default function SettingsView({ onClose }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const theme = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.theme)
  const language = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.language)
  const serverUrl = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.serverUrl)
  const bubbleColor = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.bubbleColor)
  const deviceId = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.deviceId)

  const handleDeleteAccount = async () => {
    if (!confirm(t('settings.deleteAccountConfirm'))) return
    if (!confirm(t('settings.deleteAccountConfirm2'))) return
    try {
      await api.request('/api/v1/users/me', { method: 'DELETE' })
      authStore.logout()
      onClose()
      navigate('/login')
    } catch { /* error handled by api-client */ }
  }

  const inputClass = 'w-full h-[42px] px-3.5 rounded-lg bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'
  const segBtn = (active: boolean) => cn('flex-1 h-[34px] rounded text-xs font-medium transition-colors flex items-center justify-center gap-1.5',
    active ? 'bg-[var(--color-primary)] text-white' : 'bg-[var(--color-surface-soft)] text-[var(--color-body)] hover:bg-[var(--color-hairline)]')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[380px] max-h-[580px] overflow-y-auto bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg p-6"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-headline text-xl font-semibold text-[var(--color-ink)]">{t('settings.title')}</h3>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)]"><X size={16} /></button>
        </div>

        <div className="space-y-6">
          {/* Theme */}
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">{t('settings.theme')}</label>
            <div className="flex gap-2">
              {[{ v: 'auto' as const, i: <Monitor size={14} />, l: t('settings.themeAuto') }, { v: 'light' as const, i: <Sun size={14} />, l: t('settings.themeLight') }, { v: 'dark' as const, i: <Moon size={14} />, l: t('settings.themeDark') }].map(item => (
                <button key={item.v} onClick={() => uiStore.setTheme(item.v)} className={segBtn(theme === item.v)}>{item.i} {item.l}</button>
              ))}
            </div>
          </div>

          {/* Language */}
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">{t('settings.language')}</label>
            <div className="flex gap-2">
              {[{ v: 'auto' as const, l: t('settings.languageAuto') }, { v: 'zh' as const, l: t('settings.languageZH') }, { v: 'en' as const, l: t('settings.languageEN') }].map(item => (
                <button key={item.v} onClick={() => uiStore.setLanguage(item.v)} className={segBtn(language === item.v)}>{item.l}</button>
              ))}
            </div>
          </div>

          {/* Bubble color */}
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">{t('settings.bubbleColor')}</label>
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
            <div>{t('settings.deviceID')}: <span className="font-mono select-all text-[var(--color-muted-soft)]">{deviceId}</span></div>
          </div>

          {/* Danger zone */}
          <div className="pt-4 border-t border-[var(--destructive)]/20">
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">{t('settings.dangerZone')}</label>
            <button onClick={handleDeleteAccount}
              className="w-full flex items-center justify-center gap-2 h-[42px] rounded-lg border border-red-500/20 bg-red-500/5 hover:bg-red-500/10 text-sm text-red-500 hover:text-red-600 font-medium transition-colors">
              <Trash2 size={16} /> {t('settings.deleteAccount')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
