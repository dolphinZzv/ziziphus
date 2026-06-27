export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

export function isImageURL(url: string): boolean {
  return /\.(jpg|jpeg|png|gif|webp|heic|bmp|svg)(\?.*)?$/i.test(url)
}

export function getFileIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase()
  const iconMap: Record<string, string> = {
    pdf: 'FileText', doc: 'FileText', docx: 'FileText', txt: 'FileText',
    xls: 'FileSpreadsheet', xlsx: 'FileSpreadsheet', csv: 'FileSpreadsheet',
    ppt: 'FilePresentation', pptx: 'FilePresentation',
    zip: 'FileArchive', rar: 'FileArchive', '7z': 'FileArchive', gz: 'FileArchive',
    mp3: 'FileAudio', wav: 'FileAudio', aac: 'FileAudio', flac: 'FileAudio',
    mp4: 'FileVideo', mov: 'FileVideo', avi: 'FileVideo', mkv: 'FileVideo',
    jpg: 'FileImage', jpeg: 'FileImage', png: 'FileImage', gif: 'FileImage', webp: 'FileImage',
  }
  return iconMap[ext || ''] || 'File'
}

/**
 * avatarUrl appends resize params to an image URL for avatars.
 * Uses 256×256 center-crop to reduce bandwidth while keeping quality.
 */
export function avatarUrl(url: string | undefined | null, size = 256): string | undefined {
  if (!url) return undefined
  // Skip if already has query params (e.g. external URLs) or is SVG
  if (url.includes('?') || url.endsWith('.svg')) return url
  if (url.startsWith('http') && !url.includes('/files/')) return url
  return `${url}?w=${size}&h=${size}`
}
