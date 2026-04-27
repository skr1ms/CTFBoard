/**
 * WebSocket reconnect spec - offline/online cycle.
 *
 * When the network drops, the WS socket closes and the wsStore increments
 * reconnectAttempt. The WsReconnectBanner (in AuthLayout) renders when
 * `!connected && !usingSse && reconnectAttempt > 0`.
 *
 * When the network is restored, the pending reconnect timer fires and the
 * banner disappears once connected or usingSse becomes true.
 */
import { expect, test } from '@playwright/test'

test.describe('WS reconnect banner', () => {
  test.use({ storageState: 'e2e/.auth/user1.json' })

  test('banner appears when WS drops, disappears after reconnect', async ({ page }) => {
    await page.goto('/challenges')

    // Wait until the app is fully loaded so the WS has had time to connect.
    await expect(page.locator('[data-testid="challenge-card"]').first()).toBeVisible({
      timeout: 20_000,
    })

    // --- Go offline ---
    await page.context().setOffline(true)

    // The WS socket closes almost immediately; onclose increments reconnectAttempt.
    // Banner condition: !connected && !usingSse && attempt > 0
    const banner = page.getByRole('status').filter({ hasText: /connection lost/i })
    await expect(banner).toBeVisible({ timeout: 10_000 })

    // --- Restore connectivity ---
    await page.context().setOffline(false)

    // After going back online, the pending reconnect timer fires and the WS or
    // SSE transport reconnects. Banner hides when connected or usingSse becomes true.
    // Allow up to 40 s for the full backoff + SSE fallback cycle to complete.
    await expect(banner).not.toBeVisible({ timeout: 40_000 })
  })
})
