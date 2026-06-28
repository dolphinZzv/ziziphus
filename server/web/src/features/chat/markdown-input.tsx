import { useRef, useEffect, useState, useCallback } from 'react'
import { Bold, Italic, Code, Link, Eye, List, Strikethrough } from 'lucide-react'
import { cn } from '@/lib/cn'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface Props {
  value: string
  onChange: (v: string) => void
  onSend: () => void
  placeholder?: string
  disabled?: boolean
}

export default function MarkdownInput({ value, onChange, onSend, placeholder = '', disabled }: Props) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [preview, setPreview] = useState(false)
  const [rawMode, setRawMode] = useState(false)

  // Auto-resize
  useEffect(() => {
    const el = textareaRef.current
    if (el) { el.style.height = 'auto'; el.style.height = Math.min(el.scrollHeight, 180) + 'px' }
  }, [value])

  const insertMarkup = useCallback((before: string, after = '', placeholder = '') => {
    const el = textareaRef.current
    if (!el) return

    const start = el.selectionStart
    const end = el.selectionEnd
    const selected = el.value.slice(start, end) || placeholder
    const newText = el.value.slice(0, start) + before + selected + after + el.value.slice(end)
    onChange(newText)

    // Restore cursor position after React re-render
    requestAnimationFrame(() => {
      el.focus()
      const newPos = start + before.length + selected.length + after.length
      el.setSelectionRange(newPos, newPos)
    })
  }, [onChange])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey) {
      e.preventDefault()
      onSend()
    }
    // Tab inserts spaces
    if (e.key === 'Tab') {
      e.preventDefault()
      insertMarkup('  ')
    }
  }

  const toolbarBtn = 'p-1 rounded-md hover:bg-[var(--color-hairline)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors'

  return (
    <div className="flex flex-col gap-1">
      {/* Toolbar */}
      <div className="flex items-center gap-0.5 px-1">
        <button type="button" className={toolbarBtn} title="加粗 (Ctrl+B)"
          onClick={() => insertMarkup('**', '**', '加粗文字')}>
          <Bold size={15} />
        </button>
        <button type="button" className={toolbarBtn} title="斜体 (Ctrl+I)"
          onClick={() => insertMarkup('*', '*', '斜体文字')}>
          <Italic size={15} />
        </button>
        <button type="button" className={toolbarBtn} title="删除线"
          onClick={() => insertMarkup('~~', '~~', '删除线')}>
          <Strikethrough size={15} />
        </button>
        <button type="button" className={toolbarBtn} title="代码"
          onClick={() => insertMarkup('`', '`', '代码')}>
          <Code size={15} />
        </button>
        <button type="button" className={toolbarBtn} title="链接"
          onClick={() => insertMarkup('[', '](url)', '链接文字')}>
          <Link size={15} />
        </button>
        <button type="button" className={toolbarBtn} title="列表"
          onClick={() => insertMarkup('- ', '')}>
          <List size={15} />
        </button>
        <div className="flex-1" />
        <button type="button" className={cn(toolbarBtn, preview && 'text-[var(--color-accent)] bg-[var(--color-accent)]/10')}
          title="预览" onClick={() => setPreview(!preview)}>
          <Eye size={15} />
        </button>
      </div>

      {/* Input area */}
      {preview ? (
        <button
          type="button"
          onClick={() => setPreview(false)}
          className="flex-1 min-h-[48px] max-h-[180px] overflow-y-auto px-4 py-2 rounded-xl bg-[var(--color-surface-soft)] text-sm text-left border border-[var(--color-hairline-soft)] cursor-text"
        >
          {value ? (
            <MarkdownPreview text={value} />
          ) : (
            <span className="text-[var(--color-muted-soft)]">{placeholder}</span>
          )}
        </button>
      ) : (
        <textarea
          ref={textareaRef}
          value={value}
          onChange={e => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          rows={2}
          disabled={disabled}
          className="flex-1 resize-none max-h-[180px] py-2.5 pl-4 pr-20 pb-10 bg-[var(--color-surface-soft)] text-[var(--color-ink)] text-sm placeholder:text-[var(--color-muted)] outline-none"
        />
      )}
    </div>
  )
}

function MarkdownPreview({ text }: { text: string }) {
  return (
    <div className="prose prose-sm dark:prose-invert max-w-none text-[var(--color-ink)] [&_p]:my-0 [&_code]:text-xs">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{text || ' '}</ReactMarkdown>
    </div>
  )
}
