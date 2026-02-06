export interface DialogProps {
  open: boolean
  close: () => void
}

export interface DeleteItemConfirmationProps extends DialogProps {
  name: string
  del: () => void | Promise<void>
  isFile: boolean
}

export interface DeleteMultipleDialogProps extends DialogProps {
  files: string[]
  del: () => Promise<void>
}

export interface ImageDisplayDialogProps extends DialogProps {
  title: string
  source: string
  download: () => void
}

export interface SuccessAlertProps extends DialogProps {
  text: string
}

export interface UploadFileDialogProps extends DialogProps {
  files?: File[]
}
