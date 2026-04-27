import { useEffect, useState } from 'react'
import { Link } from 'react-router'

const COUNTDOWN_SECONDS = 60

export function Page429() {
  const [seconds, setSeconds] = useState(COUNTDOWN_SECONDS)

  useEffect(() => {
    if (seconds <= 0) return
    const id = setInterval(() => setSeconds((s) => s - 1), 1000)
    return () => clearInterval(id)
  }, [seconds])

  return (
    <div className="text-center flex flex-col items-center gap-4">
      <div
        className="text-7xl font-bold text-amber-400"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        429
      </div>
      <h1 className="text-xl font-semibold text-text-primary">Rate Limit Exceeded</h1>
      <p className="text-text-muted text-sm max-w-xs">
        You've sent too many requests. The cosmic traffic control is slowing you down.
      </p>
      {seconds > 0 ? (
        <p className="text-sm text-amber-400">
          Try again in <span className="font-mono font-bold">{seconds}s</span>
        </p>
      ) : (
        <Link
          to="/"
          className="mt-2 inline-block px-4 py-2 rounded-[var(--radius-md)] bg-cosmic-blue text-sm text-white hover:bg-cosmic-blue/80 transition-colors"
        >
          Try again
        </Link>
      )}
    </div>
  )
}
