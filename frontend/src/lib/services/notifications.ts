// Notification service for desktop notifications
// Uses the Web Notification API (supported in Wails webview)

/**
 * Check if the Notification API is available in the current environment.
 */
export function isNotificationSupported(): boolean {
  return typeof window !== 'undefined' && 'Notification' in window
}

/**
 * Request notification permission from the user.
 * Returns true if permission is granted.
 */
export async function requestNotificationPermission(): Promise<boolean> {
  if (!isNotificationSupported()) {
    return false
  }

  if (Notification.permission === 'granted') {
    return true
  }

  if (Notification.permission === 'denied') {
    return false
  }

  try {
    const result = await Notification.requestPermission()
    return result === 'granted'
  } catch (e) {
    console.error('Failed to request notification permission:', e)
    return false
  }
}

/**
 * Show a desktop notification.
 * No-op if notifications are not supported or permission is not granted.
 */
export function notify(
  title: string,
  body: string,
  options?: { icon?: string; onClick?: () => void },
): void {
  if (!isNotificationSupported()) {
    return
  }

  if (Notification.permission !== 'granted') {
    return
  }

  try {
    const notification = new Notification(title, {
      body,
      icon: options?.icon,
      silent: false,
    })

    if (options?.onClick) {
      notification.onclick = () => {
        window.focus()
        options.onClick?.()
        notification.close()
      }
    }

    // Auto-close after 5 seconds
    setTimeout(() => notification.close(), 5000)
  } catch (e) {
    console.error('Failed to show notification:', e)
  }
}
