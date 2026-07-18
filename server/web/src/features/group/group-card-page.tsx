import { useEffect, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { conversationService } from '@/services/conversation-service'
import { avatarUrl } from '@/lib/file'
import type { GroupCardInfo } from '@/services/conversation-service'
import { Users, Calendar, ArrowRight } from 'lucide-react'

export default function GroupCardPage() {
  const { shareToken } = useParams<{ shareToken: string }>()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [card, setCard] = useState<GroupCardInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    if (!shareToken) {
      setError(true)
      setLoading(false)
      return
    }
    conversationService.getGroupCard(shareToken)
      .then(data => {
        setCard(data)
        setLoading(false)
      })
      .catch(() => {
        setError(true)
        setLoading(false)
      })
  }, [shareToken])

  if (loading) {
    return (
      <div className="min-h-screen bg-[var(--color-bg)] flex items-center justify-center">
        <div className="text-sm text-[var(--color-muted)]">{t('common.loading')}</div>
      </div>
    )
  }

  if (error || !card) {
    return (
      <div className="min-h-screen bg-[var(--color-bg)] flex items-center justify-center p-6">
        <div className="text-center max-w-sm">
          <div className="text-6xl mb-4">🔗</div>
          <h2 className="font-headline text-xl font-semibold text-[var(--color-ink)] mb-2">{t('group.cardNotFoundTitle') || '群组未找到'}</h2>
          <p className="text-sm text-[var(--color-muted)] mb-6">{t('group.cardNotFoundDesc') || '该分享链接无效或群组已解散'}</p>
          <Link to="/login" className="inline-flex items-center gap-2 px-5 h-10 rounded-xl bg-[var(--color-primary)] text-white text-sm font-medium hover:opacity-90 transition-opacity">
            {t('group.goToLogin') || '去登录'} <ArrowRight size={16} />
          </Link>
        </div>
      </div>
    )
  }

  const primaryColor = card.primary_color || '#0EA5E9'

  return (
    <div className="min-h-screen bg-gradient-to-b from-[var(--color-bg)] to-[var(--color-surface-soft)] flex flex-col items-center justify-center p-4 sm:p-6">
      <div className="w-full max-w-sm bg-[var(--color-surface-card)] rounded-2xl overflow-hidden shadow-xl border border-[var(--color-hairline)]"
        style={{ boxShadow: 'var(--shadow-lg)' }}>

        {/* Cover image — extends behind avatar */}
        <div className="h-48 relative"
          style={{
            background: card.cover
              ? `url(${card.cover}?w=600&h=420) center/cover`
              : `linear-gradient(135deg, ${primaryColor}, ${primaryColor}88)`,
          }}>
          {card.cover && <div className="absolute inset-0 bg-black/20" />}
        </div>

        {/* Avatar — sits on cover bottom edge */}
        <div className="flex justify-center -mt-14 mb-3">
          <div className="w-24 h-24 rounded-full border-4 border-[var(--color-surface-card)] overflow-hidden bg-[var(--color-surface-soft)]">
            {card.avatar ? (
              <img src={avatarUrl(card.avatar, 192)} alt={card.name} className="w-full h-full object-cover" />
            ) : (
              <div className="w-full h-full flex items-center justify-center text-white text-3xl font-bold"
                style={{ background: `linear-gradient(135deg, ${primaryColor}, ${primaryColor}88)` }}>
                {card.name?.charAt(0)?.toUpperCase() || 'G'}
              </div>
            )}
          </div>
        </div>

        {/* Group info */}
        <div className="px-6 pb-6 text-center">
          <h1 className="font-headline text-2xl font-bold text-[var(--color-ink)] mb-1">{card.name}</h1>

          {card.headline && (
            <p className="text-sm text-[var(--color-muted)] mb-4">{card.headline}</p>
          )}

          {/* Stats */}
          <div className="flex items-center justify-center gap-4 mb-5 text-xs text-[var(--color-muted)]">
            <span className="flex items-center gap-1.5">
              <Users size={14} />
              {card.member_count} {t('group.members') || '位成员'}
            </span>
            <span className="flex items-center gap-1.5">
              <Calendar size={14} />
              {new Date(card.created_at).toLocaleDateString()}
            </span>
          </div>

          {/* Owner */}
          <div className="text-xs text-[var(--color-muted-soft)] mb-5">
            {t('group.owner') || '群主'}：{card.owner_name}
          </div>

          {/* CTA buttons */}
          <div className="space-y-2">
            <button onClick={() => navigate('/login')}
              className="w-full h-11 rounded-xl text-white text-sm font-medium hover:opacity-90 transition-opacity"
              style={{ background: `linear-gradient(135deg, ${primaryColor}, ${primaryColor}cc)` }}>
              {t('group.joinGroup') || '登录后加入群组'}
            </button>
            <button onClick={() => navigate('/register')}
              className="w-full h-11 rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-body)] hover:bg-[var(--color-surface-soft)] transition-colors">
              {t('group.register') || '注册账号'}
            </button>
          </div>
        </div>
      </div>

      {/* Footer */}
      <p className="mt-6 text-xs text-[var(--color-muted-soft)]">Ziziphus</p>
    </div>
  )
}
