export interface LoginRequest {
  email: string
  password: string
}

export interface TokenPair {
  access_token: string
  access_expires_at?: number
  refresh_token: string
  refresh_expires_at?: number
}

export interface MeResponse {
  id: string
  username: string
  email: string
  role: string
  team_id: string | null
  created_at: string
}

export const ADMIN_ROLE = 'admin'
