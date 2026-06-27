import { useEffect, useState } from 'react'
import { sessionService } from '@/services/session-service'
import { authStore } from '@/stores/auth-store'
import type { Session, DeviceType } from '@/types/session'
import { X, Smartphone, Monitor, Tablet, Globe, LogOut } from 'lucide-react'
import { format } from 'date-fns'

interface Props { onClose: () => void }

export default function SessionList({ onClose }: Props) {
  const [sessions, setSessions] = useState<Session[]>([])
  const currentSessionId = authStore.state.sessionId

  useEffect(() => { sessionService.list().then(setSessions).catch(() => {}) }, [])

  const deviceIcon = (d: DeviceType) => {
    const cls = 'text-[var(--color-muted)]'
    switch (d) { case 0: return <Smartphone size={18} className={cls} />; case 1: return <Monitor size={18} className={cls} />; case 2: return <Globe size={18} className={cls} />; case 3: return <Tablet size={18} className={cls} />; default: return <Monitor size={18} className={cls} /> }
  }

  const deviceLabel = (d: DeviceType) => { switch (d) { case 0: return '手机'; case 1: return '电脑'; case 2: return '网页'; case 3: return '平板'; default: return '未知' } }

  const handleEndSession = async (sessionId: string) => {
    if (!confirm('确定下线该设备？')) return
    try { await sessionService.delete(sessionId); setSessions(s => s.filter(s => s.session_id !== sessionId)) } catch {}
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[400px] max-h-[500px] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg p-6 flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">设备管理</h3>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <div className="flex-1 overflow-y-auto space-y-0.5">
          {sessions.map(session => (
            <div key={session.session_id} className="flex items-center gap-3 px-3 h-12 rounded-lg hover:bg-[var(--color-surface-soft)] group">
              {deviceIcon(session.device)}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 text-sm text-[var(--color-ink)]">
                  {session.device_name || deviceLabel(session.device)}
                  {session.session_id === currentSessionId && (
                    <span className="inline-flex px-1.5 py-0.5 rounded-sm bg-[var(--color-accent)]/10 text-[var(--color-accent)] text-[10px] font-medium uppercase tracking-wider">当前</span>
                  )}
                  {session.status === 0 && <span className="w-1.5 h-1.5 rounded-full bg-[var(--success)]" />}
                </div>
                <div className="text-[11px] text-[var(--color-muted)]">
                  {session.client_ip} · {session.login_at ? format(new Date(session.login_at * 1000), 'MM/dd HH:mm') : ''}
                </div>
              </div>
              {session.session_id !== currentSessionId && (
                <button onClick={() => handleEndSession(session.session_id)}
                  className="p-1.5 rounded-lg hover:bg-[var(--destructive)]/10 opacity-0 group-hover:opacity-100 text-[var(--destructive)] transition-all"><LogOut size={14} /></button>
              )}
            </div>
          ))}
          {sessions.length === 0 && (
            <p className="text-sm text-[var(--color-muted)] text-center py-8">暂无其他设备</p>
          )}
        </div>
      </div>
    </div>
  )
}
