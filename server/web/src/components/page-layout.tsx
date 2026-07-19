import { ReactNode } from 'react'

/** Centered page layout — vertically centers content. Used by auth pages and standalone pages. */
export default function PageLayout({ children }: { children: ReactNode }) {
  return (
    <div className="h-full flex flex-col items-center sm:justify-center justify-start bg-[var(--color-canvas)] relative px-8 sm:pt-0 pt-12 gap-6">
      {children}
    </div>
  )
}
