import { useRef, useEffect, useCallback } from 'react'
import { cn } from '@/lib/cn'
import { useIsMobile } from '@/hooks/use-breakpoint'

interface Props {
  value: string
  onChange: (v: string) => void
  onSend: () => void
  placeholder?: string
  disabled?: boolean
  onFocus?: () => void
  onBlur?: () => void
}

export default function MarkdownInput({ value, onChange, onSend, placeholder = '', disabled, onFocus, onBlur }: Props) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const isComposingRef = useRef(false)
  const isMobile = useIsMobile()

  // Auto-resize
  useEffect(() => {
    if (isMobile) return
    const el = textareaRef.current
    if (el) { el.style.height = 'auto'; el.style.height = Math.min(el.scrollHeight, 160) + 'px' }
  }, [value, isMobile])

  const handleCompositionStart = () => { isComposingRef.current = true }
  const handleCompositionEnd = () => { isComposingRef.current = false }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey && !isComposingRef.current && !e.nativeEvent.isComposing) {
      e.preventDefault()
      onSend()
    }
  }

  return (
    <textarea
      ref={textareaRef}
      value={value}
      onChange={e => onChange(e.target.value)}
      onKeyDown={handleKeyDown}
      onCompositionStart={handleCompositionStart}
      onCompositionEnd={handleCompositionEnd}
      placeholder={placeholder}
      rows={1}
      disabled={disabled}
      className={cn(
        'w-full resize-none bg-transparent text-[var(--color-ink)] text-sm placeholder:text-[var(--color-muted-soft)] outline-none',
        isMobile ? 'py-2.5 pl-4 pr-[72px]' : 'min-h-[44px] max-h-[160px] py-2.5 pl-4 pr-[80px]',
      )}
      onFocus={onFocus}
      onBlur={onBlur}
    />
  )
}
