import { useEffect, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { conversationService } from '@/services/conversation-service'
import { fileService } from '@/services/file-service'
import { avatarUrl } from '@/lib/file'
import { getConvSettings, toggleConvSetting, subscribe as settingsSubscribe } from '@/stores/conversation-settings-store'
import { X, EyeOff, FileUp, Search, ArrowLeft, Image, Trash2 } from 'lucide-react'
import { useIsMobile } from '@/hooks/use-breakpoint'

interface Props { convId: string; onClose: () => void }

export default function GroupSettings({ convId, onClose }: Props) { const isMobile=useIsMobile()
  const { t } = useTranslation()
  const [showAgentResponseOnly, setShowAgentResponseOnly] = useState(
    () => getConvSettings(convId).showAgentResponseOnly
  )
  useEffect(() => {
    return settingsSubscribe(() => {
      setShowAgentResponseOnly(getConvSettings(convId).showAgentResponseOnly)
    })
  }, [convId])

  const [fileChangeNotify, setFileChangeNotify] = useState(false)
  const [discoverable, setDiscoverable] = useState(true)
  const [bgImage, setBgImage] = useState('')
  const [uploadingBg, setUploadingBg] = useState(false)
  const bgInputRef = useRef<HTMLInputElement>(null)
  useEffect(() => {
    conversationService.getSettings(convId).then(res => {
      if (res.settings?.fileChangeNotify) setFileChangeNotify(true)
      setDiscoverable(res.settings?.discoverable !== false)
      if (res.settings?.background_image) setBgImage(res.settings.background_image as string)
    }).catch(() => {})
  }, [convId])

  const handleFileChangeNotifyToggle = async () => {
    const newVal = !fileChangeNotify
    setFileChangeNotify(newVal)
    try {
      await conversationService.updateSettings(convId, { fileChangeNotify: newVal })
    } catch {
      setFileChangeNotify(!newVal)
    }
  }

  const handleDiscoverableToggle = async () => {
    const newVal = !discoverable
    setDiscoverable(newVal)
    try {
      await conversationService.updateSettings(convId, { discoverable: newVal })
    } catch {
      setDiscoverable(!newVal)
    }
  }

  const handleBgUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploadingBg(true)
    try {
      const r = await fileService.upload(file, file.name, 0)
      await conversationService.updateSettings(convId, { background_image: r.url })
      setBgImage(r.url)
    } catch (e) { console.warn('upload bg failed:', e) }
    setUploadingBg(false)
    e.target.value = ''
  }

  const updateSetting = async (key: string, value: any) => {
    try {
      const settings = await conversationService.getSettings(convId)
      const merged = { ...settings.settings, [key]: value }
      await conversationService.updateSettings(convId, merged as any)
    } catch {}
  }

  const handleBgRemove = async () => {
    setBgImage('')
    await updateSetting('background_image', '')
  }

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[360px] h-full sm:h-auto bg-[var(--color-surface-card)] rounded-none sm:rounded-xl overflow-hidden"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4">
          <h3 className="font-headline text-base font-semibold text-[var(--color-ink)]">{t('group.settingsTitle')}</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">{isMobile ? <ArrowLeft size={18} /> : <X size={16} />}</button>
        </div>

        <div className="px-5 py-4 space-y-4">
          {/* Agent display settings */}
          <label className="flex items-center justify-between cursor-pointer">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <EyeOff size={16} className="text-[var(--color-muted)] flex-shrink-0" />
              <div>
                <div className="text-sm font-medium text-[var(--color-ink)]">{t('conversation.agentDisplay')}</div>
                <div className="text-xs text-[var(--color-muted-soft)]">{t('conversation.agentDisplayHint')}</div>
              </div>
            </div>
            <button onClick={() => toggleConvSetting(convId, 'showAgentResponseOnly')}
              className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0 ml-3 ${showAgentResponseOnly ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-hairline)]'}`}>
              <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${showAgentResponseOnly ? 'left-[18px]' : 'left-0.5'}`} />
            </button>
          </label>

          {/* File change notification settings */}
          <label className="flex items-center justify-between cursor-pointer">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <FileUp size={16} className="text-[var(--color-muted)] flex-shrink-0" />
              <div>
                <div className="text-sm font-medium text-[var(--color-ink)]">{t('conversation.fileChangeNotify')}</div>
                <div className="text-xs text-[var(--color-muted-soft)]">{t('conversation.fileChangeNotifyHint')}</div>
              </div>
            </div>
            <button onClick={handleFileChangeNotifyToggle}
              className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0 ml-3 ${fileChangeNotify ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-hairline)]'}`}>
              <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${fileChangeNotify ? 'left-[18px]' : 'left-0.5'}`} />
            </button>
          </label>

          {/* Discoverable settings */}
          <label className="flex items-center justify-between cursor-pointer">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <Search size={16} className="text-[var(--color-muted)] flex-shrink-0" />
              <div>
                <div className="text-sm font-medium text-[var(--color-ink)]">{t('conversation.discoverable')}</div>
                <div className="text-xs text-[var(--color-muted-soft)]">{t('conversation.discoverableHint')}</div>
              </div>
            </div>
            <button onClick={handleDiscoverableToggle}
              className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0 ml-3 ${discoverable ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-hairline)]'}`}>
              <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${discoverable ? 'left-[18px]' : 'left-0.5'}`} />
            </button>
          </label>

          {/* Background image */}
          <div>
            <div className="flex items-center gap-2 mb-2">
              <Image size={16} className="text-[var(--color-muted)] flex-shrink-0" />
              <span className="text-xs font-medium text-[var(--color-muted)]">{t('conversation.bgImage') || '聊天背景'}</span>
            </div>
            <div className="flex items-center gap-3">
              {bgImage ? (
                <div className="relative w-16 h-16 rounded-xl overflow-hidden border border-[var(--color-hairline)] flex-shrink-0">
                  <img src={avatarUrl(bgImage, 128)} alt="" className="w-full h-full object-cover" />
                  <button onClick={handleBgRemove}
                    className="absolute top-0.5 right-0.5 p-0.5 rounded-full bg-black/40 text-white/80 hover:bg-black/60">
                    <Trash2 size={10} />
                  </button>
                </div>
              ) : (
                <button onClick={() => bgInputRef.current?.click()} disabled={uploadingBg}
                  className="w-16 h-16 rounded-xl border border-dashed border-[var(--color-hairline)] flex items-center justify-center text-[var(--color-muted)] hover:bg-[var(--color-surface-soft)] transition-colors">
                  {uploadingBg ? <span className="text-[10px]">...</span> : <Image size={18} />}
                </button>
              )}
              <span className="text-[11px] text-[var(--color-muted-soft)]">{t('conversation.bgImageHint') || '建议尺寸 1080×1920'}</span>
            </div>
            <input ref={bgInputRef} type="file" accept="image/*" onChange={handleBgUpload} className="hidden" />
          </div>
        </div>
      </div>
    </div>
  )
}
