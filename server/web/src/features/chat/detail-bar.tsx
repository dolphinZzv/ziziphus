import { useEffect, useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { conversationService } from '@/services/conversation-service'
import { userService } from '@/services/user-service'
import { authStore } from '@/stores/auth-store'
import { avatarUrl } from '@/lib/file'
import type { ConversationDetail } from '@/types/conversation'
import type { User } from '@/types/user'
import { ConvRole, ConvType } from '@/types/conversation'
import { X, Crown, Shield, UserPlus, Edit2, Bell, LogOut, Copy as CopyIcon, Cpu } from 'lucide-react'

interface Props { convId: string; onClose: () => void }

export default function DetailBar({ convId, onClose }: Props) {
  const [detail, setDetail] = useState<ConversationDetail | null>(null)
  const [userMap, setUserMap] = useState<Record<string, User>>({})
  const navigate = useNavigate()
  const currentUserId = authStore.state.user?.user_id || ''
  const [cloning, setCloning] = useState(false)

  useEffect(() => {
    conversationService.getDetail(convId).then(async d => {
      setDetail(d)
      const ids = d.members.map(m => m.user_id)
      if (ids.length > 0) {
        try { const users = await userService.batchGet(ids); setUserMap(users) } catch {}
      }
    }).catch(onClose)
  }, [convId])

  if (!detail) return null

  const me = detail.members.find(m => m.user_id === currentUserId)
  const isAdmin = me?.role === ConvRole.Admin || me?.role === ConvRole.Owner
  const isOwner = me?.role === ConvRole.Owner
  const isGroup = detail.type === ConvType.Group
  const peerUser = !isGroup ? userMap[detail.members.find(m => m.user_id !== currentUserId)?.user_id || ''] : null

  const handleClone = async () => {
    if (!confirm('克隆该群组？')) return
    setCloning(true)
    try {
      const r = await conversationService.clone(convId)
      onClose()
      navigate(`/chat/${r.conv_id}`)
    } catch {}
    setCloning(false)
  }

  return (
    <div className="border-b border-[var(--color-hairline)] bg-[var(--color-surface-card)] max-h-[240px] overflow-y-auto">
      <div className="px-4 py-3">
        <div className="flex items-center justify-between mb-3">
          <span className="text-xs font-medium text-[var(--color-muted)] uppercase tracking-wider">会话详情</span>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--color-surface-soft)]"><X size={14} /></button>
        </div>

        {/* P2P: peer info */}
        {!isGroup && peerUser && (
          <div className="flex items-center gap-3">
            {peerUser.avatar ? (
              <img src={avatarUrl(peerUser.avatar)} alt="" className="w-10 h-10 rounded-full object-cover" />
            ) : (
              <div className="w-10 h-10 rounded-full flex items-center justify-center text-white text-sm font-semibold"
                style={{ background: 'var(--color-primary)' }}>{peerUser.name?.charAt(0)?.toUpperCase() || '?'}</div>
            )}
            <div>
              <div className="text-sm font-semibold text-[var(--color-ink)]">{peerUser.name}</div>
              <div className="text-[11px] text-[var(--color-muted)]">@{peerUser.account}</div>
            </div>
          </div>
        )}

        {/* Group: info compact */}
        {isGroup && (
          <div className="space-y-2">
            <div className="flex items-center gap-3">
              {detail.avatar ? (
                <img src={avatarUrl(detail.avatar)} alt="" className="w-10 h-10 rounded-full object-cover flex-shrink-0" />
              ) : (
                <div className="w-10 h-10 rounded-full flex items-center justify-center text-white font-bold flex-shrink-0"
                  style={{ background: 'linear-gradient(135deg, var(--color-accent), #34D399)' }}>{detail.name?.charAt(0)?.toUpperCase() || 'G'}</div>
              )}
              <div>
                <div className="text-sm font-semibold text-[var(--color-ink)]">{detail.name}</div>
                <div className="text-[11px] text-[var(--color-muted)]">{detail.members.length} 成员</div>
              </div>
              {isOwner && (
                <button onClick={handleClone} disabled={cloning}
                  className="ml-auto px-2.5 py-1 rounded text-[10px] border border-dashed border-[var(--color-hairline)] text-[var(--color-muted)] hover:border-[var(--color-primary)] hover:text-[var(--color-primary)]">
                  <CopyIcon size={10} className="inline mr-1" />克隆
                </button>
              )}
            </div>
            {/* Notice preview */}
            {detail.notice && (
              <div className="text-[11px] text-[var(--color-body)] bg-[var(--color-warning)]/5 rounded px-2.5 py-1.5 leading-relaxed">
                <Bell size={10} className="inline mr-1.5 text-[var(--color-muted)]" />
                {detail.notice.slice(0, 100)}{detail.notice.length > 100 ? '...' : ''}
              </div>
            )}
            {/* Member list */}
            <div className="flex flex-wrap gap-1.5">
              {detail.members.slice(0, 15).map(m => {
                const name = userMap[m.user_id]?.name || m.user_id
                const avatar = userMap[m.user_id]?.avatar
                return (
                  <div key={m.user_id} className="flex items-center gap-1 bg-[var(--color-surface-soft)] rounded-full pl-1 pr-2 py-0.5 text-[11px]" title={name}>
                    {avatar ? (
                      <img src={avatarUrl(avatar)} alt="" className="w-4 h-4 rounded-full object-cover" />
                    ) : (
                      <div className="w-4 h-4 rounded-full flex items-center justify-center text-white text-[8px] font-bold"
                        style={{ background: m.user_type === 1 ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                        {name.charAt(0)}
                      </div>
                    )}
                    <span className="text-[var(--color-ink)] ml-0.5">{name}</span>
                    {m.role === ConvRole.Owner && <Crown size={9} className="text-[var(--warning)]" />}
                    {m.role === ConvRole.Admin && <Shield size={9} className="text-[var(--info)]" />}
                  </div>
                )
              })}
              {detail.members.length > 15 && <span className="text-[11px] text-[var(--color-muted)] self-center">+{detail.members.length - 15}</span>}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
