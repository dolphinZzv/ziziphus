import { useEffect, useRef } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { uiStore } from '@/stores/ui-store'

/** Maps routes to sheet names */
const routeToSheet: Record<string, string> = {
  '/profile': 'profile',
  '/profile/edit': 'editProfile',
  '/profile/agents': 'agents',
  '/profile/privacy': 'userSettings',
  '/profile/sessions': 'sessions',
  '/profile/settings': 'settings',
  '/contacts': 'contacts',
  '/new-chat': 'newChat',
  '/add-contact': 'addContact',
  '/create-group': 'createGroup',
  '/join-group': 'joinGroup',
}

/** Maps sheet names back to routes */
const sheetToRoute: Record<string, string> = {}
for (const [route, sheet] of Object.entries(routeToSheet)) {
  sheetToRoute[sheet] = route
}

/**
 * Synchronises activeSheet with URL.
 * - Sheet opens → push route
 * - Sheet closes → navigate(-1) (back to parent route or previous page)
 * - URL changes (browser back/forward, direct nav) → open/close sheet accordingly
 */
export default function SheetRouteSync() {
  const location = useLocation()
  const navigate = useNavigate()
  const prevSheet = useRef<string | null>(null)
  const syncing = useRef(false)

  // Listen for sheet changes → update URL
  useEffect(() => {
    return uiStore.subscribe(() => {
      if (syncing.current) return
      const sheet = uiStore.state.activeSheet
      const prev = prevSheet.current
      prevSheet.current = sheet

      // Sheet opened → push its route
      if (sheet && sheet !== prev && sheetToRoute[sheet]) {
        const target = sheetToRoute[sheet]
        if (location.pathname !== target) {
          navigate(target)
        }
      }

      // Sheet closed → go back
      if (!sheet && prev && sheetToRoute[prev]) {
        if (location.pathname.startsWith('/profile') || location.pathname === '/new-chat' || location.pathname === '/add-contact' || location.pathname === '/create-group' || location.pathname === '/join-group') {
          navigate(-1)
        }
      }
    })
  }, [location.pathname, navigate])

  // Listen for URL changes → open/close sheet
  useEffect(() => {
    const expectedSheet = routeToSheet[location.pathname]
    const currentSheet = uiStore.state.activeSheet

    if (expectedSheet && currentSheet !== expectedSheet) {
      // Route matches a sheet but sheet isn't open → open it
      syncing.current = true
      uiStore.openSheet(expectedSheet)
      syncing.current = false
    } else if (!expectedSheet && currentSheet && sheetToRoute[currentSheet]) {
      // Route doesn't match any sheet but a sheet is open → close it
      syncing.current = true
      uiStore.closeSheet()
      syncing.current = false
    }
  }, [location.pathname])

  return null
}
