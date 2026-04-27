import { subscribeWsEvents } from '@/shared/api/ws'
import { useAuthStore } from '@/shared/stores/authStore'
import { ErrorBoundary } from '@/shared/ui/error-boundary'
import { Toaster } from '@/shared/ui/toast'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './app/global.css'
import { QueryProvider } from './app/providers/QueryProvider'
import { AppRouter } from './app/router'

useAuthStore.getState().hydrate()
const unsubscribeWsEvents = subscribeWsEvents()

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    unsubscribeWsEvents()
  })
}

const root = document.getElementById('root')
if (!root) throw new Error('Root element not found')

createRoot(root).render(
  <StrictMode>
    <ErrorBoundary>
      <Toaster richColors position="top-right" closeButton />
      <QueryProvider>
        <AppRouter />
      </QueryProvider>
    </ErrorBoundary>
  </StrictMode>,
)
