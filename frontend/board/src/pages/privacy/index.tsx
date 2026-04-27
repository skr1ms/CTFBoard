import { env } from '@/shared/config/env'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { Skeleton } from '@/shared/ui/skeleton'
import { useQuery } from '@tanstack/react-query'

export function PrivacyPage() {
  const {
    data: content,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ['static', 'privacy'],
    queryFn: async () => {
      const res = await fetch(`${env.apiBaseUrl}/privacy`)
      if (!res.ok) throw new Error('Failed to load')
      return res.text()
    },
    staleTime: Infinity,
  })

  if (isLoading) {
    return (
      <div className="max-w-3xl mx-auto px-4 py-10 flex flex-col gap-4">
        {[1, 2, 3, 4, 5].map((i) => (
          <Skeleton key={i} className="h-4 w-full" />
        ))}
      </div>
    )
  }

  if (isError || !content) {
    return (
      <div className="max-w-3xl mx-auto px-4 py-10">
        <p className="text-sm text-nova-red">Privacy Policy not available.</p>
      </div>
    )
  }

  return (
    <div className="max-w-3xl mx-auto px-4 py-10">
      <MarkdownRenderer content={content} allowHtml />
    </div>
  )
}
