import { useEffect, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { authService } from '@/services/auth-service'
import { fileService } from '@/services/file-service'
import { avatarUrl } from '@/lib/file'
import type { User, WakeMode } from '@/types/user'
import { X, Plus, Edit, Trash2, Key, Copy, Check, Camera, Cpu, ArrowLeft, Search } from 'lucide-react'
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
    <div className={`${inline ? 'h-full' : 'w-[420px] max-h-[520px]'} bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl p-6 flex flex-col overflow-hidden`}
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

      {/* Search filter */}
      {agents.length > 0 && (
        <div className="mb-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)]" />
            <input type="text" value={filterQuery} onChange={e => setFilterQuery(e.target.value)}
              placeholder={t('conversation.searchPlaceholder')} className={`${inputClass} pl-8`} />
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
                <Key size={10} className={agent.api_key ? 'text-[var(--success)]' : 'text-[var(--color-muted-soft)]'} />
                {agent.api_key ? 'API Key 已配置' : '未配置'}
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      {inner}
    </div>
  )
}

function AgentEditDialog({ agent, onClose, onSaved }: { agent: User | null; onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState(agent?.name || '')
  const [avatar, setAvatar] = useState(agent?.avatar || '')
  const [wakeMode, setWakeMode] = useState<WakeMode>(agent?.wake_mode ?? 0)
  const [apiKey, setApiKey] = useState(agent?.api_key || '')
  const [showKey, setShowKey] = useState(false)
  const [copied, setCopied] = useState(false)
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const avatarInputRef = useRef<HTMLInputElement>(null)

  const handleSave = async () => {
    setSaving(true)
    try {
      const data = { name, avatar, wake_mode: wakeMode, primary_color: '#8B5CF6', secondary_color: '#A78BFA' }
      agent ? await authService.updateAgent(agent.user_id, data) : await authService.createAgent(data)
      onSaved(); onClose()
    } catch {}
    setSaving(false)
  }

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const result = await fileService.upload(file, file.name, 0)
      setAvatar(result.url)
    } catch {}
    setUploading(false)
    if (e.target) e.target.value = ''
  }

  const handleRegenerateKey = async () => {
    if (!agent || !confirm('重新生成 API Key？')) return
    try { const r = await authService.regenerateAgentKey(agent.user_id); setApiKey(r.api_key) } catch {}
  }

  const copyKey = () => { navigator.clipboard.writeText(apiKey); setCopied(true); setTimeout(() => setCopied(false), 2000) }
  const segBtn = (active: boolean) => cn('flex-1 h-[34px] rounded text-xs font-medium transition-colors',
    active ? 'bg-[var(--color-primary)] text-white' : 'bg-[var(--color-surface-soft)] text-[var(--color-body)]')
  const inputClass2 = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/30 rounded-xl" onClick={onClose}>
      <div className="w-[360px] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl p-6"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <h3 className="font-headline text-base font-semibold text-[var(--color-ink)] mb-4">{agent ? '编辑 Agent' : '创建 Agent'}</h3>
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-1.5">Agent 名称</label>
            <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Agent 名称" className={inputClass2} />
          </div>
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">头像</label>
            <div className="flex items-center gap-3">
              <button type="button"
                onClick={() => avatarInputRef.current?.click()}
                disabled={uploading}
                className="relative group flex-shrink-0"
              >
                {avatar ? (
                  <img src={avatarUrl(avatar)} alt="" className="w-12 h-12 rounded-full object-cover" />
                ) : (
                  <div className="w-12 h-12 rounded-full flex items-center justify-center text-white text-lg font-bold"
                    style={{ background: 'linear-gradient(135deg, #8B5CF6, #A78BFA)' }}>
                    {name?.charAt(0)?.toUpperCase() || 'A'}
                  </div>
                )}
                <div className="absolute inset-0 rounded-full bg-black/30 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
                  <Camera size={14} className="text-white" />
                </div>
              </button>
              <input ref={avatarInputRef} type="file" accept="image/*" onChange={handleAvatarUpload} className="hidden" />
              <div className="text-xs text-[var(--color-muted)]">
                {uploading ? '上传中...' : '点击上传头像'}
                {avatar && <button onClick={() => setAvatar('')} className="ml-2 text-[var(--destructive)] hover:underline">清除</button>}
              </div>
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">唤醒模式</label>
            <div className="flex gap-2">
              {[{ v: 0, l: '全部消息' }, { v: 1, l: '仅被@提及' }].map(m => (
                <button key={m.v} onClick={() => setWakeMode(m.v as WakeMode)} className={segBtn(wakeMode === m.v)}>{m.l}</button>
              ))}
            </div>
          </div>
          {agent && (
            <div>
              <label className="block text-xs font-medium text-[var(--color-body)] mb-1.5">API Key</label>
              {apiKey ? (
                <>
                  <div className="flex gap-1">
                    <input type={showKey ? 'text' : 'password'} value={apiKey} readOnly className={cn(inputClass2, 'font-mono text-xs')} />
                    <button onClick={() => setShowKey(!showKey)} className="px-2 text-xs text-[var(--color-muted)]">{showKey ? '隐藏' : '显示'}</button>
                    <button onClick={copyKey} className="p-2 rounded-xl hover:bg-[var(--color-surface-soft)]">{copied ? <Check size={14} className="text-[var(--success)]" /> : <Copy size={14} className="text-[var(--color-muted)]" />}</button>
                  </div>
                  <button onClick={handleRegenerateKey} className="text-[11px] text-[var(--destructive)] mt-1 hover:underline">重新生成</button>
                </>
              ) : (
                <button onClick={handleRegenerateKey} className="w-full h-[42px] rounded-xl border border-dashed border-[var(--color-primary)]/30 text-sm text-[var(--color-primary)] hover:bg-[var(--color-primary)]/5 transition-colors flex items-center justify-center gap-2">
                  <Key size={16} /> 生成 API Key
                </button>
              )}
            </div>
          )}
          <div className="flex gap-2 pt-2">
            <button onClick={onClose} className="flex-1 h-[42px] rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-body)] hover:bg-[var(--color-surface-soft)] transition-colors">取消</button>
            <button onClick={handleSave} disabled={saving} className="flex-1 h-[42px] rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40">{saving ? '保存中...' : '保存'}</button>
          </div>
        </div>
      </div>
    </div>
  )
}
