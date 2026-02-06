import React from 'react'
import { Typography, useTheme } from '@mui/material'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import DialogContentText from '@mui/material/DialogContentText'
import Button from '@mui/material/Button'
import Grid from '@mui/material/Grid'

import DescriptionIcon from '@mui/icons-material/Description'
import { Folder as FolderIcon } from '@mui/icons-material'
import type { DeleteItemConfirmationProps } from '../types/dialogs'

function DeleteItemConfirmation(props: DeleteItemConfirmationProps): React.ReactElement {
    const theme = useTheme()
    const { del, close, name, open, isFile } = props

    function handleConfirm(): void {
        void Promise.resolve(del()).then(close)
    }

    function handleShortcuts(event: React.KeyboardEvent): void {
        if (event.code === 'KeyY') handleConfirm()
        else if (event.code === 'KeyN') close()
    }

    return (
        <Dialog
            open={open}
            onClose={close}
            fullWidth
            onKeyDown={handleShortcuts}
        >
            <DialogTitle sx={{ textAlign: 'center' }}>
                Are you sure you want to delete?
            </DialogTitle>
            <DialogContent>
                <DialogContentText>
                    This action cannot be undone. This {isFile ? "file" : "folder"} will be permanently deleted.
                </DialogContentText>
                <Grid
                    container
                    spacing={1}
                    sx={{marginTop: theme.spacing(1)}}
                >
                    <Grid item xs={6} sx={{ textAlign: 'center' }}>
                        {isFile ?
                            <DescriptionIcon sx={{fontSize: "128px"}} /> :
                            <FolderIcon sx={{fontSize: "128px"}} />
                        }
                    </Grid>
                    <Grid item xs={6}>
                        <Typography>
                            {name}
                        </Typography>
                        <Typography>
                            Type: {isFile ? "File" : "Folder"}
                        </Typography>
                    </Grid>
                </Grid>
            </DialogContent>
            <DialogActions>
                <Button 
                    variant="contained"
                    aria-label="cancel"
                    onClick={close}
                >
                    Cancel
                </Button>
                <Button 
                    variant="contained"
                    aria-label="confirm"
                    onClick={handleConfirm}
                >
                    Confirm
                </Button>
            </DialogActions>
        </Dialog>
    )
}

export default DeleteItemConfirmation