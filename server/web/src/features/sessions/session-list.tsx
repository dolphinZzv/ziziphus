import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { sessionService } from '@/services/session-service'
import { authStore } from '@/stores/auth-store'
import type { Session, DeviceType } from '@/types/session'
import { X, Smartphone, Monitor, Tablet, Globe, LogOut, ArrowLeft } from 'lucide-react'
import { format } from 'date-fns'

interface Props { onClose: () => void; inline?: boolean }

export default function SessionList({ onClose, inline }: Props) {
  const { t } = useTranslation()
  const [sessions, setSessions] = useState<Session[]>([])
  const currentSessionId = authStore.state.sessionId

  useEffect(() => { sessionService.list().then(setSessions).catch(() => {}) }, [])

  const deviceIcon = (d: DeviceType) => {
    const cls = 'text-[var(--color-muted)]'
    switch (d) { case 0: return <Smartphone size={18} className={cls} />; case 1: return <Monitor size={18} className={cls} />; case 2: return <Globe size={18} className={cls} />; case 3: return <Tablet size={18} className={cls} />; default: return <Monitor size={18} className={cls} /> }
  }

  const deviceLabel = (d: DeviceType) => { switch (d) { case 0: return t('session.phone'); case 1: return t('session.desktop'); case 2: return t('session.web'); case 3: return t('session.tablet'); default: return '' } }

  const handleEndSession = async (sessionId: string) => {
    if (!confirm('确定下线该设备？')) return
    try { await sessionService.delete(sessionId); setSessions(s => s.filter(s => s.session_id !== sessionId)) } catch {}
  }

  const inner = (
    <div className={`${inline ? 'h-full' : 'w-full sm:w-[400px] max-h-[500px]'} bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl p-6 flex flex-col overflow-hidden`}
      style={inline ? {} : { boxShadow: 'var(--shadow-lg)' }} onClick={inline ? undefined : e => e.stopPropagation()}>
      <div className="flex items-center justify-between mb-5">
        <div className="flex items-center gap-2">
          {inline && (
            <button onClick={onClose} className="p-1 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">
              <ArrowLeft size={18} />
            </button>
          )}
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">{t('session.title')}</h3>
        </div>
        {!inline && <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>}
      </div>

      <div className="flex-1 overflow-y-auto space-y-1">
        {sessions.map(s => (
          <div key={s.session_id} className={`flex items-center gap-3 px-3 h-12 rounded-xl hover:bg-[var(--color-surface-soft)] group ${s.session_id === currentSessionId ? 'bg-[var(--color-primary)]/5' : ''}`}>
            {deviceIcon(s.device)}
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium text-[var(--color-ink)] flex items-center gap-2">
                {deviceLabel(s.device)} {s.device_name || ''}
                {s.session_id === currentSessionId && <span className="text-[9px] px-1.5 py-0.5 rounded-sm bg-green-500/10 text-green-600 font-medium uppercase tracking-wider">{t('session.current')}</span>}
              </div>
              <div className="text-[11px] text-[var(--color-muted)]">{s.last_active ? format(new Date(s.last_active), 'yyyy/MM/dd HH:mm') : ''}</div>
            </div>
            {s.session_id !== currentSessionId && (
              <button onClick={() => handleEndSession(s.session_id)} className="p-1.5 rounded-xl hover:bg-[var(--destructive)]/10 opacity-0 group-hover:opacity-100 text-[var(--destructive)] transition-all"><LogOut size={14} /></button>
            )}
          </div>
        ))}
        {sessions.length === 0 && <p className="text-sm text-[var(--color-muted)] text-center py-8">{t('session.noSessions')}</p>}
      </div>
    </div>
  )

  if (inline) return inner
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4" onClick={onClose}>
      {inner}
    </div>
  )
}
