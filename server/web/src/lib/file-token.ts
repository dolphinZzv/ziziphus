import { authStore } from '@/stores/auth-store'

/**
 * Append file access token as ?token= to a local file URL.
 *
 * The token is a random hex string (NOT a JWT) with a 5-minute TTL,
 * stored in Redis and bound to the authenticated user.
 *
 * Used for <img> tags, <a> download links, and <object> PDF embeds.
 * For sensitive operations that should never appear in URLs,
 * use Authorization: Bearer <file_token> directly.
 */
export function withFileToken(url: string): string {
  if (!url) return url
  const ft = authStore.state.fileToken
  if (!ft) return url
  if (!url.startsWith('/files/')) return url
  const sep = url.includes('?') ? '&' : '?'
  return `${url}${sep}token=${ft}`
}
