import { getItem, setItem, getDeviceId } from '@/lib/storage'
import i18n from '@/i18n'

type Theme = 'light' | 'dark' | 'auto'
type Language = 'zh' | 'en' | 'auto'

function resolveAutoLang(): 'zh' | 'en' {
  if (typeof navigator === 'undefined') return 'en'
  const nav = (navigator.language || '').toLowerCase()
  if (nav.startsWith('zh')) return 'zh'
  return 'en'
}

interface UIState {
  selectedConvId: string | null
  activeSheet: string | null
  isSidebarOpen: boolean
  sidebarView: string | null
  theme: Theme
  language: Language
  serverUrl: string
  bubbleColor: string
  deviceId: string
}

function applyBubbleColor(color: string) {
  const root = document.documentElement
  root.style.setProperty('--bubble-self', color)
}

function getInitialState(): UIState {
  const theme = (getItem<string>('theme', 'auto') as Theme) || 'auto'
  const language = (getItem<string>('language', 'auto') as Language) || 'auto'
  const serverUrl = getItem<string>('server_url', '')
  const bubbleColor = getItem<string>('bubble_color', '#0F172A')

  // Apply theme
  applyTheme(theme)
  // Apply language
  if (language === 'auto') {
    i18n.changeLanguage(resolveAutoLang())
  } else {
    i18n.changeLanguage(language)
  }
  // Apply bubble color
  applyBubbleColor(bubbleColor)

  return {
    selectedConvId: null,
    activeSheet: null,
    isSidebarOpen: true,
    sidebarView: null,
    theme,
    language,
    serverUrl,
    bubbleColor,
    deviceId: getDeviceId(),
  }
}

function applyTheme(theme: Theme) {
  const root = document.documentElement
  if (theme === 'dark') {
    root.classList.add('dark')
  } else if (theme === 'light') {
    root.classList.remove('dark')
  } else {
    // auto: follow system
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    root.classList.toggle('dark', prefersDark)
  }
}

let state = getInitialState()
const listeners = new Set<() => void>()
function emit() { listeners.forEach(l => l()) }

export const uiStore = {
  get state() { return state },

  subscribe(fn: () => void) {
    listeners.add(fn)
    return () => { listeners.delete(fn) }
  },

  selectConversation(convId: string | null) {
    state = { ...state, selectedConvId: convId, activeSheet: null, sidebarView: null }; emit()
  },

  openSheet(name: string) {
    state = { ...state, activeSheet: name }; emit()
  },

  closeSheet() {
    state = { ...state, activeSheet: null }; emit()
  },

  setSidebarView(view: string | null) {
    state = { ...state, sidebarView: view, activeSheet: null }; emit()
  },

  toggleSidebar() {
    state = { ...state, isSidebarOpen: !state.isSidebarOpen }; emit()
  },

  setTheme(theme: Theme) {
    setItem('theme', theme)
    applyTheme(theme)
    state = { ...state, theme }; emit()
  },

  setLanguage(lang: Language) {
    setItem('language', lang)
    if (lang === 'auto') {
      // Clear stored preference so auto-detection works on reload
      setItem('panda_ai_language', '')
      i18n.changeLanguage(resolveAutoLang())
    } else {
      setItem('panda_ai_language', lang)
      i18n.changeLanguage(lang)
    }
    state = { ...state, language: lang }; emit()
  },

  setServerUrl(url: string) {
    setItem('server_url', url)
    state = { ...state, serverUrl: url }; emit()
  },

  setBubbleColor(color: string) {
    setItem('bubble_color', color)
    applyBubbleColor(color)
    state = { ...state, bubbleColor: color }; emit()
  },
}
