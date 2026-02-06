import React from 'react'
import { useTheme } from '@mui/material/styles'
import AppBar from '@mui/material/AppBar'
import Box from '@mui/material/Box'
import Toolbar from '@mui/material/Toolbar'
import OutlinedInput from '@mui/material/OutlinedInput'
import IconButton from '@mui/material/IconButton'
import InputAdornment from '@mui/material/InputAdornment'
import ButtonGroup from '@mui/material/ButtonGroup'
import Button from '@mui/material/Button'
import Tooltip from '@mui/material/Tooltip'
import { CssBaseline } from '@mui/material'
import ArrowForwardIcon from '@mui/icons-material/ArrowForward'
import ArrowBackIosNewIcon from '@mui/icons-material/ArrowBackIosNew'
import HomeIcon from '@mui/icons-material/Home'
import CheckBoxOutlineBlankIcon from '@mui/icons-material/CheckBoxOutlineBlank'
import CheckBoxIcon from '@mui/icons-material/CheckBox'
import { LocationContext } from '../context/LocationContextWrapper'
import { CurrentFilesContext } from '../context/ContextWrappers'
import { SelectionContext } from '../context/SelectionContextWrapper'
import { ProfileContext } from '../context/ProfileContextWrapper'
import Settings from './Settings'

function LeftButtonGroup(): React.ReactElement {
    const context = React.useContext(LocationContext)

    return (
        <ButtonGroup
            variant="contained"
            color="primary"
        >
            <Button
                aria-label="back button"
                onClick={() => {
                    context.goBack()
                }}
                disabled={context.history.length === 0}
            >
                <ArrowBackIosNewIcon
                    fontSize="small"

                />
            </Button>
            <Button
                aria-label="home button"
                onClick={() => {
                    context.updateLocation('/')
                }}
            >
                <HomeIcon
                    fontSize="small"
                />
            </Button>
        </ButtonGroup>
    )
}

function SelectAllCheckbox(): React.ReactElement | null {
  const profile = React.useContext(ProfileContext)
  if (profile.settings.useDetailsView) return null
  const { files } = React.useContext(CurrentFilesContext)
  const selection = React.useContext(SelectionContext)
  const list = files.filter((f) => f.name !== '..' && f.name !== '.')
  const selectedPaths = new Set(selection.selected.map((s) => s.path))
  const allSelected = list.length > 0 && list.every((f) => selectedPaths.has(f.FullPath))
  const someSelected = selection.selected.length > 0

  function toggle(): void {
    if (allSelected) {
      selection.replaceSelection([])
    } else {
      selection.replaceSelection(list.map((f) => ({ path: f.FullPath, isFile: f.isFile })))
    }
  }

  return (
    <Tooltip title="Select all in current directory">
      <IconButton size="small" onClick={toggle} aria-label="select all" sx={{ mr: 0.5 }}>
        {allSelected ? (
          <CheckBoxIcon fontSize="small" />
        ) : (
          <CheckBoxOutlineBlankIcon fontSize="small" sx={{ opacity: someSelected ? 0.7 : 1 }} />
        )}
      </IconButton>
    </Tooltip>
  )
}

function LocationBar(): React.ReactElement {
    const theme = useTheme()
    const context = React.useContext(LocationContext)
    const [inputContent, setInputContent] = React.useState(context.currentLocation)

    React.useEffect(() => {
        setInputContent(context.currentLocation)
    }, [context.currentLocation])

    const selfRef = React.useRef<HTMLInputElement>(null)

    React.useEffect(() => {
        function handleShortcuts(event: KeyboardEvent): void {
            const isValid =
                selfRef.current !== null &&
                document.activeElement !== selfRef.current &&
                event.key === '/' &&
                (event.ctrlKey || event.altKey)
            if (isValid) {
                event.preventDefault()
                selfRef.current.focus()
                if (event.altKey) setInputContent('/')
            }
        }
        document.addEventListener('keydown', handleShortcuts)
        return () => document.removeEventListener('keydown', handleShortcuts)
    }, [])

    return (
        <Box sx={{ display: 'flex', alignItems: 'center', marginLeft: theme.spacing(1) }}>
            <SelectAllCheckbox />
            <OutlinedInput
                inputRef={selfRef}
                size="small"
                value={inputContent}
                sx={{ width: "33vw" }}
                role="searchbox"
                aria-label="search"
                endAdornment={
                    <InputAdornment position="end">
                        <IconButton
                            aria-label="go to button"
                            onClick={() => context.updateLocation(inputContent)}
                            edge="end"
                        >
                            <ArrowForwardIcon />
                        </IconButton>
                    </InputAdornment>
                }
                onChange={(event) => setInputContent(event.target.value)}
                onKeyDown={(event) => {
                    if (event.key === "Enter") context.updateLocation(inputContent)
                }}
            />
        </Box>
    )
}


function TopNav(): React.ReactElement {
    const theme = useTheme()

    return (
        <Box
            sx={{ flexGrow: 1 }}
        >
            <CssBaseline />
            <AppBar
                position="fixed"
                sx={{
                    zIndex: theme.zIndex.drawer + 1,
                    marginLeft: '240px',
                    width: 'calc(100% - 240px)',
                }}
            >
                <Toolbar sx={{ paddingLeft: theme.spacing(2) }}>
                    <LeftButtonGroup />
                    <LocationBar />
                    <Box
                        sx={{ flexGrow: 1 }}
                    />
                    <Settings />
                </Toolbar>
            </AppBar>
        </Box>
    )
}

export default TopNav
export {
    LocationContext
}