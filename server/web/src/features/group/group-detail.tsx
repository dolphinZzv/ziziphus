import { useEffect, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { conversationService } from '@/services/conversation-service'
import { conversationStore } from '@/stores/conversation-store'
import { getConvSettings, toggleConvSetting, subscribe as settingsSubscribe } from '@/stores/conversation-settings-store'
import { userService } from '@/services/user-service'
import { fileService } from '@/services/file-service'
import { avatarUrl } from '@/lib/file'
import { authStore } from '@/stores/auth-store'
import type { ConversationDetail, JoinRequest } from '@/types/conversation'
import type { User } from '@/types/user'
import { ConvRole } from '@/types/conversation'
import { X, Crown, Shield, Trash2, Check, X as XIcon, Camera, Search, Cpu, UserPlus, Bell, Edit2, EyeOff, FileUp } from 'lucide-react'

interface Props { convId: string; onClose: () => void }

export default function GroupDetail({ convId, onClose }: Props) {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<ConversationDetail | null>(null)
  const [joinRequests, setJoinRequests] = useState<JoinRequest[]>([])
  const [userMap, setUserMap] = useState<Record<string, User>>({})
  const [uploading, setUploading] = useState(false)
  const [editingName, setEditingName] = useState(false)
  const [editName, setEditName] = useState('')
  const [editingNotice, setEditingNotice] = useState(false)
  const [editNotice, setEditNotice] = useState('')
  const [showAddMember, setShowAddMember] = useState(false)
  const [addQuery, setAddQuery] = useState('')
  const [addResults, setAddResults] = useState<User[]>([])
  const [showNotice, setShowNotice] = useState(false)
  const avatarInputRef = useRef<HTMLInputElement>(null)
  const coverInputRef = useRef<HTMLInputElement>(null)
  const [uploadingCover, setUploadingCover] = useState(false)
  const currentUserId = authStore.state.user?.user_id || ''
  const [showAgentResponseOnly, setShowAgentResponseOnly] = useState(
    () => getConvSettings(convId).showAgentResponseOnly
  )
  useEffect(() => {
    return settingsSubscribe(() => {
      setShowAgentResponseOnly(getConvSettings(convId).showAgentResponseOnly)
    })
  }, [convId])
  const [fileChangeNotify, setFileChangeNotify] = useState(false)
  useEffect(() => {
    conversationService.getSettings(convId).then(res => {
      if (res.settings?.fileChangeNotify) setFileChangeNotify(true)
    }).catch(() => {})
  }, [convId])
  const handleFileChangeNotifyToggle = async () => {
    const newVal = !fileChangeNotify
    setFileChangeNotify(newVal)
    try {
      await conversationService.updateSettings(convId, { fileChangeNotify: newVal })
    } catch {
      setFileChangeNotify(!newVal)
    }
  }

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
  const u = (id: string) => userMap[id]
  const uname = (id: string) => userMap[id]?.name || id

  const actions = {
    remove: async (userId: string) => {
      if (!confirm(t('group.removeConfirm'))) return
      try { await conversationService.removeMember(convId, userId); setDetail({ ...detail, members: detail.members.filter(m => m.user_id !== userId) }) } catch {}
    },
    approve: async (userId: string) => {
      await conversationService.approveJoinRequest(convId, userId); setJoinRequests(prev => prev.filter(r => r.user_id !== userId))
    },
    reject: async (userId: string) => {
      await conversationService.rejectJoinRequest(convId, userId); setJoinRequests(prev => prev.filter(r => r.user_id !== userId))
    },
    leave: async () => {
      if (!confirm(t('group.leaveConfirm'))) return
      try { await conversationService.leave(convId); conversationStore.removeConversation(convId); onClose() } catch {}
    },
    saveName: async () => {
      const v = editName.trim()
      if (!v) return
      try { await conversationService.updateGroup(convId, { name: v }); setDetail({ ...detail, name: v }) } catch {}
      setEditingName(false)
    },
    saveNotice: async () => {
      const v = editNotice.trim()
      try { await conversationService.updateGroup(convId, { notice: v }); setDetail({ ...detail, notice: v }) } catch {}
      setEditingNotice(false)
    },
    avatar: async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]; if (!file) return; setUploading(true)
      try { const r = await fileService.upload(file, file.name, 0); await conversationService.updateGroup(convId, { avatar: r.url }); setDetail({ ...detail!, avatar: r.url }) } catch {}
      setUploading(false)
    },
    cover: async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]; if (!file) return; setUploadingCover(true)
      try { const r = await fileService.upload(file, file.name, 0); await conversationService.updateGroup(convId, { cover: r.url }); setDetail({ ...detail!, cover: r.url }) } catch {}
      setUploadingCover(false)
    },
    searchMember: async () => {
      if (!addQuery.trim()) return
      try { const users = await userService.search(addQuery.trim()); setAddResults(users.filter(x => !detail.members.some(m => m.user_id === x.user_id))) } catch {}
    },
    addMember: async (userId: string) => {
      try {
        await conversationService.addMembers(convId, [userId])
        const x = addResults.find(x => x.user_id === userId)
        if (x) setUserMap(prev => ({ ...prev, [userId]: x }))
        setDetail(d => d ? { ...d, members: [...d.members, { conv_id: convId, user_id: userId, role: ConvRole.Member, mute: false, joined_at: Date.now(), user_type: x?.type || 0, wake_mode: x?.wake_mode || 0 }] } : null)
        setAddResults(prev => prev.filter(x => x.user_id !== userId))
      } catch {}
    },
  }

  const inputSm = 'w-full h-9 px-3 rounded-xl bg-[var(--color-surface-card)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-1 focus:ring-[var(--color-primary)]'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[420px] max-h-[580px] bg-[var(--color-surface-card)] rounded-xl overflow-hidden flex flex-col"
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
          <input ref={coverInputRef} type="file" accept="image/*" onChange={actions.cover} className="hidden" />
          <button onClick={onClose} className="absolute top-3 right-3 p-1.5 rounded-xl bg-white/20 hover:bg-white/30 text-white z-10"><X size={15} /></button>
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
          <input ref={avatarInputRef} type="file" accept="image/*" onChange={actions.avatar} className="hidden" />
        </div>

        {/* Name */}
        <div className="text-center px-6 mb-4">
          {editingName ? (
            <div className="flex items-center justify-center gap-2">
              <input value={editName} onChange={e => setEditName(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter') actions.saveName(); if (e.key === 'Escape') setEditingName(false) }}
                autoFocus className="h-8 px-2 rounded bg-[var(--color-surface-soft)] text-sm font-semibold border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] w-[160px] text-center" />
              <button onClick={actions.saveName} className="text-[11px] text-[var(--color-accent)] font-medium">{t('common.save')}</button>
            </div>
          ) : (
            <button onClick={() => { if (isAdmin) { setEditName(detail.name); setEditingName(true) } }}
              className={`font-headline text-xl font-semibold ${isAdmin ? 'hover:text-[var(--color-accent)] cursor-pointer' : 'cursor-default text-[var(--color-ink)]'}`}>{detail.name}</button>
          )}
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
              {isOwner && (
                <button onClick={() => { setEditNotice(detail.notice || ''); setEditingNotice(true); setShowNotice(true) }}
                  className="text-[11px] text-[var(--color-muted)] hover:text-[var(--color-accent)] flex items-center gap-1">
                  <Edit2 size={11} />{detail.notice ? t('group.noticeEdit') : t('group.noticeAdd')}
                </button>
              )}
            </div>
            {editingNotice && (
              <div className="space-y-2 mb-2">
                <textarea value={editNotice} onChange={e => setEditNotice(e.target.value)} autoFocus rows={3} placeholder={t('group.noticePlaceholder')}
                  className="w-full resize-none rounded-xl bg-[var(--color-surface-card)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-1 focus:ring-[var(--color-primary)] px-3 py-2" />
                <div className="flex gap-2">
                  <button onClick={actions.saveNotice} className="px-3 h-8 rounded-xl bg-[var(--color-primary)] text-white text-sm">{t('common.save')}</button>
                  <button onClick={() => setEditingNotice(false)} className="px-3 h-8 rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-muted)]">{t('common.cancel')}</button>
                </div>
              </div>
            )}
            {!editingNotice && (
              <div className={`rounded-xl p-3 text-sm ${detail.notice ? 'bg-[var(--color-warning)]/5 border border-[var(--color-warning)]/10' : 'bg-[var(--color-surface-soft)]'}`}>
                {detail.notice ? <p className="text-[var(--color-body)] leading-relaxed whitespace-pre-wrap">{detail.notice}</p> : <p className="text-[var(--color-muted)] italic text-xs">{t('group.noticeEmpty')}</p>}
              </div>
            )}
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

          {/* Agent display settings */}
          <div className="border-t border-[var(--color-hairline)] pt-3 mt-3">
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

          {/* File change notification settings */}
          <div className="border-t border-[var(--color-hairline)] pt-3 mt-3">
            <label className="flex items-center justify-between cursor-pointer">
              <div className="flex items-center gap-2 flex-1 min-w-0">
                <FileUp size={14} className="text-[var(--color-muted)] flex-shrink-0" />
                <div>
                  <div className="text-xs font-medium text-[var(--color-muted)]">{t('conversation.fileChangeNotify')}</div>
                  <div className="text-[10px] text-[var(--color-muted-soft)]">{t('conversation.fileChangeNotifyHint')}</div>
                </div>
              </div>
              <button onClick={handleFileChangeNotifyToggle}
                className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0 ml-3 ${fileChangeNotify ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-hairline)]'}`}>
                <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${fileChangeNotify ? 'left-[18px]' : 'left-0.5'}`} />
              </button>
            </label>
          </div>
        </div>
      </div>
    </div>
  )
}
