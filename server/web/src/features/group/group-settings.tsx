import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { conversationService } from '@/services/conversation-service'
import { getConvSettings, toggleConvSetting, subscribe as settingsSubscribe } from '@/stores/conversation-settings-store'
import { X, EyeOff, FileUp, Search } from 'lucide-react'

interface Props { convId: string; onClose: () => void }

export default function GroupSettings({ convId, onClose }: Props) {
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
  useEffect(() => {
    conversationService.getSettings(convId).then(res => {
      if (res.settings?.fileChangeNotify) setFileChangeNotify(true)
      // discoverable defaults to true when not set
      setDiscoverable(res.settings?.discoverable !== false)
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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4" onClick={onClose}>
      <div className="w-full sm:w-[360px] bg-[var(--color-surface-card)] rounded-xl overflow-hidden"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--color-hairline)]">
          <h3 className="font-headline text-base font-semibold text-[var(--color-ink)]">{t('group.settingsTitle')}</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
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
        </div>
      </div>
    </div>
  )
}
