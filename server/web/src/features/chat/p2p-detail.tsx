import { useEffect, useState } from 'react'
import { userService } from '@/services/user-service'
import { conversationService } from '@/services/conversation-service'
import { authStore } from '@/stores/auth-store'
import type { ConversationDetail } from '@/types/conversation'
import type { User } from '@/types/user'
import { ConvType } from '@/types/conversation'
import { avatarUrl } from '@/lib/file'
import { X, Copy, Check } from 'lucide-react'

interface Props { convId: string; onClose: () => void }

export default function P2PDetail({ convId, onClose }: Props) {
  const [detail, setDetail] = useState<ConversationDetail | null>(null)
  const [peer, setPeer] = useState<User | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    conversationService.getDetail(convId).then(d => {
      setDetail(d)
      if (d.type === ConvType.P2P) {
        const peerId = d.members?.find(m => m.user_id !== authStore.state.user?.user_id)?.user_id
        if (peerId) userService.getUser(peerId).then(setPeer).catch(() => {})
      }
    }).catch(() => {})
  }, [convId])

  const copyId = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const initials = peer?.name?.charAt(0)?.toUpperCase() || '?'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[340px] bg-[var(--color-surface-card)] rounded-2xl p-6"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">聊天详情</h3>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)]"><X size={16} /></button>
        </div>

        {/* Avatar */}
        <div className="flex justify-center mb-4">
          {peer?.avatar ? (
            <img src={avatarUrl(peer.avatar)} alt="" className="w-[72px] h-[72px] rounded-full object-cover" />
          ) : (
            <div className="w-[72px] h-[72px] rounded-full flex items-center justify-center text-white text-2xl font-bold"
              style={{ background: peer?.primary_color
                ? `linear-gradient(135deg, ${peer.primary_color}, ${peer.secondary_color || peer.primary_color})`
                : 'var(--color-primary)' }}>
              {initials}
            </div>
          )}
        </div>

        {/* Info */}
        <div className="text-center space-y-2 mb-6">
          <div className="font-headline text-lg font-semibold text-[var(--color-ink)]">
            {peer?.name || '用户'}
          </div>
          <div className="text-sm text-[var(--color-muted)]">
            账号: {peer?.account || '—'}
          </div>
          <div className="flex items-center justify-center gap-1 text-[11px] text-[var(--color-muted)] font-mono select-all">
            ID: {peer?.user_id || convId}
            <button onClick={() => copyId(peer?.user_id || convId)} className="p-0.5 hover:text-[var(--color-ink)]">
              {copied ? <Check size={12} className="text-[var(--success)]" /> : <Copy size={12} />}
            </button>
          </div>
          <div className="text-sm text-[var(--color-muted)]">
            会话 ID: <span className="font-mono text-[11px] select-all">{convId}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
