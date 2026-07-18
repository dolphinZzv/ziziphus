import { useTranslation } from 'react-i18next'
import { Check, X } from 'lucide-react'
import { cn } from '@/lib/cn'

interface Props { body: string }

export default function FormResponseBubble({ body }: Props) {
  const { t } = useTranslation()
  let resp: { action: string; responder_name?: string } | null = null
  try { resp = JSON.parse(body) } catch (e) { console.error(e) }

  if (!resp) {
    return <div className="text-xs text-[var(--color-muted)]">[Response]</div>
  }

  const isApproved = resp.action === 'approve'
  const name = resp.responder_name || ''

  return (
    <div className={cn(
      'flex items-center gap-1.5 text-xs',
      isApproved ? 'text-green-600' : 'text-red-500'
    )}>
      {isApproved ? <Check size={12} /> : <X size={12} />}
      <span>{isApproved ? t('friendRequest.approvedBy', { name }) : t('friendRequest.rejectedBy', { name })}</span>
    </div>
  )
}
