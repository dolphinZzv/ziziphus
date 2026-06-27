import { useState, useRef, useSyncExternalStore } from 'react'
import { useNavigate } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import { uiStore } from '@/stores/ui-store'
import { avatarUrl } from '@/lib/file'
import { fileService } from '@/services/file-service'
import { UserType } from '@/types/user'
import { X, Edit, LogOut, Settings, Bot, Camera, Smartphone, Copy, Check } from 'lucide-react'

interface Props { onClose: () => void }

export default function ProfileView({ onClose }: Props) {
  const navigate = useNavigate()
  const user = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(user?.name || '')
  const [primaryColor, setPrimaryColor] = useState(user?.primary_color || '#0F172A')
  const [secondaryColor, setSecondaryColor] = useState(user?.secondary_color || '#64748B')
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [copied, setCopied] = useState(false)
  const avatarInputRef = useRef<HTMLInputElement>(null)

  const handleSave = async () => {
    setSaving(true)
    try { await authStore.updateProfile({ name, primary_color: primaryColor, secondary_color: secondaryColor }); setEditing(false) } catch {}
    setSaving(false)
  }
  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try { const r = await fileService.upload(file, file.name, 0); await authStore.updateProfile({ avatar: r.url }) } catch {}
    setUploading(false)
  }
  const handleLogout = () => { authStore.logout(); onClose(); navigate('/login') }
  const copyId = () => { navigator.clipboard.writeText(user?.user_id || ''); setCopied(true); setTimeout(() => setCopied(false), 2000) }

  const initials = user?.name?.charAt(0)?.toUpperCase() || '?'
  const isAgent = user?.type === UserType.Agent

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[360px] bg-[var(--color-surface-card)] rounded-lg overflow-hidden"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Gradient header */}
        <div className="h-28 flex items-end justify-between px-6 pb-5"
          style={{ background: user?.primary_color
            ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})`
            : 'var(--color-primary)' }}>
          <div />
          <div className="flex items-center gap-1">
            <button onClick={() => setEditing(!editing)} className="p-1.5 rounded-lg bg-white/20 hover:bg-white/30 text-white"><Edit size={15} /></button>
            <button onClick={onClose} className="p-1.5 rounded-lg bg-white/20 hover:bg-white/30 text-white"><X size={15} /></button>
          </div>
        </div>

        {/* Avatar — overlaps gradient */}
        <div className="flex justify-center -mt-10 mb-4">
          <button onClick={() => avatarInputRef.current?.click()} disabled={uploading}
            className="relative group cursor-pointer disabled:opacity-50">
            {user?.avatar ? (
              <img src={avatarUrl(user.avatar)} alt="" className="w-[72px] h-[72px] rounded-full object-cover border-4 border-[var(--color-surface-card)]" />
            ) : (
              <div className="w-[72px] h-[72px] rounded-full flex items-center justify-center text-white text-2xl font-bold border-4 border-[var(--color-surface-card)]"
                style={{ background: user?.primary_color ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})` : 'var(--color-primary)' }}>
                {initials}
              </div>
            )}
            <div className="absolute inset-0 rounded-full bg-black/30 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
              <Camera size={18} className="text-white" />
            </div>
          </button>
          <input ref={avatarInputRef} type="file" accept="image/*" onChange={handleAvatarUpload} className="hidden" />
        </div>

        {/* Info */}
        <div className="px-6 pb-6">
          {!editing ? (
            <div className="text-center space-y-1 mb-5">
              <div className="font-headline text-lg font-semibold text-[var(--color-ink)] flex items-center justify-center gap-1.5">
                {user?.name || '—'}
                {isAgent && <span className="text-[9px] px-1.5 py-0.5 rounded-sm bg-purple-500/10 text-purple-600 font-medium uppercase tracking-wider">Agent</span>}
              </div>
              <div className="text-sm text-[var(--color-muted)]">@{user?.account || '—'}</div>
              <div className="flex items-center justify-center gap-1 text-[11px] text-[var(--color-muted-soft)] font-mono select-all">
                {user?.user_id?.slice(0, 18)}...
                <button onClick={copyId} className="hover:text-[var(--color-ink)]">{copied ? <Check size={11} className="text-[var(--success)]" /> : <Copy size={11} />}</button>
              </div>
            </div>
          ) : (
            <div className="space-y-4 mb-5">
              <div>
                <label className="block text-xs font-medium text-[var(--color-body)] mb-1.5">昵称</label>
                <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="昵称"
                  className="w-full h-10 px-3.5 rounded-lg bg-[var(--color-surface-soft)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-1 focus:ring-[var(--color-primary)]" />
              </div>
              <div className="flex items-center gap-6">
                <div className="flex items-center gap-2">
                  <label className="text-xs text-[var(--color-body)]">主色</label>
                  <input type="color" value={primaryColor} onChange={e => setPrimaryColor(e.target.value)} className="w-7 h-7 rounded cursor-pointer" />
                </div>
                <div className="flex items-center gap-2">
                  <label className="text-xs text-[var(--color-body)]">辅色</label>
                  <input type="color" value={secondaryColor} onChange={e => setSecondaryColor(e.target.value)} className="w-7 h-7 rounded cursor-pointer" />
                </div>
              </div>
              {uploading && <p className="text-[11px] text-[var(--color-muted)]">上传头像中...</p>}
              <button onClick={handleSave} disabled={saving}
                className="w-full h-10 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40">
                {saving ? '保存中...' : '保存'}
              </button>
            </div>
          )}

          {/* Action list */}
          <div className="space-y-0.5 border-t border-[var(--color-hairline)] pt-4">
            {[
              { icon: Bot, label: 'Agent 管理', sheet: 'agents' },
              { icon: Smartphone, label: '设备管理', sheet: 'sessions' },
              { icon: Settings, label: '设置', sheet: 'settings' },
            ].map(({ icon: Icon, label, sheet }) => (
              <button key={sheet}
                onClick={() => { onClose(); setTimeout(() => uiStore.openSheet(sheet), 50) }}
                className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-sm text-[var(--color-body)] hover:text-[var(--color-ink)] transition-colors">
                <Icon size={18} /> {label}
              </button>
            ))}
            <button onClick={handleLogout}
              className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-[var(--destructive)]/10 text-sm text-[var(--destructive)] transition-colors">
              <LogOut size={18} /> 退出登录
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
