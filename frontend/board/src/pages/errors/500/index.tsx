import { Link } from 'react-router'

export function Page500() {
  return (
    <div className="text-center flex flex-col items-center gap-4">
      <div
        className="text-7xl font-bold text-nova-red"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        500
      </div>
      <h1 className="text-xl font-semibold text-text-primary">Server Error</h1>
      <p className="text-text-muted text-sm max-w-xs">
        Something exploded on our end. The engineering crew has been notified.
      </p>
      <Link to="/challenges" className="mt-2 text-sm text-cosmic-blue hover:underline">
        Back to Challenges
      </Link>
    </div>
  )
}
