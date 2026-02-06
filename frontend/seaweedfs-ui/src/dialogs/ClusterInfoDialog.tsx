import React from 'react'
import { Grid, IconButton, useTheme } from '@mui/material'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import DialogContentText from '@mui/material/DialogContentText'
import { Box } from '@mui/system'
import CloseIcon from '@mui/icons-material/Close'
import GitHubIcon from '@mui/icons-material/GitHub'

import Master from '../seaweedfs/master'
import type { DialogProps } from '../types/dialogs'
import type { ClusterInfo } from '../types/api'

function ClusterInfoDialog(props: DialogProps): React.ReactElement {
    const { open, close } = props
    const theme = useTheme()
    const [clusterInfo, setClusterInfo] = React.useState<Partial<ClusterInfo>>({})

    React.useEffect(() => {
        if (open) Master.getClusterInfo().then((data) => setClusterInfo(data))
    }, [open])

    function openGit(): void {
        window.open('https://github.com/seaweedfs/seaweedfs', '_blank')
    }

    return (
        <Dialog
            open={open}
            onClose={close}
            fullWidth
        >
            <DialogTitle>
                Cluster Info
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
                            <img height="256px" src={`${import.meta.env.BASE_URL}seaweedfs.png`} alt="seaweed logo" />
                        </Box>
                    </Grid>
                    <Grid item xs={6}>
                        <DialogContentText
                            sx={{
                                marginTop: theme.spacing(1)
                            }}
                        >
                            Version: {clusterInfo.version ?? '—'}
                        </DialogContentText>
                        <DialogContentText>
                            Datacenters: {clusterInfo.datacenters ?? '—'}
                        </DialogContentText>
                        <DialogContentText>
                            Racks: {clusterInfo.racks ?? '—'}
                        </DialogContentText>
                        <DialogContentText>
                            Nodes: {clusterInfo.nodes ?? '—'}
                        </DialogContentText>
                        <DialogContentText>
                            Storage Used: {clusterInfo.size ?? '—'}
                        </DialogContentText>
                    </Grid>
                </Grid>
            </DialogContent>
            <DialogActions sx={{ display: "flex", justifyContent: 'center' }}>
                <IconButton
                    onClick={openGit}
                    aria-label="github"
                >
                    <GitHubIcon fontSize="large" />
                </IconButton>
            </DialogActions>
        </Dialog>
    )
}

export default ClusterInfoDialog