import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface Props { text: string }

export default function TextBubble({ text }: Props) {
  return (
    <div className="prose prose-sm dark:prose-invert max-w-none break-words [&_p]:m-0 [&_p+p]:mt-1 [&_pre]:my-1 [&_pre]:text-xs [&_code]:text-xs [&_img]:rounded-lg [&_img]:max-w-[240px] [&_a]:text-blue-400 [&_a]:underline">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>
        {text || ''}
      </ReactMarkdown>
    </div>
  )
}
