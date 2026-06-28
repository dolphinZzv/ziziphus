import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { Outlet } from 'react-router-dom'
import Sidebar from '@/features/layout/sidebar'
import { uiStore } from '@/stores/ui-store'
import { wsClient } from '@/services/websocket-client'
import type { ConnectionStatus } from '@/services/websocket-client'
import { cn } from '@/lib/cn'

export default function AppLayout() {
  const { t } = useTranslation()
  const isSidebarOpen = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.isSidebarOpen)
  const theme = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.theme)
  const [connStatus, setConnStatus] = useState<ConnectionStatus>('disconnected')

  useEffect(() => {
    setConnStatus(wsClient.connectionStatus)
    return wsClient.onStatusChange(setConnStatus)
  }, [])

  // Global keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey
      // Escape → close current sheet
      if (e.key === 'Escape') uiStore.closeSheet()
      // Cmd/Ctrl + N → new chat
      if (mod && e.key === 'n') { e.preventDefault(); uiStore.openSheet('newChat') }
      // Cmd/Ctrl + K → focus search (future)
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [])

  const connected = connStatus === 'connected'

  return (
    <div className={cn('h-full w-full flex flex-col', theme === 'dark' ? 'dark' : '')}>
      {/* Connection status bar — full width top */}
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
      <div className="flex-1 flex min-h-0">
        <div className={cn(
          'w-72 flex-shrink-0 border-r h-full flex flex-col transition-transform duration-200',
          'bg-[var(--color-canvas)] border-[var(--color-hairline)]',
          !isSidebarOpen && '-translate-x-full hidden'
        )}>
          <Sidebar />
        </div>

        <div className="flex-1 h-full flex flex-col min-w-0 bg-[var(--color-surface-soft)]">
          <Outlet />
        </div>
      </div>
    </div>
  )
}
