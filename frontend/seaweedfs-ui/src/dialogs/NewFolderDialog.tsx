import React from 'react'
import { useTheme } from '@mui/material'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button'
import Alert from '@mui/material/Alert'
import FormGroup from '@mui/material/FormGroup';
import FormControlLabel from '@mui/material/FormControlLabel';
import Switch from '@mui/material/Switch'

import Filer from '../seaweedfs/filer'
import { LocationContext } from '../context/LocationContextWrapper'
import { getFullPath } from '../seaweedfs/file'
import type { DialogProps } from '../types/dialogs'

function NewFolderDialog(props: DialogProps): React.ReactElement {
    const { open, close } = props
    const context = React.useContext(LocationContext)
    const theme = useTheme()

    const [folder, setFolder] = React.useState('')
    const [goTo, setGoTo] = React.useState(false)
    const [error, setError] = React.useState('')

    function handleClose() {
        setFolder('')
        setGoTo(false)
        setError('')
        close()
    }

    async function submit() {
        setError('')
        try {
            const fullPath = getFullPath(folder, context.currentLocation)
            await Filer.createFolder(fullPath)
            context.refresh()
            if (goTo) {
                context.updateLocation(fullPath)
            }
            handleClose()
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Create folder failed')
        }
    }

    function isValid() {
        return folder !== ''
    }

    return (
        <Dialog
            open={open}
            onClose={handleClose}
            fullWidth
        >
            <DialogTitle sx={{ textAlign: 'center' }}>
                Create Folder
            </DialogTitle>
            <DialogContent>
                {error ? <Alert severity="error" sx={{ mb: 1 }} onClose={() => setError('')}>{error}</Alert> : null}
                <TextField
                    required
                    fullWidth
                    sx={{marginTop: theme.spacing(1)}}
                    label="Folder Name"
                    inputProps={{"aria-label": "folder name"}}
                    role="textbox"
                    value={folder}
                    onChange={(event) =>{
                        setFolder(event.target.value)
                    }}
                />
                <FormGroup>
                    <FormControlLabel
                        sx={{marginTop: theme.spacing(1)}} 
                        control={
                            <Switch
                                checked={goTo}
                                onChange={event => {
                                    setGoTo(event.target.checked)
                                }}
                                inputProps={{ "aria-label": "go to new folder"}}
                            />
                        } 
                        label="Go To New Folder"
                        labelPlacement="start"
                    />
                </FormGroup>
            </DialogContent>
            <DialogActions>
                <Button 
                    variant="contained"
                    aria-label="close"
                    onClick={handleClose}
                >
                    Close
                </Button>
                <Button 
                    variant="contained"
                    aria-label="submit"
                    onClick={submit}
                    disabled={!isValid()}
                >
                    Submit
                </Button>
            </DialogActions>
        </Dialog>
    )
}

export default NewFolderDialog