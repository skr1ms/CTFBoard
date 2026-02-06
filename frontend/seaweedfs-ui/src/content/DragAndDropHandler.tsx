import React from 'react'
import UploadFileDialog from '../dialogs/UploadFileDialog'

function DragAndDropHandler(): React.ReactElement {
    const [open, setOpen] = React.useState(false)
    const [files, setFiles] = React.useState<File[]>([])

    React.useEffect(() => {
        document.addEventListener('drop', handleDrop as EventListener)
        document.addEventListener('dragover', handleDrag as EventListener)
        return () => {
            document.removeEventListener('drop', handleDrop as EventListener)
            document.removeEventListener('dragover', handleDrag as EventListener)
        }
    }, [])

    function close(): void {
        setFiles([])
        setOpen(false)
    }

    function handleDrag(event: DragEvent): void {
        event.stopPropagation()
        event.preventDefault()
    }

    function handleDrop(event: DragEvent): void {
        event.stopPropagation()
        event.preventDefault()
        const tempFiles: File[] = []
        for (const item of event.dataTransfer?.items ?? []) {
            if (item.kind === 'file') {
                const file = item.getAsFile()
                if (file) tempFiles.push(file)
            }
        }
        if (tempFiles.length) {
            setFiles(tempFiles)
            setOpen(true)
        }
    }

    return (
        <React.Fragment>
            <UploadFileDialog
                files={files}
                open={open}
                close={close}
            />
        </React.Fragment>
    )
}

export default DragAndDropHandler