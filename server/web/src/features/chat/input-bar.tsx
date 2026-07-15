import { useState, useRef, useEffect, useSyncExternalStore } from 'react'
import { chatStore } from '@/stores/chat-store'
import { api } from '@/services/api-client'
import { fileService } from '@/services/file-service'
import { conversationService } from '@/services/conversation-service'
import { userService } from '@/services/user-service'
import { ContentType } from '@/types/message'
import { cn } from '@/lib/cn'
import { avatarUrl } from '@/lib/file'
import { Send, Paperclip, Image, X, AtSign } from 'lucide-react'
import MarkdownInput from './markdown-input'
import { useTranslation } from 'react-i18next'

interface Props { convId: string; isP2p?: boolean }

export default function InputBar({ convId, isP2p }: Props) {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  const [uploading, setUploading] = useState(false)
  const [members, setMembers] = useState<Array<{ id: string; name: string; avatar?: string }>>([])
  const [showMention, setShowMention] = useState(false)
  const [mentionFilter, setMentionFilter] = useState('')
  const [showAttachMenu, setShowAttachMenu] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const replyTo = useSyncExternalStore(chatStore.subscribe, () => chatStore.getReplyTo(convId))

  // Restore draft when switching conversations
  useEffect(() => {
    const draft = chatStore.getDraft(convId)
    setText(draft)
  }, [convId])

  // Load members + webhooks for @mention
  useEffect(() => {
    Promise.allSettled([
      conversationService.getDetail(convId).then(d => {
        const ids = d.members.map(m => m.user_id)
        if (ids.length > 1) {
          userService.batchGet(ids).then(users => {
            setMembers(prev => {
              const userItems = Object.entries(users).map(([id, u]) => ({ id, name: u.name || id, avatar: u.avatar || '' }))
              // Merge: keep existing webhook items, replace user items
              const whItems = prev.filter(m => m.id.startsWith('wh:'))
              return [...userItems, ...whItems]
            })
          }).catch(() => {})
        }
      }),
      api.request<Array<{ id: number; name: string }>>(`/api/v1/conversations/${convId}/webhooks`).then(whs => {
        setMembers(prev => {
          const whItems = whs.map(wh => ({ id: `wh:${wh.id}`, name: `@${wh.name}` }))
          const userItems = prev.filter(m => !m.id.startsWith('wh:'))
          return [...userItems, ...whItems]
        })
      }).catch(() => {}),
    ])
  }, [convId])

  const handleSend = () => {
    const trimmed = text.trim()
    if (!trimmed || uploading) return
    // Extract @mentions from text
    const mentionIds: string[] = []
    const atMatches = trimmed.match(/@(\S+)/g)
    if (atMatches) {
      atMatches.forEach(m => {
        const name = m.slice(1)
        const member = members.find(mm => mm.name === name || mm.id === name)
        if (member) mentionIds.push(member.id)
      })
    }
    chatStore.sendMessage(convId, trimmed, ContentType.Text, replyTo?.msg_id || 0, mentionIds)
    chatStore.setDraft(convId, '')
    chatStore.setReplyTo(convId, null)
    setText('')
  }

  const handleChange = (v: string) => { setText(v); chatStore.setDraft(convId, v) }

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>, type: 'image' | 'file') => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    setShowAttachMenu(false)
    try {
      const result = await fileService.upload(file, file.name, type === 'image' ? 0 : 1, undefined, convId)
      const body = JSON.stringify({ url: result.url, name: file.name, size: file.size, file_id: result.file_id })
      chatStore.sendMessage(convId, body, type === 'image' ? ContentType.Image : ContentType.File)
    } catch { /* ignore */ }
    setUploading(false)
    if (e.target) e.target.value = ''
  }

  // Handle paste event for images
  const handlePaste = async (e: React.ClipboardEvent) => {
    const items = e.clipboardData?.items
    if (!items) return
    for (let i = 0; i < items.length; i++) {
      if (items[i].type.startsWith('image/')) {
        e.preventDefault()
        const file = items[i].getAsFile()
        if (file) await uploadFile(file, 'image')
        return
      }
    }
  }

  // Handle drag-drop
  const handleDragOver = (e: React.DragEvent) => { e.preventDefault(); setDragOver(true) }
  const handleDragLeave = () => setDragOver(false)
  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer?.files?.[0]
    if (!file) return
    const type = file.type.startsWith('image/') ? 'image' : 'file'
    await uploadFile(file, type)
  }

  const uploadFile = async (file: File, type: 'image' | 'file') => {
    setUploading(true)
    try {
      const result = await fileService.upload(file, file.name, type === 'image' ? 0 : 1, undefined, convId)
      const body = JSON.stringify({ url: result.url, name: file.name, size: file.size, file_id: result.file_id })
      chatStore.sendMessage(convId, body, type === 'image' ? ContentType.Image : ContentType.File)
    } catch { /* ignore */ }
    setUploading(false)
  }

  // Insert @mention into text
  const insertMention = (name: string) => {
    const before = text.slice(0, text.lastIndexOf('@'))
    setText(before + `@${name} `)
    setShowMention(false)
    setMentionFilter('')
  }

  // Listen for @ in text changes (skip for P2P)
  const handleMentionChange = (v: string) => {
    handleChange(v)
    if (isP2p) return
    const cursorPos = v.length
    const atIdx = v.lastIndexOf('@', cursorPos)
    if (atIdx >= 0 && (atIdx === 0 || /\s/.test(v[atIdx - 1]))) {
      const afterAt = v.slice(atIdx + 1, cursorPos)
      if (!afterAt.includes(' ')) {
        setMentionFilter(afterAt.toLowerCase())
        setShowMention(true)
        return
      }
    }
    setShowMention(false)
  }

  const filtered = mentionFilter
    ? members.filter(m => m.name.toLowerCase().includes(mentionFilter) || m.id.toLowerCase().includes(mentionFilter)).slice(0, 5)
    : members.slice(0, 5)

  return (
    <div className="flex-shrink-0 border-t border-[var(--color-hairline)] bg-[var(--color-surface-card)] relative"
      onPaste={handlePaste}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}>
      {/* Reply preview */}
      {replyTo && (
        <div className="flex items-center gap-2 px-4 py-2 bg-[var(--color-surface-soft)] border-b border-[var(--color-hairline)]">
          <div className="flex-1 min-w-0">
            <div className="text-[11px] text-[var(--color-accent)] font-medium">回复 {replyTo.sender_name}</div>
            <div className="text-xs text-[var(--color-muted)] truncate">{replyTo.body?.slice(0, 80)}</div>
          </div>
          <button onClick={() => chatStore.setReplyTo(convId, null)} className="p-1 rounded hover:bg-[var(--color-hairline)] text-[var(--color-muted)]"><X size={14} /></button>
        </div>
      )}

      {/* Mention popup — hide for P2P */}
      {!isP2p && showMention && members.length > 0 && (
        <div className="relative">
          <div className="absolute bottom-0 left-4 right-4 z-10 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl mb-1 max-h-[160px] overflow-y-auto"
            style={{ boxShadow: 'var(--shadow-md)' }}>
            {filtered.length > 0 ? filtered.map(m => (
              <button key={m.id} type="button"
                onClick={() => insertMention(m.name)}
                className="w-full flex items-center gap-3 px-3 py-2 hover:bg-[var(--color-surface-soft)] text-sm text-[var(--color-ink)]">
                {m.avatar ? (
                  <img src={avatarUrl(m.avatar, 48)} alt="" className="w-6 h-6 rounded-full object-cover flex-shrink-0" />
                ) : (
                  <div className="w-6 h-6 rounded-full bg-[var(--color-muted)]/20 flex items-center justify-center text-xs font-semibold flex-shrink-0">
                    {m.name.charAt(0)}
                  </div>
                )}
                <span>{m.name}</span>
              </button>
            )) : (
              <div className="px-3 py-2 text-xs text-[var(--color-muted)]">无匹配成员</div>
            )}
          </div>
        </div>
      )}

      {/* Input area — macOS style: buttons at bottom-right */}
      <div className="pt-3">
        <div className="relative">
          <MarkdownInput
            value={text}
            onChange={handleMentionChange}
            onSend={handleSend}
            placeholder={t('chat.inputPlaceholder') + (isP2p ? '' : ' @用户名 提及')}
            disabled={uploading}
          />

          {/* Bottom-right action buttons */}
          <div className="flex items-center gap-1 absolute bottom-2 right-2">
            {!isP2p && (
            <button type="button"
              onClick={() => { setMentionFilter(''); setShowMention(!showMention) }}
              className="p-1.5 rounded-xl hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
              title="@提及">
              <AtSign size={17} />
            </button>
          )}

            {/* Attachment button + popover */}
            <div className="relative">
              <button type="button"
                onClick={() => setShowAttachMenu(!showAttachMenu)}
                className="p-1.5 rounded-xl hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
                title="附件">
                <Paperclip size={17} />
              </button>
              {showAttachMenu && (
                <>
                  <div className="fixed inset-0 z-10" onClick={() => setShowAttachMenu(false)} />
                  <div className="absolute bottom-full right-0 mb-1 w-36 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl z-20 py-1"
                    style={{ boxShadow: 'var(--shadow-md)' }}>
                    <button type="button"
                      onClick={() => imageInputRef.current?.click()}
                      className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                      <Image size={15} /> 图片
                    </button>
                    <button type="button"
                      onClick={() => fileInputRef.current?.click()}
                      className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                      <Paperclip size={15} /> 文件
                    </button>
                  </div>
                </>
              )}
            </div>

            {/* Send button */}
            <button
              onClick={handleSend}
              disabled={!text.trim() || uploading}
              className={cn(
                'p-1.5 rounded-xl transition-colors',
                text.trim()
                  ? 'bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white'
                  : 'text-[var(--color-muted-soft)] cursor-default'
              )}
              title="发送">
              <Send size={17} />
            </button>
          </div>
        </div>

        <input ref={imageInputRef} type="file" accept="image/*" onChange={e => handleFileSelect(e, 'image')} className="hidden" />
        <input ref={fileInputRef} type="file" multiple onChange={e => handleFileSelect(e, 'file')} className="hidden" />
      </div>

        {/* Drag overlay */}
        {dragOver && (
          <div className="absolute inset-0 z-50 bg-[var(--color-primary)]/10 border-2 border-dashed border-[var(--color-primary)] rounded-xl flex items-center justify-center pointer-events-none">
            <span className="text-sm text-[var(--color-primary)] font-medium">释放以上传文件</span>
          </div>
        )}
    </div>
  )
}
