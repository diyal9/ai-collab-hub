export const APP_BASE = '/ai-collab-hub'

/** 将 redirect 参数规范为 vue-router 路径（去掉部署前缀） */
export function normalizeRedirect(raw: string | null | undefined): string {
  if (!raw || typeof raw !== 'string') return '/'

  let path = raw.trim()
  if (!path) return '/'

  if (path.startsWith(APP_BASE)) {
    path = path.slice(APP_BASE.length) || '/'
  }

  if (!path.startsWith('/')) {
    path = `/${path}`
  }

  if (path.startsWith('/login')) {
    return '/'
  }

  return path
}
