import { useEffect, useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { conversationService } from '@/services/conversation-service'
import { userService } from '@/services/user-service'
import { fileService } from '@/services/file-service'
import { useTranslation } from 'react-i18next'
import { avatarUrl } from '@/lib/file'
import { authStore } from '@/stores/auth-store'
import type { ConversationDetail, JoinRequest } from '@/types/conversation'
import type { User } from '@/types/user'
import { ConvRole } from '@/types/conversation'
import { X, Crown, Shield, Trash2, Check, X as XIcon, Camera, LogOut, Search, Cpu, UserPlus, Bell, Edit2, Copy } from 'lucide-react'

interface Props { convId: string; onClose: () => void }

const inputSm = 'w-full h-9 px-3 rounded-lg bg-[var(--color-surface-card)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-1 focus:ring-[var(--color-primary)]'

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
  const [tab, setTab] = useState<'members' | 'notice' | 'requests'>('members')
  const avatarInputRef = useRef<HTMLInputElement>(null)
  const coverInputRef = useRef<HTMLInputElement>(null)
  const [uploadingCover, setUploadingCover] = useState(false)
  const navigate = useNavigate()
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
  const user = (id: string) => userMap[id]
  const userName = (id: string) => userMap[id]?.name || id

  const handleRemove = async (userId: string) => {
    if (!confirm(t('group.removeConfirm'))) return
    try { await conversationService.removeMember(convId, userId); setDetail({ ...detail, members: detail.members.filter(m => m.user_id !== userId) }) } catch {}
  }
  const handleApprove = async (userId: string) => {
    await conversationService.approveJoinRequest(convId, userId); setJoinRequests(prev => prev.filter(r => r.user_id !== userId))
  }
  const handleReject = async (userId: string) => {
    await conversationService.rejectJoinRequest(convId, userId); setJoinRequests(prev => prev.filter(r => r.user_id !== userId))
  }
  const handleLeave = async () => {
    if (!confirm(t('group.leaveConfirm'))) return
    try { await conversationService.leave(convId); onClose() } catch {}
  }
  const handleSaveName = async () => {
    const t = editName.trim()
    if (!t) return
    try { await conversationService.updateGroup(convId, { name: t }); setDetail({ ...detail, name: t }) } catch {}
    setEditingName(false)
  }
  const handleSaveNotice = async () => {
    const t = editNotice.trim()
    try { await conversationService.updateGroup(convId, { notice: t }); setDetail({ ...detail, notice: t }) } catch {}
    setEditingNotice(false)
  }
  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try { const r = await fileService.upload(file, file.name, 0); await conversationService.updateGroup(convId, { avatar: r.url }); setDetail({ ...detail!, avatar: r.url }) } catch {}
    setUploading(false)
  }
  const handleCoverUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploadingCover(true)
    try { const r = await fileService.upload(file, file.name, 0); await conversationService.updateGroup(convId, { cover: r.url }); setDetail({ ...detail!, cover: r.url }) } catch {}
    setUploadingCover(false)
  }
  const handleAddMemberSearch = async () => {
    if (!addQuery.trim()) return
    try { const users = await userService.search(addQuery.trim()); setAddResults(users.filter(u => !detail.members.some(m => m.user_id === u.user_id))) } catch {}
  }
  const handleAddMember = async (userId: string) => {
    try {
      await conversationService.addMembers(convId, [userId])
      const u = addResults.find(x => x.user_id === userId)
      if (u) setUserMap(prev => ({ ...prev, [userId]: u }))
      setDetail(d => d ? { ...d, members: [...d.members, { conv_id: convId, user_id: userId, role: ConvRole.Member, mute: false, joined_at: Date.now(), user_type: u?.type || 0, wake_mode: u?.wake_mode || 0 }] } : null)
      setAddResults(prev => prev.filter(x => x.user_id !== userId))
    } catch {}
  }

  const tabBtn = (t: typeof tab, label: string, badge?: number) => (
    <button onClick={() => setTab(t)} className={`flex-1 h-9 rounded-lg text-xs font-medium transition-colors ${tab === t ? 'bg-[var(--color-primary)] text-white' : 'bg-[var(--color-surface-soft)] text-[var(--color-body)] hover:bg-[var(--color-hairline)]'}`}>
      {label}{badge ? <span className="ml-1 opacity-70">{badge}</span> : ''}
    </button>
  )

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[420px] max-h-[580px] bg-[var(--color-surface-card)] rounded-lg overflow-hidden flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Header with cover as background */}
        <div className="h-28 flex items-end justify-between px-6 pb-5 relative"
          style={{ background: detail.cover ? `url(${detail.cover}?w=840&h=224) center/cover` : `linear-gradient(135deg, var(--color-primary), var(--color-muted))` }}>
          {detail.cover && <div className="absolute inset-0 bg-black/20" />}
          {isAdmin && (
            <button onClick={() => coverInputRef.current?.click()} disabled={uploadingCover}
              className="absolute top-3 left-3 p-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white z-10 transition-colors">
              <Camera size={14} />
            </button>
          )}
          <input ref={coverInputRef} type="file" accept="image/*" onChange={handleCoverUpload} className="hidden" />
          <div />
          <div className="flex items-center gap-1 relative z-10">
            <button onClick={onClose} className="p-1.5 rounded-lg bg-white/20 hover:bg-white/30 text-white"><X size={15} /></button>
          </div>
        </div>

        {/* Avatar — overlaps header */}
        <div className="flex justify-center -mt-10 mb-4">
          <button onClick={() => isAdmin ? avatarInputRef.current?.click() : undefined} disabled={uploading}
            className="relative group cursor-pointer disabled:opacity-50">
            {detail.avatar ? (
              <img src={avatarUrl(detail.avatar)} alt="" className="w-[72px] h-[72px] rounded-full object-cover border-4 border-[var(--color-surface-card)]" />
            ) : (
              <div className="w-[72px] h-[72px] rounded-full flex items-center justify-center text-white text-2xl font-bold border-4 border-[var(--color-surface-card)]"
                style={{ background: 'linear-gradient(135deg, var(--color-accent), #34D399)' }}>{detail.name?.charAt(0)?.toUpperCase() || 'G'}</div>
            )}
            {isAdmin && <div className="absolute inset-0 rounded-full bg-black/30 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity"><Camera size={18} className="text-white" /></div>}
          </button>
          <input ref={avatarInputRef} type="file" accept="image/*" onChange={handleAvatarUpload} className="hidden" />
        </div>

        {/* Info */}
        <div className="px-6 pb-4">
          <div className="text-center space-y-1 mb-4">
            {editingName ? (
              <div className="flex items-center justify-center gap-2">
                <input value={editName} onChange={e => setEditName(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') handleSaveName(); if (e.key === 'Escape') setEditingName(false) }} autoFocus className="h-8 px-2 rounded bg-[var(--color-surface-soft)] text-sm font-semibold border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] w-[160px] text-center" />
                <button onClick={handleSaveName} className="text-[11px] text-[var(--color-accent)] font-medium flex-shrink-0">保存</button>
              </div>
            ) : (
              <button onClick={() => { if (isAdmin) { setEditName(detail.name); setEditingName(true) } }}
                className={`font-headline text-lg font-semibold ${isAdmin ? 'hover:text-[var(--color-accent)] cursor-pointer' : 'cursor-default text-[var(--color-ink)]'}`}>{detail.name}</button>
            )}
            <div className="text-sm text-[var(--color-muted)]">{t('group.memberCount', { count: detail.members.length })}</div>
          </div>

          {/* Tab bar */}
          <div className="flex gap-1.5 border-t border-[var(--color-hairline)] pt-4 mb-3">
            {tabBtn('members', t('group.members'))}
            {tabBtn('notice', t('group.notice'))}
            {isAdmin && joinRequests.length > 0 && tabBtn('requests', t('group.requests'), joinRequests.length)}
          </div>
        </div>

        {/* Tab content */}
        <div className="flex-1 overflow-y-auto px-6 pb-4">
          {tab === 'members' && (
            <div className="space-y-3">
              {/* Add member */}
              {isAdmin && (
                <div>
                  {!showAddMember ? (
                    <button onClick={() => setShowAddMember(true)}
                      className="w-full h-9 rounded-lg border border-dashed border-[var(--color-hairline)] text-xs text-[var(--color-muted)] hover:border-[var(--color-primary)] hover:text-[var(--color-primary)] transition-colors flex items-center justify-center gap-1.5">
                      <UserPlus size={13} /> 添加成员
                    </button>
                  ) : (
                    <div className="space-y-2 bg-[var(--color-surface-soft)] rounded-lg p-3">
                      <div className="flex gap-2">
                        <input type="text" value={addQuery} onChange={e => setAddQuery(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleAddMemberSearch() } }}
                          placeholder={t('group.searchMember')} autoFocus className={inputSm} />
                        <button onClick={handleAddMemberSearch} className="px-3 h-9 rounded-lg bg-[var(--color-primary)] text-white text-xs flex-shrink-0">搜索</button>
                        <button onClick={() => setShowAddMember(false)} className="px-2 h-9 text-xs text-[var(--color-muted)] flex-shrink-0">取消</button>
                      </div>
                      {addResults.length > 0 && (
                        <div className="max-h-[120px] overflow-y-auto space-y-0.5">
                          {addResults.map(u => (
                            <div key={u.user_id} className="flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-[var(--color-surface-card)]"><span className="text-xs flex-1">{u.name}</span><button onClick={() => handleAddMember(u.user_id)} className="px-2 py-0.5 rounded bg-[var(--color-accent)] text-white text-[10px]">添加</button></div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}

              {/* Member list */}
              {detail.members.map(member => {
                const name = userName(member.user_id)
                const avatar = userMap[member.user_id]?.avatar
                return (
                  <div key={member.user_id} className="flex items-center gap-3 py-2 -mx-1 px-1 rounded-lg hover:bg-[var(--color-surface-soft)] group">
                    <div className="relative flex-shrink-0">
                      {avatar ? (
                        <img src={avatarUrl(avatar)} alt="" className="w-9 h-9 rounded-full object-cover" />
                      ) : (
                        <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-xs font-semibold"
                          style={{ background: member.user_type === 1 ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                          {name.charAt(0).toUpperCase()}
                        </div>
                      )}
                      {member.user_type === 1 && <span className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-purple-500 flex items-center justify-center border-2 border-[var(--color-surface-card)]"><Cpu size={7} className="text-white" /></span>}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="text-[13px] font-medium text-[var(--color-ink)] flex items-center gap-1.5">
                        <span className="truncate">{member.nickname || name}</span>
                        {member.role === ConvRole.Owner && <Crown size={11} className="text-[var(--warning)] flex-shrink-0" />}
                        {member.role === ConvRole.Admin && <Shield size={11} className="text-[var(--info)] flex-shrink-0" />}
                        {member.user_id === currentUserId && <span className="text-[10px] text-[var(--color-muted)]">(我)</span>}
                      </div>
                      <div className="text-[11px] text-[var(--color-muted)]">@{userMap[member.user_id]?.account || member.user_id.slice(0, 12) + '...'}</div>
                    </div>
                    {isAdmin && member.user_id !== currentUserId && member.role !== ConvRole.Owner && (
                      <button onClick={() => handleRemove(member.user_id)} className="p-1 rounded opacity-0 group-hover:opacity-100 text-[var(--destructive)] transition-all"><Trash2 size={14} /></button>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {tab === 'notice' && (
            <div>
              {editingNotice ? (
                <div className="space-y-3">
                  <textarea value={editNotice} onChange={e => setEditNotice(e.target.value)} autoFocus rows={4} placeholder={t('group.noticePlaceholder')}
                    className="w-full resize-none rounded-lg bg-[var(--color-surface-card)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-1 focus:ring-[var(--color-primary)] px-3 py-2" />
                  <div className="flex gap-2">
                    <button onClick={handleSaveNotice} className="px-4 h-9 rounded-lg bg-[var(--color-primary)] text-white text-sm">保存</button>
                    <button onClick={() => setEditingNotice(false)} className="px-4 h-9 rounded-lg border border-[var(--color-hairline)] text-sm text-[var(--color-muted)]">取消</button>
                  </div>
                </div>
              ) : (
                <div className={`rounded-lg p-4 text-sm ${detail.notice ? 'bg-[var(--color-warning)]/5 border border-[var(--color-warning)]/10' : 'bg-[var(--color-surface-soft)]'}`}>
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs font-medium text-[var(--color-muted)] uppercase tracking-wider flex items-center gap-1.5"><Bell size={12} /> 群公告</span>
                    {isOwner && <button onClick={() => { setEditNotice(detail.notice || ''); setEditingNotice(true) }} className="text-[11px] text-[var(--color-muted)] hover:text-[var(--color-accent)] flex items-center gap-1"><Edit2 size={11} />{detail.notice ? t('group.noticeEdit') : t('group.noticeAdd')}</button>}
                  </div>
                  {detail.notice ? <p className="text-[var(--color-body)] leading-relaxed whitespace-pre-wrap">{detail.notice}</p> : <p className="text-[var(--color-muted)] italic text-xs">暂无公告，群主可在此添加</p>}
                </div>
              )}
            </div>
          )}

          {tab === 'requests' && joinRequests.length > 0 && (
            <div className="space-y-1">
              {joinRequests.map(req => (
                <div key={req.user_id} className="flex items-center gap-3 py-2 px-1 rounded-lg">
                  <div className="w-8 h-8 rounded-full bg-[var(--color-muted)]/20 flex items-center justify-center text-xs font-semibold flex-shrink-0">
                    {userName(req.user_id).charAt(0).toUpperCase()}
                  </div>
                  <div className="flex-1 min-w-0 text-[13px] font-medium text-[var(--color-ink)]">{userName(req.user_id)}</div>
                  <button onClick={() => handleApprove(req.user_id)} className="p-1.5 rounded-lg hover:bg-[var(--success)]/10 text-[var(--success)]"><Check size={15} /></button>
                  <button onClick={() => handleReject(req.user_id)} className="p-1.5 rounded-lg hover:bg-[var(--destructive)]/10 text-[var(--destructive)]"><XIcon size={15} /></button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 pb-5 space-y-2">
          {isOwner && (
            <button onClick={async () => {
              if (!confirm(t('group.cloneConfirm'))) return
              try {
                const r = await conversationService.clone(convId)
                onClose()
                navigate(`/chat/${r.conv_id}`)
              } catch {}
            }}
              className="w-full h-10 rounded-lg border border-dashed border-[var(--color-hairline)] text-sm text-[var(--color-muted)] hover:border-[var(--color-primary)] hover:text-[var(--color-primary)] transition-colors flex items-center justify-center gap-2">
              <Copy size={14} /> 克隆群组
            </button>
          )}
          <button onClick={handleLeave}
            className="w-full h-10 rounded-lg border border-[var(--destructive)]/20 text-sm text-[var(--destructive)] hover:bg-[var(--destructive)]/5 transition-colors flex items-center justify-center gap-2">
            <LogOut size={14} /> 退出群组
          </button>
        </div>
      </div>
    </div>
  )
}
