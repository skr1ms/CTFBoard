import { api, isApiError } from '@/shared/api/client'
import type { operations } from '@/shared/api/schema.d'
import { useAuthStore } from '@/shared/stores/authStore'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

type CommentItem = NonNullable<
  operations['GetChallengesChallengeIDComments']['responses']['200']['content']['application/json']
>[number]

type RatingItem = NonNullable<
  operations['GetChallengesChallengeIDRatings']['responses']['200']['content']['application/json']
>[number]

type SolutionData = NonNullable<
  operations['GetChallengesChallengeIDSolution']['responses']['200']['content']['application/json']
>

function formatRelative(iso: string | undefined, now: number): string {
  if (!iso) return ''
  const d = new Date(iso)
  const diff = (now - d.getTime()) / 1000
  if (diff < 60) return 'just now'
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return d.toLocaleDateString()
}

// ---------------------------------------------------------------------------
// Comments section (TASK-039)
// ---------------------------------------------------------------------------
export function CommentsSection({ challengeId }: { challengeId: string }) {
  const user = useAuthStore((s) => s.user)
  const qc = useQueryClient()

  const { data: comments = [], isLoading } = useQuery({
    queryKey: ['challenge', challengeId, 'comments'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}/comments', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) throw error
      return (data ?? []) as CommentItem[]
    },
    staleTime: 0,
  })

  const { mutateAsync: postComment, isPending: posting } = useMutation({
    mutationFn: async (content: string) => {
      const { error } = await api.POST('/challenges/{challengeID}/comments', {
        params: { path: { challengeID: challengeId } },
        body: { content },
      })
      if (error) throw error
    },
    onSuccess: () =>
      void qc.invalidateQueries({ queryKey: ['challenge', challengeId, 'comments'] }),
  })

  const { mutateAsync: deleteComment, isPending: deleting } = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/comments/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () =>
      void qc.invalidateQueries({ queryKey: ['challenge', challengeId, 'comments'] }),
  })

  const [text, setText] = useState('')
  const MAX = 2000
  const [now] = useState(() => Date.now())

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!text.trim()) return
    try {
      await postComment(text.trim())
      setText('')
      toast.success('Comment posted')
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Post failed')
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <h3 className="text-sm font-semibold text-text-muted uppercase tracking-wide">Comments</h3>

      {/* Comment list */}
      {isLoading ? (
        <div className="flex flex-col gap-3">
          {[1, 2].map((i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : comments.length === 0 ? (
        <p className="text-sm text-text-muted">No comments yet. Be the first!</p>
      ) : (
        <div className="flex flex-col gap-3">
          {comments.map((c) => (
            <div
              key={c.id}
              className="flex flex-col gap-1 px-3 py-2.5 rounded-[var(--radius-md)] bg-space-dark border border-space-border"
            >
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs font-medium text-cosmic-blue">
                  {c.username ?? 'Anonymous'}
                </span>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-text-muted">
                    {formatRelative(c.created_at, now)}
                  </span>
                  {c.user_id === user?.id && (
                    <button
                      onClick={() =>
                        deleteComment(c.id ?? '').catch((err) =>
                          toast.error(isApiError(err) ? err.message : 'Delete failed'),
                        )
                      }
                      disabled={deleting}
                      className="text-xs text-text-muted hover:text-nova-red transition-colors disabled:opacity-40"
                    >
                      Delete
                    </button>
                  )}
                </div>
              </div>
              <p className="text-sm text-text-secondary whitespace-pre-wrap break-words">
                {c.content}
              </p>
            </div>
          ))}
        </div>
      )}

      {/* Post form */}
      <form onSubmit={handleSubmit} className="flex flex-col gap-2">
        <div className="relative">
          <textarea
            rows={3}
            value={text}
            onChange={(e) => setText(e.target.value.slice(0, MAX))}
            placeholder="Write a comment…"
            className="w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 resize-none"
          />
          <span
            className={`absolute bottom-2 right-2 text-xs ${text.length >= MAX ? 'text-nova-red' : 'text-text-muted'}`}
          >
            {text.length}/{MAX}
          </span>
        </div>
        <div className="flex justify-end">
          <button
            type="submit"
            disabled={posting || !text.trim()}
            className="px-3 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
          >
            {posting ? 'Posting…' : 'Post comment'}
          </button>
        </div>
      </form>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Ratings section (TASK-040)
// ---------------------------------------------------------------------------
function StarRating({
  value,
  onChange,
  readonly,
}: {
  value: number
  onChange?: (v: number) => void
  readonly?: boolean
}) {
  const [hovered, setHovered] = useState(0)
  const display = readonly ? value : hovered || value

  return (
    <div className="flex gap-0.5">
      {[1, 2, 3, 4, 5].map((star) => (
        <button
          key={star}
          type="button"
          disabled={readonly}
          onClick={() => onChange?.(star)}
          onMouseEnter={() => !readonly && setHovered(star)}
          onMouseLeave={() => !readonly && setHovered(0)}
          className={`text-xl transition-colors ${readonly ? 'cursor-default' : 'cursor-pointer hover:scale-110'} ${display >= star ? 'text-solar-yellow' : 'text-space-border'}`}
          aria-label={`${star} star${star !== 1 ? 's' : ''}`}
        >
          ★
        </button>
      ))}
    </div>
  )
}

export function RatingsSection({ challengeId, solved }: { challengeId: string; solved: boolean }) {
  const qc = useQueryClient()

  const { data: ratings = [], isLoading } = useQuery({
    queryKey: ['challenge', challengeId, 'ratings'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}/ratings', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) throw error
      return (data ?? []) as RatingItem[]
    },
    staleTime: 0,
  })

  const { mutateAsync: submitRating, isPending: submitting } = useMutation({
    mutationFn: async ({ value, review }: { value: number; review?: string }) => {
      const body: { value: number; review?: string } = { value }
      if (review?.trim()) body.review = review.trim()
      const { error } = await api.PUT('/challenges/{challengeID}/rating', {
        params: { path: { challengeID: challengeId } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['challenge', challengeId, 'ratings'] })
      toast.success('Rating submitted')
    },
  })

  const [myRating, setMyRating] = useState(0)
  const [myReview, setMyReview] = useState('')

  const avg =
    ratings.length > 0
      ? (ratings.reduce((s, r) => s + (r.value ?? 0), 0) / ratings.length).toFixed(1)
      : null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!myRating) return
    try {
      await submitRating({ value: myRating, review: myReview })
      setMyReview('')
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Rating failed')
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <h3 className="text-sm font-semibold text-text-muted uppercase tracking-wide">Ratings</h3>
        {avg && (
          <div className="flex items-center gap-1">
            <span className="text-solar-yellow">★</span>
            <span className="text-sm font-medium text-text-primary">{avg}</span>
            <span className="text-xs text-text-muted">({ratings.length})</span>
          </div>
        )}
      </div>

      {/* Submit rating (solved only) */}
      {solved ? (
        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-3 p-3 rounded-[var(--radius-md)] border border-space-border bg-space-dark/40"
        >
          <p className="text-xs text-text-muted">Rate this challenge:</p>
          <StarRating value={myRating} onChange={setMyRating} />
          <textarea
            rows={2}
            value={myReview}
            onChange={(e) => setMyReview(e.target.value)}
            placeholder="Optional review…"
            className="w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 resize-none"
          />
          <div className="flex justify-end">
            <button
              type="submit"
              disabled={submitting || !myRating}
              className="px-3 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
            >
              {submitting ? 'Submitting…' : 'Submit rating'}
            </button>
          </div>
        </form>
      ) : (
        <p className="text-xs text-text-muted">Solve this challenge to rate it.</p>
      )}

      {/* Ratings list */}
      {isLoading ? (
        <div className="flex flex-col gap-2">
          {[1, 2].map((i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      ) : ratings.length === 0 ? (
        <p className="text-sm text-text-muted">No ratings yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {ratings.map((r) => (
            <div
              key={r.id}
              className="flex flex-col gap-1 px-3 py-2 rounded-[var(--radius-md)] bg-space-dark border border-space-border"
            >
              <StarRating value={r.value ?? 0} readonly />
              {r.review && <p className="text-sm text-text-secondary">{r.review}</p>}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Solution / writeup section (TASK-041)
// ---------------------------------------------------------------------------
export function SolutionSection({ challengeId }: { challengeId: string }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['challenge', challengeId, 'solution'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}/solution', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) throw error
      return data as SolutionData | null
    },
    staleTime: 60_000,
    retry: false,
  })

  if (isLoading) return <Skeleton className="h-32 w-full" />

  if (error) {
    const msg = isApiError(error) ? error.message : ''
    if (msg.toLowerCase().includes('disabled') || msg.toLowerCase().includes('writeup')) {
      return <p className="text-sm text-text-muted">Writeups are disabled for this competition.</p>
    }
    return <p className="text-sm text-text-muted">Solution not available.</p>
  }

  if (!data?.content && (!data?.files || data.files.length === 0)) {
    return <p className="text-sm text-text-muted">No solution posted yet.</p>
  }

  return (
    <div className="flex flex-col gap-4">
      <h3 className="text-sm font-semibold text-text-muted uppercase tracking-wide">Solution</h3>

      {data?.content && (
        <div className="rounded-[var(--radius-md)] border border-space-border bg-space-dark/40 p-4">
          <MarkdownRenderer content={data.content} />
        </div>
      )}

      {data?.files && data.files.length > 0 && (
        <div className="flex flex-col gap-2">
          <p className="text-xs font-medium text-text-muted uppercase tracking-wide">Files</p>
          <div className="flex flex-wrap gap-2">
            {data.files.map((f) => (
              <a
                key={f.id}
                href={f.url ?? '#'}
                download={f.filename}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 px-3 py-2 rounded-[var(--radius-md)] border border-space-border bg-space-dark hover:border-cosmic-blue/50 hover:bg-cosmic-blue/5 transition-colors text-sm text-text-secondary hover:text-text-primary"
              >
                <svg
                  className="w-4 h-4 shrink-0 text-cosmic-blue"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                >
                  <path d="M.5 9.9a.5.5 0 0 1 .5.5v2.5a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-2.5a.5.5 0 0 1 1 0v2.5a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2v-2.5a.5.5 0 0 1 .5-.5z" />
                  <path d="M7.646 11.854a.5.5 0 0 0 .708 0l3-3a.5.5 0 0 0-.708-.708L8.5 10.293V1.5a.5.5 0 0 0-1 0v8.793L5.354 8.146a.5.5 0 1 0-.708.708l3 3z" />
                </svg>
                <span className="truncate max-w-[160px]">{f.filename ?? 'Download'}</span>
              </a>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
