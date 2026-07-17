import Sidebar from '@/features/layout/sidebar'

/** Standalone page for the conversation list (used on mobile) */
export default function ConversationsPage() {
  return (
    <div className="h-full w-full bg-[var(--color-canvas)]">
      <Sidebar />
    </div>
  )
}
