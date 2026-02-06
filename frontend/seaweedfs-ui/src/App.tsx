import React from 'react'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import { AuthContext } from './context/AuthContextWrapper'
import ContextWrappers from './context/ContextWrappers'
import LoginPage from './pages/LoginPage'
import TopNav from './nav/TopNav'
import SideDrawer from './nav/SideDrawer'
import FileExplorer from './content/FileExplorer'
import DragAndDropHandler from './content/DragAndDropHandler'

function App(): React.ReactElement {
  const auth = React.useContext(AuthContext)
  const wantsLight = window.matchMedia('(prefers-color-scheme: light)').matches
  const mode: 'light' | 'dark' = wantsLight ? 'light' : 'dark'
  const theme = createTheme({
    palette: {
      mode: mode,
    },
  })

  if (!auth.isAuthenticated) {
    return (
      <ThemeProvider theme={theme}>
        <LoginPage />
      </ThemeProvider>
    )
  }

  return (
    <ThemeProvider theme={theme}>
      <ContextWrappers>
        <TopNav />
        <SideDrawer />
        <FileExplorer />
        <DragAndDropHandler />
      </ContextWrappers>
    </ThemeProvider>
  )
}

export default App
