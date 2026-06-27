import { useEffect, useState } from 'react'
import { userService } from '@/services/user-service'
import { avatarUrl } from '@/lib/file'
import type { User } from '@/types/user'
import { UserType } from '@/types/user'
import { Copy, Check, Loader2, Bot } from 'lucide-react'

interface Props {
  userId: string
  onClose: () => void
}

export default function UserCard({ userId, onClose }: Props) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    userService.getUser(userId).then(u => { setUser(u); setLoading(false) }).catch(() => setLoading(false))
  }, [userId])

  const copyId = () => {
    navigator.clipboard.writeText(userId)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const initials = (user?.name || userId).charAt(0).toUpperCase()
  const isAgent = user?.type === UserType.Agent

  return (
    <div className="w-[240px] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg p-4"
      style={{ boxShadow: 'var(--shadow-md)' }}>
      {loading ? (
          <div className="flex items-center justify-center py-4"><Loader2 size={16} className="animate-spin text-[var(--color-muted)]" /></div>
        ) : user ? (
          <div className="text-center space-y-3">
            {/* Avatar */}
            <div className="flex justify-center">
              {user.avatar ? (
                <img src={avatarUrl(user.avatar)} alt="" className="w-14 h-14 rounded-full object-cover" />
              ) : (
                <div className="w-14 h-14 rounded-full flex items-center justify-center text-white text-lg font-bold"
                  style={{ background: user.primary_color
                    ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})`
                    : isAgent ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : 'var(--color-primary)' }}>
                  {initials}
                </div>
              )}
            </div>

            {/* Name + type badge */}
            <div>
              <div className="font-headline text-base font-semibold text-[var(--color-ink)] flex items-center justify-center gap-1.5">
                {user.name}
                {isAgent && <Bot size={14} className="text-purple-500" />}
              </div>
              <div className="text-xs text-[var(--color-muted)] mt-0.5">@{user.account}</div>
            </div>

            {/* ID */}
            <div className="flex items-center justify-center gap-1 text-[11px] text-[var(--color-muted-soft)] font-mono select-all">
              {userId.slice(0, 16)}...
              <button onClick={copyId} className="hover:text-[var(--color-ink)]">
                {copied ? <Check size={12} className="text-[var(--success)]" /> : <Copy size={12} />}
              </button>
            </div>
          </div>
        ) : (
          <p className="text-xs text-[var(--color-muted)] text-center py-2">加载失败</p>
        )}
    </div>
  )
}
