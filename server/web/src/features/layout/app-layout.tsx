import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { Outlet, useLocation } from 'react-router-dom'
import Sidebar from '@/features/layout/sidebar'
import { SheetWrapper } from '@/features/layout/lazy-sheets'
import { authStore } from '@/stores/auth-store'
import { uiStore } from '@/stores/ui-store'
import { wsClient } from '@/services/websocket-client'
import { MessageType } from '@/types/ws'
import type { ConnectionStatus } from '@/services/websocket-client'
import type { MsgPushPayload } from '@/types/ws'
import { cn } from '@/lib/cn'
import { useIsMobile, useIsTablet } from '@/hooks/use-breakpoint'

export default function AppLayout() {
  const { t } = useTranslation()
  const location = useLocation()
  const isSidebarOpen = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.isSidebarOpen)
  const theme = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.theme)
  const [connStatus, setConnStatus] = useState<ConnectionStatus>('disconnected')
  const [sidebarWidth, setSidebarWidth] = useState(288)
  const isMobile = useIsMobile()
  const isTablet = useIsTablet()
  const activeSheet = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.activeSheet)

  useEffect(() => {
    setConnStatus(wsClient.connectionStatus)
    return wsClient.onStatusChange(setConnStatus)
  }, [])

  // Desktop notifications
  useEffect(() => {
    if (!('Notification' in window)) return
    if (Notification.permission !== 'granted' && Notification.permission !== 'denied') {
      Notification.requestPermission()
    }
    const user = authStore.state.user
    const handler = wsClient.on(MessageType.MsgPush, (payload: unknown) => {
      const push = payload as MsgPushPayload
      if (document.hasFocus()) return
      if (push.sender_id === user?.user_id) return
      if (Notification.permission !== 'granted') return
      const body = push.body || ''
      new Notification(push.sender_name || '新消息', {
        body: body.length > 120 ? body.slice(0, 120) + '...' : body,
        icon: '/favicon.ico',
      })
    })
    return () => handler?.()
  }, [])

  // Global keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey
      if (e.key === 'Escape') uiStore.closeSheet()
      if (mod && e.key === 'n') { e.preventDefault(); uiStore.openSheet('newChat') }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [])

  const connected = connStatus === 'connected'

  // Computed widths
  const sideW = isMobile ? '100%' : (isTablet ? 240 : sidebarWidth)

  return (
    <div className={cn('h-full w-full flex flex-col', theme === 'dark' ? 'dark' : '')}>
      {/* Connection status bar */}
      {!connected && (
        <div className={cn(
          'h-6 flex items-center justify-center text-[11px] font-medium flex-shrink-0',
          connStatus === 'connecting' && 'bg-[var(--warning)]/10 text-[var(--warning)]',
          connStatus === 'recovering' && 'bg-[var(--warning)]/10 text-[var(--warning)]',
          connStatus === 'disconnected' && 'bg-[var(--destructive)]/10 text-[var(--destructive)]',
        )}>
          <span className={cn('w-1.5 h-1.5 rounded-full mr-1.5',
            (connStatus === 'connecting' || connStatus === 'recovering') && 'bg-[var(--warning)] animate-pulse',
            connStatus === 'disconnected' && 'bg-[var(--destructive)]',
          )} />
          {connStatus === 'connecting' ? t('connection.connecting') : connStatus === 'recovering' ? t('connection.recovering') : t('connection.disconnected')}
        </div>
      )}

      {/* Body */}
      <div className="flex-1 flex min-h-0 relative">
        {/* Mobile sidebar backdrop */}
        {isMobile && isSidebarOpen && (
          <div
            className="fixed inset-0 z-40 bg-black/30"
            onClick={() => uiStore.toggleSidebar()}
          />
        )}

        {/* Sidebar */}
        <div
          className={cn(
            // Desktop/tablet: inline
            'flex-shrink-0 h-full flex flex-col border-r border-[var(--color-hairline)] bg-[var(--color-canvas)]',
            // Mobile: fixed overlay with transition
            isMobile && 'fixed inset-y-0 left-0 z-50 transition-transform duration-200',
            isMobile && (isSidebarOpen ? 'translate-x-0' : '-translate-x-full'),
            // Non-mobile: inline, hidden when toggled off
            !isMobile && 'transition-transform duration-200',
            !isMobile && !isSidebarOpen && '-translate-x-full hidden',
          )}
          style={{ width: sideW }}
        >
          <Sidebar />
        </div>

        {/* Drag handle (desktop only) */}
        {!isMobile && !isTablet && isSidebarOpen && (
          <div className="relative flex-shrink-0" style={{ width: 0 }}>
            <div
              className="absolute -left-1 top-0 bottom-0 w-2 cursor-col-resize group z-10"
              onMouseDown={e => {
                e.preventDefault(); e.stopPropagation()
                const sx = e.clientX; const sw = sidebarWidth
                document.body.style.userSelect = 'none'
                const mv = (ev: MouseEvent) => { ev.preventDefault(); setSidebarWidth(Math.max(200, Math.min(400, sw + ev.clientX - sx))) }
                const up = () => { document.body.style.userSelect = ''; document.removeEventListener('mousemove', mv); document.removeEventListener('mouseup', up) }
                document.addEventListener('mousemove', mv); document.addEventListener('mouseup', up)
              }}
            />
          </div>
        )}

        {/* Chat area */}
        <div className={cn(
          'flex-1 h-full flex flex-col min-w-0 bg-[var(--color-surface-soft)]',
          isMobile && isSidebarOpen && 'hidden'
        )}>
          {/* Mobile: no header when sidebar is visible (conversation list is fullscreen) */}
          <Outlet />
        </div>
      </div>

      {/* Lazy-loaded sheets (rendered at root level, outside sidebar) */}
      {['newChat','addContact','createGroup','joinGroup','profile','settings','userSettings','agents','sessions','contacts','shortcuts'].map(name => (
        <SheetWrapper key={name} name={name} activeSheet={activeSheet} onClose={() => uiStore.closeSheet()} />
      ))}
    </div>
  )
}
