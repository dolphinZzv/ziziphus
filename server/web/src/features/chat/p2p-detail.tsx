import { useEffect, useState, useRef, useSyncExternalStore } from 'react'
import { userService } from '@/services/user-service'
import { conversationService } from '@/services/conversation-service'
import { fileService } from '@/services/file-service'
import { useTranslation } from 'react-i18next'
import { contactRequestService } from '@/services/contact-request-service'
import { authStore } from '@/stores/auth-store'
import { contactStore } from '@/stores/contact-store'
import type { ConversationDetail } from '@/types/conversation'
import type { User } from '@/types/user'
import { ConvType } from '@/types/conversation'
import { UserType } from '@/types/user'
import { avatarUrl } from '@/lib/file'
import { X, Copy, Check, Camera, UserPlus, UserCheck, Cpu } from 'lucide-react'

interface Props { convId: string; onClose: () => void }

export default function P2PDetail({ convId, onClose }: Props) {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<ConversationDetail | null>(null)
  const [peer, setPeer] = useState<User | null>(null)
  const [memberMap, setMemberMap] = useState<Record<string, User>>({})
  const [copied, setCopied] = useState(false)
  const [uploadingCover, setUploadingCover] = useState(false)
  const [requestSent, setRequestSent] = useState(false)
  const coverInputRef = useRef<HTMLInputElement>(null)
  const contacts = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.contacts)
  const currentUserId = authStore.state.user?.user_id || ''
  const peerId = detail?.members?.find(m => m.user_id !== currentUserId)?.user_id || ''
  const isContact = contacts.some(c => c.user_id === peerId)
  const isAgent = peer?.type === UserType.Agent

  useEffect(() => {
    conversationService.getDetail(convId).then(async d => {
      setDetail(d)
      if (d.type === ConvType.P2P) {
        const peerId = d.members?.find(m => m.user_id !== authStore.state.user?.user_id)?.user_id
        if (peerId) userService.getUser(peerId).then(setPeer).catch(() => {})
        // Fetch both members for member list
        const ids = d.members.map(m => m.user_id)
        if (ids.length > 0) {
          try { const users = await userService.batchGet(ids); setMemberMap(users) } catch {}
        }
      }
    }).catch(() => {})
  }, [convId])

  const copyId = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleCoverUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploadingCover(true)
    try { const r = await fileService.upload(file, file.name, 0); await conversationService.updateGroup(convId, { cover: r.url }); setDetail({ ...detail!, cover: r.url }) } catch {}
    setUploadingCover(false)
  }

  const handleAddContact = async () => {
    if (!peerId) return
    try { await contactRequestService.send(peerId); setRequestSent(true) } catch {}
  }

  const initials = peer?.name?.charAt(0)?.toUpperCase() || '?'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[360px] bg-[var(--color-surface-card)] rounded-lg overflow-hidden"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Header with cover as background */}
        <div className="h-28 flex items-end justify-between px-6 pb-5 relative"
          style={{ background: detail?.cover
            ? `url(${detail.cover}?w=720&h=224) center/cover`
            : peer?.primary_color
              ? `linear-gradient(135deg, ${peer.primary_color}, ${peer.secondary_color || peer.primary_color})`
              : 'var(--color-primary)' }}>
          {detail?.cover && <div className="absolute inset-0 bg-black/20" />}
          <button onClick={() => coverInputRef.current?.click()} disabled={uploadingCover}
            className="absolute top-3 left-3 p-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white z-10 transition-colors">
            <Camera size={14} />
          </button>
          <input ref={coverInputRef} type="file" accept="image/*" onChange={handleCoverUpload} className="hidden" />
          <div />
          <button onClick={onClose} className="p-1.5 rounded-lg bg-white/20 hover:bg-white/30 text-white relative z-10"><X size={15} /></button>
        </div>

        {/* Avatar — overlaps header */}
        <div className="flex justify-center -mt-10 mb-4">
          {peer?.avatar ? (
            <img src={avatarUrl(peer.avatar)} alt="" className="w-[72px] h-[72px] rounded-full object-cover border-4 border-[var(--color-surface-card)]" />
          ) : (
            <div className="w-[72px] h-[72px] rounded-full flex items-center justify-center text-white text-2xl font-bold border-4 border-[var(--color-surface-card)]"
              style={{ background: peer?.primary_color ? `linear-gradient(135deg, ${peer.primary_color}, ${peer.secondary_color || peer.primary_color})` : 'var(--color-primary)' }}>
              {initials}
            </div>
          )}
        </div>

        {/* Info */}
        <div className="px-6 pb-6">
          <div className="text-center space-y-1 mb-5">
            <div className="font-headline text-lg font-semibold text-[var(--color-ink)] flex items-center justify-center gap-1.5">
              {peer?.name || 'User'}
              {isAgent && <span className="text-[9px] px-1.5 py-0.5 rounded-sm bg-purple-500/10 text-purple-600 font-medium uppercase tracking-wider">Agent</span>}
            </div>
            <div className="text-sm text-[var(--color-muted)]">@{peer?.account || '—'}</div>
            <div className="flex items-center justify-center gap-1 text-[11px] text-[var(--color-muted-soft)] font-mono select-all">
              {peerId.slice(0, 18)}...
              <button onClick={() => copyId(peerId)} className="hover:text-[var(--color-ink)]">
                {copied ? <Check size={11} className="text-[var(--success)]" /> : <Copy size={11} />}
              </button>
            </div>
          </div>

          {/* Action list */}
          <div className="space-y-0.5 border-t border-[var(--color-hairline)] pt-4">
            {isContact ? (
              <button disabled
                className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg bg-[var(--success)]/5 text-sm text-[var(--success)]">
                <UserCheck size={18} /> {t('conversation.alreadyFriends')}
              </button>
            ) : requestSent ? (
              <button disabled
                className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg bg-[var(--color-muted)]/5 text-sm text-[var(--color-muted)]">
                <Check size={18} /> {t('conversation.requestSent')}
              </button>
            ) : (
              <button onClick={handleAddContact}
                className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-sm text-[var(--color-body)] hover:text-[var(--color-ink)] transition-colors">
                <UserPlus size={18} /> {t('conversation.addFriend')}
              </button>
            )}
          </div>

          {/* Members */}
          {detail && detail.members.length > 0 && (
            <div className="border-t border-[var(--color-hairline)] pt-4 mt-4">
              <div className="text-xs font-medium text-[var(--color-muted)] mb-3">{t('conversation.members')}</div>
              <div className="space-y-1">
                {detail.members.map(member => {
                  const u = memberMap[member.user_id]
                  const name = u?.name || member.user_id
                  const avatar = u?.avatar
                  const isAI = u?.type === UserType.Agent
                  return (
                    <div key={member.user_id} className="flex items-center gap-3 py-2 px-1 rounded-lg">
                      <div className="relative flex-shrink-0">
                        {avatar ? (
                          <img src={avatarUrl(avatar)} alt="" className="w-9 h-9 rounded-full object-cover" />
                        ) : (
                          <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-xs font-semibold"
                            style={{ background: isAI ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : u?.primary_color ? `linear-gradient(135deg, ${u.primary_color}, ${u.secondary_color || u.primary_color})` : 'var(--color-primary)' }}>
                            {name.charAt(0).toUpperCase()}
                          </div>
                        )}
                        {isAI && (
                          <div className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-purple-500 flex items-center justify-center border-2 border-[var(--color-surface-card)]">
                            <Cpu size={7} className="text-white" />
                          </div>
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="text-[13px] font-medium text-[var(--color-ink)] truncate">{name}</div>
                        <div className="text-[11px] text-[var(--color-muted)]">@{u?.account || member.user_id.slice(0, 12) + '...'}</div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}