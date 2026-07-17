import { useEffect, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { authService } from '@/services/auth-service'
import { fileService } from '@/services/file-service'
import { avatarUrl } from '@/lib/file'
import type { User, WakeMode } from '@/types/user'
import { X, Plus, Edit, Trash2, Key, Copy, Check, Camera, Cpu, ArrowLeft, Search, Eye, EyeOff } from 'lucide-react'
import { cn } from '@/lib/cn'

interface Props { onClose: () => void; inline?: boolean }

export default function AgentList({ onClose, inline }: Props) {
  const { t } = useTranslation()
  const [agents, setAgents] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<User | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [filterQuery, setFilterQuery] = useState('')

  const load = async () => { setLoading(true); try { setAgents(await authService.listAgents()) } catch {}; setLoading(false) }
  useEffect(() => { load() }, [])

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除该 Agent？')) return
    try { await authService.deleteAgent(id); load() } catch {}
  }

  const filteredAgents = filterQuery.trim()
    ? agents.filter(a => a.name?.toLowerCase().includes(filterQuery.trim().toLowerCase()))
    : agents

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  const inner = (
    <div className={`${inline ? 'h-full' : 'w-full sm:w-[420px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)]'} bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6 flex flex-col overflow-hidden`}
      style={inline ? {} : { boxShadow: 'var(--shadow-lg)' }} onClick={inline ? undefined : e => e.stopPropagation()}>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          {inline && (
            <button onClick={onClose} className="p-1 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">
              <ArrowLeft size={18} />
            </button>
          )}
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">Agent 管理</h3>
        </div>
        <div className="flex items-center gap-1">
          {agents.length < 10 && <button onClick={() => setShowCreate(true)} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><Plus size={16} /></button>}
          {!inline && <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>}
        </div>
      </div>

      {agents.length > 0 && (
        <div className="mb-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)]" />
            <input type="text" value={filterQuery} onChange={e => setFilterQuery(e.target.value)}
              placeholder={t('agent.searchPlaceholder')} className={`${inputClass} pl-8`} />
            {filterQuery && (
              <button onClick={() => setFilterQuery('')} className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-[var(--color-muted)]"><X size={12} /></button>
            )}
          </div>
        </div>
      )}

      <div className="flex-1 overflow-y-auto space-y-0.5">
        {filteredAgents.length === 0 && !loading ? (
          <p className="text-sm text-[var(--color-muted)] text-center py-8">{t('conversation.noMatch')}</p>
        ) : filteredAgents.map(agent => (
          <div key={agent.user_id} className="flex items-center gap-3 px-3 h-12 rounded-xl hover:bg-[var(--color-surface-soft)] group">
            <div className="relative flex-shrink-0">
              {agent.avatar ? (
                <img src={avatarUrl(agent.avatar)} alt="" className="w-9 h-9 rounded-full object-cover" />
              ) : (
                <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-semibold"
                  style={{ background: 'linear-gradient(135deg, #8B5CF6, #A78BFA)' }}>
                  {agent.name?.charAt(0)?.toUpperCase() || 'A'}
                </div>
              )}
              <span className="absolute -bottom-0.5 -right-0.5 w-4 h-4 rounded-full bg-blue-500 flex items-center justify-center border border-[var(--color-surface-card)]">
                <Cpu size={9} className="text-white" />
              </span>
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium text-[var(--color-ink)] flex items-center gap-1.5">{agent.name} <span className="text-[9px] px-1 py-0.5 rounded-sm bg-purple-500/10 text-purple-600 dark:text-purple-400 font-medium uppercase tracking-wider">Agent</span></div>
              <div className="text-[11px] text-[var(--color-muted)] flex items-center gap-1">
                {agent.api_key ? 'API Key 已配置' : 'API Key 未配置'}
              </div>
            </div>
            <button onClick={() => setEditing(agent)} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] opacity-0 group-hover:opacity-100 text-[var(--color-muted)] transition-all"><Edit size={14} /></button>
            <button onClick={() => handleDelete(agent.user_id)} className="p-1.5 rounded-xl hover:bg-[var(--destructive)]/10 opacity-0 group-hover:opacity-100 text-[var(--destructive)] transition-all"><Trash2 size={14} /></button>
          </div>
        ))}
        {agents.length >= 10 && <p className="text-xs text-[var(--color-muted)] text-center py-2">最多创建 10 个 Agent</p>}
      </div>

      {(showCreate || editing) && (
        <AgentEditDialog agent={editing} onClose={() => { setShowCreate(false); setEditing(null) }} onSaved={load} />
      )}
    </div>
  )

  if (inline) return inner
  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      {inner}
    </div>
  )
}

function AgentEditDialog({ agent, onClose, onSaved }: { agent: User | null; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState(agent?.name || '')
  const [headline, setHeadline] = useState(agent?.headline || '')
  const [avatar, setAvatar] = useState(agent?.avatar || '')
  const [cover, setCover] = useState(agent?.cover || '')
  const [primaryColor, setPrimaryColor] = useState(agent?.primary_color || '#8B5CF6')
  const [secondaryColor, setSecondaryColor] = useState(agent?.secondary_color || '#A78BFA')
  const [wakeMode, setWakeMode] = useState<WakeMode>(agent?.wake_mode ?? 0)
  const [apiKey, setApiKey] = useState(agent?.api_key || '')
  const [showKey, setShowKey] = useState(false)
  const [copied, setCopied] = useState(false)
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadingCover, setUploadingCover] = useState(false)
  const avatarInputRef = useRef<HTMLInputElement>(null)
  const coverInputRef = useRef<HTMLInputElement>(null)

  const handleSave = async () => {
    if (!name.trim()) return
    setSaving(true)
    try {
      const data = { name, headline, avatar, cover, wake_mode: wakeMode, primary_color: primaryColor, secondary_color: secondaryColor }
      agent ? await authService.updateAgent(agent.user_id, data) : await authService.createAgent(data)
      onSaved(); onClose()
    } catch {}
    setSaving(false)
  }

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]; if (!file) return
    setUploading(true)
    try { const r = await fileService.upload(file, file.name, 0); setAvatar(r.url) } catch {}
    setUploading(false); e.target.value = ''
  }

  const handleCoverUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]; if (!file) return
    setUploadingCover(true)
    try { const r = await fileService.upload(file, file.name, 0); setCover(r.url) } catch {}
    setUploadingCover(false); e.target.value = ''
  }

  const handleRegenerateKey = async () => {
    if (!agent || !confirm('重新生成 API Key？')) return
    try { const r = await authService.regenerateAgentKey(agent.user_id); setApiKey(r.api_key) } catch {}
  }

  const copyKey = () => { navigator.clipboard.writeText(apiKey); setCopied(true); setTimeout(() => setCopied(false), 2000) }

  const iCls = 'w-full h-10 px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[400px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] overflow-y-auto bg-[var(--color-surface-card)] rounded-none sm:rounded-xl"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Banner */}
        <div className="h-24 relative"
          style={{ background: cover ? `url(${cover}?w=800&h=192) center/cover` : `linear-gradient(135deg, ${primaryColor}, ${secondaryColor})` }}>
          {cover && <div className="absolute inset-0 bg-black/20" />}
          <button onClick={() => coverInputRef.current?.click()} disabled={uploadingCover}
            className="absolute top-3 left-3 p-1.5 rounded-xl bg-white/10 hover:bg-white/20 text-white/70 hover:text-white z-10">
            <Camera size={14} />
          </button>
          <input ref={coverInputRef} type="file" accept="image/*" onChange={handleCoverUpload} className="hidden" />
          <div className="absolute top-3 right-3 flex items-center gap-1 z-10">
            <button onClick={onClose} className="p-1.5 rounded-xl bg-white/20 hover:bg-white/30 text-white"><X size={15} /></button>
          </div>
        </div>

        {/* Avatar — overlaps banner */}
        <div className="flex justify-center -mt-9 mb-3">
          <button onClick={() => avatarInputRef.current?.click()} disabled={uploading}
            className="relative group cursor-pointer">
            {avatar ? (
              <img src={avatarUrl(avatar, 160)} alt="" className="w-[72px] h-[72px] rounded-full object-cover " />
            ) : (
              <div className="w-[72px] h-[72px] rounded-full flex items-center justify-center text-white text-xl font-bold "
                style={{ background: `linear-gradient(135deg, ${primaryColor}, ${secondaryColor})` }}>
                {name?.charAt(0)?.toUpperCase() || 'A'}
              </div>
            )}
            <div className="absolute inset-0 rounded-full bg-black/30 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
              <Camera size={16} className="text-white" />
            </div>
          </button>
          <input ref={avatarInputRef} type="file" accept="image/*" onChange={handleAvatarUpload} className="hidden" />
        </div>

        {/* Form */}
        <div className="px-5 pb-5 space-y-3">
          <div className="text-center">
            <h3 className="font-headline text-base font-semibold text-[var(--color-ink)]">{agent ? '编辑 Agent' : '创建 Agent'}</h3>
          </div>

          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-1 block">Agent 名称</label>
            <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Agent 名称" className={iCls} />
          </div>

          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-1 block">简介</label>
            <input type="text" value={headline} onChange={e => setHeadline(e.target.value)} placeholder="Agent 简介（选填）" maxLength={120} className={iCls} />
          </div>

          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-2 block">唤醒模式</label>
            <div className="flex rounded-xl bg-[var(--color-surface-soft)] p-0.5">
              {[{ v: 0, l: '全部消息' }, { v: 1, l: '仅被 @提及' }].map(m => (
                <button key={m.v} onClick={() => setWakeMode(m.v as WakeMode)}
                  className={cn('flex-1 h-8 rounded-lg text-xs font-medium transition-colors',
                    wakeMode === m.v ? 'bg-[var(--color-surface-card)] text-[var(--color-ink)] shadow-sm' : 'text-[var(--color-muted)] hover:text-[var(--color-body)]'
                  )}>{m.l}</button>
              ))}
            </div>
          </div>

          <div>
            <label className="text-[11px] text-[var(--color-muted)] mb-2 block">主题色</label>
            <div className="flex items-center gap-3">
              <label className="flex items-center gap-1.5 text-xs text-[var(--color-muted)]">主色 <input type="color" value={primaryColor} onChange={e => setPrimaryColor(e.target.value)} className="w-6 h-6 rounded cursor-pointer border-0" /></label>
              <label className="flex items-center gap-1.5 text-xs text-[var(--color-muted)]">辅色 <input type="color" value={secondaryColor} onChange={e => setSecondaryColor(e.target.value)} className="w-6 h-6 rounded cursor-pointer border-0" /></label>
            </div>
          </div>

          {agent && (
            <div>
              <label className="text-[11px] text-[var(--color-muted)] mb-1.5 block">API Key</label>
              {apiKey ? (
                <div className="space-y-1.5">
                  <div className="flex gap-1">
                    <input type={showKey ? 'text' : 'password'} value={apiKey} readOnly
                      className={cn(iCls, 'font-mono text-xs flex-1')} />
                    <button onClick={() => setShowKey(!showKey)}
                      className="px-2 text-xs text-[var(--color-muted)] hover:text-[var(--color-ink)]">
                      {showKey ? <EyeOff size={14} /> : <Eye size={14} />}
                    </button>
                    <button onClick={copyKey} className="p-2 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">
                      {copied ? <Check size={14} className="text-[var(--success)]" /> : <Copy size={14} />}
                    </button>
                  </div>
                  <button onClick={handleRegenerateKey} className="text-[11px] text-[var(--destructive)] hover:underline">重新生成 Key</button>
                </div>
              ) : (
                <button onClick={handleRegenerateKey}
                  className="w-full h-10 rounded-xl border border-dashed border-[var(--color-primary)]/30 text-sm text-[var(--color-primary)] hover:bg-[var(--color-primary)]/5 transition-colors flex items-center justify-center gap-2">
                  <Key size={16} /> 生成 API Key
                </button>
              )}
            </div>
          )}

          <div className="flex gap-2 pt-1">
            <button onClick={onClose}
              className="flex-1 h-10 rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-body)] hover:bg-[var(--color-surface-soft)] transition-colors">取消</button>
            <button onClick={handleSave} disabled={saving || !name.trim()}
              className="flex-1 h-10 rounded-xl text-sm font-medium transition-colors disabled:opacity-40"
              style={{ background: `linear-gradient(135deg, ${primaryColor}, ${secondaryColor})`, color: '#FFFFFF' }}>
              {saving ? '保存中...' : '保存'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
