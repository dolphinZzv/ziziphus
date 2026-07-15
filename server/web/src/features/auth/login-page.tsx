import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import { getSavedAccounts, saveAccount, removeSavedAccount } from '@/lib/storage'
import { X, Eye, EyeOff } from 'lucide-react'
import AuthFooter from './auth-footer'

export default function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const isLoading = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoading)
  const error = useSyncExternalStore(authStore.subscribe, () => authStore.state.error)
  const mfaChallenge = useSyncExternalStore(authStore.subscribe, () => authStore.state.mfaChallenge)
  const isLoggedIn = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoggedIn)

  const [account, setAccount] = useState('')
  const [password, setPassword] = useState('')
  const [mfaCode, setMfaCode] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [remember, setRemember] = useState(true)
  const [savedAccounts, setSavedAccounts] = useState(getSavedAccounts)

  // Redirect when logged in
  useEffect(() => { if (isLoggedIn) navigate('/chat', { replace: true }) }, [isLoggedIn, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!account.trim() || !password.trim()) return
    try {
      await authStore.login(account.trim(), password)
      if (remember) saveAccount(account.trim())
      setSavedAccounts(getSavedAccounts())
    } catch { /* error handled by store */ }
  }

  const handleMfaVerify = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!mfaCode.trim()) return
    try {
      await authStore.mfaVerify(mfaCode.trim())
      if (isLoggedIn) navigate('/chat', { replace: true })
    } catch { /* error handled by store */ }
  }

  const fillAccount = (acc: string) => setAccount(acc)
  const deleteAccount = (acc: string, e: React.MouseEvent) => {
    e.stopPropagation(); removeSavedAccount(acc); setSavedAccounts(getSavedAccounts())
  }

  const inputClass = 'w-full h-12 px-4 rounded-xl bg-[var(--color-surface-soft)] text-[var(--color-ink)] text-sm placeholder:text-[var(--color-muted)] outline-none border border-transparent focus:border-[var(--color-primary)]/40 focus:bg-[var(--color-surface-card)] transition-colors'

  // MFA verification step
  if (mfaChallenge) {
    const isTOTP = mfaChallenge.mfaType === 1
    const mfaTypeLabel = isTOTP ? t('auth.mfaTOTPLabel') : t('auth.mfaEmailLabel')
    const mfaHint = isTOTP
      ? t('auth.mfaTOTPHint')
      : t('auth.mfaEmailHint') + (mfaChallenge.maskedEmail ? ` (${mfaChallenge.maskedEmail})` : '')
    return (
      <div className="h-full flex flex-col items-center justify-center bg-[var(--color-canvas)] relative px-8 gap-8">
        <div className="text-center">
          <h1 className="font-headline text-[28px] font-bold text-[var(--color-ink)]">ziziphus</h1>
          <p className="text-sm text-[var(--color-muted)] mt-2">{t('auth.mfaTitle')}</p>
          <div className="inline-block mt-2 px-3 py-1 rounded-full bg-[var(--color-primary)]/10 text-[var(--color-primary)] text-xs font-medium">
            {mfaTypeLabel}
          </div>
          <p className="text-xs text-[var(--color-muted)] mt-3">{mfaHint}</p>
        </div>
        <form onSubmit={handleMfaVerify} className="w-full max-w-[320px] flex flex-col gap-4">
          <input type="text" value={mfaCode} onChange={e => setMfaCode(e.target.value)}
            placeholder={t('auth.mfaCode')} maxLength={6}
            className={`${inputClass} text-center tracking-[6px] font-mono text-lg`} autoFocus />
          {error && <span className="text-xs text-[var(--destructive)] text-center">{error}</span>}
          <button type="submit" disabled={isLoading || !mfaCode.trim()}
            className="w-full h-11 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40">
            {isLoading ? t('auth.verifying') : t('auth.verify')}
          </button>
        </form>
        <AuthFooter />
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col items-center justify-center bg-[var(--color-canvas)] relative px-8 gap-8">
      {/* Logo */}
      <div className="text-center">
        <h1 className="font-headline text-[28px] font-bold text-[var(--color-ink)]">ziziphus</h1>
      </div>

      <form onSubmit={handleSubmit} className="w-full max-w-[320px] flex flex-col gap-4">
        {/* Account */}
        <div className="relative">
          <input type="text" value={account} onChange={e => setAccount(e.target.value)}
            placeholder={t("auth.account")} className={inputClass} autoComplete="username" />
          {savedAccounts.length > 0 && !account && (
            <div className="absolute top-full left-0 right-0 mt-1 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl z-10 overflow-hidden"
              style={{ boxShadow: 'var(--shadow-md)' }}>
              {savedAccounts.map(acc => (
                <button key={acc} type="button" onClick={() => fillAccount(acc)}
                  className="w-full flex items-center justify-between px-4 py-2.5 hover:bg-[var(--color-surface-soft)] text-sm text-[var(--color-ink)]">
                  {acc}
                  <X size={13} className="text-[var(--color-muted)] hover:text-[var(--destructive)]" onClick={e => deleteAccount(acc, e)} />
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Password */}
        <div className="relative">
          <input type={showPassword ? 'text' : 'password'} value={password} onChange={e => setPassword(e.target.value)}
            placeholder={t("auth.password")} className={`${inputClass} pr-10`} autoComplete="current-password" />
          <button type="button" onClick={() => setShowPassword(!showPassword)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)] hover:text-[var(--color-ink)]">
            {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
          </button>
        </div>

        {/* Remember + Error */}
        <div className="flex items-center justify-between">
          <label className="flex items-center gap-1.5 text-xs text-[var(--color-muted)] cursor-pointer">
            <input type="checkbox" checked={remember} onChange={e => setRemember(e.target.checked)}
              className="w-3.5 h-3.5 rounded accent-[var(--color-primary)]" />
            {t("auth.rememberAccount")}
          </label>
          {error && <span className="text-xs text-[var(--destructive)]">{error}</span>}
        </div>

        {/* Submit */}
        <button type="submit" disabled={isLoading}
          className="w-full h-11 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40">
          {isLoading ? t('auth.loggingIn') : t('auth.login')}
        </button>
      </form>

      <Link to="/register" className="text-xs text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors">
        {t('auth.switchToRegister')}
      </Link>

      <AuthFooter />
    </div>
  )
}
