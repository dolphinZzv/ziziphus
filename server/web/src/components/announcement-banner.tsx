import { useEffect, useState } from 'react'
import { X, Megaphone } from 'lucide-react'

interface Announcement {
  enabled: boolean
  title: string
  body: string
  url: string
}

export default function AnnouncementBanner() {
  const [announcement, setAnnouncement] = useState<Announcement | null>(null)
  const [dismissed, setDismissed] = useState(false)

  useEffect(() => {
    fetch('/api/v1/announcement')
      .then(r => r.json())
      .then(json => {
        if (json.code === 0 && json.data?.enabled) {
          setAnnouncement(json.data)
        }
      })
      .catch(() => { /* announcement fetch is non-critical */ })
  }, [])

  if (!announcement || dismissed) return null

  return (
    <div className="flex items-center gap-3 px-4 py-2.5 text-sm"
      style={{ background: 'var(--color-accent)', color: '#fff' }}>
      <Megaphone size={16} />
      <div className="flex-1 min-w-0">
        {announcement.title && <span className="font-semibold mr-2">{announcement.title}</span>}
        <span className="opacity-90">{announcement.body}</span>
        {announcement.url && (
          <a href={announcement.url} target="_blank" rel="noopener noreferrer"
            className="ml-2 underline underline-offset-2 hover:opacity-80 whitespace-nowrap">
            详情
          </a>
        )}
      </div>
      <button onClick={() => setDismissed(true)} className="shrink-0 p-1 rounded hover:bg-white/20">
        <X size={14} />
      </button>
    </div>
  )
}
