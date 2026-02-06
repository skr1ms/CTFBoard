import React from 'react'
import type { LocationContextValue } from '../types/context'

const defaultLocationContext: LocationContextValue = {
  currentLocation: '/',
  history: [],
  refreshCount: 0,
  updateLocation: () => {},
  goBack: () => {},
  refresh: () => {},
}

const STORAGE_PATH_KEY = 'seaweedfs-ui-currentPath'

function getStoredPath(): string {
  try {
    const s = localStorage.getItem(STORAGE_PATH_KEY)
    return s && s.startsWith('/') ? s : '/'
  } catch {
    return '/'
  }
}

const LocationContext = React.createContext<LocationContextValue>(defaultLocationContext)

interface LocationContextWrapperProps {
  children: React.ReactNode
}

function LocationContextWrapper(props: LocationContextWrapperProps) {
  const [locationState, setLocationState] = React.useState(() => ({
    currentLocation: getStoredPath(),
    history: [] as string[],
    refreshCount: 0,
  }))

  const updateLocation = React.useCallback((newLocation: string) => {
    const path = newLocation.startsWith('/') ? newLocation : `/${newLocation}`
    try {
      localStorage.setItem(STORAGE_PATH_KEY, path)
    } catch {}
    setLocationState((prev) => ({
      history: [...prev.history, prev.currentLocation],
      currentLocation: path,
      refreshCount: prev.refreshCount,
    }))
  }, [])

  const goBack = React.useCallback(() => {
    setLocationState((prev) => {
      const newHistory = [...prev.history]
      const newLocation = newHistory.pop() ?? '/'
      try {
        localStorage.setItem(STORAGE_PATH_KEY, newLocation)
      } catch {}
      return {
        history: newHistory,
        currentLocation: newLocation,
        refreshCount: prev.refreshCount,
      }
    })
  }, [])

  const refresh = React.useCallback(() => {
    setLocationState((prev) => ({
      ...prev,
      refreshCount: prev.refreshCount + 1,
    }))
  }, [])

  const value: LocationContextValue = {
    ...locationState,
    updateLocation,
    goBack,
    refresh,
  }

  return (
    <LocationContext.Provider value={value}>
      {props.children}
    </LocationContext.Provider>
  )
}

export default LocationContextWrapper
export { LocationContext }
