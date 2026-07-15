import { useState, useRef, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import { uiStore } from '@/stores/ui-store'
import { avatarUrl } from '@/lib/file'
import { fileService } from '@/services/file-service'
import { UserType } from '@/types/user'
import { X, Edit, LogOut, Settings, Bot, Camera, Smartphone, Copy, Check, Shield } from 'lucide-react'
import ProfileEditView from './profile-edit-view'

interface Props { onClose: () => void }

export default function ProfileView({ onClose }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const [showEdit, setShowEdit] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadingCover, setUploadingCover] = useState(false)
  const [copied, setCopied] = useState(false)
  const avatarInputRef = useRef<HTMLInputElement>(null)
  const coverInputRef = useRef<HTMLInputElement>(null)

  const handleAvatar = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]; if (!file) return
    setUploading(true)
    try { const r = await fileService.upload(file, file.name, 0); await authStore.updateProfile({ avatar: r.url }) } catch {}
    setUploading(false)
  }
  const handleCover = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]; if (!file) return
    setUploadingCover(true)
    try { const r = await fileService.upload(file, file.name, 0); await authStore.updateProfile({ cover: r.url }) } catch {}
    setUploadingCover(false)
  }
  const handleLogout = () => { authStore.logout(); onClose(); navigate('/login') }
  const copyId = () => { navigator.clipboard.writeText(user?.user_id || ''); setCopied(true); setTimeout(() => setCopied(false), 2000) }

  const initials = user?.name?.charAt(0)?.toUpperCase() || '?'
  const isAgent = user?.type === UserType.Agent

  return (
    <>
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[360px] bg-[var(--color-surface-card)] rounded-xl overflow-hidden"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Banner */}
        <div className="h-28 relative"
          style={{ background: user?.cover
            ? `url(${user.cover}?w=720&h=224) center/cover`
            : `linear-gradient(135deg, ${user?.primary_color || 'var(--color-primary)'}, ${user?.secondary_color || user?.primary_color || 'var(--color-muted)'})` }}>
          {user?.cover && <div className="absolute inset-0 bg-black/20" />}
          <button onClick={() => coverInputRef.current?.click()} disabled={uploadingCover}
            className="absolute top-3 left-3 p-1.5 rounded-xl bg-white/10 hover:bg-white/20 text-white/70 hover:text-white z-10 transition-colors">
            <Camera size={14} />
          </button>
          <input ref={coverInputRef} type="file" accept="image/*" onChange={handleCover} className="hidden" />
          <div className="absolute top-3 right-3 flex items-center gap-1 z-10">
            <button onClick={() => setShowEdit(true)} className="p-1.5 rounded-xl bg-white/20 hover:bg-white/30 text-white">
              <Edit size={15} />
            </button>
            <button onClick={onClose} className="p-1.5 rounded-xl bg-white/20 hover:bg-white/30 text-white">
              <X size={15} />
            </button>
          </div>
        </div>

        {/* Avatar — overlaps banner */}
        <div className="flex justify-center -mt-10 mb-3">
          <button onClick={() => avatarInputRef.current?.click()} disabled={uploading}
            className="relative group cursor-pointer">
            {user?.avatar ? (
              <img src={avatarUrl(user.avatar, 160)} alt="" className="w-20 h-20 rounded-full object-cover border-[3px] border-[var(--color-surface-card)] shadow-sm" />
            ) : (
              <div className="w-20 h-20 rounded-full flex items-center justify-center text-white text-2xl font-bold border-[3px] border-[var(--color-surface-card)] shadow-sm"
                style={{ background: `linear-gradient(135deg, ${user?.primary_color || 'var(--color-primary)'}, ${user?.secondary_color || user?.primary_color || 'var(--color-muted)'})` }}>
                {initials}
              </div>
            )}
            <div className="absolute inset-0 rounded-full bg-black/30 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
              <Camera size={16} className="text-white" />
            </div>
          </button>
          <input ref={avatarInputRef} type="file" accept="image/*" onChange={handleAvatar} className="hidden" />
        </div>

        {/* Name + Info */}
        <div className="text-center px-6 mb-4">
          <div className="font-headline text-xl font-semibold text-[var(--color-ink)] flex items-center justify-center gap-1.5">
            {user?.name || '—'}
            {isAgent && <span className="text-[9px] px-1.5 py-0.5 rounded-sm bg-purple-500/10 text-purple-600 font-medium uppercase tracking-wider">Agent</span>}
          </div>
          {user?.headline && <div className="text-sm text-[var(--color-muted)] mt-1">{user.headline}</div>}
          <div className="text-sm text-[var(--color-muted)]">@{user?.account || '—'}</div>
        </div>

        {/* ID */}
        <div className="px-6 mb-4">
          <div className="flex items-center justify-center gap-1 text-[11px] text-[var(--color-muted-soft)] font-mono select-all">
            {user?.user_id}
            <button onClick={copyId} className="hover:text-[var(--color-ink)]">
              {copied ? <Check size={11} className="text-[var(--success)]" /> : <Copy size={11} />}
            </button>
          </div>
        </div>

        {/* Actions */}
        <div className="px-4 pb-4 space-y-0.5">
          <div className="border-t border-[var(--color-hairline)] pt-3 space-y-0.5">
            {[
              { icon: Bot, label: t('profile.agentMgmt'), sheet: 'agents' },
              { icon: Shield, label: t('profile.userSettings'), sheet: 'userSettings' },
              { icon: Smartphone, label: t('profile.deviceMgmt'), sheet: 'sessions' },
              { icon: Settings, label: t('profile.appSettings'), sheet: 'settings' },
            ].map(({ icon: Icon, label, sheet }) => (
              <button key={sheet}
                onClick={() => { onClose(); setTimeout(() => uiStore.openSheet(sheet), 50) }}
                className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-sm text-[var(--color-body)] hover:text-[var(--color-ink)] transition-colors">
                <Icon size={18} /> {label}
              </button>
            ))}
            <button onClick={handleLogout}
              className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-[var(--destructive)]/10 text-sm text-[var(--destructive)] transition-colors">
              <LogOut size={18} /> {t('profile.logout')}
            </button>
          </div>
        </div>
      </div>
    </div>
    {showEdit && <ProfileEditView onClose={() => setShowEdit(false)} />}
    </>
  )
}
