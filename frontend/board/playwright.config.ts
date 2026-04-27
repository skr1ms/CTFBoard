import { defineConfig, devices } from '@playwright/test'
import { execSync } from 'child_process'
import * as fs from 'fs'

/**
 * Resolve the Chromium executable path.
 *
 * On NixOS, PLAYWRIGHT_BROWSERS_PATH is set to a nix-store path that may not
 * contain the headless-shell binary (it requires a nix-shell environment to
 * build the wrapper). We detect this and fall back to the system Chromium.
 */
function resolveChromiumPath(): string | undefined {
  // Explicit override always wins
  const explicit = process.env['PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH']
  if (explicit && fs.existsSync(explicit)) return explicit

  // Check if the nix-store browser path is missing
  const browsersPath = process.env['PLAYWRIGHT_BROWSERS_PATH']
  if (browsersPath) {
    const headlessShell = `${browsersPath}/chromium_headless_shell-1217/chrome-headless-shell-linux64/chrome-headless-shell`
    if (!fs.existsSync(headlessShell)) {
      // Fall back to system Chromium
      try {
        const systemChromium = execSync('which chromium || which google-chrome', { encoding: 'utf8' }).trim()
        if (systemChromium && fs.existsSync(systemChromium)) return systemChromium
      } catch {
        // ignore
      }
    }
  }

  return undefined
}

const executablePath = resolveChromiumPath()

const chromiumOptions = {
  ...devices['Desktop Chrome'],
  ...(executablePath ? { executablePath } : {}),
}

export default defineConfig({
  testDir: './e2e/specs',
  globalSetup: './e2e/global-setup.ts',
  fullyParallel: true,
  workers: 2,
  retries: 1,
  timeout: 30_000,
  use: {
    baseURL: 'http://localhost:4173',
  },
  projects: [
    {
      name: 'chromium',
      use: chromiumOptions,
    },
  ],
  webServer: {
    command: 'bun run preview',
    url: 'http://localhost:4173',
    reuseExistingServer: true,
    timeout: 30_000,
    env: {
      VITE_BACKEND_PORT: process.env['VITE_BACKEND_PORT'] ?? '8090',
    },
  },
})
