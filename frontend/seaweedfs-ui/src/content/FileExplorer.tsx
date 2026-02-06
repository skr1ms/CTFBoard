import React from 'react'
import Toolbar from '@mui/material/Toolbar'
import CssBaseline from '@mui/material/CssBaseline'
import { LocationContext } from '../context/LocationContextWrapper'
import { CurrentFilesContext } from '../context/ContextWrappers'
import Filer from '../seaweedfs/filer'
import DetailsView from './DetailsView'
import CardView from './cardview/CardView'
import { ProfileContext } from '../context/ProfileContextWrapper'
import type { FileEntry } from '../types/models'

interface FileExplorerProps {
  files?: FileEntry[]
}

function FileExplorer(props: FileExplorerProps): React.ReactElement {
  const context = React.useContext(LocationContext)
  const { setFiles: setCurrentDirFiles } = React.useContext(CurrentFilesContext)
  const [files, setFiles] = React.useState<FileEntry[]>(props.files ?? [])

  const profile = React.useContext(ProfileContext)

  React.useEffect(() => {
    async function loadFiles(): Promise<void> {
      const tempFiles = await Filer.getFiles(context.currentLocation)
      const output = tempFiles.filter(
        (f) => profile.settings.showDotFiles || !f.name.startsWith('.')
      )
      setFiles(output)
      setCurrentDirFiles(output)
    }
    loadFiles()
  }, [context.currentLocation, profile.settings.showDotFiles, context.refreshCount, setCurrentDirFiles])


  return (
    <>
      <Toolbar />
      <CssBaseline />
      {profile.settings.useDetailsView ? (
        <DetailsView files={files} />
      ) : (
        <CardView files={files} />
      )}
    </>
  )
}

export default FileExplorer