import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
  /** Rendered when a ChunkLoadError occurs (stale JS chunks after deploy). */
  chunkErrorFallback?: ReactNode
  /** Rendered for any other unhandled render error. */
  fallback?: ReactNode
}

interface State {
  error: Error | null
}

/**
 * Class-based error boundary that distinguishes ChunkLoadErrors (stale chunks
 * after a new deploy) from generic render errors and shows appropriate UI.
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack)
  }

  render() {
    const { error } = this.state
    if (!error) return this.props.children

    if (error.name === 'ChunkLoadError') {
      return (
        this.props.chunkErrorFallback ?? (
          <div className="flex min-h-screen items-center justify-center p-8 text-center">
            <div className="flex flex-col items-center gap-4">
              <p className="text-text-muted text-sm">A new version of the app is available.</p>
              <button
                onClick={() => window.location.reload()}
                className="text-sm text-cosmic-blue hover:underline"
              >
                Reload page
              </button>
            </div>
          </div>
        )
      )
    }

    return (
      this.props.fallback ?? (
        <div className="flex min-h-screen items-center justify-center p-8 text-center">
          <div className="flex flex-col items-center gap-4">
            <span className="text-5xl font-bold text-nova-red">500</span>
            <p className="text-text-muted text-sm">Something went wrong.</p>
            <button
              onClick={() => window.location.reload()}
              className="text-sm text-cosmic-blue hover:underline"
            >
              Reload page
            </button>
          </div>
        </div>
      )
    )
  }
}
