import { useState, useEffect, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { X, ChevronLeft, ChevronRight, Download, ZoomIn, ZoomOut } from 'lucide-react'
import { withFileToken } from '@/lib/file-token'

interface Props {
  images: string[]
  currentIndex: number
  onClose: () => void
}

export default function ImageViewer({ images, currentIndex, onClose }: Props) {
  const [index, setIndex] = useState(currentIndex)
  const [scale, setScale] = useState(1)

  const prev = useCallback(() => setIndex(i => Math.max(0, i - 1)), [])
  const next = useCallback(() => setIndex(i => Math.min(images.length - 1, i + 1)), [images.length])
  const zoomIn = () => setScale(s => Math.min(s * 1.5, 5))
  const zoomOut = () => setScale(s => Math.max(s / 1.5, 0.5))

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
    if (e.key === 'ArrowLeft') prev()
    if (e.key === 'ArrowRight') next()
  }, [onClose, prev, next])

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = ''
    }
  }, [handleKeyDown])

  useEffect(() => { setScale(1) }, [index])

  const viewer = (
    <div className="fixed inset-0 z-50 bg-black/95 flex items-center justify-center">
      {/* Close button */}
      <button onClick={onClose} className="absolute top-4 right-4 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white z-10">
        <X size={20} />
      </button>

      {/* Counter */}
      <div className="absolute top-4 left-1/2 -translate-x-1/2 text-white text-sm">
        {index + 1} / {images.length}
      </div>

      {/* Zoom controls */}
      <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex gap-2 z-10">
        <button onClick={zoomOut} className="p-2 rounded-full bg-white/10 hover:bg-white/20 text-white"><ZoomOut size={18} /></button>
        <button onClick={() => setScale(1)} className="p-2 rounded-full bg-white/10 hover:bg-white/20 text-white text-xs">1:1</button>
        <button onClick={zoomIn} className="p-2 rounded-full bg-white/10 hover:bg-white/20 text-white"><ZoomIn size={18} /></button>
        <button onClick={() => window.open(withFileToken(images[index]), '_blank')} className="p-2 rounded-full bg-white/10 hover:bg-white/20 text-white"><Download size={18} /></button>
      </div>

      {/* Navigation */}
      {images.length > 1 && (
        <>
          <button onClick={prev} disabled={index === 0} className="absolute left-4 top-1/2 -translate-y-1/2 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white disabled:opacity-30"><ChevronLeft size={24} /></button>
          <button onClick={next} disabled={index === images.length - 1} className="absolute right-4 top-1/2 -translate-y-1/2 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white disabled:opacity-30"><ChevronRight size={24} /></button>
        </>
      )}

      {/* Image */}
      <img loading="lazy" decoding="async"
        src={withFileToken(images[index])}
        alt=""
        className="max-w-[90vw] max-h-[85vh] object-contain transition-transform duration-200 select-none"
        style={{ transform: `scale(${scale})` }}
        draggable={false}
        onClick={() => setScale(s => s === 1 ? 2 : 1)}
      />
    </div>
  )

  return createPortal(viewer, document.body)
}
