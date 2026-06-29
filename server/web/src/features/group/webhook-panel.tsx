import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { webhookService } from '@/services/webhook-service'
import type { ConvWebhook } from '@/types/webhook'
import { Globe, Plus, Trash2, Key, Shield, Activity, Clipboard } from 'lucide-react'

interface Props { convId: string }

export default function WebhookPanel({ convId }: Props) {
  const { t } = useTranslation()
  const [webhooks, setWebhooks] = useState<ConvWebhook[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<ConvWebhook | null>(null)
  const [form, setForm] = useState({ name: '', callback_url: '', cidr_whitelist: '', require_audit: false })
  const [created, setCreated] = useState<{ token: string; api_key: string } | null>(null)
  const [pendingMsgs, setPendingMsgs] = useState<any[]>([])
  const [showPending, setShowPending] = useState(false)

  useEffect(() => {
    webhookService.list(convId).then(setWebhooks).catch(() => {})
    webhookService.pendingMessages(convId).then(setPendingMsgs).catch(() => {})
  }, [convId])

  const resetForm = () => setForm({ name: '', callback_url: '', cidr_whitelist: '', require_audit: false })

  const openCreate = () => { setEditing(null); resetForm(); setShowForm(true) }
  const openEdit = (wh: ConvWebhook) => {
    setEditing(wh)
    setForm({ name: wh.name, callback_url: wh.callback_url || '', cidr_whitelist: (wh.cidr_whitelist || []).join(', '), require_audit: wh.require_audit })
    setShowForm(true)
  }

  const handleSave = async () => {
    if (!form.name.trim()) return
    try {
      const data = { name: form.name, callback_url: form.callback_url, cidr_whitelist: form.cidr_whitelist ? form.cidr_whitelist.split(',').map(s => s.trim()).filter(Boolean) : [], require_audit: form.require_audit }
      if (editing) {
        await webhookService.update(convId, editing.id, data)
        setWebhooks(prev => prev.map(w => w.id === editing.id ? { ...w, ...data } : w))
      } else {
        const result = await webhookService.create(convId, data)
        setWebhooks(prev => [...prev, { ...result, require_audit: result.require_audit }])
        setCreated({ token: result.token, api_key: result.api_key })
      }
      setShowForm(false)
    } catch {} // api error
  }

  const handleDelete = async (id: number) => {
    if (!confirm(t('conversation.webhookDeleteConfirm'))) return
    try { await webhookService.delete(convId, id); setWebhooks(prev => prev.filter(w => w.id !== id)) } catch {}
  }

  const handleApprove = async (msgId: number) => {
    try { await webhookService.approveMessage(msgId); setPendingMsgs(prev => prev.filter(m => m.msg_id !== msgId)) } catch {}
  }
  const handleReject = async (msgId: number) => {
    try { await webhookService.rejectMessage(msgId); setPendingMsgs(prev => prev.filter(m => m.msg_id !== msgId)) } catch {}
  }

  const iCls = 'w-full h-8 px-2 rounded-lg text-xs bg-[var(--color-surface-soft)] border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)]'

  return (
    <div className="border-t border-[var(--color-hairline)] pt-3 mt-3">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2"><Globe size={14} className="text-[var(--color-muted)]" /><span className="text-xs font-medium text-[var(--color-muted)]">{t('conversation.webhookTitle')}</span></div>
        <button onClick={openCreate} className="p-1 rounded-xl hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--color-accent)]"><Plus size={14} /></button>
      </div>

      {webhooks.length === 0 && <div className="text-[10px] text-[var(--color-muted-soft)] text-center py-2">{t('conversation.webhookNoData')}</div>}
      {webhooks.map(wh => (
        <div key={wh.id} className="flex items-start gap-2 py-2 border-b border-[var(--color-hairline-soft)] last:border-0">
          <div className="flex-1 min-w-0">
            <div className="text-xs font-medium text-[var(--color-ink)]">@{wh.name}</div>
            {wh.callback_url && <div className="text-[10px] text-[var(--color-muted)] truncate">{wh.callback_url}</div>}
            <div className="flex items-center gap-2 mt-1 text-[9px] text-[var(--color-muted-soft)]">
              <span className="flex items-center gap-0.5"><Key size={9} /> Key</span>
              {wh.cidr_whitelist?.length ? <span className="flex items-center gap-0.5"><Shield size={9} /> CIDR</span> : null}
              {wh.require_audit ? <span className="flex items-center gap-0.5"><Activity size={9} /> Review</span> : null}
            </div>
          </div>
          <div className="flex items-center gap-1 flex-shrink-0">
            <button onClick={() => openEdit(wh)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--color-accent)]">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17 3a2.85 2.85 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>
            </button>
            <button onClick={() => handleDelete(wh.id)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--destructive)]"><Trash2 size={12} /></button>
          </div>
        </div>
      ))}

      {/* Form modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={() => setShowForm(false)}>
          <div className="w-[380px] bg-[var(--color-surface-card)] rounded-xl p-4" style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
            <div className="text-sm font-medium text-[var(--color-ink)] mb-3">{editing ? t('conversation.webhookEdit') : t('conversation.webhookAdd')}</div>
            <div className="space-y-2.5">
              <div>
                <label className="text-[11px] text-[var(--color-muted)]">{t('conversation.webhookName')} *</label>
                <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} className={iCls} />
                <div className="text-[9px] text-[var(--color-muted-soft)] mt-0.5">{t('conversation.webhookNameHint')}</div>
              </div>
              <div>
                <label className="text-[11px] text-[var(--color-muted)]">{t('conversation.webhookCallbackURL')}</label>
                <input value={form.callback_url} onChange={e => setForm({ ...form, callback_url: e.target.value })} className={iCls} />
                <div className="text-[9px] text-[var(--color-muted-soft)] mt-0.5">{t('conversation.webhookCallbackHint')}</div>
              </div>
              <div>
                <label className="text-[11px] text-[var(--color-muted)]">{t('conversation.webhookCIDR')}</label>
                <input value={form.cidr_whitelist} onChange={e => setForm({ ...form, cidr_whitelist: e.target.value })} className={iCls} />
                <div className="text-[9px] text-[var(--color-muted-soft)] mt-0.5">{t('conversation.webhookCIDRHint')}</div>
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={form.require_audit} onChange={e => setForm({ ...form, require_audit: e.target.checked })} className="w-3.5 h-3.5 accent-[var(--color-primary)]" />
                <span className="text-[11px] text-[var(--color-muted)]">{t('conversation.webhookRequireAudit')}</span>
              </label>
              {!editing && <div className="text-[10px] text-[var(--color-muted-soft)] bg-[var(--color-surface-soft)] rounded-lg p-2">{t('conversation.webhookCreatedOnce')}</div>}
            </div>
            <div className="flex items-center justify-end gap-2 mt-4">
              <button onClick={() => setShowForm(false)} className="px-3 py-1.5 rounded-lg text-xs text-[var(--color-muted)] hover:bg-[var(--color-hairline)]">{t('common.cancel')}</button>
              <button onClick={handleSave} className="px-3 py-1.5 rounded-lg text-xs bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)]">{t('common.save')}</button>
            </div>
          </div>
        </div>
      )}

      {/* Secret display modal */}
      {created && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={() => setCreated(null)}>
          <div className="w-[380px] bg-[var(--color-surface-card)] rounded-xl p-4" style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
            <div className="text-sm font-medium text-[var(--color-ink)] mb-2">{t('conversation.webhookCreateSuccess')}</div>
            <div className="text-[10px] text-[var(--color-muted-soft)] mb-3">{t('conversation.webhookCreatedOnce')}</div>
            {created.token && (
              <div className="flex items-center justify-between mb-2">
                <div className="text-xs font-mono text-[var(--color-ink)] select-all flex-1 truncate mr-2">{created.token}</div>
                <button onClick={() => navigator.clipboard.writeText(created.token)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-[var(--color-muted)]"><Clipboard size={14} /></button>
              </div>
            )}
            <div className="flex items-center justify-between mb-3">
              <div className="text-xs font-mono text-[var(--color-ink)] select-all flex-1 truncate mr-2">{created.api_key}</div>
              <button onClick={() => navigator.clipboard.writeText(created.api_key)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-[var(--color-muted)]"><Clipboard size={14} /></button>
            </div>
            <button onClick={() => setCreated(null)} className="w-full h-8 rounded-lg text-xs bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)]">{t('common.close')}</button>
          </div>
        </div>
      )}

      {/* Pending messages */}
      {pendingMsgs.length > 0 && (
        <div className="border-t border-[var(--color-hairline)] pt-3 mt-3">
          <button onClick={() => setShowPending(!showPending)} className="flex items-center gap-2 text-xs font-medium text-[var(--color-muted)] mb-1">
            <Activity size={14} /> {t('conversation.webhookPendingTitle')} ({pendingMsgs.length})
          </button>
          {showPending && pendingMsgs.map(pm => (
            <div key={pm.msg_id} className="flex items-center gap-2 py-1.5 border-b border-[var(--color-hairline-soft)] last:border-0">
              <div className="flex-1 min-w-0 text-[11px] text-[var(--color-ink)] truncate">msg #{pm.msg_id}</div>
              <button onClick={() => handleApprove(pm.msg_id)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-green-500"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12"/></svg></button>
              <button onClick={() => handleReject(pm.msg_id)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-red-500"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
