import React from 'react'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogContentText from '@mui/material/DialogContentText'
import DialogActions from '@mui/material/DialogActions'
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button'
import type { SelectedItem } from '../types/context'

interface CopyMoveDialogProps {
  open: boolean
  close: () => void
  currentLocation: string
  items: SelectedItem[]
  onCopy: (destPath: string) => Promise<void>
  onMove: (destPath: string) => Promise<void>
}

function CopyMoveDialog(props: CopyMoveDialogProps): React.ReactElement {
  const { open, close, currentLocation, items, onCopy, onMove } = props
  const [destPath, setDestPath] = React.useState(currentLocation)
  const [loading, setLoading] = React.useState(false)
  const [moveConfirmOpen, setMoveConfirmOpen] = React.useState(false)

  React.useEffect(() => {
    if (open) {
      setDestPath(currentLocation)
      setMoveConfirmOpen(false)
    }
  }, [open, currentLocation])

  async function handleCopy(): Promise<void> {
    setLoading(true)
    try {
      await onCopy(destPath.trim() || '/')
      close()
    } finally {
      setLoading(false)
    }
  }

  function openMoveConfirm(): void {
    setMoveConfirmOpen(true)
  }

  async function handleMove(): Promise<void> {
    setLoading(true)
    try {
      await onMove(destPath.trim() || '/')
      setMoveConfirmOpen(false)
      close()
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <Dialog open={open} onClose={close} fullWidth maxWidth="sm">
        <DialogTitle>Copy / Move to folder</DialogTitle>
        <DialogContent>
          <TextField
            fullWidth
            label="Destination path"
            value={destPath}
            onChange={(e) => setDestPath(e.target.value)}
            margin="normal"
            autoFocus
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={close}>Cancel</Button>
          <Button onClick={() => void handleCopy()} disabled={loading}>
            Copy here
          </Button>
          <Button variant="contained" onClick={openMoveConfirm} disabled={loading}>
            Move here
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={moveConfirmOpen} onClose={() => setMoveConfirmOpen(false)}>
        <DialogTitle>Confirm move</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to perform this action? Items will be moved and cannot be recovered from their original location.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setMoveConfirmOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void handleMove()} disabled={loading}>
            Move
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}

export default CopyMoveDialog
