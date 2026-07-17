import { useEffect, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { conversationService } from '@/services/conversation-service'
import { conversationStore } from '@/stores/conversation-store'
import { userService } from '@/services/user-service'
import { fileService } from '@/services/file-service'
import { avatarUrl } from '@/lib/file'
import { authStore } from '@/stores/auth-store'
import type { ConversationDetail, JoinRequest } from '@/types/conversation'
import type { User } from '@/types/user'
import { ConvRole } from '@/types/conversation'
import { X, Check, X as XIcon, Camera, Bell, Edit2 } from 'lucide-react'
import GroupEditView from './group-edit-view'

interface Props { convId: string; onClose: () => void }

export default function GroupBasicInfo({ convId, onClose }: Props) {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<ConversationDetail | null>(null)
  const [joinRequests, setJoinRequests] = useState<JoinRequest[]>([])
  const [userMap, setUserMap] = useState<Record<string, User>>({})
  const [uploading, setUploading] = useState(false)
  const [showEdit, setShowEdit] = useState(false)
  const avatarInputRef = useRef<HTMLInputElement>(null)
  const coverInputRef = useRef<HTMLInputElement>(null)
  const [uploadingCover, setUploadingCover] = useState(false)
  const currentUserId = authStore.state.user?.user_id || ''

  useEffect(() => {
    conversationService.getDetail(convId).then(async d => {
      setDetail(d)
      const ids = d.members.map(m => m.user_id)
      if (ids.length > 0) {
        try { const users = await userService.batchGet(ids); setUserMap(users) } catch {}
      }
    }).catch(() => {})
    conversationService.listJoinRequests(convId).then(async reqs => {
      setJoinRequests(reqs)
      const ids = reqs.map(r => r.user_id)
      if (ids.length > 0) {
        try { const users = await userService.batchGet(ids); setUserMap(prev => ({ ...prev, ...users })) } catch {}
      }
    }).catch(() => {})
  }, [convId])

  if (!detail) return null

  const me = detail.members.find(m => m.user_id === currentUserId)
  const isAdmin = me?.role === ConvRole.Admin || me?.role === ConvRole.Owner
  const isOwner = me?.role === ConvRole.Owner
  const uname = (id: string) => userMap[id]?.name || id

  const actions = {
    removeMember: async (userId: string) => {
      if (!confirm(t('group.removeConfirm'))) return
      try { await conversationService.removeMember(convId, userId); setDetail({ ...detail, members: detail.members.filter(m => m.user_id !== userId) }) } catch {}
    },
    approve: async (userId: string) => {
      await conversationService.approveJoinRequest(convId, userId); setJoinRequests(prev => prev.filter(r => r.user_id !== userId))
    },
    reject: async (userId: string) => {
      await conversationService.rejectJoinRequest(convId, userId); setJoinRequests(prev => prev.filter(r => r.user_id !== userId))
    },
    uploadAvatar: async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]; if (!file) return; setUploading(true)
      try { const r = await fileService.upload(file, file.name, 0); await conversationService.updateGroup(convId, { avatar: r.url }); setDetail({ ...detail!, avatar: r.url }) } catch {}
      setUploading(false)
    },
    uploadCover: async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]; if (!file) return; setUploadingCover(true)
      try { const r = await fileService.upload(file, file.name, 0); await conversationService.updateGroup(convId, { cover: r.url }); setDetail({ ...detail!, cover: r.url }) } catch {}
      setUploadingCover(false)
    },
  }

  return (
    <>
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[420px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] bg-[var(--color-surface-card)] rounded-none sm:rounded-xl overflow-hidden flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Banner */}
        <div className="h-28 relative"
          style={{ background: detail.cover ? `url(${detail.cover}?w=840&h=224) center/cover` : `linear-gradient(135deg, var(--color-primary), var(--color-muted))` }}>
          {detail.cover && <div className="absolute inset-0 bg-black/20" />}
          {isAdmin && (
            <button onClick={() => coverInputRef.current?.click()} disabled={uploadingCover}
              className="absolute top-3 left-3 p-1.5 rounded-xl bg-white/10 hover:bg-white/20 text-white/70 hover:text-white z-10">
              <Camera size={14} />
            </button>
          )}
          <input ref={coverInputRef} type="file" accept="image/*" onChange={actions.uploadCover} className="hidden" />
          <div className="absolute top-3 right-3 flex items-center gap-1 z-10">
            {isAdmin && (
              <button onClick={() => setShowEdit(true)} className="p-1.5 rounded-xl bg-white/20 hover:bg-white/30 text-white">
                <Edit2 size={15} />
              </button>
            )}
            <button onClick={onClose} className="p-1.5 rounded-xl bg-white/20 hover:bg-white/30 text-white">
              <X size={15} />
            </button>
          </div>
        </div>

        {/* Avatar — overlaps banner */}
        <div className="flex justify-center -mt-10 mb-3">
          <button onClick={() => isAdmin ? avatarInputRef.current?.click() : undefined} disabled={uploading}
            className="relative group cursor-pointer">
            {detail.avatar ? (
              <img src={avatarUrl(detail.avatar, 160)} alt="" className="w-20 h-20 rounded-full object-cover border-[3px] border-[var(--color-surface-card)] shadow-sm" />
            ) : (
              <div className="w-20 h-20 rounded-full flex items-center justify-center text-white text-2xl font-bold border-[3px] border-[var(--color-surface-card)] shadow-sm"
                style={{ background: 'linear-gradient(135deg, var(--color-accent), #34D399)' }}>{detail.name?.charAt(0)?.toUpperCase() || 'G'}</div>
            )}
            {isAdmin && <div className="absolute inset-0 rounded-full bg-black/30 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity"><Camera size={16} className="text-white" /></div>}
          </button>
          <input ref={avatarInputRef} type="file" accept="image/*" onChange={actions.uploadAvatar} className="hidden" />
        </div>

        {/* Name + Headline */}
        <div className="text-center px-6 mb-4">
          <div className="font-headline text-xl font-semibold text-[var(--color-ink)]">
            {detail.name}
          </div>
          {detail.headline && <div className="text-sm text-[var(--color-muted)] mt-1">{detail.headline}</div>}
          <div className="text-sm text-[var(--color-muted)] mt-0.5">{t('group.memberCount', { count: detail.members.length })}</div>
        </div>

        {/* Content area */}
        <div className="flex-1 overflow-y-auto px-6 pb-4">
          {/* Notice */}
          <div className="border-t border-[var(--color-hairline)] pt-3 mt-3">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-[var(--color-muted)] uppercase tracking-wider flex items-center gap-1.5">
                <Bell size={12} /> {t('group.notice')}
              </span>
            </div>
            <div className={`rounded-xl p-3 text-sm ${detail.notice ? 'bg-[var(--color-warning)]/5' : 'bg-[var(--color-surface-soft)]'}`}>
              {detail.notice ? <p className="text-[var(--color-body)] leading-relaxed whitespace-pre-wrap">{detail.notice}</p> : <p className="text-[var(--color-muted)] italic text-xs">{t('group.noticeEmpty')}</p>}
            </div>
          </div>


          {/* Join Requests */}
          {isAdmin && joinRequests.length > 0 && (
            <div className="border-t border-[var(--color-hairline)] pt-3 mt-3">
              <span className="text-xs font-medium text-[var(--color-muted)] uppercase tracking-wider">{t('group.requests')} ({joinRequests.length})</span>
              <div className="space-y-1 mt-2">
                {joinRequests.map(req => (
                  <div key={req.user_id} className="flex items-center gap-3 py-1.5 px-1 rounded-xl">
                    <div className="w-8 h-8 rounded-full bg-[var(--color-muted)]/20 flex items-center justify-center text-xs font-semibold flex-shrink-0">
                      {uname(req.user_id).charAt(0).toUpperCase()}
                    </div>
                    <div className="flex-1 min-w-0 text-[13px] font-medium text-[var(--color-ink)]">{uname(req.user_id)}</div>
                    <button onClick={() => actions.approve(req.user_id)} className="p-1.5 rounded-xl hover:bg-[var(--success)]/10 text-[var(--success)]"><Check size={15} /></button>
                    <button onClick={() => actions.reject(req.user_id)} className="p-1.5 rounded-xl hover:bg-[var(--destructive)]/10 text-[var(--destructive)]"><XIcon size={15} /></button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
    {showEdit && (
      <GroupEditView
        convId={convId}
        name={detail.name}
        headline={detail.headline || ''}
        notice={detail.notice || ''}
        onClose={() => setShowEdit(false)}
        onSaved={data => setDetail(prev => prev ? { ...prev, name: data.name, headline: data.headline, notice: data.notice } : null)}
      />
    )}
    </>
  )
}
