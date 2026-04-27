import { env } from '@/shared/config/env'
import { Avatar } from '@/shared/ui/avatar'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card } from '@/shared/ui/card'
import { CodeBlock } from '@/shared/ui/code-block'
import { ConfirmationDialog } from '@/shared/ui/confirmation-dialog'
import { DateTimePicker } from '@/shared/ui/date-time-picker'
import { Dropdown } from '@/shared/ui/dropdown'
import { FileUpload } from '@/shared/ui/file-upload'
import { Input } from '@/shared/ui/input'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { Modal } from '@/shared/ui/modal'
import { Pagination } from '@/shared/ui/pagination'
import { SearchInput } from '@/shared/ui/search-input'
import { SkeletonText } from '@/shared/ui/skeleton'
import { Spinner } from '@/shared/ui/spinner'
import { StarRating } from '@/shared/ui/star-rating'
import { Table } from '@/shared/ui/table'
import { Tabs } from '@/shared/ui/tabs'
import { Toaster } from '@/shared/ui/toast'
import { useState } from 'react'

type Row = { id: number; name: string; score: number }

const rows: Row[] = Array.from({ length: 5 }, (_, i) => ({
  id: i + 1,
  name: `Team ${i + 1}`,
  score: (5 - i) * 1000,
}))

const SAMPLE_MD = `## Markdown\n\n**Bold** and *italic* text.\n\n\`inline code\`\n\n> A blockquote\n`
const SAMPLE_CODE = `function solve(input: string): string {\n  return input.split('').reverse().join('')\n}`

export function App() {
  const [modalOpen, setModalOpen] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [page, setPage] = useState(1)
  const [tab, setTab] = useState('all')
  const [rating, setRating] = useState(3)
  const [pickedDate, setPickedDate] = useState<Date | undefined>(new Date())
  const [query, setQuery] = useState('')

  return (
    <div className="min-h-full p-8 flex flex-col gap-8 max-w-3xl mx-auto">
      <Toaster />

      <h1
        style={{ fontFamily: 'var(--font-display)' }}
        className="text-3xl font-bold text-cosmic-blue"
      >
        {env.appName} UI Kit
      </h1>

      {/* Buttons & Spinner */}
      <div className="flex flex-wrap gap-2">
        <Button variant="primary">Primary</Button>
        <Button variant="secondary">Secondary</Button>
        <Button variant="danger">Danger</Button>
        <Button variant="ghost">Ghost</Button>
        <Button variant="primary" loading>
          Loading
        </Button>
        <Spinner />
      </div>

      {/* Inputs */}
      <div className="flex flex-wrap gap-2">
        <Input label="Username" placeholder="Enter username" />
        <Input label="Error" placeholder="..." error="Required" />
      </div>

      {/* Search */}
      <SearchInput value={query} onChange={setQuery} placeholder="Search challenges…" />

      {/* Badges */}
      <div className="flex flex-wrap gap-2">
        <Badge>Default</Badge>
        <Badge variant="blue">Blue</Badge>
        <Badge variant="purple">Purple</Badge>
        <Badge variant="green">Solved</Badge>
        <Badge variant="red">Error</Badge>
      </div>

      {/* Avatar */}
      <div className="flex gap-3 items-center">
        <Avatar alt="Alice" size="sm" />
        <Avatar alt="Bob" size="md" />
        <Avatar alt="Carol" size="lg" />
        <Avatar src="https://i.pravatar.cc/80" alt="Dave" size="xl" />
      </div>

      {/* Star Rating */}
      <StarRating value={rating} onChange={setRating} />

      {/* Card */}
      <Card glow>
        <p className="text-text-primary">Card with glow hover</p>
        <SkeletonText lines={2} className="mt-2" />
      </Card>

      {/* Code Block */}
      <CodeBlock code={SAMPLE_CODE} language="typescript" />

      {/* Markdown */}
      <MarkdownRenderer content={SAMPLE_MD} />

      {/* File Upload */}
      <FileUpload onFile={(file) => console.log(file)} maxSizeMb={5} />

      {/* Date Time Picker */}
      <DateTimePicker
        label="Event start"
        {...(pickedDate !== undefined ? { value: pickedDate } : {})}
        onChange={setPickedDate}
      />

      {/* Modal */}
      <Button onClick={() => setModalOpen(true)}>Open Modal</Button>
      <Modal open={modalOpen} onClose={() => setModalOpen(false)} title="Test Modal">
        <p className="text-text-secondary">Modal content. Press Esc or click overlay to close.</p>
      </Modal>

      {/* Confirmation Dialog */}
      <Button variant="danger" onClick={() => setConfirmOpen(true)}>
        Delete something
      </Button>
      <ConfirmationDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => setConfirmOpen(false)}
        title="Confirm deletion"
        message="This action cannot be undone."
        confirmLabel="Delete"
        danger
      />

      {/* Table */}
      <Table<Row>
        columns={[
          { key: 'rank', header: '#', cell: (_, i) => i + 1 },
          { key: 'name', header: 'Team', cell: (r) => r.name },
          { key: 'score', header: 'Score', cell: (r) => r.score },
        ]}
        data={rows}
        keyFn={(r) => r.id}
      />

      {/* Pagination */}
      <Pagination page={page} totalPages={10} onChange={setPage} />

      {/* Tabs */}
      <Tabs
        tabs={[
          { value: 'all', label: 'All' },
          { value: 'web', label: 'Web' },
          { value: 'crypto', label: 'Crypto' },
        ]}
        value={tab}
        onChange={setTab}
      />

      {/* Dropdown */}
      <Dropdown
        trigger={<Button variant="secondary">Actions ▾</Button>}
        items={[
          { value: 'edit', label: 'Edit' },
          { value: 'delete', label: 'Delete', danger: true },
        ]}
        onSelect={(v) => console.log(v)}
      />
    </div>
  )
}
