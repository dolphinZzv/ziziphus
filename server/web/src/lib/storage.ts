const PREFIX = 'ziziphus_'

export function getItem<T>(key: string, fallback?: T): T | undefined {
  try {
    const raw = localStorage.getItem(PREFIX + key)
    if (raw === null) return fallback
    return JSON.parse(raw) as T
  } catch {
    return fallback
  }
}

export function setItem<T>(key: string, value: T): void {
  try {
    localStorage.setItem(PREFIX + key, JSON.stringify(value))
  } catch { /* storage full - ignore */ }
}

export function removeItem(key: string): void {
  localStorage.removeItem(PREFIX + key)
}

// Device ID
export function getDeviceId(): string {
  let id = localStorage.getItem(PREFIX + 'device_id')
  if (!id) {
    id = 'web_' + safeUUID().slice(0, 8)
    localStorage.setItem(PREFIX + 'device_id', id)
  }
  return id
}

// Saved accounts
export function getSavedAccounts(): string[] {
  return getItem<string[]>('saved_accounts', [])!
}

export function saveAccount(account: string): void {
  const accounts = getSavedAccounts().filter(a => a !== account)
  accounts.unshift(account)
  setItem('saved_accounts', accounts.slice(0, 5))
}

export function removeSavedAccount(account: string): void {
  const accounts = getSavedAccounts().filter(a => a !== account)
  setItem('saved_accounts', accounts)
}

// Drafts
export function getDraft(convId: string): string {
  return getItem<string>(`draft_${convId}`, '')!
}

export function setDraft(convId: string, text: string): void {
  if (text.trim()) {
    setItem(`draft_${convId}`, text)
  } else {
    removeItem(`draft_${convId}`)
  }
}

// Server URL
export function getServerUrl(): string {
  return getItem<string>('server_url', '')!
}

export function setServerUrl(url: string): void {
  setItem('server_url', url)
}

// Client seq counter
let _clientSeq = 0
export function nextClientSeq(): number {
  _clientSeq++
  return _clientSeq
}

// safeUUID returns a UUID v4 string. Works in both secure (HTTPS) and
// insecure (HTTP) contexts. Falls back to a manual implementation when
// crypto.randomUUID is unavailable.
export function safeUUID(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  // Fallback for HTTP contexts
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = (Math.random() * 16) | 0
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16)
  })
}
