import React from 'react'
import { useTheme } from '@mui/material'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button'

import Filer from '../seaweedfs/filer'
import { LocationContext } from '../context/LocationContextWrapper'
import { getFullPath } from '../seaweedfs/file'
import type { DialogProps } from '../types/dialogs'

const blankForm = { name: '', content: '' }

function TextFileDialog(props: DialogProps): React.ReactElement {
    const { open, close } = props
    const context = React.useContext(LocationContext)
    const theme = useTheme()
    const [form, setForm] = React.useState(blankForm)

    function handleClose(): void {
        setForm(blankForm)
        close()
    }

    async function submit(): Promise<void> {
        const fullPath = getFullPath(form.name, context.currentLocation)
        const file = new File([form.content], form.name, { type: 'text/richtext' })
        await Filer.uploadFile(fullPath, file)
        context.refresh()
        handleClose()
    }

    function isValid(): boolean {
        return form.name !== '' && form.content !== ''
    }

    return(
        <Dialog
            open={open}
            onClose={handleClose}
            fullWidth
            maxWidth="md"
        >
            <DialogTitle sx={{ textAlign: 'center' }}>
                Create File
            </DialogTitle>
            <DialogContent>
                <TextField
                    required
                    fullWidth
                    sx={{marginTop: theme.spacing(1)}}
                    label="File Name"
                    inputProps={{"aria-label": "file name"}}
                    role="textbox"
                    value={form.name}
                    onChange={(event) =>{
                        setForm({
                            ...form,
                            name: event.target.value
                        })
                    }}
                />
                <TextField
                    fullWidth
                    required
                    sx={{marginTop: theme.spacing(1)}}
                    label="File Content"
                    inputProps={{"aria-label": "file content"}}
                    role="textbox"
                    multiline
                    maxRows={16}
                    minRows={4}
                    value={form.content}
                    onChange={(event) =>{
                        setForm({
                            ...form,
                            content: event.target.value
                        })
                    }}
                />
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

export default TextFileDialog