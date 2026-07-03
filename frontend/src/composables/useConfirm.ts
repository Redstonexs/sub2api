import { reactive } from 'vue'

export interface ConfirmOptions {
  /** Dialog title; ConfirmDialogHost falls back to t('common.confirm') when empty */
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

interface ConfirmState {
  show: boolean
  title: string
  message: string
  confirmText: string | undefined
  cancelText: string | undefined
  danger: boolean
}

// Module-scope singleton: every caller shares the one dialog rendered by ConfirmDialogHost
const state = reactive<ConfirmState>({
  show: false,
  title: '',
  message: '',
  confirmText: undefined,
  cancelText: undefined,
  danger: false
})

let pendingResolve: ((value: boolean) => void) | null = null

function confirm(options: ConfirmOptions): Promise<boolean> {
  // A new request while one is open cancels the previous one
  pendingResolve?.(false)

  state.title = options.title ?? ''
  state.message = options.message
  state.confirmText = options.confirmText
  state.cancelText = options.cancelText
  state.danger = options.danger ?? false
  state.show = true

  return new Promise<boolean>((resolve) => {
    pendingResolve = resolve
  })
}

function resolveConfirm(result: boolean) {
  state.show = false
  const resolve = pendingResolve
  pendingResolve = null
  resolve?.(result)
}

export function useConfirm() {
  return { confirm }
}

// Internal API for ConfirmDialogHost
export function useConfirmHost() {
  return { state, resolveConfirm }
}
