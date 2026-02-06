import React from 'react'
import * as authApi from '../api/auth'
import { ADMIN_ROLE } from '../types/auth'

const STORAGE_KEY = 'seaweedfs_ui_access_token'

export interface AuthContextValue {
  token: string | null
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  isAuthenticated: boolean
}

const defaultAuth: AuthContextValue = {
  token: null,
  login: async () => {},
  logout: () => {},
  isAuthenticated: false,
}

const AuthContext = React.createContext<AuthContextValue>(defaultAuth)

interface AuthContextWrapperProps {
  children: React.ReactNode
}

function AuthContextWrapper(props: AuthContextWrapperProps): React.ReactElement {
  const [token, setToken] = React.useState<string | null>(() =>
    sessionStorage.getItem(STORAGE_KEY)
  )

  const login = React.useCallback(async (email: string, password: string) => {
    const pair = await authApi.login(email, password)
    const accessToken = pair.access_token
    const me = await authApi.getMe(accessToken)
    if (me.role !== ADMIN_ROLE) {
      throw new Error('Only admin can access')
    }
    sessionStorage.setItem(STORAGE_KEY, accessToken)
    setToken(accessToken)
  }, [])

  const logout = React.useCallback(() => {
    sessionStorage.removeItem(STORAGE_KEY)
    setToken(null)
  }, [])

  const value: AuthContextValue = {
    token,
    login,
    logout,
    isAuthenticated: !!token,
  }

  return (
    <AuthContext.Provider value={value}>
      {props.children}
    </AuthContext.Provider>
  )
}

export default AuthContextWrapper
export { AuthContext }
