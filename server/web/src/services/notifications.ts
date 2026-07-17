let granted = false

export async function requestNotificationPermission() {
  if (!('Notification' in window)) return false
  if (Notification.permission === 'granted') { granted = true; return true }
  if (Notification.permission === 'denied') return false
  const result = await Notification.requestPermission()
  granted = result === 'granted'
  return granted
}

export function isNotificationGranted() {
  return granted || Notification.permission === 'granted'
}

export function showMessageNotification(title: string, body: string, convId: string) {
  if (!isNotificationGranted()) return
  if (document.visibilityState === 'visible') return // don't notify if tab is focused
  try {
    const n = new Notification(title, {
      body,
      icon: '/favicon.ico',
      tag: convId, // group by conversation
      requireInteraction: false,
    })
    n.onclick = () => {
      window.focus()
      // navigate handled by caller or direct location
      const chatPath = `/conversations/${convId}`
      if (window.location.hash?.includes(chatPath)) return
      window.location.hash = chatPath
      n.close()
    }
    // Auto-close after 5 seconds
    setTimeout(() => n.close(), 5000)
  } catch { /* browser may not support */ }
}
