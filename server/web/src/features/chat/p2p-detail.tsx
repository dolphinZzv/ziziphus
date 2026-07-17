import { useEffect, useState, useRef, useSyncExternalStore } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { userService } from '@/services/user-service'
import { conversationService } from '@/services/conversation-service'
import { conversationStore } from '@/stores/conversation-store'
import { getConvSettings, subscribe as settingsSubscribe } from '@/stores/conversation-settings-store'
import { fileService } from '@/services/file-service'
import { contactRequestService } from '@/services/contact-request-service'
import { authStore } from '@/stores/auth-store'
import { contactStore } from '@/stores/contact-store'
import { ArrowLeft, X, Camera, Check, Copy, UserPlus, LogOut, EyeOff, Cpu } from 'lucide-react'
import { useIsMobile } from '@/hooks/use-breakpoint'
import { UserType } from '@/types/user'
import { ConvType } from '@/types/conversation'
import { avatarUrl } from '@/lib/file'

interface Props { convId: string; onClose: () => void }

export default function P2PDetail({ convId, onClose }: Props) { const isMobile=useIsMobile()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<ConversationDetail | null>(null)
  const [peer, setPeer] = useState<User | null>(null)
  const [memberMap, setMemberMap] = useState<Record<string, User>>({})
  const [copied, setCopied] = useState(false)
  const [uploadingCover, setUploadingCover] = useState(false)
  const [requestSent, setRequestSent] = useState(false)
  const coverInputRef = useRef<HTMLInputElement>(null)
  const [showAgentResponseOnly, setShowAgentResponseOnly] = useState(
    () => getConvSettings(convId).showAgentResponseOnly
  )
  useEffect(() => {
    return settingsSubscribe(() => {
      setShowAgentResponseOnly(getConvSettings(convId).showAgentResponseOnly)
    })
  }, [convId])
  const contacts = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.contacts)
  const currentUserId = authStore.state.user?.user_id || ''
  const peerId = detail?.members?.find(m => m.user_id !== currentUserId)?.user_id || ''
  const isContact = contacts.some(c => c.user_id === peerId)
  const isAgent = peer?.type === UserType.Agent

  useEffect(() => {
    conversationService.getDetail(convId).then(async d => {
      setDetail(d)
      if (d.type === ConvType.P2P) {
        const pid = d.members?.find(m => m.user_id !== authStore.state.user?.user_id)?.user_id
        if (pid) userService.getUser(pid).then(setPeer).catch(() => {})
        const ids = d.members.map(m => m.user_id)
        if (ids.length > 0) {
          try { const users = await userService.batchGet(ids); setMemberMap(users) } catch {}
        }
      }
    }).catch(() => {})
  }, [convId])

  const copyId = (text: string) => {
    navigator.clipboard.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 2000)
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

  const handleLeave = async () => {
    if (!confirm(t('group.leaveConfirm'))) return
    try { await conversationService.leave(convId); conversationStore.removeConversation(convId); onClose(); navigate('/conversations') } catch {}
  }

  const initials = peer?.name?.charAt(0)?.toUpperCase() || '?'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[360px] h-full sm:h-auto bg-[var(--color-surface-card)] rounded-none sm:rounded-xl overflow-hidden"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Banner */}
        <div className="h-28 relative"
          style={{ background: detail?.cover
            ? `url(${detail.cover}?w=720&h=224) center/cover`
            : peer?.primary_color
              ? `linear-gradient(135deg, ${peer.primary_color}, ${peer.secondary_color || peer.primary_color})`
              : 'var(--color-primary)' }}>
          {detail?.cover && <div className="absolute inset-0 bg-black/20" />}
          <button onClick={() => coverInputRef.current?.click()} disabled={uploadingCover}
            className="absolute top-3 left-3 p-1.5 rounded-xl bg-white/10 hover:bg-white/20 text-white/70 hover:text-white z-10 transition-colors">
            <Camera size={14} />
          </button>
          <input ref={coverInputRef} type="file" accept="image/*" onChange={handleCoverUpload} className="hidden" />
          <button onClick={onClose} className="absolute top-3 right-3 p-1.5 rounded-xl bg-white/20 hover:bg-white/30 text-white z-10">{isMobile ? <ArrowLeft size={18} /> : <X size={15} />}</button>
        </div>

        {/* Avatar — overlaps banner */}
        <div className="flex justify-center -mt-10 mb-3">
          {peer?.avatar ? (
            <img src={avatarUrl(peer.avatar, 160)} alt="" className="w-20 h-20 rounded-full object-cover " />
          ) : (
            <div className="w-20 h-20 rounded-full flex items-center justify-center text-white text-2xl font-bold "
              style={{ background: peer?.primary_color ? `linear-gradient(135deg, ${peer.primary_color}, ${peer.secondary_color || peer.primary_color})` : 'var(--color-primary)' }}>{initials}</div>
          )}
        </div>

        {/* Name */}
        <div className="text-center px-6 mb-4">
          <div className="font-headline text-xl font-semibold text-[var(--color-ink)] flex items-center justify-center gap-1.5">
            {peer?.name || 'User'}
            {isAgent && <span className="text-[9px] px-1.5 py-0.5 rounded-sm bg-purple-500/10 text-purple-600 font-medium uppercase tracking-wider">Agent</span>}
          </div>
          <div className="text-sm text-[var(--color-muted)] mt-0.5">@{peer?.account || '—'}</div>
        </div>

        {/* Info */}
        <div className="px-6 pb-4 space-y-4">
          <div className="flex items-center justify-center gap-1 text-[11px] text-[var(--color-muted-soft)] font-mono select-all">
            {peerId.slice(0, 18)}...
            <button onClick={() => copyId(peerId)} className="hover:text-[var(--color-ink)]">{copied ? <Check size={11} className="text-[var(--success)]" /> : <Copy size={11} />}</button>
          </div>

          {/* Action */}
          {!isContact && (
            <div className="pt-3">
              {requestSent ? (
                <button disabled className="w-full flex items-center justify-center gap-2 h-10 rounded-xl bg-[var(--color-muted)]/5 text-sm text-[var(--color-muted)] font-medium">
                  <Check size={16} /> {t('conversation.requestSent')}
                </button>
              ) : (
                <button onClick={handleAddContact} className="w-full flex items-center justify-center gap-2 h-10 rounded-xl border border-[var(--color-hairline)] hover:bg-[var(--color-surface-soft)] text-sm text-[var(--color-body)] hover:text-[var(--color-ink)] transition-colors font-medium">
                  <UserPlus size={16} /> {t('conversation.addFriend', '添加好友')}
                </button>
              )}
            </div>
          )}

          {/* Members */}
          {detail && detail.members.length > 0 && (
            <div className="pt-3">
              <div className="text-xs font-medium text-[var(--color-muted)] mb-2">{t('conversation.members')}</div>
              <div className="space-y-1">
                {detail.members.map(member => {
                  const mu = memberMap[member.user_id]
                  const name = mu?.name || member.user_id
                  const avatar = mu?.avatar
                  const isAI = mu?.type === UserType.Agent
                  return (
                    <div key={member.user_id} className="flex items-center gap-3 py-1.5 px-1 rounded-xl">
                      <div className="relative flex-shrink-0">
                        {avatar ? (
                          <img src={avatarUrl(avatar)} alt="" className="w-8 h-8 rounded-full object-cover" />
                        ) : (
                          <div className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-semibold"
                            style={{ background: isAI ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : mu?.primary_color ? `linear-gradient(135deg, ${mu.primary_color}, ${mu.secondary_color || mu.primary_color})` : 'var(--color-primary)' }}>
                            {name.charAt(0).toUpperCase()}
                          </div>
                        )}
                        {isAI && <div className="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full bg-purple-500 flex items-center justify-center border-2 border-[var(--color-surface-card)]"><Cpu size={6} className="text-white" /></div>}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="text-[13px] font-medium text-[var(--color-ink)] truncate">{name}</div>
                        <div className="text-[11px] text-[var(--color-muted)]">@{mu?.account || member.user_id.slice(0, 12) + '...'}</div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* Agent display settings */}
          <div className="pt-3">
            <label className="flex items-center justify-between cursor-pointer">
              <div className="flex items-center gap-2 flex-1 min-w-0">
                <EyeOff size={14} className="text-[var(--color-muted)] flex-shrink-0" />
                <div>
                  <div className="text-xs font-medium text-[var(--color-muted)]">{t('conversation.agentDisplay')}</div>
                  <div className="text-[10px] text-[var(--color-muted-soft)]">{t('conversation.agentDisplayHint')}</div>
                </div>
              </div>
              <button onClick={() => toggleConvSetting(convId, 'showAgentResponseOnly')}
                className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0 ml-3 ${showAgentResponseOnly ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-hairline)]'}`}>
                <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${showAgentResponseOnly ? 'left-[18px]' : 'left-0.5'}`} />
              </button>
            </label>
          </div>

          {/* Leave */}
          <div className="pt-3">
            <button onClick={handleLeave}
              className="w-full flex items-center justify-center gap-2 h-10 rounded-xl border border-[var(--destructive)]/20 text-sm text-[var(--destructive)] hover:bg-[var(--destructive)]/5 transition-colors font-medium">
              <LogOut size={16} /> {t('group.leave')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
