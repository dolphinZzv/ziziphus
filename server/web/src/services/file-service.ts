import { uploadFile as upload } from './api-client'
import { api } from './api-client'


export interface ConvFileInfo {
  file_id: string
  url: string
  size: number
  name: string
  content_type: number
  width?: number
  height?: number
  uploader_id: string
  uploader_name?: string
  conv_id?: string
  created_at: number
}

export const fileService = {
  upload(fileData: Blob, fileName: string, fileType: number, onProgress?: (p: number) => void, convId?: string, folderId?: number) {
    return upload(fileData, fileName, fileType, onProgress, convId, folderId)
  },

  getFileUrl(fileId: string): string {
    const base = (window as unknown as Record<string, string>).__API_BASE__ || ''
    return `${base}/files/${fileId}`
  },

  listConvFiles(convId: string, page = 1, size = 50) {
    return api.request<{ items: ConvFileInfo[]; total: number; page: number; size: number }>(
      `/api/v1/conversations/${convId}/files`,
      { query: { page, size } }
    )
  },

  deleteConvFile(convId: string, fileId: string) {
    return api.request(`/api/v1/conversations/${convId}/files/${fileId}`, { method: 'DELETE' })
  },

  deleteFolder(convId: string, folderId: number) {
    return api.request(`/api/v1/conversations/${convId}/folders/${folderId}`, { method: 'DELETE' })
  },

  moveFile(convId: string, fileId: string, folderId: number) {
    return api.request(`/api/v1/conversations/${convId}/files/${fileId}/move`, { method: 'PUT', body: { folder_id: folderId } })
  },

  moveFolder(convId: string, folderId: number, parentId: number) {
    return api.request(`/api/v1/conversations/${convId}/folders/${folderId}/move`, { method: 'PUT', body: { parent_id: parentId } })
  },

  renameFolder(convId: string, folderId: number, name: string) {
    return api.request(`/api/v1/conversations/${convId}/folders/${folderId}/rename`, { method: 'PUT', body: { name } })
  },
}
