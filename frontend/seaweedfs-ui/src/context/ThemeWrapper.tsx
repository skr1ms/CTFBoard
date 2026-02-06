import React from 'react'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import { ProfileContext } from './ProfileContextWrapper'

interface ThemeWrapperProps {
  children: React.ReactNode
}

function ThemeWrapper(props: ThemeWrapperProps) {
  const profile = React.useContext(ProfileContext)
  const mode = profile.settings.useDarkMode ? 'dark' : 'light'
  const cardBackground = profile.settings.useDarkMode ? '#212121' : '#bdbdbd'

  const theme = createTheme({
    palette: {
      mode,
      background: {
        default: mode === 'dark' ? '#121212' : '#fff',
        paper: mode === 'dark' ? '#1e1e1e' : '#fff',
      },
    },
    components: {
      MuiCard: {
        styleOverrides: {
          root: { backgroundColor: cardBackground },
        },
      },
    },
  })

  return <ThemeProvider theme={theme}>{props.children}</ThemeProvider>
}

export default ThemeWrapper
