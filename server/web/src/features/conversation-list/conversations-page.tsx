import Sidebar from '@/features/layout/sidebar'
import EmptyChat from '@/features/chat/empty-chat'
import { useIsMobile } from '@/hooks/use-breakpoint'

/** Standalone page for the conversation list.
 *  Mobile: full screen conversation list.
 *  Desktop/tablet: sidebar + empty state on the right. */
export default function ConversationsPage() {
  const isMobile = useIsMobile()

  if (isMobile) {
    return (
      <div className="h-full w-full bg-[var(--color-canvas)]">
        <Sidebar />
      </div>
    )
  }

  return (
    <div className="h-full w-full flex">
      {/* Sidebar */}
      <div className="flex-shrink-0 h-full flex flex-col border-r border-[var(--color-hairline)] bg-[var(--color-canvas)]"
        style={{ width: 288 }}>
        <Sidebar />
      </div>
      {/* Empty state */}
      <EmptyChat />
    </div>
  )
}
