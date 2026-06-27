export interface FileInfo {
  file_id: string
  url: string
  thumbnail_url?: string
  size: number
  name: string
  content_type: number // 0=image, 1=file, 2=audio, 3=video
  width?: number
  height?: number
  uploader_id: string
  created_at: number
}
