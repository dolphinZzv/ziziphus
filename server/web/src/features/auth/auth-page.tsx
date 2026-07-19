import { useEffect, useRef, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import { getSavedAccounts, saveAccount, removeSavedAccount } from '@/lib/storage'
import { api } from '@/services/api-client'
import { X, Eye, EyeOff, ArrowLeft, CheckCircle } from 'lucide-react'
import PageLayout from '@/components/page-layout'
import AuthFooter from './auth-footer'

type Panel = 'login' | 'register' | 'forgot' | 'forgot-reset' | 'forgot-done'

export default function AuthPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()

  const pathPanel: Panel = location.pathname === '/register' ? 'register'
    : location.pathname === '/forgot-password' ? 'forgot'
    : 'login'
  const [panel, setPanel] = useState<Panel>(pathPanel)

  const isLoading = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoading)
  const error = useSyncExternalStore(authStore.subscribe, () => authStore.state.error)
  const mfaChallenge = useSyncExternalStore(authStore.subscribe, () => authStore.state.mfaChallenge)
  const isLoggedIn = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoggedIn)

  // Login state
  const [account, setAccount] = useState('')
  const [password, setPassword] = useState('')
  const [mfaCode, setMfaCode] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [remember] = useState(true)
  const [localLoginError, setLocalLoginError] = useState('')
  const [savedAccounts, setSavedAccounts] = useState(getSavedAccounts)
  const [termsDialog, setTermsDialog] = useState<'login' | 'register' | null>(null)
  const pendingAction = useRef<(() => Promise<void>) | null>(null)
  const [agreeLoginTerms, setAgreeLoginTerms] = useState(false)

  // Register state
  const [regAccount, setRegAccount] = useState('')
  const [regName, setRegName] = useState('')
  const [regPassword, setRegPassword] = useState('')
  const [regConfirm, setRegConfirm] = useState('')
  const [regEmail, setRegEmail] = useState('')
  const [localRegError, setLocalRegError] = useState('')
  const [agreeTerms, setAgreeTerms] = useState(false)

  // Forgot password state
  const [fpAccount, setFpAccount] = useState('')
  const [fpUserId, setFpUserId] = useState('')
  const [fpCode, setFpCode] = useState('')
  const [fpNewPassword, setFpNewPassword] = useState('')
  const [fpConfirm, setFpConfirm] = useState('')
  const [fpIsLoading, setFpIsLoading] = useState(false)
  const [fpError, setFpError] = useState('')
  const [fpSuccessMsg, setFpSuccessMsg] = useState('')

  // OAuth
  const [oauthProviders, setOAuthProviders] = useState<string[]>([])

  // App info
  const [appName, setAppName] = useState('Ziziphus')
  const [appHeadline, setAppHeadline] = useState('')

  useEffect(() => {
    fetch('/api/v1/app/info')
      .then(r => r.json())
      .then(d => {
        if (d.data?.name) setAppName(d.data.name)
        if (d.data?.headline) setAppHeadline(d.data.headline)
        if (d.data?.oauth) {
          const providers: string[] = []
          if (d.data.oauth.github) providers.push('github')
          if (d.data.oauth.google) providers.push('google')
          setOAuthProviders(providers)
        }
      })
      .catch(() => {})
  }, [])

  useEffect(() => { if (isLoggedIn) navigate('/conversations', { replace: true }) }, [isLoggedIn, navigate])

  // Sync URL path
  const goPanel = (p: Panel) => {
    setPanel(p)
    setLocalLoginError('')
    setLocalRegError('')
    setFpError('')
    if (p === 'login') navigate('/login', { replace: true })
    else if (p === 'register') navigate('/register', { replace: true })
  }

  // ---- OAuth ----
  const handleOAuthLogin = (provider: string) => async () => {
    try {
      const res = await api.request<{ url: string }>(`/api/v1/auth/${provider}/login`)
      window.location.href = res.url
    } catch (e) {
      console.error('OAuth login failed', e)
    }
  }

  // ---- Login ----
  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!account.trim()) { setLocalLoginError(t('auth.accountRequired', '请填写账号')); return }
    if (!password.trim()) { setLocalLoginError(t('auth.passwordRequired', '请填写密码')); return }
    if (!agreeLoginTerms) {
      setLocalLoginError('')
      pendingAction.current = async () => {
        await authStore.login(account.trim(), password)
        if (remember) saveAccount(account.trim())
        setSavedAccounts(getSavedAccounts())
      }
      setTermsDialog('login')
      return
    }
    try {
      await authStore.login(account.trim(), password)
      if (remember) saveAccount(account.trim())
      setSavedAccounts(getSavedAccounts())
    } catch { /* error handled by store */ }
  }
  const handleMfaVerify = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!mfaCode.trim()) return
    try { await authStore.mfaVerify(mfaCode.trim()) } catch (e) { console.error(e) }
  }
  const fillAccount = (acc: string) => setAccount(acc)
  const deleteAccount = (acc: string, e: React.MouseEvent) => {
    e.stopPropagation(); removeSavedAccount(acc); setSavedAccounts(getSavedAccounts())
  }

  // ---- Register ----
  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setLocalRegError('')
    if (!regAccount.trim()) { setLocalRegError(t('auth.accountRequired', '请填写账号')); return }
    if (!regName.trim()) { setLocalRegError(t('auth.nameRequired', '请填写昵称')); return }
    if (!regPassword.trim()) { setLocalRegError(t('auth.passwordRequired', '请填写密码')); return }
    if (regPassword.trim().length < 8) { setLocalRegError(t('auth.passwordTooShort', '密码至少8位')); return }
    if (regPassword.trim().length > 72) { setLocalRegError(t('auth.passwordTooLong', '密码最多72位')); return }
    if (regPassword !== regConfirm) { setLocalRegError(t('auth.passwordMismatch')); return }
    if (!agreeTerms) {
      setLocalRegError('')
      pendingAction.current = async () => {
        await authStore.register(regAccount.trim(), regName.trim(), regPassword.trim(), regEmail.trim() || undefined)
      }
      setTermsDialog('register')
      return
    }
    try { await authStore.register(regAccount.trim(), regName.trim(), regPassword.trim(), regEmail.trim() || undefined) } catch (e) { console.error(e) }
  }

  // ---- Forgot Password ----
  const handleFpRequest = async (e: React.FormEvent) => {
    e.preventDefault()
    setFpError('')
    if (!fpAccount.trim()) { setFpError(t('common.required', '请填写账号或邮箱')); return }
    setFpIsLoading(true)
    try {
      const result = await api.request<{ user_id: string; code?: string }>(
        '/api/v1/users/password-reset/send-code',
        { method: 'POST', body: { account_or_email: fpAccount.trim() } }
      )
      setFpUserId(result.user_id)
      setFpSentAccount(fpAccount.trim())
      setFpSuccessMsg(t('auth.sendCodeSuccess', '验证码已发送到你的邮箱，请查收'))
      setPanel('forgot-reset')
    } catch (err: unknown) {
      setFpError(err instanceof Error ? err.message : t('auth.requestResetFailed', '请求失败'))
    } finally { setFpIsLoading(false) }
  }

  const handleFpReset = async (e: React.FormEvent) => {
    e.preventDefault()
    setFpError('')
    if (!fpCode.trim() || !fpNewPassword.trim() || !fpConfirm.trim()) {
      setFpError(t('auth.fieldRequired', '请填写所有字段')); return
    }
    if (fpNewPassword.length < 8) { setFpError(t('auth.passwordTooShort', '密码至少8位')); return }
    if (fpNewPassword.length > 72) { setFpError(t('auth.passwordTooLong', '密码最多72位')); return }
    if (fpNewPassword !== fpConfirm) { setFpError(t('auth.passwordMismatch')); return }
    setFpIsLoading(true)
    try {
      await api.request('/api/v1/users/password-reset/reset',
        { method: 'POST', body: { user_id: fpUserId, code: fpCode.trim(), new_password: fpNewPassword } })
      setPanel('forgot-done')
    } catch (err: unknown) {
      setFpError(err instanceof Error ? err.message : t('auth.resetFailed', '重置失败'))
    } finally { setFpIsLoading(false) }
  }

  const inputClass = 'w-full h-12 px-4 rounded-xl bg-[var(--color-surface-soft)] text-[var(--color-ink)] text-sm placeholder:text-[var(--color-muted)] outline-none border border-transparent focus:border-[var(--color-primary)]/40 focus:bg-[var(--color-surface-card)] transition-colors'

  // MFA
  if (mfaChallenge) {
    const isTOTP = mfaChallenge.mfaType === 1
    const mfaTypeLabel = isTOTP ? t('auth.mfaTOTPLabel') : t('auth.mfaEmailLabel')
    const mfaHint = isTOTP ? t('auth.mfaTOTPHint') : t('auth.mfaEmailHint') + (mfaChallenge.maskedEmail ? ` (${mfaChallenge.maskedEmail})` : '')
    return (
      <PageLayout>
        <div className="w-full max-w-[340px] flex flex-col items-center gap-6">
          <ZLogo name={appName} />
          <p className="text-sm text-[var(--color-muted)] -mt-3">{t('auth.mfaTitle')}</p>
          <div className="px-3 py-1 rounded-full bg-[var(--color-primary)]/10 text-[var(--color-primary)] text-xs font-medium">{mfaTypeLabel}</div>
          <p className="text-xs text-[var(--color-muted)] -mt-2">{mfaHint}</p>
          <form onSubmit={handleMfaVerify} className="w-full flex flex-col gap-3">
            <input className={inputClass} type="text" inputMode="numeric" autoComplete="one-time-code" maxLength={6}
              placeholder={t('auth.mfaCode') || '验证码'} value={mfaCode} onChange={e => setMfaCode(e.target.value)} autoFocus />
            <button type="submit" disabled={!mfaCode.trim() || isLoading}
              className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 disabled:opacity-50 transition-all cursor-pointer">
              {isLoading ? t('auth.verifying') : t('auth.verify') || '验证'}
            </button>
            {error && <p className="text-xs text-red-500 text-center">{error}</p>}
          </form>
        </div>
      </PageLayout>
    )
  }

  const displayLoginError = localLoginError || error
  const displayRegError = localRegError || error

  return (
    <PageLayout>
      <div className="w-full max-w-[340px] flex flex-col items-center gap-8">
        <ZLogo name={appName} headline={appHeadline} />

        <div className="w-full animate-msg-in" key={panel}>
          {/* Login */}
          {panel === 'login' && (
            <div className="flex flex-col gap-4">
              <form onSubmit={handleLogin} className="flex flex-col gap-3">
                <div className="relative">
                  <input className={inputClass + ' pr-10'} type="text" placeholder={t('auth.account')}
                    value={account} onChange={e => setAccount(e.target.value)} disabled={isLoading}
                    autoFocus autoComplete="username" onFocus={() => setSavedAccounts(getSavedAccounts())} />
                  {!account && savedAccounts.length > 0 && !isLoading && (
                    <div className="absolute top-full left-0 right-0 mt-1 rounded-xl bg-[var(--color-surface-card)] border border-[var(--color-hairline)] z-10 overflow-hidden"
                      style={{ boxShadow: 'var(--shadow-md)' }}>
                      {savedAccounts.map(acc => (
                        <button key={acc} type="button"
                          className="w-full px-4 py-2.5 text-left text-sm text-[var(--color-ink)] hover:bg-[var(--color-surface-soft)] transition-colors flex items-center justify-between group"
                          onClick={() => fillAccount(acc)}>
                          <span>{acc}</span>
                          <span className="text-[var(--color-muted)] hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                            onClick={e => deleteAccount(acc, e)}><X size={14} /></span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
                <div className="relative">
                  <input className={inputClass + ' pr-10'} type={showPassword ? 'text' : 'password'}
                    placeholder={t('auth.password')} value={password} onChange={e => setPassword(e.target.value)}
                    disabled={isLoading} autoComplete="current-password" />
                  <button type="button" tabIndex={-1}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
                    onClick={() => setShowPassword(!showPassword)}>
                    {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                  </button>
                </div>
                <label className="flex items-start gap-2 cursor-pointer">
                  <input type="checkbox" checked={agreeLoginTerms} onChange={e => setAgreeLoginTerms(e.target.checked)}
                    className="mt-0.5 w-4 h-4 rounded border-[var(--color-hairline)] text-[var(--color-primary)] focus:ring-[var(--color-primary)]/20 accent-[var(--color-primary)]" />
                  <span className="text-xs text-[var(--color-muted)] leading-relaxed">
                    {t('auth.agreeTerms', 'I agree to the')}{' '}
                    <a href="/privacy" target="_blank" className="text-[var(--color-primary)] hover:underline">{t('auth.privacy', 'Privacy Policy')}</a>
                    {' '}{t('common.and', 'and')}{' '}
                    <a href="/terms" target="_blank" className="text-[var(--color-primary)] hover:underline">{t('auth.terms', 'Terms of Service')}</a>
                  </span>
                </label>
                <button type="submit" disabled={isLoading}
                  className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 disabled:opacity-50 transition-all cursor-pointer">
                  {isLoading ? t('auth.loggingIn') : t('auth.login')}
                </button>
                {displayLoginError && <p className="text-xs text-red-500 text-center">{displayLoginError}</p>}
                <div className="flex justify-end text-xs">
                  <button type="button" onClick={() => goPanel('forgot')}
                    className="text-[var(--color-primary)] hover:underline">{t('auth.forgotPassword', '忘记密码？')}</button>
                </div>
                <p className="text-xs text-[var(--color-muted)] text-center pt-1">
                  {t('auth.noAccount')}{' '}
                  <button type="button" onClick={() => goPanel('register')}
                    className="text-[var(--color-primary)] hover:underline font-medium">{t('auth.register')}</button>
                </p>
              </form>
              {/* OAuth — below the form */}
              {oauthProviders.length > 0 && (
                <>
                  <div className="flex items-center gap-3">
                    <div className="flex-1 h-px bg-[var(--color-hairline)]" />
                    <span className="text-xs text-[var(--color-muted)]">{t('auth.or')}</span>
                    <div className="flex-1 h-px bg-[var(--color-hairline)]" />
                  </div>
                  <div className="flex flex-col gap-2">
                    {oauthProviders.includes('github') && (
                      <button type="button" onClick={handleOAuthLogin('github')}
                        className="flex items-center justify-center gap-2 w-full h-11 rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-body)] hover:bg-[var(--color-surface-soft)] transition-colors cursor-pointer">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/></svg> {t('auth.oauthGithub', 'Sign in with GitHub')}
                      </button>
                    )}
                    {oauthProviders.includes('google') && (
                      <button type="button" onClick={handleOAuthLogin('google')}
                        className="flex items-center justify-center gap-2 w-full h-11 rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-body)] hover:bg-[var(--color-surface-soft)] transition-colors cursor-pointer">
                        <svg width="18" height="18" viewBox="0 0 24 24"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg> {t('auth.oauthGoogle', 'Sign in with Google')}
                      </button>
                    )}
                  </div>
                </>
              )}
            </div>
          )}

          {/* Register */}
          {panel === 'register' && (
            <form onSubmit={handleRegister} className="flex flex-col gap-3">
              <input type="text" value={regAccount} onChange={e => setRegAccount(e.target.value)}
                placeholder={t('auth.account')} className={inputClass} autoComplete="username" autoFocus />
              <input type="text" value={regName} onChange={e => setRegName(e.target.value)}
                placeholder={t('auth.name')} className={inputClass} />
              <input type="email" value={regEmail} onChange={e => setRegEmail(e.target.value)}
                placeholder={t('auth.email')} className={inputClass} autoComplete="email" />
              <input type="password" value={regPassword} onChange={e => setRegPassword(e.target.value)}
                placeholder={t('auth.password')} className={inputClass} autoComplete="new-password" />
              <input type="password" value={regConfirm} onChange={e => setRegConfirm(e.target.value)}
                placeholder={t('auth.confirmPassword')} className={inputClass} autoComplete="new-password" />
              {displayRegError && (
                <div className="text-xs text-[var(--destructive)] bg-[var(--destructive)]/10 rounded-xl px-3 py-2">{displayRegError}</div>
              )}
              <label className="flex items-start gap-2 cursor-pointer">
                <input type="checkbox" checked={agreeTerms} onChange={e => setAgreeTerms(e.target.checked)}
                  className="mt-0.5 w-4 h-4 rounded border-[var(--color-hairline)] text-[var(--color-primary)] focus:ring-[var(--color-primary)]/20 accent-[var(--color-primary)]" />
                <span className="text-xs text-[var(--color-muted)] leading-relaxed">
                  {t('auth.agreeTerms', '我同意')}{' '}
                  <a href="/privacy" target="_blank" className="text-[var(--color-primary)] hover:underline">{t('auth.privacy', '隐私政策')}</a>
                  {' '}{t('common.and', '和')}{' '}
                  <a href="/terms" target="_blank" className="text-[var(--color-primary)] hover:underline">{t('auth.terms', '服务条款')}</a>
                </span>
              </label>
              <button type="submit" disabled={isLoading}
                className="w-full h-11 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40">
                {isLoading ? t('auth.registering') : t('auth.register')}
              </button>
              <p className="text-xs text-[var(--color-muted)] text-center pt-1">
                {t('auth.hasAccount', '已有账号？')}{' '}
                <button type="button" onClick={() => goPanel('login')}
                  className="text-[var(--color-primary)] hover:underline font-medium">{t('auth.login')}</button>
              </p>
            </form>
          )}

          {/* Forgot password — request code */}
          {panel === 'forgot' && (
            <div>
              <button type="button" onClick={() => goPanel('login')}
                className="flex items-center gap-1 text-xs text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors mb-4">
                <ArrowLeft size={14} /> {t('auth.backToLogin', '返回登录')}
              </button>
              <p className="text-sm text-[var(--color-muted)] mb-1">{t('auth.forgotPasswordTitle', '找回密码')}</p>
              <p className="text-xs text-[var(--color-muted)] mb-4">{t('auth.forgotPasswordHint', '输入账号或邮箱来接收重置验证码')}</p>
              <form onSubmit={handleFpRequest} className="flex flex-col gap-3">
                <input className={inputClass} type="text"
                  placeholder={t('auth.accountOrEmail', '账号或邮箱')}
                  value={fpAccount} onChange={e => setFpAccount(e.target.value)} disabled={fpIsLoading} autoFocus />
                {fpError && <p className="text-xs text-red-500 text-center">{fpError}</p>}
                <button type="submit" disabled={fpIsLoading}
                  className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 disabled:opacity-50 transition-all cursor-pointer mt-1">
                  {fpIsLoading ? t('auth.sending', '发送中...') : t('auth.sendCode', '发送验证码')}
                </button>
              </form>
            </div>
          )}

          {/* Forgot password — reset */}
          {panel === 'forgot-reset' && (
            <div>
              <button type="button" onClick={() => goPanel('forgot')}
                className="flex items-center gap-1 text-xs text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors mb-4">
                <ArrowLeft size={14} /> {t('auth.backToRequest', '重新发送验证码')}
              </button>
              <p className="text-sm text-[var(--color-muted)] mb-1">{t('auth.resetPasswordTitle', '重置密码')}</p>
              <p className="text-xs text-[var(--color-muted)] mb-2">{t('auth.resetPasswordHint', '请输入验证码和新密码')}</p>
              {fpSuccessMsg && (
                <p className="mb-3 text-xs text-green-600 bg-green-50 dark:text-green-400 dark:bg-green-900/20 rounded-xl px-3 py-2">{fpSuccessMsg}</p>
              )}
              <form onSubmit={handleFpReset} className="flex flex-col gap-3">
                <input className={inputClass} type="text" inputMode="numeric" maxLength={6}
                  placeholder={t('auth.verificationCode', '验证码')}
                  value={fpCode} onChange={e => setFpCode(e.target.value)} disabled={fpIsLoading} autoFocus />
                <input className={inputClass} type="password"
                  placeholder={t('auth.newPassword', '新密码')}
                  value={fpNewPassword} onChange={e => setFpNewPassword(e.target.value)} disabled={fpIsLoading}
                  autoComplete="new-password" />
                <input className={inputClass} type="password"
                  placeholder={t('auth.confirmPassword')}
                  value={fpConfirm} onChange={e => setFpConfirm(e.target.value)} disabled={fpIsLoading}
                  autoComplete="new-password" />
                {fpError && <p className="text-xs text-red-500 text-center">{fpError}</p>}
                <button type="submit" disabled={fpIsLoading}
                  className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 disabled:opacity-50 transition-all cursor-pointer mt-1">
                  {fpIsLoading ? t('auth.resetting', '重置中...') : t('auth.resetPassword', '重置密码')}
                </button>
              </form>
            </div>
          )}

          {/* Forgot password — done */}
          {panel === 'forgot-done' && (
            <div className="flex flex-col items-center gap-4 text-center">
              <CheckCircle size={48} className="text-[var(--success)]" />
              <div>
                <p className="text-base font-semibold text-[var(--color-ink)]">{t('auth.resetSuccess', '密码重置成功')}</p>
                <p className="text-sm text-[var(--color-muted)] mt-1">{t('auth.resetSuccessHint', '请使用新密码登录')}</p>
              </div>
              <button onClick={() => { setFpAccount(''); setFpCode(''); setFpNewPassword(''); setFpConfirm(''); goPanel('login') }}
                className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 transition-all cursor-pointer mt-2">
                {t('auth.backToLogin', '返回登录')}
              </button>
            </div>
          )}
        </div>
      </div>

      <AuthFooter />

      {/* Terms dialog */}
      {termsDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={() => { setTermsDialog(null); pendingAction.current = null }}>
          <div className="w-[90%] sm:w-[360px] bg-[var(--color-surface-card)] rounded-xl p-6" style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
            <h3 className="text-base font-semibold text-[var(--color-ink)] mb-3">{t('auth.termsTitle', 'Terms of Service')}</h3>
            <p className="text-xs text-[var(--color-muted)] leading-relaxed mb-4">
              {t('auth.termsContent', 'Please read and agree to the following terms to continue:')}
            </p>
            <div className="flex items-center justify-center gap-3 mb-5 text-xs">
              <a href="/privacy" target="_blank" className="text-[var(--color-primary)] hover:underline">{t('auth.privacy', 'Privacy Policy')}</a>
              <span className="text-[var(--color-hairline)]">·</span>
              <a href="/terms" target="_blank" className="text-[var(--color-primary)] hover:underline">{t('auth.terms', 'Terms of Service')}</a>
            </div>
            <div className="flex gap-3">
              <button onClick={() => { setTermsDialog(null); pendingAction.current = null }}
                className="flex-1 h-11 rounded-xl border border-[var(--color-hairline)] text-sm text-[var(--color-muted)] hover:bg-[var(--color-surface-soft)] transition-colors cursor-pointer">
                {t('auth.disagree', 'Disagree')}
              </button>
              <button onClick={async () => {
                if (termsDialog === 'login') setAgreeLoginTerms(true)
                else setAgreeTerms(true)
                setTermsDialog(null)
                const action = pendingAction.current
                pendingAction.current = null
                try { await action?.() } catch { /* handled by store */ }
              }}
                className="flex-1 h-11 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 transition-all cursor-pointer">
                {t('auth.agree', 'Agree')}
              </button>
            </div>
          </div>
        </div>
      )}
    </PageLayout>
  )
}

function ZLogo({ name, headline }: { name: string; headline?: string }) {
  return (
    <div className="flex flex-col items-center gap-3">
      <div className="w-14 h-14 rounded-2xl flex items-center justify-center text-white text-3xl font-extrabold font-headline tracking-tighter"
        style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-accent))' }}>Z</div>
      <h1 className="font-headline text-xl font-bold text-[var(--color-ink)] tracking-tight">{name}</h1>
      {headline && <p className="text-[13px] text-[var(--color-muted)] -mt-1">{headline}</p>}
    </div>
  )
}
