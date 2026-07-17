import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { conversationService } from '@/services/conversation-service'
import { userService } from '@/services/user-service'
import { avatarUrl } from '@/lib/file'
import { authStore } from '@/stores/auth-store'
import type { ConversationDetail } from '@/types/conversation'
import type { User } from '@/types/user'
import { ConvRole } from '@/types/conversation'
import { X, Crown, Shield, Trash2, Cpu, Search } from 'lucide-react'

interface Props { convId: string; onClose: () => void }

export default function MemberListView({ convId, onClose }: Props) {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<ConversationDetail | null>(null)
  const [userMap, setUserMap] = useState<Record<string, User>>({})
  const [filterQuery, setFilterQuery] = useState('')
  const currentUserId = authStore.state.user?.user_id || ''

  const load = () => {
    conversationService.getDetail(convId).then(async d => {
      setDetail(d)
      const ids = d.members.map(m => m.user_id)
      if (ids.length > 0) {
        try { const users = await userService.batchGet(ids); setUserMap(users) } catch {}
      }
    }).catch(() => {})
  }
  useEffect(() => { load() }, [convId])

  if (!detail) return null

  const me = detail.members.find(m => m.user_id === currentUserId)
  const isAdmin = me?.role === ConvRole.Admin || me?.role === ConvRole.Owner
  const u = (id: string) => userMap[id]
  const uname = (id: string) => userMap[id]?.name || id

  const filteredMembers = filterQuery.trim()
    ? detail.members.filter(m => {
        const userName = uname(m.user_id)
        const q = filterQuery.trim().toLowerCase()
        return userName.toLowerCase().includes(q) || m.nickname?.toLowerCase().includes(q)
      })
    : detail.members

  const handleRemove = async (userId: string) => {
    if (!confirm(t('group.removeConfirm'))) return
    try {
      await conversationService.removeMember(convId, userId)
      setDetail({ ...detail, members: detail.members.filter(m => m.user_id !== userId) })
    } catch {}
  }

  return (
    <>
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[380px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] bg-[var(--color-surface-card)] rounded-none sm:rounded-xl overflow-hidden flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--color-hairline)]">
          <h3 className="font-headline text-base font-semibold text-[var(--color-ink)]">
            {t('group.members')} ({detail.members.length})
          </h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <div className="px-4 pt-3 pb-2">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)]" />
            <input type="text" value={filterQuery} onChange={e => setFilterQuery(e.target.value)}
              placeholder="搜索成员..."
              className="w-full h-9 pl-8 pr-3 rounded-xl bg-[var(--color-surface-soft)] text-xs border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)]" />
            {filterQuery && (
              <button onClick={() => setFilterQuery('')} className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-[var(--color-muted)]"><X size={12} /></button>
            )}
          </div>
        </div>

        <div className="flex-1 overflow-y-auto px-3 pb-3 space-y-0.5">
          {filteredMembers.length === 0 ? (
            <p className="text-sm text-[var(--color-muted)] text-center py-8">{t('conversation.noMatch')}</p>
          ) : filteredMembers.map(member => {
            const name = uname(member.user_id)
            const avatar = u(member.user_id)?.avatar
            return (
              <div key={member.user_id} className="flex items-center gap-3 px-3 py-2 rounded-xl hover:bg-[var(--color-surface-soft)] group">
                <div className="relative flex-shrink-0">
                  {avatar ? (
                    <img src={avatarUrl(avatar)} alt="" className="w-9 h-9 rounded-full object-cover" />
                  ) : (
                    <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-xs font-semibold"
                      style={{ background: member.user_type === 1 ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                      {name.charAt(0).toUpperCase()}
                    </div>
                  )}
                  {member.user_type === 1 && (
                    <span className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-purple-500 flex items-center justify-center border-2 border-[var(--color-surface-card)]">
                      <Cpu size={7} className="text-white" />
                    </span>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-[var(--color-ink)] flex items-center gap-1.5">
                    <span className="truncate">{member.nickname || name}</span>
                    {member.role === ConvRole.Owner && <Crown size={11} className="text-[var(--warning)] flex-shrink-0" />}
                    {member.role === ConvRole.Admin && <Shield size={11} className="text-[var(--info)] flex-shrink-0" />}
                    {member.user_id === currentUserId && <span className="text-[10px] text-[var(--color-muted)]">{t('group.me')}</span>}
                  </div>
                  <div className="text-[11px] text-[var(--color-muted)]">@{u(member.user_id)?.account || member.user_id.slice(0, 12) + '...'}</div>
                </div>
                {isAdmin && member.user_id !== currentUserId && member.role !== ConvRole.Owner && (
                  <button onClick={() => handleRemove(member.user_id)}
                    className="p-1 rounded-lg hover:bg-[var(--destructive)]/10 text-[var(--color-muted)] hover:text-[var(--destructive)]">
                    <Trash2 size={14} />
                  </button>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
    </>
  )
}
