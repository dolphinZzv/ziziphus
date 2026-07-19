import { useState, useEffect, useSyncExternalStore } from 'react'
import { authStore } from '@/stores/auth-store'
import { mfaService, type MFAStatus } from '@/services/mfa-service'
import { X, Shield, Check, Copy } from 'lucide-react'

interface Props { onClose: () => void }

export default function PrivacyView({ onClose }: Props) {
  const user = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const [discoverable, setDiscoverable] = useState(user?.discoverable ?? true)
  const [allowDirectChat, setAllowDirectChat] = useState(user?.allow_direct_chat ?? true)
  const [email, setEmail] = useState(user?.email || '')
  const [savingEmail, setSavingEmail] = useState(false)
  const [emailCode, setEmailCode] = useState('')
  const [showEmailVerify, setShowEmailVerify] = useState(false)
  const [emailVerifyError, setEmailVerifyError] = useState('')

  // MFA state
  const [mfa, setMFA] = useState<MFAStatus>({ enabled: false, mfa_type: 0 })
  const [mfaType, setMFAType] = useState(1) // 1=totp, 2=email
  const [totpURI, setTotpURI] = useState('')
  const [totpSecret, setTotpSecret] = useState('')
  const [verifyCode, setVerifyCode] = useState('')
  const [showSetup, setShowSetup] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => { mfaService.getStatus().then(setMFA).catch(() => {}) }, [])

  const handleToggleDiscoverable = async () => {
    const v = !discoverable; setDiscoverable(v)
    try { await authStore.updateProfile({ discoverable: v }) } catch { setDiscoverable(!v) }
  }
  const handleToggleDirectChat = async () => {
    const v = !allowDirectChat; setAllowDirectChat(v)
    try { await authStore.updateProfile({ allow_direct_chat: v }) } catch { setAllowDirectChat(!v) }
  }
  const handleSaveEmail = async () => {
    if (!email.trim() || email.trim() === user?.email) return
    setSavingEmail(true)
    try {
      await mfaService.sendEmailCode(email.trim())
      setShowEmailVerify(true)
      setEmailVerifyError('')
    } catch (e: any) { setEmailVerifyError(e?.message || 'Failed') }
    setSavingEmail(false)
  }

  const handleConfirmEmail = async () => {
    if (!emailCode.trim()) return
    try {
      const r = await mfaService.confirmEmail(emailCode.trim())
      await authStore.updateProfile({ email: r.email })
      setShowEmailVerify(false)
      setEmailCode('')
    } catch (e: any) { setEmailVerifyError(e?.message || 'Invalid code') }
  }

  const handleEnableMFA = async (type: number) => {
    setMFAType(type)
    try {
      const r = await mfaService.setup(type)
      if (type === 1) { setTotpSecret(r.secret || ''); setTotpURI(r.uri || '') }
      setShowSetup(true)
    } catch (e: any) { alert(e?.message || 'Failed to set up MFA') }
  }

  const handleVerifyMFA = async () => {
    try {
      await mfaService.verify(verifyCode)
      setShowSetup(false); setVerifyCode('')
      setMFA({ enabled: true, mfa_type: mfaType })
    } catch (e: any) { alert(e?.message || 'Invalid code') }
  }

  const handleDisableMFA = async () => {
    try {
      await mfaService.disable()
      setMFA({ enabled: false, mfa_type: 0 })
    } catch (e) { console.error(e) }
  }

  const copySecret = () => { navigator.clipboard.writeText(totpSecret); setCopied(true); setTimeout(() => setCopied(false), 2000) }

  const toggleClass = (active: boolean) => `relative w-9 h-5 rounded-full transition-colors flex-shrink-0 ml-3 ${active ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-hairline)]'}`
  const toggleDot = (active: boolean) => `absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${active ? 'left-[18px]' : 'left-0.5'}`

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[420px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] overflow-y-auto bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-headline text-xl font-semibold text-[var(--color-ink)]">用户设置</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <div className="space-y-6">
          {/* Privacy */}
          <div className="bg-[var(--color-surface-soft)] rounded-xl p-4 space-y-4">
            <label className="flex items-center justify-between cursor-pointer">
              <div className="flex-1 min-w-0"><span className="text-sm text-[var(--color-body)]">允许通过搜索找到我</span></div>
              <button onClick={handleToggleDiscoverable} className={toggleClass(discoverable)}><span className={toggleDot(discoverable)} /></button>
            </label>
            <div className="mt-3" />
            <label className="flex items-center justify-between cursor-pointer">
              <div className="flex-1 min-w-0"><span className="text-sm text-[var(--color-body)]">允许直接发起会话</span></div>
              <button onClick={handleToggleDirectChat} className={toggleClass(allowDirectChat)}><span className={toggleDot(allowDirectChat)} /></button>
            </label>
          </div>

          {/* Email */}
          <div>
            <label className="block text-xs font-medium text-[var(--color-body)] mb-2">邮箱</label>
            <div className="flex gap-2">
              <input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="email@example.com"
                className="flex-1 h-10 px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)]" disabled={showEmailVerify} />
              <button onClick={handleSaveEmail} disabled={savingEmail || showEmailVerify}
                className="px-4 h-10 rounded-xl bg-[var(--color-primary)] text-white text-sm font-medium disabled:opacity-40">{savingEmail ? '...' : '保存'}</button>
            </div>
            {showEmailVerify && (
              <div className="mt-2 space-y-2 bg-[var(--color-surface-soft)] rounded-xl p-3">
                <p className="text-xs text-[var(--color-muted)]">验证码已发送至 {email}（测试环境显示在下方）</p>
                <div className="flex gap-2">
                  <input type="text" value={emailCode} onChange={e => setEmailCode(e.target.value)} maxLength={6}
                    placeholder="输入 6 位验证码"
                    className="flex-1 h-9 px-3 rounded-xl bg-[var(--color-surface-card)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] text-center tracking-[4px] font-mono" />
                  <button onClick={handleConfirmEmail} className="px-4 h-9 rounded-xl bg-[var(--color-primary)] text-white text-sm">验证</button>
                  <button onClick={() => { setShowEmailVerify(false); setEmailCode(''); setEmail(user?.email || '') }}
                    className="px-3 h-9 text-xs text-[var(--color-muted)]">取消</button>
                </div>
                {emailVerifyError && <p className="text-xs text-[var(--destructive)]">{emailVerifyError}</p>}
              </div>
            )}
          </div>

          {/* MFA */}
          <div>
            <label className="flex items-center gap-2 text-xs font-medium text-[var(--color-body)] mb-3">
              <Shield size={14} /> 多因素认证 (MFA)
            </label>
            {mfa.enabled ? (
              <div className="bg-[var(--color-surface-soft)] rounded-xl p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-[var(--color-body)]">{mfa.mfa_type === 1 ? 'TOTP 认证' : '邮箱认证'} · 已开启</span>
                  <button onClick={handleDisableMFA} className="text-xs text-[var(--destructive)] hover:underline">关闭</button>
                </div>
              </div>
            ) : showSetup ? (
              <div className="bg-[var(--color-surface-soft)] rounded-xl p-4 space-y-3">
                {mfaType === 1 && (
                  <div className="space-y-2">
                    <p className="text-xs text-[var(--color-muted)]">使用 Authenticator App 扫描二维码或手动输入密钥</p>
                    {totpURI && (
                      <div className="flex justify-center">
                        <img loading="lazy" decoding="async" src={`https://api.qrserver.com/v1/create-qr-code/?size=160x160&data=${encodeURIComponent(totpURI)}`} alt="TOTP QR" className="w-40 h-40 rounded-xl bg-white p-2" />
                      </div>
                    )}
                    <div className="flex items-center gap-2">
                      <code className="flex-1 text-[11px] bg-[var(--color-surface-card)] px-2 py-1 rounded font-mono select-all">{totpSecret}</code>
                      <button onClick={copySecret} className="p-1 text-[var(--color-muted)]">{copied ? <Check size={14} className="text-[var(--success)]" /> : <Copy size={14} />}</button>
                    </div>
                  </div>
                )}
                <div className="flex gap-2">
                  <input type="text" value={verifyCode} onChange={e => setVerifyCode(e.target.value)} maxLength={6} placeholder="输入 6 位验证码"
                    className="flex-1 h-9 px-3 rounded-xl bg-[var(--color-surface-card)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] text-center tracking-[4px] font-mono" />
                  <button onClick={handleVerifyMFA} className="px-4 h-9 rounded-xl bg-[var(--color-primary)] text-white text-sm">验证</button>
                  <button onClick={() => setShowSetup(false)} className="px-3 h-9 text-xs text-[var(--color-muted)]">取消</button>
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                <button onClick={() => handleEnableMFA(1)} className="w-full h-10 rounded-xl border border-dashed border-[var(--color-hairline)] text-sm text-[var(--color-body)] hover:border-[var(--color-primary)] hover:text-[var(--color-primary)] transition-colors flex items-center justify-center gap-2">
                  开启 TOTP 认证（Authenticator App）
                </button>
                <button onClick={() => { if (!email) { alert('请先设置邮箱'); return } handleEnableMFA(2) }}
                  className="w-full h-10 rounded-xl border border-dashed border-[var(--color-hairline)] text-sm text-[var(--color-body)] hover:border-[var(--color-primary)] hover:text-[var(--color-primary)] transition-colors flex items-center justify-center gap-2">
                  开启邮箱验证码认证
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
