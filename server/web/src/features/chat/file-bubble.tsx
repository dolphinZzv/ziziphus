import { useTranslation } from 'react-i18next'
import { FileText, Download } from 'lucide-react'
import { formatFileSize } from '@/lib/file'

interface Props { body: string }

export default function FileBubble({ body }: Props) {
  const { t } = useTranslation()
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

  const handleDownload = () => {
    if (url) window.open(url, '_blank')
  }

  return (
    <button
      onClick={handleDownload}
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
  )
}
