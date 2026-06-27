import { getDateLabel } from '@/lib/time'

interface Props { timestamp: number }

export default function DateSeparator({ timestamp }: Props) {
  const label = getDateLabel(timestamp)
  if (!label) return null

  return (
    <div className="flex items-center justify-center my-4">
      <span className="inline-flex px-3 py-0.5 rounded-sm bg-[var(--color-surface-soft)] text-[var(--color-muted)] text-[11px] font-medium uppercase tracking-wider">
        {label}
      </span>
    </div>
  )
}
