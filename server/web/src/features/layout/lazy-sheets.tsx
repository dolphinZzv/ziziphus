import { lazy, Suspense } from 'react'
import { useTranslation } from 'react-i18next'

// Lazy-load all modal/sheet components for code splitting
const NewChatDialog = lazy(() => import('@/features/conversation-list/new-chat-dialog'))
const CreateGroupDialog = lazy(() => import('@/features/conversation-list/create-group-dialog'))
const JoinGroupDialog = lazy(() => import('@/features/conversation-list/join-group-dialog'))
const ProfileView = lazy(() => import('@/features/profile/profile-view'))
const SettingsView = lazy(() => import('@/features/settings/settings-view'))
const PrivacyView = lazy(() => import('@/features/settings/privacy-view'))
const AgentList = lazy(() => import('@/features/agents/agent-list'))
const SessionList = lazy(() => import('@/features/sessions/session-list'))
const ShortcutsView = lazy(() => import('@/features/settings/shortcuts-view'))
const MemberListView = lazy(() => import('@/features/group/member-list-view'))
const ContactList = lazy(() => import('@/features/contacts/contact-list'))
const AddContactDialog = lazy(() => import('@/features/conversation-list/add-contact-dialog'))

const SheetFallback = () => {
  const { t } = useTranslation()
  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30">
      <div className="p-6 text-[var(--color-muted)]">{t('common.loading')}</div>
    </div>
  )
}

interface SheetWrapperProps {
  name: string
  activeSheet: string | null
  onClose: () => void
}

export function SheetWrapper({ name, activeSheet, onClose }: SheetWrapperProps) {
  if (activeSheet !== name) return null

  return (
    <Suspense fallback={<SheetFallback />}>
      {name === 'newChat' && <NewChatDialog onClose={onClose} />}
      {name === 'createGroup' && <CreateGroupDialog onClose={onClose} />}
      {name === 'joinGroup' && <JoinGroupDialog onClose={onClose} />}
      {name === 'profile' && <ProfileView onClose={onClose} />}
      {name === 'settings' && <SettingsView onClose={onClose} />}
      {name === 'agents' && <AgentList onClose={onClose} />}
      {name === 'sessions' && <SessionList onClose={onClose} />}
      {name === 'contacts' && <ContactList onClose={onClose} />}
      {name === 'addContact' && <AddContactDialog onClose={onClose} />}
      {name === 'userSettings' && <PrivacyView onClose={onClose} />}
      {name === 'shortcuts' && <ShortcutsView onClose={onClose} />}
    </Suspense>
  )
}
