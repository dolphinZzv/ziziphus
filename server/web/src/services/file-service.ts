import { uploadFile as upload } from './api-client'
import type { FileInfo } from '@/types/file'

export const fileService = {
  upload(fileData: Blob, fileName: string, fileType: number, onProgress?: (p: number) => void) {
    return upload(fileData, fileName, fileType, onProgress)
  },

  getFileUrl(fileId: string): string {
    const base = (window as unknown as Record<string, string>).__API_BASE__ || ''
    return `${base}/files/${fileId}`
  },
}
