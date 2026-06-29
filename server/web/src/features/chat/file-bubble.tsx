import { useState, useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, Download, X, ZoomIn, ZoomOut, ExternalLink } from 'lucide-react'
import { formatFileSize } from '@/lib/file'

interface Props { body: string }

export default function FileBubble({ body }: Props) {
  const { t } = useTranslation()
  const [viewerOpen, setViewerOpen] = useState(false)
  const [zoom, setZoom] = useState(1)
  let name = t('chat.file')
  let size = 0
  let url = ''

  try {
    const parsed = JSON.parse(body)
    name = parsed.name || parsed.file_name || t('chat.file')
    size = parsed.size || 0
    url = parsed.url || ''
  } catch {
    name = body
  }

  const isPDF = name.toLowerCase().endsWith('.pdf') && !!url

  const handleClick = () => {
    if (isPDF) {
      setViewerOpen(true)
      setZoom(1)
    } else if (url) {
      window.open(url, '_blank')
    }
  }

  const zoomIn = () => setZoom(z => Math.min(z * 1.5, 5))
  const zoomOut = () => setZoom(z => Math.max(z / 1.5, 0.5))

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') setViewerOpen(false)
  }, [])

  useEffect(() => {
    if (viewerOpen) {
      document.addEventListener('keydown', handleKeyDown)
      document.body.style.overflow = 'hidden'
    }
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = ''
    }
  }, [viewerOpen, handleKeyDown])

  return (
    <>
      <button
        onClick={handleClick}
        className="flex items-center gap-3 p-2 rounded-xl bg-white/10 hover:bg-white/20 transition-colors min-w-[200px]"
      >
        <div className="w-12 h-12 rounded-xl bg-white/20 flex items-center justify-center flex-shrink-0">
          <FileText size={22} />
        </div>
        <div className="flex-1 min-w-0 text-left">
          <div className="text-sm truncate">{name}</div>
          {size > 0 && <div className="text-[11px] opacity-60">{formatFileSize(size)}</div>}
        </div>
        <Download size={16} className="flex-shrink-0 opacity-60" />
      </button>

      {/* Full-screen PDF viewer (like image preview) */}
      {viewerOpen && (
        <div className="fixed inset-0 z-50 bg-black/95 flex flex-col">
          {/* Top bar */}
          <div className="flex items-center justify-between px-4 py-3 flex-shrink-0">
            <div className="flex items-center gap-2 min-w-0">
              <FileText size={16} className="text-white/70 flex-shrink-0" />
              <span className="text-sm text-white/80 truncate">{name}</span>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={zoomOut}
                className="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white/80 hover:text-white transition-colors"
                title={t('chat.zoomOut')}
              >
                <ZoomOut size={16} />
              </button>
              <span className="text-xs text-white/60 min-w-[36px] text-center">{Math.round(zoom * 100)}%</span>
              <button
                onClick={zoomIn}
                className="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white/80 hover:text-white transition-colors"
                title={t('chat.zoomIn')}
              >
                <ZoomIn size={16} />
              </button>
              <a
                href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white/80 hover:text-white transition-colors"
                title={t('chat.openInNewTab')}
              >
                <ExternalLink size={16} />
              </a>
              <button
                onClick={() => setViewerOpen(false)}
                className="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white/80 hover:text-white transition-colors"
              >
                <X size={18} />
              </button>
            </div>
          </div>

          {/* PDF viewer */}
          <div className="flex-1 min-h-0 px-4 pb-4">
            <object
              data={`${url}#zoom=${Math.round(zoom * 100)}`}
              type="application/pdf"
              className="w-full h-full rounded-lg"
              title={name}
            >
              <div className="flex flex-col items-center justify-center h-full text-white/60 gap-4">
                <FileText size={48} />
                <p className="text-sm">{t('chat.pdfNotSupported')}</p>
                <a
                  href={url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 text-white text-sm transition-colors"
                >
                  {t('chat.openInNewTab')}
                </a>
              </div>
            </object>
          </div>
        </div>
      )}
    </>
  )
}
