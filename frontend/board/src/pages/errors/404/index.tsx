import { Link } from 'react-router'

export function Page404() {
  return (
    <div className="text-center flex flex-col items-center gap-4">
      <div
        className="text-7xl font-bold bg-gradient-to-r from-cosmic-blue to-cosmic-purple bg-clip-text text-transparent"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        404
      </div>
      <h1 className="text-xl font-semibold text-text-primary">Lost in Space</h1>
      <p className="text-text-muted text-sm max-w-xs">
        The page you're looking for has drifted into a black hole.
      </p>
      <Link to="/challenges" className="mt-2 text-sm text-cosmic-blue hover:underline">
        Return to base
      </Link>
    </div>
  )
}
