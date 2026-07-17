import { ReactNode } from 'react'
import AnnouncementBanner from '@/components/announcement-banner'

/**
 * AppShell wraps the entire application:
 * - Top: announcement banner (fixed, non-scrolling)
 * - Below: content area (flex-1, fills remaining height)
 * Both auth pages and the chat layout render inside the content area.
 */
export default function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="h-full w-full flex flex-col">
      <div className="flex-shrink-0">
        <AnnouncementBanner />
      </div>
      <div className="flex-1 min-h-0">
        {children}
      </div>
    </div>
  )
}
