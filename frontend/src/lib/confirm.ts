import { reactive } from 'vue'

interface ConfirmState {
  open: boolean
  title: string
  message: string
  confirmText: string
  danger: boolean
  resolve: ((v: boolean) => void) | null
}

export const confirmState = reactive<ConfirmState>({
  open: false, title: '请确认', message: '', confirmText: '确认', danger: false, resolve: null,
})

export function askConfirm(
  message: string,
  opts: { title?: string; confirmText?: string; danger?: boolean } = {},
): Promise<boolean> {
  return new Promise((resolve) => {
    confirmState.title = opts.title || '请确认'
    confirmState.message = message
    confirmState.confirmText = opts.confirmText || '确认'
    confirmState.danger = !!opts.danger
    confirmState.resolve = resolve
    confirmState.open = true
  })
}

export function resolveConfirm(v: boolean) {
  confirmState.open = false
  const r = confirmState.resolve
  confirmState.resolve = null
  if (r) r(v)
}
