import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { webhookService } from '@/services/webhook-service'
import type { ConvWebhook } from '@/types/webhook'
import { X, Globe, Plus, Trash2, Key, Shield, Copy, Check, ArrowLeft } from 'lucide-react'
import { useIsMobile } from '@/hooks/use-breakpoint'

interface Props { convId: string; onClose: () => void }

export default function WebhookPanel({ convId, onClose }: Props) { const isMobile=useIsMobile()
  const { t } = useTranslation()
  const [webhooks, setWebhooks] = useState<ConvWebhook[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<ConvWebhook | null>(null)
  const [form, setForm] = useState({ name: '', callback_url: '', cidr_whitelist: '', headers: '' })
  const [saveError, setSaveError] = useState('')
  const [testingId, setTestingId] = useState<number | null>(null)
  const [testResults, setTestResults] = useState<Record<number, { ok: boolean; msg: string }>>({})
  const [copiedCode, setCopiedCode] = useState('')

  useEffect(() => {
    webhookService.list(convId).then(setWebhooks).catch(() => {})
  }, [convId])

  const resetForm = () => { setForm({ name: '', callback_url: '', cidr_whitelist: '', headers: '' }); setSaveError('') }

  const openCreate = () => { setEditing(null); resetForm(); setShowForm(true) }
  const openEdit = (wh: ConvWebhook) => {
    setEditing(wh)
    setForm({ name: wh.name, callback_url: wh.callback_url || '', cidr_whitelist: (wh.cidr_whitelist || []).join(', '), headers: (wh.headers || []).map(h => `${h.key}: ${h.value}`).join('\n') })
    setShowForm(true)
  }

  const handleSave = async () => {
    if (!form.name.trim()) { setSaveError('名称不能为空'); return }
    setSaveError('')
    try {
      const data = { name: form.name, callback_url: form.callback_url, cidr_whitelist: form.cidr_whitelist ? form.cidr_whitelist.split(',').map(s => s.trim()).filter(Boolean) : [], headers: form.headers ? form.headers.split('\n').map(line => { const idx = line.indexOf(':'); return idx > 0 ? { key: line.slice(0, idx).trim(), value: line.slice(idx + 1).trim() } : null }).filter(Boolean) as { key: string; value: string }[] : [] }
      if (editing) {
        await webhookService.update(convId, editing.id, data)
        setWebhooks(prev => prev.map(w => w.id === editing.id ? { ...w, ...data } : w))
      } else {
        const result = await webhookService.create(convId, data)
        setWebhooks(prev => [...prev, result])
      }
      setShowForm(false)
    } catch (e: any) {
      setSaveError(e?.message || '保存失败，请检查权限和输入')
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm(t('conversation.webhookDeleteConfirm'))) return
    try { await webhookService.delete(convId, id); setWebhooks(prev => prev.filter(w => w.id !== id)) } catch (e) { console.error(e) }
  }

  const handleTest = async (id: number) => {
    setTestingId(id)
    setTestResults(prev => ({ ...prev, [id]: { ok: false, msg: '测试中...' } }))
    try {
      await webhookService.test(convId, id)
      setTestResults(prev => ({ ...prev, [id]: { ok: true, msg: '✅ 发送成功' } }))
    } catch (e: any) {
      setTestResults(prev => ({ ...prev, [id]: { ok: false, msg: `❌ ${e?.message || '失败'}` } }))
    }
    setTestingId(null)
    setTimeout(() => setTestResults(prev => { const n = { ...prev }; delete n[id]; return n }), 3000)
  }

  const copyText = (label: string, text: string) => {
    navigator.clipboard.writeText(text)
    setCopiedCode(label)
    setTimeout(() => setCopiedCode(''), 2000)
  }

  const iCls = 'w-full h-8 px-2 rounded-lg text-xs bg-[var(--color-surface-soft)] border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)]'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[420px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] bg-[var(--color-surface-card)] rounded-none sm:rounded-xl overflow-hidden flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4">
          <div className="flex items-center gap-2">
            <Globe size={16} className="text-[var(--color-muted)]" />
            <h3 className="font-headline text-base font-semibold text-[var(--color-ink)]">{t('conversation.webhookTitle')}</h3>
          </div>
          <div className="flex items-center gap-1">
            <button onClick={openCreate} className="p-1.5 rounded-xl hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--color-accent)]"><Plus size={16} /></button>
            <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">{isMobile ? <ArrowLeft size={18} /> : <X size={16} />}</button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-5 py-3">
          {webhooks.length === 0 && <div className="text-xs text-[var(--color-muted-soft)] text-center py-6">{t('conversation.webhookNoData')}</div>}
          {webhooks.map(wh => (
            <div key={wh.id} className="flex items-start gap-2 py-2.5 border-b border-[var(--color-hairline-soft)] last:border-0">
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium text-[var(--color-ink)]">@{wh.name}</div>
                {wh.callback_url && <div className="text-[11px] text-[var(--color-muted)] truncate">{wh.callback_url}</div>}
                <div className="flex items-center gap-2 mt-1 text-[10px] text-[var(--color-muted-soft)]">
                  <span className="flex items-center gap-0.5"><Key size={9} /> Key</span>
                  {wh.cidr_whitelist?.length ? <span className="flex items-center gap-0.5"><Shield size={9} /> CIDR</span> : null}
                </div>
              </div>
              <div className="flex items-center gap-1 flex-shrink-0">
                <button onClick={() => handleTest(wh.id)} disabled={testingId === wh.id}
                  className="p-1 rounded hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-green-500 disabled:opacity-40"
                  title="测试">
                  {testingId === wh.id
                    ? <span className="block w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" />
                    : <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>}
                </button>
                {testResults[wh.id] && (
                  <span className={`text-[10px] ${testResults[wh.id].ok ? 'text-green-500' : 'text-red-500'}`}>
                    {testResults[wh.id].msg}
                  </span>
                )}
                <button onClick={() => openEdit(wh)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--color-accent)]">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17 3a2.85 2.85 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>
                </button>
                <button onClick={() => handleDelete(wh.id)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--destructive)]"><Trash2 size={12} /></button>
              </div>
            </div>
          ))}

          {/* Form modal */}
          {showForm && (
            <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={() => setShowForm(false)}>
              <div className="w-full sm:w-[380px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] overflow-y-auto bg-[var(--color-surface-card)] rounded-none sm:rounded-xl p-4" style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
                <div className="flex items-center justify-between mb-3">
                <span className="text-sm font-medium text-[var(--color-ink)]">{editing ? t('conversation.webhookEdit') : t('conversation.webhookAdd')}</span>
                <button onClick={() => setShowForm(false)} className="p-1 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={15} /></button>
              </div>
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
                    <label className="text-[11px] text-[var(--color-muted)]">{t('conversation.webhookHeaders')}</label>
                    <textarea value={form.headers} onChange={e => setForm({ ...form, headers: e.target.value })} rows={3} placeholder="X-Custom: value" className={iCls} style={{ height: 'auto', resize: 'vertical' }} />
                    <div className="text-[9px] text-[var(--color-muted-soft)] mt-0.5">{t('conversation.webhookHeadersHint')}</div>
                  </div>
                  <div>
                    <label className="text-[11px] text-[var(--color-muted)]">{t('conversation.webhookCIDR')}</label>
                    <input value={form.cidr_whitelist} onChange={e => setForm({ ...form, cidr_whitelist: e.target.value })} className={iCls} />
                    <div className="text-[9px] text-[var(--color-muted-soft)] mt-0.5">{t('conversation.webhookCIDRHint')}</div>
                  </div>
                  {saveError && <div className="text-[11px] text-red-500 bg-red-50 rounded-lg px-2 py-1">{saveError}</div>}
                  {editing && (() => {
                    const baseUrl = window.location.origin
                    const apiKey = editing.api_key || '<api_key>'
                    const curlCmd = `curl -X POST \\\n  "${baseUrl}/api/v1/webhooks/receive" \\\n  -H "Authorization: Bearer ${apiKey}" \\\n  -H "Content-Type: application/json" \\\n  -d '{"body": "Hello from webhook", "content_type": 0}'`
                    const hasCallback = !!(editing.callback_url || form.callback_url)
                    let socatCmd = ''
                    let socatPort = '8080'
                    if (hasCallback) {
                      try { const u = new URL(editing.callback_url || form.callback_url!); socatPort = u.port || '8080'; socatCmd = `socat TCP-LISTEN:${socatPort},reuseaddr,fork -` } catch (e) { console.error(e) }
                    }
                    return (
                    <>
                    {/* API Key */}
                    <div className=" space-y-1.5">
                      <div className="flex items-center justify-between">
                        <label className="text-[10px] text-[var(--color-muted)]">API Key</label>
                        <button onClick={() => copyText('apikey', apiKey)} className="text-[var(--color-muted)] hover:text-[var(--color-ink)] p-0.5">
                          {copiedCode === 'apikey' ? <Check size={11} className="text-[var(--success)]" /> : <Copy size={11} />}
                        </button>
                      </div>
                      <code className="block bg-[var(--color-surface-soft)] rounded-lg p-1.5 text-[10px] text-[var(--color-ink)] break-all select-all font-mono">{apiKey}</code>
                    </div>
                    {/* Usage examples */}
                    <details open className="text-[10px]">
                      <summary className="text-[var(--color-muted)] cursor-pointer hover:text-[var(--color-ink)]">{t('conversation.webhookUsageExamples')}</summary>
                      <div className="mt-2 space-y-2">
                        <div>
                          <div className="flex items-center justify-between mb-1">
                            <span className="text-[var(--color-muted)]">curl</span>
                            <button onClick={() => copyText('curl', curlCmd)}
                              className="text-[var(--color-muted)] hover:text-[var(--color-ink)] p-0.5">
                              {copiedCode === 'curl' ? <Check size={12} className="text-[var(--success)]" /> : <Copy size={12} />}
                            </button>
                          </div>
                          <code className="block bg-[var(--color-surface-soft)] rounded-lg p-2 text-[var(--color-ink)] break-all select-all">{curlCmd}</code>
                        </div>
                        {hasCallback && socatCmd && (
                        <div>
                          <div className="flex items-center justify-between mb-1">
                            <span className="text-[var(--color-muted)]">socat</span>
                            <button onClick={() => copyText('socat', socatCmd)}
                              className="text-[var(--color-muted)] hover:text-[var(--color-ink)] p-0.5">
                              {copiedCode === 'socat' ? <Check size={12} className="text-[var(--success)]" /> : <Copy size={12} />}
                            </button>
                          </div>
                          <code className="block bg-[var(--color-surface-soft)] rounded-lg p-2 text-[var(--color-ink)] break-all select-all">{socatCmd}</code>
                        </div>
                        )}
                      </div>
                    </details>
                    </>
                    )})()}
                </div>
                <div className="flex items-center justify-end gap-2 mt-4">
                  <button onClick={() => setShowForm(false)} className="px-3 py-1.5 rounded-lg text-xs text-[var(--color-muted)] hover:bg-[var(--color-hairline)]">{t('common.cancel')}</button>
                  <button onClick={handleSave} className="px-3 py-1.5 rounded-lg text-xs bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)]">{t('common.save')}</button>
                </div>
              </div>
            </div>
          )}

        </div>
      </div>
    </div>
  )
}
