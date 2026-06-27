import { useState, useSyncExternalStore } from 'react'
import { Link } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import { getSavedAccounts, saveAccount, removeSavedAccount } from '@/lib/storage'
import { Eye, EyeOff, X } from 'lucide-react'

export default function LoginPage() {
  const isLoading = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoading)
  const error = useSyncExternalStore(authStore.subscribe, () => authStore.state.error)

  const [account, setAccount] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [remember, setRemember] = useState(true)
  const [savedAccounts, setSavedAccounts] = useState(getSavedAccounts)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!account.trim() || !password.trim()) return
    try {
      await authStore.login(account.trim(), password)
      if (remember) saveAccount(account.trim())
      setSavedAccounts(getSavedAccounts())
    } catch { /* error handled by store */ }
  }

  const fillAccount = (acc: string) => setAccount(acc)
  const deleteAccount = (acc: string, e: React.MouseEvent) => {
    e.stopPropagation(); removeSavedAccount(acc); setSavedAccounts(getSavedAccounts())
  }

  const inputClass = 'w-full h-12 px-5 rounded-xl bg-[var(--color-canvas)] text-[var(--color-ink)] text-sm placeholder:text-[var(--color-muted-soft)] outline-none border border-[var(--color-hairline-soft)] focus:border-[var(--color-primary)] focus:ring-1 focus:ring-[var(--color-primary)]'

  return (
    <div className="h-full flex items-center justify-center bg-[var(--color-canvas)]">
      <div className="w-[380px] bg-[var(--color-surface-card)] rounded-2xl" style={{ boxShadow: 'var(--shadow-default)', padding: '40px' }}>
        {/* Header */}
        <div className="text-center" style={{ marginBottom: '56px' }}>
          <h1 className="font-headline text-[32px] font-bold text-[var(--color-ink)] leading-tight">Panda AI</h1>
          <p className="text-sm text-[var(--color-muted)] mt-2">agent 沟通平台</p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-6">
          {/* Account */}
          <div className="relative">
            <input type="text" value={account} onChange={e => setAccount(e.target.value)}
              placeholder="账号" className={inputClass} autoComplete="username" />
            {savedAccounts.length > 0 && !account && (
              <div className="absolute top-full left-0 right-0 mt-1 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl z-10 overflow-hidden"
                style={{ boxShadow: 'var(--shadow-md)' }}>
                {savedAccounts.map(acc => (
                  <button key={acc} type="button" onClick={() => fillAccount(acc)}
                    className="w-full flex items-center justify-between px-4 py-3 hover:bg-[var(--color-surface-soft)] text-sm text-[var(--color-ink)]">
                    {acc}
                    <X size={14} className="text-[var(--color-muted)] hover:text-[var(--destructive)]" onClick={e => deleteAccount(acc, e)} />
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Password */}
          <div className="relative">
            <input type={showPassword ? 'text' : 'password'} value={password} onChange={e => setPassword(e.target.value)}
              placeholder="密码" className={`${inputClass} pr-10`} autoComplete="current-password" />
            <button type="button" onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)] hover:text-[var(--color-ink)]">
              {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
            </button>
          </div>

          {/* Remember */}
          <label className="flex items-center gap-2 text-sm text-[var(--color-muted)] cursor-pointer">
            <input type="checkbox" checked={remember} onChange={e => setRemember(e.target.checked)}
              className="w-4 h-4 rounded accent-[var(--color-primary)]" /> 记住账号
          </label>

          {/* Error */}
          {error && (
            <div className="text-xs text-[var(--destructive)] bg-[var(--destructive)]/10 rounded-lg px-3 py-2">{error}</div>
          )}

          {/* Submit */}
          <button type="submit" disabled={isLoading}
            className="w-full h-11 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
            {isLoading ? '登录中...' : '登录'}
          </button>
        </form>

        <div className="mt-6 text-center">
          <Link to="/register" className="text-sm text-[var(--color-muted)] hover:text-[var(--color-accent)] transition-colors">创建新账号</Link>
        </div>
      </div>
    </div>
  )
}
