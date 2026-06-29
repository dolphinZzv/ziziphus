import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface Props { text: string; highlight?: string }

function HighlightedMarkdown({ text, keyword }: { text: string; keyword?: string }) {
  if (!keyword) {
    return (
      <ReactMarkdown remarkPlugins={[remarkGfm]}>
        {text || ''}
      </ReactMarkdown>
    )
  }
  // For markdown with search highlight: render as normal markdown but highlight text nodes
  // Use a simpler approach: highlight in the raw text before markdown rendering
  const escaped = keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const highlighted = text.replace(
    new RegExp(`(${escaped})`, 'gi'),
    '<mark class="bg-[var(--color-primary)]/30 dark:bg-[var(--color-primary)]/40 text-inherit rounded-sm px-0.5">$1</mark>'
  )
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]}>
      {highlighted}
    </ReactMarkdown>
  )
}

export default function TextBubble({ text, highlight }: Props) {
  return (
    <div className="prose prose-sm dark:prose-invert max-w-none break-words [&_p]:m-0 [&_p+p]:mt-1 [&_pre]:my-1 [&_pre]:text-xs [&_code]:text-xs [&_img]:rounded-xl [&_img]:max-w-[240px] [&_a]:text-blue-400 [&_a]:underline">
      <HighlightedMarkdown text={text || ''} keyword={highlight} />
    </div>
  )
}
