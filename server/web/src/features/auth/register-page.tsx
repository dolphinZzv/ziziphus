import { useState, useSyncExternalStore } from 'react'
import { Link } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'

export default function RegisterPage() {
  const isLoading = useSyncExternalStore(authStore.subscribe, () => authStore.state.isLoading)
  const error = useSyncExternalStore(authStore.subscribe, () => authStore.state.error)

  const [account, setAccount] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [localError, setLocalError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLocalError('')
    if (!account.trim() || !name.trim() || !password.trim()) { setLocalError('请填写所有字段'); return }
    if (password !== confirmPassword) { setLocalError('两次密码不一致'); return }
    try { await authStore.register(account.trim(), name.trim(), password) } catch {}
  }

  const displayError = localError || error
  const inputClass = 'w-full h-12 px-5 rounded-xl bg-[var(--color-canvas)] text-[var(--color-ink)] text-sm placeholder:text-[var(--color-muted-soft)] outline-none border border-[var(--color-hairline-soft)] focus:border-[var(--color-primary)] focus:ring-1 focus:ring-[var(--color-primary)]'

  return (
    <div className="h-full flex items-center justify-center bg-[var(--color-canvas)]">
      <div className="w-[380px] bg-[var(--color-surface-card)] rounded-2xl" style={{ boxShadow: 'var(--shadow-default)', padding: '40px' }}>
        <div className="text-center" style={{ marginBottom: '56px' }}>
          <h1 className="font-headline text-[32px] font-bold text-[var(--color-ink)] leading-tight">Panda AI</h1>
          <p className="text-sm text-[var(--color-muted)] mt-2">创建新账号</p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-6">
          <input type="text" value={account} onChange={e => setAccount(e.target.value)} placeholder="账号" className={inputClass} autoComplete="username" />
          <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="昵称" className={inputClass} />
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="密码" className={inputClass} autoComplete="new-password" />
          <input type="password" value={confirmPassword} onChange={e => setConfirmPassword(e.target.value)} placeholder="确认密码" className={inputClass} autoComplete="new-password" />

          {displayError && (
            <div className="text-xs text-[var(--destructive)] bg-[var(--destructive)]/10 rounded-lg px-3 py-2">{displayError}</div>
          )}

          <button type="submit" disabled={isLoading}
            className="w-full h-11 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
            {isLoading ? '注册中...' : '注册'}
          </button>
        </form>

        <div className="mt-6 text-center">
          <Link to="/login" className="text-sm text-[var(--color-muted)] hover:text-[var(--color-accent)] transition-colors">已有账号？去登录</Link>
        </div>
      </div>
    </div>
  )
}
