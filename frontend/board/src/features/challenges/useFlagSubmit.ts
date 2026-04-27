import { api, apiErrorMessage, isApiError } from '@/shared/api/client'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useEffect, useReducer } from 'react'
import { useInvalidateChallenges } from './useChallenges'

export type SubmitState =
  | { type: 'idle' }
  | { type: 'correct' }
  | { type: 'incorrect'; message: string }
  | { type: 'rate_limited'; seconds: number }
  | { type: 'max_attempts' }
  | { type: 'submission_closed'; message: string }
  | { type: 'error'; message: string }

type Action =
  | { type: 'SUBMIT_CORRECT' }
  | { type: 'SUBMIT_INCORRECT'; message: string }
  | { type: 'SUBMIT_RATE_LIMIT'; seconds: number }
  | { type: 'SUBMIT_MAX_ATTEMPTS' }
  | { type: 'SUBMIT_CLOSED'; message: string }
  | { type: 'SUBMIT_ERROR'; message: string }
  | { type: 'TICK' }
  | { type: 'RESET' }

const SUBMISSION_CLOSED_CODES = new Set([
  'SUBMISSION_NOT_ALLOWED',
  'COMPETITION_ENDED',
  'COMPETITION_PAUSED',
  'COMPETITION_NOT_STARTED',
  'COMPETITION_FROZEN',
])

function reducer(state: SubmitState, action: Action): SubmitState {
  switch (action.type) {
    case 'SUBMIT_CORRECT':
      return { type: 'correct' }
    case 'SUBMIT_INCORRECT':
      return { type: 'incorrect', message: action.message }
    case 'SUBMIT_RATE_LIMIT':
      return { type: 'rate_limited', seconds: action.seconds }
    case 'SUBMIT_MAX_ATTEMPTS':
      return { type: 'max_attempts' }
    case 'SUBMIT_CLOSED':
      return { type: 'submission_closed', message: action.message }
    case 'SUBMIT_ERROR':
      return { type: 'error', message: action.message }
    case 'TICK':
      if (state.type !== 'rate_limited') return state
      if (state.seconds <= 1) return { type: 'idle' }
      return { ...state, seconds: state.seconds - 1 }
    case 'RESET':
      return { type: 'idle' }
  }
}

export function useFlagSubmit(challengeId: string) {
  const invalidate = useInvalidateChallenges()
  const qc = useQueryClient()
  const [submitState, dispatch] = useReducer(reducer, { type: 'idle' })

  const isRateLimited = submitState.type === 'rate_limited'

  useEffect(() => {
    if (!isRateLimited) return
    const id = setInterval(() => {
      dispatch({ type: 'TICK' })
    }, 1000)
    return () => clearInterval(id)
  }, [isRateLimited])

  const { mutate, isPending } = useMutation({
    mutationFn: async (flag: string) => {
      const { data, error } = await api.POST('/challenges/{challengeID}/submit', {
        params: { path: { challengeID: challengeId } },
        body: { flag },
      })
      if (error) throw error
      return data
    },
    onSuccess: (data) => {
      if (data?.correct) {
        dispatch({ type: 'SUBMIT_CORRECT' })
        invalidate()
        qc.invalidateQueries({ queryKey: ['scoreboard'] })
        qc.invalidateQueries({ queryKey: ['my-team'] })
        qc.invalidateQueries({ queryKey: ['notifications'] })
        const pts = (data as { points?: number }).points
        toast.success(pts != null ? `Correct flag! +${pts} pts` : 'Correct flag!')
      } else {
        const msg = data?.message ?? 'Incorrect flag.'
        dispatch({ type: 'SUBMIT_INCORRECT', message: msg })
        toast.error(msg)
      }
    },
    onError: (err) => {
      const code = isApiError(err) ? err.code : undefined
      if (code === 'RATE_LIMIT_EXCEEDED') {
        const raw = (err as { retryAfter?: unknown }).retryAfter
        const seconds = typeof raw === 'number' && raw > 0 ? Math.ceil(raw) : 30
        dispatch({ type: 'SUBMIT_RATE_LIMIT', seconds })
        toast.error(`Rate limited - try again in ${seconds} second${seconds === 1 ? '' : 's'}`)
      } else if (code === 'MAX_ATTEMPTS_REACHED') {
        dispatch({ type: 'SUBMIT_MAX_ATTEMPTS' })
        toast.error('Maximum attempts reached for this challenge')
      } else if (code && SUBMISSION_CLOSED_CODES.has(code)) {
        const msg = apiErrorMessage(err, 'Submissions are currently closed.')
        dispatch({ type: 'SUBMIT_CLOSED', message: msg })
        toast.error(msg)
        qc.invalidateQueries({ queryKey: ['competition-status'] })
      } else {
        const msg = apiErrorMessage(err, 'Submission failed.')
        dispatch({ type: 'SUBMIT_ERROR', message: msg })
        toast.error(msg)
      }
    },
  })

  const reset = () => dispatch({ type: 'RESET' })

  return { submitState, isPending, submit: mutate, reset }
}
