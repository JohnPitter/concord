// Toast notification store using Svelte 5 runes

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: string
  type: ToastType
  title: string
  message?: string
  duration: number
}

let toasts = $state<Toast[]>([])
let nextId = 0

export function getToasts() {
  return {
    get list() { return toasts },
  }
}

export function addToast(type: ToastType, title: string, message?: string, duration = 5000): void {
  const id = `toast-${++nextId}`
  const toast: Toast = { id, type, title, message, duration }
  toasts = [...toasts, toast]

  if (duration > 0) {
    setTimeout(() => removeToast(id), duration)
  }
}

export function removeToast(id: string): void {
  toasts = toasts.filter(t => t.id !== id)
}

export function toastSuccess(title: string, message?: string): void {
  addToast('success', title, message)
}

export function toastError(title: string, message?: string): void {
  addToast('error', title, message, 8000)
}

export function toastWarning(title: string, message?: string): void {
  addToast('warning', title, message)
}

export function toastInfo(title: string, message?: string): void {
  addToast('info', title, message)
}
