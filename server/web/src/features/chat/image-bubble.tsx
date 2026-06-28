import { useState, lazy, Suspense } from 'react'
import { Image } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useTranslation } from 'react-i18next'

const ImageViewer = lazy(() => import('@/components/image-viewer'))

interface Props { body: string; msgId: number }

function thumbUrl(original: string, size = 400): string {
  if (!original || original.includes('?') || original.startsWith('data:') || original.startsWith('blob:')) return original
  if (original.endsWith('.svg')) return original
  return `${original}?w=${size}&h=${size}`
}

export default function ImageBubble({ body, msgId }: Props) {
  const { t } = useTranslation()
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState(false)
  const [viewer, setViewer] = useState(false)

  let url = body
  try {
    const parsed = JSON.parse(body)
    url = parsed.url || parsed.thumbnail_url || body
  } catch {}

  if (error) {
    return (
      <div className="w-[200px] h-[150px] bg-[var(--color-surface-soft)] rounded-xl flex items-center justify-center text-[var(--color-muted)]">
        <Image size={24} />
      </div>
    )
  }

  return (
    <>
      <div className="relative max-w-[400px]">
        {!loaded && (
          <div className="w-[200px] h-[150px] bg-[var(--color-surface-soft)] rounded-xl animate-pulse flex items-center justify-center">
            <Image size={24} className="text-[var(--color-muted)]" />
          </div>
        )}
        <img
          src={thumbUrl(url)}
          alt=""
          className={cn('rounded-xl max-w-full max-h-[360px] object-cover cursor-pointer hover:opacity-90 transition-opacity', !loaded && 'hidden')}
          onLoad={() => setLoaded(true)}
          onError={() => setError(true)}
          onClick={() => setViewer(true)}
        />
      </div>
      {viewer && (
        <Suspense fallback={null}>
          <ImageViewer images={[url]} currentIndex={0} onClose={() => setViewer(false)} />
        </Suspense>
      )}
    </>
  )
}
