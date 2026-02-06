import React from 'react'
import { Grid, IconButton, useTheme } from '@mui/material'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import DialogContentText from '@mui/material/DialogContentText'
import { Box } from '@mui/system'

import CloseIcon from '@mui/icons-material/Close'
import type { DialogProps } from '../types/dialogs'

function AboutDialog(props: DialogProps): React.ReactElement {
    const { open, close } = props
    const theme = useTheme()

    return (
        <Dialog
            open={open}
            onClose={close}
            fullWidth
        >
            <DialogTitle>
                About
                <IconButton
                    onClick={close}
                    aria-label="close"
                    sx={{
                        position: 'absolute',
                        right: 8,
                        top: 8,
                        color: (theme) => theme.palette.grey[500],
                    }}
                >
                    <CloseIcon fontSize="large" />
                </IconButton>
            </DialogTitle>
            <DialogContent dividers>
                <Grid container spacing={1}>
                    <Grid item xs={6}>
                        <Box sx={{ display: "flex", justifyContent: 'center' }}>
                            <img height="256px" src={`${import.meta.env.BASE_URL}seaweedfs.png`} alt="SeaweedFS" />
                        </Box>
                    </Grid>
                    <Grid item xs={6}>
                        <DialogContentText
                            sx={{
                                marginTop: theme.spacing(1)
                            }}
                        >
                            SeaweedFS UI — web interface for viewing and managing files in SeaweedFS Filer.
                        </DialogContentText>
                        <DialogContentText>
                            React {React.version}
                        </DialogContentText>
                    </Grid>
                </Grid>
            </DialogContent>
        </Dialog>
    )
}

export default AboutDialog
