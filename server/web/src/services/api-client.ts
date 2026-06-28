import { getServerUrl, getItem } from '@/lib/storage'
import type { APIResponse } from '@/types/api'

function getBaseURL(): string {
  const custom = getServerUrl()
  return custom || window.location.origin
}

async function request<T>(
  path: string,
  options: {
    method?: string
    body?: unknown
    query?: Record<string, string | number | undefined>
    headers?: Record<string, string>
    retries?: number
  } = {}
): Promise<T> {
  const { method = 'GET', body, query, headers = {}, retries = 0 } = options
  const maxAttempts = retries > 0 ? retries + 1 : 2 // default: 1 initial + 1 retry

  const url = new URL(getBaseURL() + path)
  if (query) {
    Object.entries(query).forEach(([k, v]) => {
      if (v !== undefined) url.searchParams.set(k, String(v))
    })
  }

  const token = getItem<string>('token', '')
  const reqHeaders: Record<string, string> = { 'Content-Type': 'application/json', ...headers }
  if (token) reqHeaders['Authorization'] = `Bearer ${token}`

  let resp: Response
  let lastErr: APIError | null = null
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      resp = await fetch(url.toString(), {
        method,
        headers: reqHeaders,
        body: body ? JSON.stringify(body) : undefined,
      })
            break
    } catch {
      lastErr = new APIError(-1, '网络连接失败，请检查服务器地址')
            if (attempt < maxAttempts - 1) {
        await new Promise(r => setTimeout(r, 1000 * (attempt + 1))) // exponential backoff
      }
          }
  }
  if (!resp!) throw lastErr || new APIError(-1, '请求失败')

  if (resp.status === 401) {
    throw new APIError(401, 'Unauthorized, please login again')
  }

  // Handle non-2xx that aren't JSON
  const text = await resp.text()
  let json: APIResponse<T>
  try {
    json = JSON.parse(text)
  } catch {
    if (!resp.ok) {
      throw new APIError(resp.status, `服务器错误 (${resp.status})`)
    }
    throw new APIError(-1, '服务器返回异常数据')
  }

  if (json.code !== 0) {
    throw new APIError(json.code, json.msg || '请求失败')
  }
  return json.data as T
}

export function uploadFile(
  fileData: Blob,
  fileName: string,
  fileType: number,
  onProgress?: (progress: number) => void
): Promise<{ file_id: string; url: string }> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    const formData = new FormData()
    formData.append('file', fileData, fileName)
    formData.append('file_type', String(fileType))

    const token = getItem<string>('token', '')
    xhr.open('POST', getBaseURL() + '/api/v1/files/upload')
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () => {
      try {
        const json: APIResponse<{ file_id: string; url: string }> = JSON.parse(xhr.responseText)
        if (json.code === 0 && json.data) resolve(json.data)
        else reject(new APIError(json.code, json.msg || '上传失败'))
      } catch { reject(new APIError(-1, '上传失败')) }
    }
    xhr.onerror = () => reject(new APIError(-1, 'Network error'))
    xhr.send(formData)
  })
}

export class APIError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
    this.name = 'APIError'
  }
}

export const api = { request, uploadFile }
