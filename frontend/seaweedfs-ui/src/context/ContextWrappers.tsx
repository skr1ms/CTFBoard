import React from 'react'
import LocationContextWrapper from './LocationContextWrapper'
import SelectionContextWrapper from './SelectionContextWrapper'
import ThemeWrapper from './ThemeWrapper'
import ProfileContextWrapper from './ProfileContextWrapper'
import type { FileEntry } from '../types/models'

const CurrentFilesContext = React.createContext<{
  files: FileEntry[]
  setFiles: (f: FileEntry[]) => void
}>({ files: [], setFiles: () => {} })

function CurrentFilesContextWrapper(props: { children: React.ReactNode }) {
  const [files, setFiles] = React.useState<FileEntry[]>([])
  return (
    <CurrentFilesContext.Provider value={{ files, setFiles }}>
      {props.children}
    </CurrentFilesContext.Provider>
  )
}

export { CurrentFilesContext }

interface ContextWrappersProps {
  children: React.ReactNode
}

function ContextWrappers(props: ContextWrappersProps) {
  return (
    <LocationContextWrapper>
      <CurrentFilesContextWrapper>
        <ProfileContextWrapper>
          <SelectionContextWrapper>
            <ThemeWrapper>{props.children}</ThemeWrapper>
          </SelectionContextWrapper>
        </ProfileContextWrapper>
      </CurrentFilesContextWrapper>
    </LocationContextWrapper>
  )
}

export default ContextWrappers
