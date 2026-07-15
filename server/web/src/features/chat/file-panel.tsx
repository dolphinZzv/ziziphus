import { useEffect, useState, useRef } from 'react'
import { fileService } from '@/services/file-service'
import type { ConvFileInfo } from '@/services/file-service'
import { api } from '@/services/api-client'
import { formatTime } from '@/lib/time'
import { X, Upload, Download, FileText, Image, Film, Folder, FolderPlus, ChevronRight, MoreHorizontal, Info, Edit2 } from 'lucide-react'

interface Props { convId: string; onClose: () => void; width?: number; onWidthChange?: (w: number) => void }
interface FolderNode { name: string; path: string; mod_time: number }

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`
}

const TYPE_ICONS = [Image, FileText, FileText, Film] as const

export default function FilePanel({ convId, onClose, width }: Props) {
  const [entries, setEntries] = useState<ConvFileInfo[]>([])
  const [folders, setFolders] = useState<FolderNode[]>([])
  const [currentPath, setCurrentPath] = useState('') // "" = root
  const [loading, setLoading] = useState(true)
  const [uploading, setUploading] = useState(false)
  const [newFolderInput, setNewFolderInput] = useState(false)
  const [newFolderName, setNewFolderName] = useState('')
  const [menuFileId, setMenuFileId] = useState<string | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const [dragFolderOver, setDragFolderOver] = useState<string | null>(null)
  const [infoFile, setInfoFile] = useState<ConvFileInfo | null>(null)
  const [renamePath, setRenamePath] = useState<string | null>(null)
  const [renameName, setRenameName] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const loadData = async () => {
    setLoading(true)
    try {
      const [dirs, fls] = await Promise.all([
        api.request<FolderNode[]>(`/api/v1/conversations/${convId}/folders?parent_path=${encodeURIComponent(currentPath)}`),
        currentPath === ''
          ? api.request<{ items: ConvFileInfo[]; total: number }>(`/api/v1/conversations/${convId}/files`)
          : api.request<{ items: ConvFileInfo[]; total: number }>(`/api/v1/conversations/${convId}/folders/files?path=${encodeURIComponent(currentPath)}`),
      ])
      setFolders(dirs)
      setEntries(fls.items)
    } catch {}
    setLoading(false)
  }

  useEffect(() => { loadData() }, [convId, currentPath])

  const uploadFile = async (file: File) => {
    setUploading(true)
    try { await fileService.upload(file, file.name, file.type.startsWith('image/') ? 0 : 1, undefined, convId, currentPath); loadData() } catch {}
    setUploading(false)
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    await uploadFile(file); e.target.value = ''
  }

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault(); setDragOver(false)
    const files = e.dataTransfer?.files
    if (!files) return
    for (let i = 0; i < files.length; i++) await uploadFile(files[i])
  }

  const handleDragOver = (e: React.DragEvent) => { e.preventDefault(); setDragOver(true) }
  const handleDragLeave = () => setDragOver(false)

  const goRoot = () => setCurrentPath('')

  const createFolder = async () => {
    const n = newFolderName.trim()
    if (!n) return
    try {
      await api.request(`/api/v1/conversations/${convId}/folders`, { method: 'POST', body: { name: n, parent_path: currentPath } })
      setNewFolderName(''); setNewFolderInput(false)
      loadData()
    } catch {}
  }

  const deleteFile = async (id: string) => {
    try { await fileService.deleteConvFile(convId, id); setEntries(f => f.filter(x => x.file_id !== id)) } catch {}
    setMenuFileId(null)
  }

  const deleteFolder = async (path: string) => {
    if (!confirm('删除此文件夹？')) return
    try { await fileService.deleteFolder(convId, path); loadData() } catch {}
  }

  // Breadcrumb segments
  const pathSegments = currentPath ? currentPath.split('/').filter(Boolean) : []
  const breadcrumbs: { label: string; path: string }[] = [{ label: '/', path: '' }]
  let buildPath = ''
  for (const seg of pathSegments) {
    buildPath = buildPath ? `${buildPath}/${seg}` : seg
    breadcrumbs.push({ label: seg, path: buildPath })
  }

  const rowClass = 'flex items-center gap-2 px-3 h-10 hover:bg-[var(--color-surface-soft)] transition-colors cursor-pointer group rounded'

  return (
    <div className="bg-[var(--color-surface-card)] border-l border-[var(--color-hairline)] flex flex-col h-full flex-shrink-0 text-sm relative" style={{ width: width || 260 }}
      onDragOver={handleDragOver} onDragLeave={handleDragLeave} onDrop={handleDrop}>
      {/* Header */}
      <div className="h-12 flex items-center justify-between px-4 border-b border-[var(--color-hairline)] flex-shrink-0">
        <div className="flex items-center gap-2 text-[13px] font-medium text-[var(--color-ink)]">
          文件 <span className="text-[var(--color-muted)] font-normal">{entries.length + folders.length}</span>
        </div>
        <div className="flex items-center gap-0.5">
          <button onClick={() => setNewFolderInput(true)} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]" title="新建文件夹">
            <FolderPlus size={15} />
          </button>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={15} /></button>
        </div>
      </div>

      {/* New folder input */}
      {newFolderInput && (
        <div className="px-3 py-2 border-b border-[var(--color-hairline)]">
          <div className="flex gap-1.5">
            <input type="text" value={newFolderName} onChange={e => setNewFolderName(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') createFolder(); if (e.key === 'Escape') setNewFolderInput(false) }}
              placeholder="文件夹名称" className="flex-1 h-8 px-2.5 rounded-md text-xs bg-[var(--color-surface-soft)] border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)]" autoFocus />
            <button onClick={createFolder} className="px-3 h-8 rounded-md text-xs font-medium bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)] transition-colors">创建</button>
          </div>
        </div>
      )}

      {/* Breadcrumb path */}
      <div className="flex items-center gap-1 px-3 py-2 border-b border-[var(--color-hairline)] text-[11px] text-[var(--color-muted)] overflow-x-auto">
        {breadcrumbs.map((b, i) => (
          <span key={b.path} className="flex items-center gap-1">
            {i > 0 && <ChevronRight size={10} />}
            <button onClick={() => setCurrentPath(b.path)}
              className={`flex items-center gap-1 hover:text-[var(--color-ink)] whitespace-nowrap ${b.path === currentPath ? 'text-[var(--color-ink)] font-medium' : ''}`}>
              {i === 0 ? <Folder size={14} className="text-[var(--color-accent)] flex-shrink-0" /> : null}
              {b.label}
            </button>
          </span>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto py-1">
        {loading ? (
          <div className="px-3 py-8 text-xs text-center text-[var(--color-muted)]">加载中...</div>
        ) : folders.length === 0 && entries.length === 0 ? (
          <div className="px-3 py-12 text-xs text-center text-[var(--color-muted)] space-y-2">
            <Folder size={28} className="mx-auto opacity-20" />
            <p>暂无文件</p>
          </div>
        ) : (
          <>
            {/* Folders */}
            {folders.map(f => {
              const isDropTarget = dragFolderOver === f.path
              const isRenaming = renamePath === f.path
              return (
                <div key={f.path}>
                  <div className={`${rowClass} ${isDropTarget ? 'bg-[var(--color-primary)]/10 ring-1 ring-[var(--color-primary)]' : ''}`}
                    draggable
                    onDragStart={e => { e.dataTransfer.setData('text/folder', f.path); e.dataTransfer.effectAllowed = 'move' }}
                    onDragOver={e => { e.preventDefault(); e.stopPropagation(); setDragFolderOver(f.path) }}
                    onDragLeave={() => setDragFolderOver(null)}
                    onDrop={async e => {
                      e.preventDefault(); setDragFolderOver(null)
                      const fileId = e.dataTransfer.getData('text/file')
                      const srcFolder = e.dataTransfer.getData('text/folder')
                      if (fileId) { try { await fileService.moveFile(convId, fileId, f.path); } catch {} }
                      if (srcFolder && srcFolder !== f.path) { try { await fileService.moveFolder(convId, srcFolder, f.path); } catch {} }
                      loadData()
                    }}
                    onClick={() => setCurrentPath(f.path)}>
                    <Folder size={15} className="text-[var(--color-accent)] flex-shrink-0" />
                    {isRenaming ? (
                      <input type="text" value={renameName}
                        onChange={e => setRenameName(e.target.value)}
                        onKeyDown={async e => {
                          if (e.key === 'Enter') { e.stopPropagation(); const n = renameName.trim(); if (n) { await fileService.renameFolder(convId, f.path, n); setRenamePath(null); loadData() } }
                          if (e.key === 'Escape') { e.stopPropagation(); setRenamePath(null) }
                        }}
                        onBlur={() => setRenamePath(null)}
                        onClick={e => e.stopPropagation()}
                        className="flex-1 h-6 px-2 rounded text-xs bg-[var(--color-surface-soft)] border border-[var(--color-primary)] focus:outline-none" autoFocus />
                    ) : (
                      <span className="flex-1 text-[13px] truncate">{f.name}</span>
                    )}
                    <div className="relative flex items-center">
                      <button onClick={e => { e.stopPropagation(); setMenuFileId(menuFileId === `folder:${f.path}` ? null : `folder:${f.path}`) }}
                        className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-[var(--color-hairline)] text-[var(--color-muted)]">
                        <MoreHorizontal size={13} />
                      </button>
                      {menuFileId === `folder:${f.path}` && (
                        <>
                          <div className="fixed inset-0 z-10" onClick={() => setMenuFileId(null)} />
                          <div className="absolute right-0 top-7 w-28 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg z-20 py-1 shadow-lg text-xs"
                            style={{ boxShadow: 'var(--shadow-md)' }}>
                            <button onClick={e => { e.stopPropagation(); setRenamePath(f.path); setRenameName(f.name); setMenuFileId(null) }}
                              className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                              <Edit2 size={11} /> 重命名
                            </button>
                            <button onClick={e => { e.stopPropagation(); deleteFolder(f.path); setMenuFileId(null) }}
                              className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-[var(--destructive)]/10 text-[var(--destructive)]">
                              <X size={11} /> 删除
                            </button>
                          </div>
                        </>
                      )}
                    </div>
                  </div>
                </div>
              )
            })}

            {/* Files */}
            {entries.map(f => {
              const Icon = TYPE_ICONS[f.content_type] || FileText
              return (
                <div key={f.file_id} className={rowClass}
                  draggable
                  onDragStart={e => { e.dataTransfer.setData('text/file', f.file_id); e.dataTransfer.effectAllowed = 'move' }}>
                  <Icon size={14} className="text-[var(--color-muted)] flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="text-[13px] truncate">{f.name}</div>
                    <div className="text-[10px] text-[var(--color-muted)]">
                      {formatSize(f.size)}
                      {f.uploader_name && ` · ${f.uploader_name}`}
                    </div>
                  </div>
                  <div className="relative flex items-center">
                    <button onClick={e => { e.stopPropagation(); setMenuFileId(menuFileId === f.file_id ? null : f.file_id) }}
                      className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-[var(--color-hairline)] text-[var(--color-muted)]">
                      <MoreHorizontal size={13} />
                    </button>
                    {menuFileId === f.file_id && (
                      <>
                        <div className="fixed inset-0 z-10" onClick={() => setMenuFileId(null)} />
                        <div className="absolute right-0 top-7 w-36 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg z-20 py-1 shadow-lg text-xs"
                          style={{ boxShadow: 'var(--shadow-md)' }}>
                          <a href={f.url} download={f.name}
                            className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                            <Download size={11} /> 下载
                          </a>
                          <button onClick={() => { setInfoFile(f); setMenuFileId(null) }}
                            className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                            <Info size={11} /> 文件简介
                          </button>
                          <button onClick={() => { deleteFile(f.file_id); setMenuFileId(null) }}
                            className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-[var(--destructive)]/10 text-[var(--destructive)]">
                            <X size={11} /> 删除
                          </button>
                        </div>
                      </>
                    )}
                  </div>
                </div>
              )
            })}
          </>
        )}
      </div>

      {/* File info modal */}
      {infoFile && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/20" onClick={() => setInfoFile(null)}>
          <div className="w-56 bg-[var(--color-surface-card)] rounded-xl p-5 shadow-xl" onClick={e => e.stopPropagation()} style={{ boxShadow: 'var(--shadow-lg)' }}>
            <div className="flex items-start justify-between mb-3">
              <div className="text-[13px] font-medium text-[var(--color-ink)] break-all">{infoFile.name}</div>
              <button onClick={() => setInfoFile(null)} className="p-1 rounded hover:bg-[var(--color-surface-soft)] flex-shrink-0"><X size={13} /></button>
            </div>
            <div className="space-y-1.5 text-[11px] text-[var(--color-muted)]">
              <div className="flex justify-between"><span>大小</span><span className="text-[var(--color-ink)]">{formatSize(infoFile.size)}</span></div>
              <div className="flex justify-between"><span>类型</span><span className="text-[var(--color-ink)]">{['图片','文件','音频','视频'][infoFile.content_type] || '文件'}</span></div>
              <div className="flex justify-between"><span>上传者</span><span className="text-[var(--color-ink)]">{infoFile.uploader_name || infoFile.uploader_id}</span></div>
              <div className="flex justify-between"><span>时间</span><span className="text-[var(--color-ink)]">{formatTime(infoFile.created_at)}</span></div>
              {infoFile.width && infoFile.height && (
                <div className="flex justify-between"><span>尺寸</span><span className="text-[var(--color-ink)]">{infoFile.width} × {infoFile.height}</span></div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Drag overlay */}
      {dragOver && (
        <div className="absolute inset-0 z-50 bg-[var(--color-primary)]/10 border-2 border-dashed border-[var(--color-primary)] rounded flex items-center justify-center pointer-events-none">
          <div className="text-center text-[var(--color-primary)]">
            <Upload size={28} className="mx-auto mb-1" />
            <p className="text-xs font-medium">释放以上传</p>
          </div>
        </div>
      )}

      {/* Upload */}
      <div className="px-3 pb-3 pt-2 border-t border-[var(--color-hairline)]">
        <button onClick={() => fileInputRef.current?.click()} disabled={uploading}
          className="w-full h-8 rounded-md bg-[var(--color-surface-soft)] hover:bg-[var(--color-hairline)] text-[13px] text-[var(--color-muted)] flex items-center justify-center gap-1.5 transition-colors">
          <Upload size={13} /> {uploading ? '上传中...' : '上传文件'}
        </button>
        <input ref={fileInputRef} type="file" onChange={handleUpload} className="hidden" />
      </div>
    </div>
  )
}
