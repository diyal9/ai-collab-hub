import { ElMessage } from 'element-plus'

export function notifySuccess(message: string): void {
  ElMessage.success(message)
}

export function notifyError(message: string): void {
  ElMessage.error(message)
}

export function notifyWarning(message: string): void {
  ElMessage.warning(message)
}

export function apiErrorMessage(error: unknown, fallback = '操作失败，请稍后重试'): string {
  if (error && typeof error === 'object' && 'response' in error) {
    const resp = (error as { response?: { data?: { error?: string; message?: string }; status?: number } }).response
    const msg = resp?.data?.error || resp?.data?.message
    if (msg) return msg
    if (resp?.status === 403) return '权限不足'
    if (resp?.status === 404) return '请求的资源不存在'
  }
  if (error instanceof Error && error.message) return error.message
  return fallback
}

export function showApiError(error: unknown, fallback?: string): void {
  notifyError(apiErrorMessage(error, fallback))
}
