import { useState } from 'react'
import type { AgentTimelineBody, AgentTimelineEntry } from '@/types/agent_timeline'
import { AgentStepStatus, AgentStepType } from '@/types/agent_timeline'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Brain, Wrench, FileText, MessageCircle, ChevronDown, ChevronRight, Loader2 } from 'lucide-react'

interface Props { body: string }

export default function AgentTimelineBubble({ body }: Props) {
  let timeline: AgentTimelineBody
  try {
    timeline = JSON.parse(body)
  } catch {
    return <div className="text-sm italic opacity-60">[Agent 消息]</div>
  }

  const statusColor = () => {
    const s = timeline.status
    if (s === AgentStepStatus.Running) return 'var(--warning)'
    if (s === AgentStepStatus.Success) return 'var(--success)'
    if (s === AgentStepStatus.Error) return 'var(--destructive)'
    return 'var(--color-muted)'
  }

  const statusLabel = () => {
    const s = timeline.status
    if (s === AgentStepStatus.Running) return '运行中'
    if (s === AgentStepStatus.Success) return '已完成'
    if (s === AgentStepStatus.Error) return '错误'
    return '等待中'
  }

  return (
    <div className="min-w-[260px] space-y-1.5">
      {/* Status header */}
      <div className="flex items-center gap-1.5 mb-2">
        <span className="w-1.5 h-1.5 rounded-full flex-shrink-0"
          style={{ background: statusColor() }} />
        <span className="text-[10px] font-medium uppercase tracking-wider" style={{ color: statusColor() }}>
          {statusLabel()}
        </span>
      </div>

      {timeline.entries?.map((entry, i) => (
        <TimelineEntry key={entry.id || i} entry={entry} />
      ))}
    </div>
  )
}

function TimelineEntry({ entry }: { entry: AgentTimelineEntry }) {
  const [expanded, setExpanded] = useState(
    entry.type === AgentStepType.Response || entry.status === AgentStepStatus.Error
  )

  const icon = () => {
    const cls = 'flex-shrink-0 mt-0.5'
    switch (entry.type) {
      case AgentStepType.Thinking: return <Brain size={14} className={`${cls} text-purple-500`} />
      case AgentStepType.ToolCall: return <Wrench size={14} className={`${cls} text-orange-500`} />
      case AgentStepType.ToolResult: return <FileText size={14} className={`${cls} text-green-500`} />
      case AgentStepType.Response: return <MessageCircle size={14} className={`${cls} text-blue-500`} />
      default: return <MessageCircle size={14} className={`${cls}`} />
    }
  }

  const statusDot = () => {
    if (entry.status === AgentStepStatus.Running) return <Loader2 size={10} className="text-orange-500 animate-spin flex-shrink-0" />
    if (entry.status === AgentStepStatus.Success) return <span className="w-1.5 h-1.5 rounded-full bg-green-500 flex-shrink-0" />
    if (entry.status === AgentStepStatus.Error) return <span className="w-1.5 h-1.5 rounded-full bg-red-500 flex-shrink-0" />
    return <span className="w-1.5 h-1.5 rounded-full bg-gray-400 flex-shrink-0" />
  }

  const renderContent = () => {
    if (!entry.content) return null

    if (entry.type === AgentStepType.Response) {
      return (
        <div className="mt-1.5 prose prose-xs dark:prose-invert max-w-none text-inherit opacity-90 [&_p]:my-0.5 [&_pre]:text-[11px] [&_code]:text-[11px]">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{entry.content}</ReactMarkdown>
        </div>
      )
    }

    if (entry.type === AgentStepType.ToolCall || entry.type === AgentStepType.ToolResult) {
      // Try to pretty-print JSON tool input/output
      let formatted = entry.content
      try {
        const parsed = JSON.parse(entry.content)
        formatted = JSON.stringify(parsed, null, 2)
      } catch {}
      return (
        <div className="mt-1.5 p-2 bg-black/5 dark:bg-white/5 rounded text-[11px] whitespace-pre-wrap break-words font-mono opacity-80 max-h-[200px] overflow-y-auto">
          {formatted}
        </div>
      )
    }

    // thinking content — plain text
    return (
      <div className="mt-1.5 text-[11px] whitespace-pre-wrap break-words opacity-70 italic">
        {entry.content}
      </div>
    )
  }

  return (
    <div className="text-xs">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1 w-full text-left hover:opacity-80 py-0.5"
      >
        {expanded ? <ChevronDown size={10} className="flex-shrink-0 opacity-50" /> : <ChevronRight size={10} className="flex-shrink-0 opacity-50" />}
        {icon()}
        <span className="truncate flex-1">{entry.title || entry.type}</span>
        {statusDot()}
      </button>
      {expanded && renderContent()}
    </div>
  )
}
