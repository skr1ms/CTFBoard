import React from 'react'
import { useTheme } from '@mui/material'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import DialogContentText from '@mui/material/DialogContentText'
import Button from '@mui/material/Button'
import Accordion from '@mui/material/Accordion'
import AccordionSummary from '@mui/material/AccordionSummary'
import AccordionDetails from '@mui/material/AccordionDetails'
import Typography from '@mui/material/Typography'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem';
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'

import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import ArrowRightIcon from '@mui/icons-material/ArrowRight'
import type { DeleteMultipleDialogProps } from '../types/dialogs'

function DeleteMultipleDialog(props: DeleteMultipleDialogProps): React.ReactElement {
    const theme = useTheme()
    const { del, close, files, open } = props

    function handleConfirm(): void {
        void del().then(close)
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
            sx={{ zIndex: 1400 }}
        >
            <DialogTitle sx={{ textAlign: 'center' }}>
                Are you sure you want to delete?
            </DialogTitle>
            <DialogContent>
                <DialogContentText>
                    This action cannot be undone. Deleted items cannot be recovered.
                </DialogContentText>
                <DialogContentText sx={{ mt: 1, fontWeight: 500 }}>
                    {files.length} item{files.length !== 1 ? 's' : ''} will be permanently deleted.
                </DialogContentText>
                <Accordion
                    sx={{
                        marginTop: theme.spacing(1)
                    }}
                >
                    <AccordionSummary
                        expandIcon={<ExpandMoreIcon />}
                        aria-label="show files"
                    >
                        <Typography>Files</Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <List dense>
                            {files.map(fileName => {
                                return (
                                    <ListItem key={fileName}>
                                        <ListItemIcon>
                                            <ArrowRightIcon />
                                        </ListItemIcon>
                                        <ListItemText
                                            primary={fileName}
                                        />
                                    </ListItem>
                                )
                            })}
                        </List>
                    </AccordionDetails>
                </Accordion>
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

export default DeleteMultipleDialog