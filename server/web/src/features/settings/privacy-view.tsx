import { useState, useSyncExternalStore } from 'react'
import { authStore } from '@/stores/auth-store'
import { X } from 'lucide-react'

interface Props { onClose: () => void }

export default function PrivacyView({ onClose }: Props) {
  const user = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const [discoverable, setDiscoverable] = useState(user?.discoverable ?? true)
  const [allowDirectChat, setAllowDirectChat] = useState(user?.allow_direct_chat ?? true)

  const handleToggleDiscoverable = async () => {
    const v = !discoverable
    setDiscoverable(v)
    try { await authStore.updateProfile({ discoverable: v }) } catch { setDiscoverable(!v) }
  }
  const handleToggleDirectChat = async () => {
    const v = !allowDirectChat
    setAllowDirectChat(v)
    try { await authStore.updateProfile({ allow_direct_chat: v }) } catch { setAllowDirectChat(!v) }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[380px] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg p-6"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-headline text-xl font-semibold text-[var(--color-ink)]">用户设置</h3>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)]"><X size={16} /></button>
        </div>

        <div className="space-y-4">
          <div className="bg-[var(--color-surface-soft)] rounded-lg p-4 space-y-4">
            <label className="flex items-center justify-between cursor-pointer">
              <div className="flex-1 min-w-0">
                <span className="text-sm text-[var(--color-body)]">允许通过搜索找到我</span>
                <p className="text-[11px] text-[var(--color-muted)] mt-0.5">关闭后其他用户无法通过搜索查找到你</p>
              </div>
              <button onClick={handleToggleDiscoverable}
                className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0 ml-3 ${discoverable ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-hairline)]'}`}>
                <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${discoverable ? 'left-[18px]' : 'left-0.5'}`} />
              </button>
            </label>
            <div className="border-t border-[var(--color-hairline)]" />
            <label className="flex items-center justify-between cursor-pointer">
              <div className="flex-1 min-w-0">
                <span className="text-sm text-[var(--color-body)]">允许直接发起会话</span>
                <p className="text-[11px] text-[var(--color-muted)] mt-0.5">关闭后需要先成为好友才能发起会话</p>
              </div>
              <button onClick={handleToggleDirectChat}
                className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0 ml-3 ${allowDirectChat ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-hairline)]'}`}>
                <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${allowDirectChat ? 'left-[18px]' : 'left-0.5'}`} />
              </button>
            </label>
          </div>
        </div>
      </div>
    </div>
  )
}
