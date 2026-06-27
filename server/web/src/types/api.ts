export interface APIResponse<T = unknown> {
  code: number
  msg: string
  data: T | null
}

export interface PaginatedData<T = unknown> {
  items: T[]
  total: number
  page: number
  size: number
}
