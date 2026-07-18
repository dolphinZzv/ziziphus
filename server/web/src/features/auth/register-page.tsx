import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import PageLayout from '@/components/page-layout'
import AuthFooter from './auth-footer'

export default function RegisterPage() {
  const { t } = useTranslation()
  const isLoading = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoading)
  const error = useSyncExternalStore(authStore.subscribe, () => authStore.state.error)

  const [account, setAccount] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [appName, setAppName] = useState('Ziziphus')
  const [email, setEmail] = useState('')
  const [localError, setLocalError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLocalError('')
    if (!account.trim()) { setLocalError(t('auth.accountRequired', '请填写账号')); return }
    if (!name.trim()) { setLocalError(t('auth.nameRequired', '请填写昵称')); return }
    if (!password.trim()) { setLocalError(t('auth.passwordRequired', '请填写密码')); return }
    if (password.trim().length < 8) { setLocalError(t('auth.passwordTooShort', '密码至少8位')); return }
    if (password !== confirmPassword) { setLocalError(t('auth.passwordMismatch')); return }
    try { await authStore.register(account.trim(), name.trim(), password.trim(), email.trim() || undefined) } catch (e) { console.error(e) }
  }

  const displayError = localError || error

  const inputClass = 'w-full h-12 px-4 rounded-xl bg-[var(--color-surface-soft)] text-[var(--color-ink)] text-sm placeholder:text-[var(--color-muted)] outline-none border border-transparent focus:border-[var(--color-primary)]/40 focus:bg-[var(--color-surface-card)] transition-colors'

  useEffect(() => {
    fetch('/api/v1/app/info')
      .then(r => r.json())
      .then(d => {
        if (d.data?.name) setAppName(d.data.name)
        if (d.data?.headline) setAppHeadline(d.data.headline)
      })
      .catch(() => { /* use default */ })
  }, [])

  return (
    <PageLayout>
      <div className="w-full max-w-[340px] flex flex-col items-center gap-8">
        {/* Z Logo */}
        <div className="flex flex-col items-center gap-3">
          <div className="w-14 h-14 rounded-2xl flex items-center justify-center text-white text-3xl font-extrabold font-headline tracking-tighter"
            style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-accent))' }}>
            Z
          </div>
          <h1 className="font-headline text-xl font-bold text-[var(--color-ink)] tracking-tight">{appName}</h1>
        </div>

      <form onSubmit={handleSubmit} className="w-full flex flex-col gap-3">
        <input type="text" value={account} onChange={e => setAccount(e.target.value)}
          placeholder={t("auth.account")} className={inputClass} autoComplete="username" />
        <input type="text" value={name} onChange={e => setName(e.target.value)}
          placeholder={t("auth.name")} className={inputClass} />
        <input type="email" value={email} onChange={e => setEmail(e.target.value)}
          placeholder={t("auth.email")} className={inputClass} autoComplete="email" />
        <input type="password" value={password} onChange={e => setPassword(e.target.value)}
          placeholder={t("auth.password")} className={inputClass} autoComplete="new-password" />
        <input type="password" value={confirmPassword} onChange={e => setConfirmPassword(e.target.value)}
          placeholder={t("auth.confirmPassword")} className={inputClass} autoComplete="new-password" />

        {displayError && (
          <div className="text-xs text-[var(--destructive)] bg-[var(--destructive)]/10 rounded-xl px-3 py-2">{displayError}</div>
        )}

        <button type="submit" disabled={isLoading}
          className="w-full h-11 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40">
          {isLoading ? t('auth.registering') : t('auth.register')}
        </button>
      </form>
      </div>

      <Link to="/login" className="text-xs text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors">
        {t('auth.switchToLogin')}
      </Link>

      <AuthFooter />
    </PageLayout>
  )
}
