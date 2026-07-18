import { useEffect, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { sessionService } from '@/services/session-service'
import type { Session } from '@/types/session'
import { DeviceType } from '@/types/session'
import { Smartphone, Monitor, Tablet, Globe, LogOut } from 'lucide-react'

function getLabel(d: DeviceType): string {
  switch (d) {
    case DeviceType.Phone: return '手机'
    case DeviceType.Desktop: return '电脑'
    case DeviceType.Web: return '网页'
    case DeviceType.Tablet: return '平板'
    default: return ''
  }
}

const ICONS: Record<number, typeof Smartphone> = {
  [DeviceType.Phone]: Smartphone,
  [DeviceType.Desktop]: Monitor,
  [DeviceType.Web]: Globe,
  [DeviceType.Tablet]: Tablet,
}

export default function DeviceIndicator() {
  const { t } = useTranslation()
  const [list, setList] = useState<Session[]>([])
  const [open, setOpen] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)
  const btnRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    const load = () => sessionService.list().then(setList).catch(() => {})
    const tid = setTimeout(load, 100)
    const iid = setInterval(load, 60_000)
    return () => { clearTimeout(tid); clearInterval(iid) }
  }, [])

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node) &&
          btnRef.current && !btnRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  if (list.length === 0) return null

  return (
    <div className="fixed bottom-4 left-4 z-40 max-md:hidden">
      <button
        ref={btnRef}
        onClick={() => setOpen(o => !o)}
        className="flex items-center gap-1.5 px-2.5 h-8 rounded-lg bg-[var(--color-surface-card)] border border-[var(--color-hairline)] text-[11px] text-[var(--color-muted)] hover:bg-[var(--color-surface-soft)] transition-colors cursor-pointer shadow-sm"
        title={t('session.title')}
      >
        <Smartphone size={12} className="text-[var(--color-muted)]" />
        <span className="font-medium">{list.length}{t('session.deviceUnit')}</span>
      </button>

      {open && (
        <div
          ref={panelRef}
          className="absolute bottom-full left-0 mb-2 z-50 w-72 rounded-xl bg-[var(--color-surface-card)] border border-[var(--color-hairline)] overflow-hidden"
          style={{ boxShadow: 'var(--shadow-lg)' }}
        >
          <div className="px-4 py-3 border-b border-[var(--color-hairline)]">
            <h4 className="text-sm font-semibold text-[var(--color-ink)]">{t('session.title')}</h4>
            <p className="text-[11px] text-[var(--color-muted)] mt-0.5">{list.length} {t('session.deviceUnit')}</p>
          </div>
          <div className="max-h-64 overflow-y-auto py-1">
            {list.map(s => {
              const Icon = ICONS[s.device] || Monitor
              return (
                <div key={s.session_id} className="flex items-center gap-3 px-4 h-11 hover:bg-[var(--color-surface-soft)] group">
                  <Icon size={15} className="text-[var(--color-muted)]" />
                  <div className="flex-1 min-w-0">
                    <div className="text-xs font-medium text-[var(--color-ink)] flex items-center gap-1.5">
                      {s.device_name || getLabel(s.device)}
                    </div>
                    <div className="text-[10px] text-[var(--color-muted-soft)]">
                      {s.last_active ? new Date(s.last_active).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) : ''}
                    </div>
                  </div>
                  <button
                    onClick={() => {
                      sessionService.delete(s.session_id).then(() => setList(prev => prev.filter(x => x.session_id !== s.session_id))).catch(() => {})
                    }}
                    className="p-1 rounded hover:bg-[var(--destructive)]/10 text-[var(--destructive)] opacity-0 group-hover:opacity-100 transition-all"
                    title={t('session.endSession')}
                  >
                    <LogOut size={12} />
                  </button>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
