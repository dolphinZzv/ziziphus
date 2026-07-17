import { useEffect, useSyncExternalStore, Suspense, lazy } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import './i18n'
import { authStore } from '@/stores/auth-store'
import { uiStore } from '@/stores/ui-store'
import AppLayout from '@/features/layout/app-layout'
import AppShell from '@/features/layout/app-shell'
import LoginPage from '@/features/auth/login-page'
import RegisterPage from '@/features/auth/register-page'
import ForgotPasswordPage from '@/features/auth/forgot-password-page'
import EmptyChat from '@/features/chat/empty-chat'
import ErrorBoundary from '@/components/error-boundary'
import ProfilePage from '@/features/profile/profile-page'

const ChatView = lazy(() => import('@/features/chat/chat-view'))

const PageFallback = () => (
  <div className="h-full flex items-center justify-center text-sm text-[var(--color-muted)]">加载中...</div>
)

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
        <Route path="/login" element={isLoggedIn ? <Navigate to="/chat" replace /> : <LoginPage />} />
        <Route path="/register" element={isLoggedIn ? <Navigate to="/chat" replace /> : <RegisterPage />} />
        <Route path="/forgot-password" element={isLoggedIn ? <Navigate to="/chat" replace /> : <ForgotPasswordPage />} />
        <Route path="/" element={<AuthGuard><ErrorBoundary><AppLayout /></ErrorBoundary></AuthGuard>}>
          <Route index element={<Navigate to="/chat" replace />} />
          <Route path="chat" element={<Suspense fallback={<PageFallback />}><EmptyChat /></Suspense>} />
          <Route path="chat/:convId" element={<Suspense fallback={<PageFallback />}><ChatView /></Suspense>} />
          <Route path="profile" element={<ProfilePage />} />
        </Route>
        <Route path="*" element={<Navigate to="/chat" replace />} />
        </Routes>
      </AppShell>
    </BrowserRouter>
  )
}
