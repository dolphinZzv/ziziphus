import { useEffect, useSyncExternalStore, Suspense, lazy } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import './i18n'
import { authStore } from '@/stores/auth-store'
import { uiStore } from '@/stores/ui-store'
import AppShell from '@/features/layout/app-shell'
import ErrorBoundary from '@/components/error-boundary'

const AppLayout = lazy(() => import('@/features/layout/app-layout'))
const AuthPage = lazy(() => import('@/features/auth/auth-page'))
const ConversationsPage = lazy(() => import('@/features/conversation-list/conversations-page'))
const GroupCardPage = lazy(() => import('@/features/group/group-card-page'))
const ChatView = lazy(() => import('@/features/chat/chat-view'))

const PageFallback = () => (
  <div className="h-full flex items-center justify-center text-sm text-[var(--color-muted)]">加载中...</div>
)

/** Suspense-wrapped ChatView — reused across /conversations/:convId sub-routes */
function LazyChatView() {
  return <Suspense fallback={<PageFallback />}><ChatView /></Suspense>
}

function AuthGuard({ children }: { children: React.ReactNode }) {
  const isLoggedIn = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoggedIn)
  if (!isLoggedIn) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  const isLoggedIn = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoggedIn)

  useEffect(() => {
    authStore.checkExistingSession()
  }, [])

  // Listen for system theme changes
  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => {
      if (uiStore.state.theme === 'system') {
        document.documentElement.classList.toggle('dark', mq.matches)
      }
    }
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [])

  return (
    <BrowserRouter>
      <AppShell>
        <Routes>
        <Route path="/login" element={isLoggedIn ? <Navigate to="/conversations" replace /> : <Suspense fallback={<PageFallback />}><AuthPage /></Suspense>} />
        <Route path="/register" element={isLoggedIn ? <Navigate to="/conversations" replace /> : <Suspense fallback={<PageFallback />}><AuthPage /></Suspense>} />
        <Route path="/forgot-password" element={isLoggedIn ? <Navigate to="/conversations" replace /> : <Suspense fallback={<PageFallback />}><AuthPage /></Suspense>} />
        <Route path="/" element={<AuthGuard><ErrorBoundary><Suspense fallback={<PageFallback />}><AppLayout /></Suspense></ErrorBoundary></AuthGuard>}>
          <Route index element={<Navigate to="/conversations" replace />} />
          <Route path="conversations" element={<Suspense fallback={<PageFallback />}><ConversationsPage /></Suspense>} />
          <Route path="conversations/:convId" element={<LazyChatView />} />
          <Route path="conversations/:convId/info" element={<LazyChatView />} />
          <Route path="conversations/:convId/settings" element={<LazyChatView />} />
          <Route path="conversations/:convId/webhooks" element={<LazyChatView />} />
          <Route path="conversations/:convId/members" element={<LazyChatView />} />
          <Route path="conversations/:convId/add-member" element={<LazyChatView />} />
          <Route path="conversations/:convId/detail" element={<LazyChatView />} />
          <Route path="conversations/:convId/history" element={<LazyChatView />} />
          <Route path="profile" element={<ConversationsPage />} />
          <Route path="profile/edit" element={<ConversationsPage />} />
          <Route path="profile/agents" element={<ConversationsPage />} />
          <Route path="profile/privacy" element={<ConversationsPage />} />
          <Route path="profile/sessions" element={<ConversationsPage />} />
          <Route path="profile/settings" element={<ConversationsPage />} />
          <Route path="contacts" element={<ConversationsPage />} />
          <Route path="new-chat" element={<ConversationsPage />} />
          <Route path="add-contact" element={<ConversationsPage />} />
          <Route path="create-group" element={<ConversationsPage />} />
          <Route path="join-group" element={<ConversationsPage />} />
        </Route>
        <Route path="/group-card/:shareToken" element={<Suspense fallback={<PageFallback />}><GroupCardPage /></Suspense>} />
        <Route path="*" element={<Navigate to="/conversations" replace />} />
        </Routes>
      </AppShell>
    </BrowserRouter>
  )
}

