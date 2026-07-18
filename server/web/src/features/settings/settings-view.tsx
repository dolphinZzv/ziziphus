import { useSyncExternalStore } from 'react'
import { useNavigate } from 'react-router-dom'
import { uiStore } from '@/stores/ui-store'
import { authStore } from '@/stores/auth-store'
import { api } from '@/services/api-client'
import { X, Monitor, Sun, Moon, Trash2, ArrowLeft, Bell, ChevronDown } from 'lucide-react'
import { requestNotificationPermission, isNotificationGranted } from '@/services/notifications'
import { cn } from '@/lib/cn'
import { useTranslation } from 'react-i18next'

interface Props { onClose: () => void; inline?: boolean }
const BUBBLE_COLORS = ['#0F172A', '#059669', '#0EA5E9', '#22C55E', '#EAB308', '#EF4444', '#8B5CF6', '#64748B']

export default function SettingsView({ onClose, inline }: Props) {
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

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'
  const segBtn = (active: boolean) => cn('flex-1 h-[34px] rounded text-xs font-medium transition-colors flex items-center justify-center gap-1.5',
    active ? 'bg-[var(--color-primary)] text-white' : 'bg-[var(--color-surface-soft)] text-[var(--color-body)] hover:bg-[var(--color-hairline)]')

  const inner = (
    <div className={`${inline ? 'h-full' : 'w-full sm:w-[380px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)]'} overflow-y-auto bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6`}
      style={inline ? {} : { boxShadow: 'var(--shadow-lg)' }} onClick={inline ? undefined : e => e.stopPropagation()}>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          {inline && (
            <button onClick={onClose} className="p-1 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">
              <ArrowLeft size={18} />
            </button>
          )}
          <h3 className="font-headline text-xl font-semibold text-[var(--color-ink)]">{t('settings.title')}</h3>
        </div>
        {!inline && <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)]"><X size={16} /></button>}
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
          <div className="relative">
            <select
              value={language}
              onChange={e => uiStore.setLanguage(e.target.value)}
              className="appearance-none w-full h-10 pl-3 pr-10 rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-body)] bg-[var(--color-surface-card)] outline-none focus:border-[var(--color-primary)] cursor-pointer"
            >
              {[{ v: 'auto', l: t('settings.languageAuto') }, { v: 'zh', l: t('settings.languageZH') }, { v: 'en', l: t('settings.languageEN') }, { v: 'ja', l: t('settings.languageJA') }, { v: 'fr', l: t('settings.languageFR') }, { v: 'de', l: t('settings.languageDE') }, { v: 'es', l: t('settings.languageES') }, { v: 'ko', l: t('settings.languageKO') }, { v: 'ru', l: t('settings.languageRU') }].map(item => (
                <option key={item.v} value={item.v} className="bg-[var(--color-surface-card)] text-[var(--color-ink)]">{item.l}</option>
              ))}
            </select>
            <ChevronDown size={16} className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-muted-soft)]" />
          </div>
        </div>

        {/* Notifications */}
        <div>
          <label className="block text-xs font-medium text-[var(--color-body)] mb-2">{t('settings.notifications')}</label>
          <button onClick={() => requestNotificationPermission().then(g => g && alert(t('settings.notifyGranted') || 'Notifications enabled'))}
            className="w-full h-10 rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-body)] hover:bg-[var(--color-surface-soft)] transition-colors flex items-center justify-center gap-2">
            {isNotificationGranted() ? t('settings.notifyEnabled') : t('settings.notifyEnable')}
          </button>
        </div>

        {/* Bubble color */}
        <div>
          <label className="block text-xs font-medium text-[var(--color-body)] mb-2">{t('settings.bubbleColor')}</label>
          <div className="flex gap-2 flex-wrap">
            {BUBBLE_COLORS.map(color => (
              <button key={color} onClick={() => uiStore.setBubbleColor(color)}
                className={cn('w-8 h-8 rounded-xl transition-transform', bubbleColor === color && 'ring-2 ring-offset-1 ring-[var(--color-primary)] scale-110')}
                style={{ background: color }} />
            ))}
          </div>
        </div>

        {/* Device info */}
        <div className="text-[11px] text-[var(--color-muted)] space-y-1 pt-2">
          <div>{t('settings.deviceID')}: <span className="font-mono select-all text-[var(--color-muted-soft)]">{deviceId}</span></div>
        </div>

        {/* Danger zone */}
        <div className="pt-4 border-t border-[var(--destructive)]/20">
          <label className="block text-xs font-medium text-[var(--color-body)] mb-2">{t('settings.dangerZone')}</label>
          <button onClick={handleDeleteAccount}
            className="w-full flex items-center justify-center gap-2 h-[42px] rounded-xl border border-red-500/20 bg-red-500/5 hover:bg-red-500/10 text-sm text-red-500 hover:text-red-600 font-medium transition-colors">
            <Trash2 size={16} /> {t('settings.deleteAccount')}
          </button>
        </div>
      </div>
    </div>
  )

  if (inline) return inner
  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      {inner}
    </div>
  )
}
