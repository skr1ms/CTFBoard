import { Button } from '@/shared/ui/button'
import { Modal } from '@/shared/ui/modal'

export interface ConfirmationDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  title?: string
  message: React.ReactNode
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  loading?: boolean
}

export function ConfirmationDialog({
  open,
  onClose,
  onConfirm,
  title = 'Confirm action',
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  danger = false,
  loading = false,
}: ConfirmationDialogProps) {
  return (
    <Modal open={open} onClose={onClose} title={title} maxWidth="max-w-md">
      <div className="flex flex-col gap-5">
        <div className="text-text-secondary text-sm leading-relaxed">{message}</div>
        <div className="flex gap-3 justify-end">
          <Button variant="ghost" onClick={onClose} disabled={loading}>
            {cancelLabel}
          </Button>
          <Button
            data-testid="hint-confirm-yes"
            variant={danger ? 'danger' : 'primary'}
            onClick={onConfirm}
            loading={loading}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </Modal>
  )
}
