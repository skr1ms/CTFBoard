import { Link } from 'react-router'

export function Page403() {
  return (
    <div className="text-center flex flex-col items-center gap-4">
      <div
        className="text-7xl font-bold text-cosmic-purple"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        403
      </div>
      <h1 className="text-xl font-semibold text-text-primary">Access Denied</h1>
      <p className="text-text-muted text-sm max-w-xs">
        You don't have permission to access this area of the galaxy.
      </p>
      <Link to="/challenges" className="mt-2 text-sm text-cosmic-blue hover:underline">
        Back to Challenges
      </Link>
    </div>
  )
}
