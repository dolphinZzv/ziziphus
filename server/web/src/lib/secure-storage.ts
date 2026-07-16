/**
 * Secure storage using sessionStorage (cleared when tab is closed).
 * Use for sensitive data like auth tokens and user profiles.
 * Falls back to a no-op in environments where sessionStorage is unavailable.
 */

const store: Storage | null =
  typeof sessionStorage !== 'undefined' ? sessionStorage : null

const PREFIX = 'ziziphus_'

export function getItem<T>(key: string, fallback?: T): T | undefined {
  if (!store) return fallback
  try {
    const raw = store.getItem(PREFIX + key)
    if (raw === null) return fallback
    return JSON.parse(raw) as T
  } catch {
    return fallback
  }
}

export function setItem<T>(key: string, value: T): void {
  if (!store) return
  try {
    store.setItem(PREFIX + key, JSON.stringify(value))
  } catch { /* storage full — ignore */ }
}

export function removeItem(key: string): void {
  if (!store) return
  try {
    store.removeItem(PREFIX + key)
  } catch { /* ignore */ }
}
