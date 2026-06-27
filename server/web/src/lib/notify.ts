let unread = 0
let titleTimer: ReturnType<typeof setTimeout> | null = null
const baseTitle = document.title

// Play a subtle notification sound
export function notifySound() {
  try {
    const ctx = new (window.AudioContext || (window as any).webkitAudioContext)()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.frequency.value = 800
    osc.type = 'sine'
    gain.gain.setValueAtTime(0.15, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.15)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.15)
    setTimeout(() => ctx.close(), 200)
  } catch { /* AudioContext may not be available */ }
}

// Flash document.title with unread count
export function notifyTitle(count: number) {
  if (document.hidden) {
    unread += count
    const flash = () => {
      document.title = unread > 0 ? `(${unread}) ${baseTitle}` : baseTitle
      if (unread > 0) {
        titleTimer = setTimeout(() => {
          document.title = baseTitle
          titleTimer = setTimeout(flash, 800)
        }, 800)
      }
    }
    flash()
  }
}

// Reset title when tab becomes visible
if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) {
      unread = 0
      if (titleTimer) clearTimeout(titleTimer)
      document.title = baseTitle
    }
  })
}
