import { cn } from '@/lib/cn'

export function SkeletonRow({ className }: { className?: string }) {
  return (
    <div className={cn('flex items-center gap-3 px-4 h-[52px]', className)}>
      <div className="w-10 h-10 rounded-full bg-[var(--color-hairline)] animate-pulse flex-shrink-0" />
      <div className="flex-1 space-y-2">
        <div className="h-3.5 w-2/5 rounded-sm bg-[var(--color-hairline)] animate-pulse" />
        <div className="h-2.5 w-3/5 rounded-sm bg-[var(--color-hairline-soft)] animate-pulse" />
      </div>
    </div>
  )
}

export function SkeletonChat() {
  return (
    <div className="px-4 py-2 space-y-1">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className={cn('flex gap-2', i % 2 === 0 ? 'justify-end' : 'justify-start')}>
          {i % 2 !== 0 && <div className="w-8 h-8 rounded-full bg-[var(--color-hairline)] animate-pulse flex-shrink-0 mt-1" />}
          <div className={cn('rounded-xl px-3 py-2 space-y-1.5', i % 2 === 0 ? 'bg-[var(--color-primary)]/20' : 'bg-[var(--color-surface-card)]')}
            style={{ width: `${Math.floor(Math.random() * 40 + 30)}%` }}>
            <div className="h-3 rounded-sm bg-[var(--color-hairline)] animate-pulse" style={{ width: '100%' }} />
            <div className="h-3 rounded-sm bg-[var(--color-hairline)] animate-pulse" style={{ width: '60%' }} />
          </div>
          {i % 2 === 0 && <div className="w-8 flex-shrink-0" />}
        </div>
      ))}
    </div>
  )
}
