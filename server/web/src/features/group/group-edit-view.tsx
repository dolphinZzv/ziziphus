import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { conversationService } from '@/services/conversation-service'
import { X } from 'lucide-react'

interface Props { convId: string; name: string; headline: string; notice: string; onClose: () => void; onSaved?: (data: { name: string; headline: string; notice: string }) => void }

export default function GroupEditView({ convId, name, headline, notice, onClose, onSaved }: Props) {
  const { t } = useTranslation()
  const [editName, setEditName] = useState(name)
  const [editHeadline, setEditHeadline] = useState(headline)
  const [editNotice, setEditNotice] = useState(notice)
  const [saving, setSaving] = useState(false)

  const handleSave = async () => {
    setSaving(true)
    try {
      const data = { name: editName.trim(), headline: editHeadline.trim(), notice: editNotice.trim() }
      await conversationService.updateGroup(convId, data)
      onSaved?.(data)
      onClose()
    } catch {}
    setSaving(false)
  }

  const iCls = 'w-full rounded-xl bg-[var(--color-surface-soft)] text-sm text-[var(--color-ink)] outline-none border border-[var(--color-hairline)] focus:border-[var(--color-primary)] px-3.5'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[360px] bg-[var(--color-surface-card)] rounded-xl overflow-hidden"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--color-hairline)]">
          <h3 className="font-headline text-base font-semibold text-[var(--color-ink)]">{t('group.editTitle')}</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <div className="px-5 py-4 space-y-3">
          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-1 block">{t('group.name')}</label>
            <input type="text" value={editName} onChange={e => setEditName(e.target.value)} className={`${iCls} h-10`} />
          </div>
          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-1 block">{t('group.headline')}</label>
            <input type="text" value={editHeadline} onChange={e => setEditHeadline(e.target.value)} maxLength={120} className={`${iCls} h-10`} />
          </div>
          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-1 block">{t('group.notice')}</label>
            <textarea value={editNotice} onChange={e => setEditNotice(e.target.value)} rows={4} placeholder={t('group.noticePlaceholder')}
              className={`${iCls} py-2.5 resize-none`} />
          </div>
          <button onClick={handleSave} disabled={saving || !editName.trim()}
            className="w-full h-10 rounded-xl bg-[var(--color-primary)] text-white text-sm font-medium transition-colors disabled:opacity-40">
            {saving ? t('profile.saving') : t('profile.save')}
          </button>
        </div>
      </div>
    </div>
  )
}
