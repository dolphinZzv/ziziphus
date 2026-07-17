import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import { getSavedAccounts, saveAccount, removeSavedAccount } from '@/lib/storage'
import { X, Eye, EyeOff } from 'lucide-react'
import PageLayout from '@/components/page-layout'
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
  const [remember] = useState(true)
  const [localError, setLocalError] = useState('')
  const [savedAccounts, setSavedAccounts] = useState(getSavedAccounts)
  const [appName, setAppName] = useState('Ziziphus')
  const [appHeadline, setAppHeadline] = useState('')

  // Fetch app info from server
  useEffect(() => {
    fetch('/api/v1/app/info')
      .then(r => r.json())
      .then(d => {
        if (d.data?.name) setAppName(d.data.name)
        if (d.data?.headline) setAppHeadline(d.data.headline)
      })
      .catch(() => { /* use default */ })
  }, [])

  // Redirect when logged in
  useEffect(() => { if (isLoggedIn) navigate('/conversations', { replace: true }) }, [isLoggedIn, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!account.trim()) { setLocalError(t('auth.accountRequired', '请填写账号')); return }
    if (!password.trim()) { setLocalError(t('auth.passwordRequired', '请填写密码')); return }
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
      if (isLoggedIn) navigate('/conversations', { replace: true })
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
      <PageLayout>
        <div className="w-full max-w-[340px] flex flex-col items-center gap-6">
          {/* Z Logo */}
          <div className="flex flex-col items-center gap-3">
            <div className="w-14 h-14 rounded-2xl flex items-center justify-center text-white text-3xl font-extrabold font-headline tracking-tighter"
              style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-accent))' }}>
              Z
            </div>
            <h1 className="font-headline text-xl font-bold text-[var(--color-ink)] tracking-tight">{appName}</h1>
          </div>

          <p className="text-sm text-[var(--color-muted)] -mt-3">{t('auth.mfaTitle')}</p>
          <div className="px-3 py-1 rounded-full bg-[var(--color-primary)]/10 text-[var(--color-primary)] text-xs font-medium">
            {mfaTypeLabel}
          </div>
          <p className="text-xs text-[var(--color-muted)] -mt-2">{mfaHint}</p>

          {/* MFA code input */}
          <form onSubmit={handleMfaVerify} className="w-full flex flex-col gap-3">
          <input
            className={inputClass}
            type="text"
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            placeholder={t('auth.mfaCode') || '验证码'}
            value={mfaCode}
            onChange={e => setMfaCode(e.target.value)}
            autoFocus
          />
          <button
            type="submit"
            disabled={!mfaCode.trim() || isLoading}
            className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 disabled:opacity-50 transition-all cursor-pointer"
          >
            {isLoading ? t('auth.verifying') : t('auth.verify') || '验证'}
          </button>
          {error && <p className="text-xs text-red-500 text-center">{error}</p>}
          </form>
        </div>
      </PageLayout>
    )
  }

  return (
    <PageLayout>
      <div className="w-full max-w-[340px] flex flex-col items-center gap-8">
        {/* Z Logo */}
        <div className="flex flex-col items-center gap-3">
          <div className="w-14 h-14 rounded-2xl flex items-center justify-center text-white text-3xl font-extrabold font-headline tracking-tighter"
            style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-accent))' }}>
            Z
          </div>
          <h1 className="font-headline text-xl font-bold text-[var(--color-ink)] tracking-tight">
            {appName}
          </h1>
          {appHeadline && (
            <p className="text-[13px] text-[var(--color-muted)] -mt-1">{appHeadline}</p>
          )}
        </div>

      <form onSubmit={handleSubmit} className="w-full flex flex-col gap-3">
        {/* Account */}
        <div className="relative">
          <input
            className={inputClass + ' pr-10'}
            type="text"
            placeholder={t('auth.account')}
            value={account}
            onChange={e => setAccount(e.target.value)}
            disabled={isLoading}
            autoFocus
            autoComplete="username"
            onFocus={() => setSavedAccounts(getSavedAccounts())}
          />
          {/* Saved accounts dropdown */}
          {!account && savedAccounts.length > 0 && !isLoading && (
            <div className="absolute top-full left-0 right-0 mt-1 rounded-xl bg-[var(--color-surface-card)] border border-[var(--color-border)] shadow-lg z-10 overflow-hidden">
              {savedAccounts.map(acc => (
                <button
                  key={acc}
                  type="button"
                  className="w-full px-4 py-2.5 text-left text-sm text-[var(--color-ink)] hover:bg-[var(--color-surface-soft)] transition-colors flex items-center justify-between group"
                  onClick={() => fillAccount(acc)}
                >
                  <span>{acc}</span>
                  <span
                    className="text-[var(--color-muted)] hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                    onClick={e => deleteAccount(acc, e)}
                  >
                    <X size={14} />
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Password */}
        <div className="relative">
          <input
            className={inputClass + ' pr-10'}
            type={showPassword ? 'text' : 'password'}
            placeholder={t('auth.password')}
            value={password}
            onChange={e => setPassword(e.target.value)}
            disabled={isLoading}
            autoComplete="current-password"
          />
          <button
            type="button"
            className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
            onClick={() => setShowPassword(!showPassword)}
            tabIndex={-1}
          >
            {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
          </button>
        </div>

        {/* Login button */}
        <button
          type="submit"
          disabled={isLoading}
          className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 disabled:opacity-50 transition-all cursor-pointer mt-1"
        >
          {isLoading ? t('auth.loggingIn') : t('auth.login')}
        </button>

        {/* Error */}
        {(localError || error) && <p className="text-xs text-red-500 text-center">{localError || error}</p>}

        {/* Forgot password link */}
        <div className="flex justify-end -mt-2">
          <Link to="/forgot-password" className="text-xs text-[var(--color-primary)] hover:underline">
            {t('auth.forgotPassword', '忘记密码？')}
          </Link>
        </div>

        {/* Register link */}
        <p className="text-xs text-[var(--color-muted)] text-center">
          {t('auth.noAccount')}{' '}
          <Link to="/register" className="text-[var(--color-primary)] hover:underline font-medium">
            {t('auth.register')}
          </Link>
        </p>
      </form>
      </div>

      <AuthFooter />
    </PageLayout>
  )
}
