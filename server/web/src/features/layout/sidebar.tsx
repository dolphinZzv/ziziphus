import { useEffect, useSyncExternalStore, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import { conversationStore } from '@/stores/conversation-store'
import { uiStore } from '@/stores/ui-store'
import { wsClient } from '@/services/websocket-client'
import { MessageType } from '@/types/ws'
import ConversationList from '@/features/conversation-list/conversation-list'
import { avatarUrl } from '@/lib/file'
import { Plus, User, Users, UserPlus, MessageCircle, Search } from 'lucide-react'
import { useIsMobile } from '@/hooks/use-breakpoint'
import { useTranslation } from 'react-i18next'

export default function Sidebar() {
  const navigate = useNavigate()
  const user = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const { t } = useTranslation()
  const isMobile = useIsMobile()
  const [showMenu, setShowMenu] = useState(false)

  useEffect(() => {
    if (!showMenu) return
    const handler = (e: MouseEvent) => {
      if (!(e.target as HTMLElement).closest('.plus-menu-container')) setShowMenu(false)
    }
    document.addEventListener('click', handler)
    return () => document.removeEventListener('click', handler)
  }, [showMenu])

  useEffect(() => {
    conversationStore.load()
  }, [])

  useEffect(() => {
    const u1 = wsClient.on(MessageType.MsgPush, () => conversationStore.refresh())
    const u2 = wsClient.on(MessageType.SessionOnline, () => conversationStore.refresh())
    const u3 = wsClient.on(MessageType.SessionOffline, () => conversationStore.refresh())
    return () => { u1?.(); u2?.(); u3?.() }
  }, [])

  return (
    <>
      {/* User header */}
      <div className="flex items-center justify-between px-3 h-12">
        <button
          onClick={() => uiStore.openSheet('profile')}
          className="flex items-center gap-2 pl-2 text-sm font-medium text-[var(--color-ink)] hover:opacity-80 min-w-0 btn-press"
        >
          {user?.avatar ? (
            <img src={avatarUrl(user.avatar)} alt="" className="w-8 h-8 rounded-full object-cover" />
          ) : (
            <div className="w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-semibold"
              style={{ background: user?.primary_color
                ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})`
                : 'var(--color-primary)' }}>
              {user?.name?.charAt(0)?.toUpperCase() || '?'}
            </div>
          )}
          <span className="truncate max-w-[140px]">{user?.name || user?.account || 'User'}</span>
        </button>

        <div className="flex items-center gap-0.5">
          <button
            onClick={() => uiStore.openSheet('contacts')}
            className="p-2 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
            title={t('sidebar.contacts')}>
            <User size={18} />
          </button>
          <div className="relative plus-menu-container">
            <button
              onClick={() => setShowMenu(!showMenu)}
              className="p-2 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
            >
              <Plus size={18} />
            </button>
            {showMenu && (
              <div className="absolute right-0 top-full mt-1 w-44 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl z-[100] py-1"
                style={{ boxShadow: 'var(--shadow-md)' }}>
                {[
                  { icon: MessageCircle, label: t('sidebar.newChat'), sheet: 'newChat' },
                  { icon: Search, label: t('sidebar.addContact'), sheet: 'addContact' },
                  { icon: Users, label: t('sidebar.createGroup'), sheet: 'createGroup' },
                  { icon: UserPlus, label: t('sidebar.joinGroup'), sheet: 'joinGroup' },
                ].map(item => (
                  <button key={item.sheet}
                    onClick={() => { uiStore.openSheet(item.sheet); setShowMenu(false) }}
                    className="w-full flex items-center gap-3 px-4 py-2.5 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)] hover:text-[var(--color-ink)] transition-colors">
                    <item.icon size={16} /> {item.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Conversation list */}
      <div className="flex-1 overflow-hidden">
        <ConversationList />
      </div>
    </>
  )
}
