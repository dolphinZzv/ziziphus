import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '@/services/api-client'
import AuthFooter from './auth-footer'

type Step = 'request' | 'reset' | 'done'

export default function ForgotPasswordPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [step, setStep] = useState<Step>('request')
  const [accountOrEmail, setAccountOrEmail] = useState('')
  const [userId, setUserId] = useState('')
  const [code, setCode] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [successMsg, setSuccessMsg] = useState('')
  const [appName, setAppName] = useState('Ziziphus')

  useEffect(() => {
    fetch('/api/v1/app/info')
      .then(r => r.json())
      .then(d => { if (d.data?.name) setAppName(d.data.name) })
      .catch(() => { /* use default */ })
  }, [])

  const inputClass = 'w-full h-12 px-4 rounded-xl bg-[var(--color-surface-soft)] text-[var(--color-ink)] text-sm placeholder:text-[var(--color-muted)] outline-none border border-transparent focus:border-[var(--color-primary)]/40 focus:bg-[var(--color-surface-card)] transition-colors'

  const handleRequestCode = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!accountOrEmail.trim()) {
      setError(t('common.required', '请填写账号或邮箱'))
      return
    }
    setIsLoading(true)
    try {
      const result = await api.request<{ user_id: string; code?: string }>(
        '/api/v1/users/password-reset/send-code',
        { method: 'POST', body: { account_or_email: accountOrEmail.trim() } }
      )
      setUserId(result.user_id)
      setSuccessMsg(t('auth.sendCodeSuccess', '验证码已发送到你的邮箱，请查收'))
      setStep('reset')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('auth.requestResetFailed', '请求失败'))
    } finally {
      setIsLoading(false)
    }
  }

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!code.trim() || !newPassword.trim() || !confirmPassword.trim()) {
      setError(t('auth.fieldRequired'))
      return
    }
    if (newPassword.length < 8) {
      setError(t('auth.passwordTooShort', '密码至少8位'))
      return
    }
    if (newPassword !== confirmPassword) {
      setError(t('auth.passwordMismatch'))
      return
    }
    setIsLoading(true)
    try {
      await api.request<{ status: string }>(
        '/api/v1/users/password-reset/reset',
        { method: 'POST', body: { user_id: userId, code: code.trim(), new_password: newPassword } }
      )
      setStep('done')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('auth.resetFailed', '重置失败'))
    } finally {
      setIsLoading(false)
    }
  }

  // Step 1: Request code
  if (step === 'request') {
    return (
      <div className="h-full flex flex-col items-center justify-center bg-[var(--color-canvas)] relative px-8 gap-8">
        <div className="text-center">
          <h1 className="font-headline text-[28px] font-bold text-[var(--color-ink)]">{appName}</h1>
          <p className="text-sm text-[var(--color-muted)] mt-3">{t('auth.forgotPasswordTitle', '找回密码')}</p>
          <p className="text-xs text-[var(--color-muted)] mt-1">{t('auth.forgotPasswordHint', '输入账号或邮箱来接收重置验证码')}</p>
        </div>

        <form onSubmit={handleRequestCode} className="w-full max-w-[320px] flex flex-col gap-4">
          <input
            className={inputClass}
            type="text"
            placeholder={t('auth.accountOrEmail', '账号或邮箱')}
            value={accountOrEmail}
            onChange={e => setAccountOrEmail(e.target.value)}
            disabled={isLoading}
            autoFocus
          />

          {error && <p className="text-xs text-red-500 text-center">{error}</p>}

          <button
            type="submit"
            disabled={isLoading || !accountOrEmail.trim()}
            className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 disabled:opacity-50 transition-all cursor-pointer mt-1"
          >
            {isLoading ? t('auth.sending', '发送中...') : t('auth.sendCode', '发送验证码')}
          </button>
        </form>

        <Link to="/login" className="text-xs text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors">
          {t('auth.backToLogin', '返回登录')}
        </Link>

        <AuthFooter />
      </div>
    )
  }

  // Step 2: Reset password with code
  if (step === 'reset') {
    return (
      <div className="h-full flex flex-col items-center justify-center bg-[var(--color-canvas)] relative px-8 gap-8">
        <div className="text-center">
          <h1 className="font-headline text-[28px] font-bold text-[var(--color-ink)]">{appName}</h1>
          <p className="text-sm text-[var(--color-muted)] mt-3">{t('auth.resetPasswordTitle', '重置密码')}</p>
          <p className="text-xs text-[var(--color-muted)] mt-1">{t('auth.resetPasswordHint', '请输入验证码和新密码')}</p>
          {successMsg && (
            <p className="mt-3 text-xs text-green-600 bg-green-50 dark:text-green-400 dark:bg-green-900/20 rounded-xl px-3 py-2">{successMsg}</p>
          )}
        </div>

        <form onSubmit={handleResetPassword} className="w-full max-w-[320px] flex flex-col gap-4">
          <input
            className={inputClass}
            type="text"
            inputMode="numeric"
            maxLength={6}
            placeholder={t('auth.verificationCode', '验证码')}
            value={code}
            onChange={e => setCode(e.target.value)}
            disabled={isLoading}
            autoFocus
          />
          <input
            className={inputClass}
            type="password"
            placeholder={t('auth.newPassword', '新密码')}
            value={newPassword}
            onChange={e => setNewPassword(e.target.value)}
            disabled={isLoading}
            autoComplete="new-password"
          />
          <input
            className={inputClass}
            type="password"
            placeholder={t('auth.confirmPassword')}
            value={confirmPassword}
            onChange={e => setConfirmPassword(e.target.value)}
            disabled={isLoading}
            autoComplete="new-password"
          />

          {error && <p className="text-xs text-red-500 text-center">{error}</p>}

          <button
            type="submit"
            disabled={isLoading || !code.trim() || !newPassword.trim() || !confirmPassword.trim()}
            className="w-full h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 disabled:opacity-50 transition-all cursor-pointer mt-1"
          >
            {isLoading ? t('auth.resetting', '重置中...') : t('auth.resetPassword', '重置密码')}
          </button>
        </form>

        <button type="button" className="text-xs text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors" onClick={() => setStep('request')}>
          {t('auth.backToRequest', '重新发送验证码')}
        </button>
      </div>
    )
  }

  // Step 3: Done
  return (
    <div className="h-full flex flex-col items-center justify-center bg-[var(--color-canvas)] relative px-8 gap-8">
      <div className="text-center">
        <h1 className="font-headline text-[28px] font-bold text-[var(--color-ink)]">{appName}</h1>
        <p className="text-sm text-[var(--color-muted)] mt-3">{t('auth.resetSuccess', '密码重置成功')}</p>
        <p className="text-xs text-[var(--color-muted)] mt-1">{t('auth.resetSuccessHint', '请使用新密码登录')}</p>
      </div>

      <button
        onClick={() => navigate('/login')}
        className="w-full max-w-[320px] h-12 rounded-xl bg-[var(--color-primary)] text-white text-sm font-semibold hover:opacity-90 transition-all cursor-pointer"
      >
        {t('auth.backToLogin', '返回登录')}
      </button>

      <AuthFooter />
    </div>
  )
}
