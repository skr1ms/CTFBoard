import { cn } from '@/shared/lib/utils'
import ReactMarkdown from 'react-markdown'
import rehypeRaw from 'rehype-raw'
import rehypeSanitize from 'rehype-sanitize'

export interface MarkdownRendererProps {
  content: string
  className?: string
  allowHtml?: boolean
}

export function MarkdownRenderer({ content, className, allowHtml = false }: MarkdownRendererProps) {
  return (
    <div
      className={cn(
        'prose prose-invert max-w-none',
        'prose-headings:text-text-primary prose-headings:font-semibold',
        'prose-p:text-text-secondary prose-p:leading-relaxed',
        'prose-a:text-cosmic-blue prose-a:no-underline hover:prose-a:underline',
        'prose-strong:text-text-primary',
        'prose-code:text-cosmic-purple prose-code:bg-space-dark prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-code:text-sm prose-code:font-mono prose-code:before:content-none prose-code:after:content-none',
        'prose-pre:bg-space-dark prose-pre:border prose-pre:border-space-border prose-pre:rounded-lg prose-pre:overflow-x-auto',
        'prose-blockquote:border-l-cosmic-blue prose-blockquote:text-text-muted prose-blockquote:not-italic',
        'prose-ul:text-text-secondary prose-ol:text-text-secondary',
        'prose-li:text-text-secondary',
        'prose-hr:border-space-border',
        'prose-table:border-collapse prose-thead:border-space-border prose-tbody:border-space-border',
        'prose-th:border prose-th:border-space-border prose-th:bg-space-card prose-th:px-3 prose-th:py-2 prose-th:text-text-primary',
        'prose-td:border prose-td:border-space-border prose-td:px-3 prose-td:py-2 prose-td:text-text-secondary',
        className,
      )}
    >
      <ReactMarkdown rehypePlugins={allowHtml ? [rehypeRaw, rehypeSanitize] : [rehypeSanitize]}>
        {content}
      </ReactMarkdown>
    </div>
  )
}
