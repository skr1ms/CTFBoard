import React from 'react'
import Snackbar from '@mui/material/Snackbar'
import Alert from '@mui/material/Alert'
import IconButton from '@mui/material/IconButton'
import CloseIcon from '@mui/icons-material/Close'
import type { SuccessAlertProps } from '../types/dialogs'

interface ActionProps {
  close: () => void
}

function Action(props: ActionProps): React.ReactElement {
    const { close } = props
    return (
        <IconButton
            aria-label="close alert"
            color="inherit"
            size="small"
            onClick={close}
        >
            <CloseIcon fontSize="inherit" />
        </IconButton>
    )
}

function SuccessAlert(props: SuccessAlertProps): React.ReactElement {
    const { open, close, text } = props

    return (
        <Snackbar
            open={open}
            autoHideDuration={5000}
            onClose={close}
            anchorOrigin={{
                vertical: "bottom",
                horizontal: "center"
            }}
        >
            <Alert
                variant="filled"
                severity="success"
                action={<Action close={close} />}
            >
                {text}
            </Alert>
        </Snackbar>
    )

}

export default SuccessAlert