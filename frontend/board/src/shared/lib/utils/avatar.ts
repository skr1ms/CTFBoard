import { env } from '@/shared/config/env'

export function avatarSrc(path?: string | null): string | null {
  if (!path) return null
  return `${env.apiBaseUrl}/avatars/${path}`
}
