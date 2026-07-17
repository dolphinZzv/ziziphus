import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import ProfileView from './profile-view'

export default function ProfilePage() {
  const navigate = useNavigate()
  const { t } = useTranslation()

  return (
    <div className="h-full w-full flex flex-col bg-[var(--color-surface-card)]">
      {/* Top navigation bar */}
      <div className="flex items-center h-12 px-3 border-b border-[var(--color-hairline)] bg-[var(--color-canvas)] flex-shrink-0">
        <button
          onClick={() => navigate(-1)}
          className="p-2 -ml-2 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-ink)] transition-colors flex items-center gap-2"
        >
          <ArrowLeft size={18} />
          <span className="text-sm font-medium">{t('profile.profile', '个人资料')}</span>
        </button>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto">
        <ProfileView variant="page" />
      </div>
    </div>
  )
}
