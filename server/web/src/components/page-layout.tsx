import { ReactNode } from 'react'

/** Centered page layout — vertically centers content. Used by auth pages and standalone pages. */
export default function PageLayout({ children }: { children: ReactNode }) {
  return (
    <div className="h-full flex flex-col items-center justify-center bg-[var(--color-canvas)] relative px-8 gap-8">
      {children}
    </div>
  )
}
