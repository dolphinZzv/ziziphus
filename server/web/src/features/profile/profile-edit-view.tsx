import { useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { authStore } from '@/stores/auth-store'
import { X } from 'lucide-react'

interface Props { onClose: () => void }

export default function ProfileEditView({ onClose }: Props) {
  const { t } = useTranslation()
  const user = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const [name, setName] = useState(user?.name || '')
  const [headline, setHeadline] = useState(user?.headline || '')
  const [primaryColor, setPrimaryColor] = useState(user?.primary_color || '#0F172A')
  const [secondaryColor, setSecondaryColor] = useState(user?.secondary_color || '#64748B')
  const [saving, setSaving] = useState(false)

  const handleSave = async () => {
    setSaving(true)
    try { await authStore.updateProfile({ name, headline, primary_color: primaryColor, secondary_color: secondaryColor }); onClose() } catch {}
    setSaving(false)
  }

  const inputCls = 'w-full h-10 px-3.5 rounded-xl bg-[var(--color-surface-soft)] text-sm text-[var(--color-ink)] outline-none border border-[var(--color-hairline)] focus:border-[var(--color-primary)]'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[340px] h-full sm:h-auto bg-[var(--color-surface-card)] rounded-none sm:rounded-xl overflow-hidden"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        <div className="flex items-center justify-between px-5 py-4">
          <h3 className="font-headline text-base font-semibold text-[var(--color-ink)]">{t('profile.editTitle')}</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <div className="px-5 py-4 space-y-3">
          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-1 block">{t('profile.nickname')}</label>
            <input type="text" value={name} onChange={e => setName(e.target.value)}
              placeholder={t('profile.nicknamePlaceholder')} className={inputCls} />
          </div>
          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-1 block">{t('profile.headlinePlaceholder')}</label>
            <input type="text" value={headline} onChange={e => setHeadline(e.target.value)}
              placeholder={t('profile.headlinePlaceholder')} maxLength={120} className={inputCls} />
          </div>
          <div className="flex items-center gap-6">
            <label className="flex items-center gap-2 text-xs text-[var(--color-muted)]">{t('profile.primaryColor')}
              <input type="color" value={primaryColor} onChange={e => setPrimaryColor(e.target.value)} className="w-7 h-7 rounded cursor-pointer" /></label>
            <label className="flex items-center gap-2 text-xs text-[var(--color-muted)]">{t('profile.secondaryColor')}
              <input type="color" value={secondaryColor} onChange={e => setSecondaryColor(e.target.value)} className="w-7 h-7 rounded cursor-pointer" /></label>
          </div>
          <button onClick={handleSave} disabled={saving}
            className="w-full h-10 rounded-xl bg-[var(--color-primary)] text-white text-sm font-medium transition-colors disabled:opacity-40">
            {saving ? t('profile.saving') : t('profile.save')}
          </button>
        </div>
      </div>
    </div>
  )
}
