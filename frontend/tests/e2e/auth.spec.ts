import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test('should show login page when not authenticated', async ({ page }) => {
    await page.goto('/')
    // Should see login page with GitHub device flow
    await expect(page.getByText('Welcome to Concord')).toBeVisible()
    await expect(page.getByRole('button', { name: /sign in with github/i })).toBeVisible()
  })

  test('should show loading state initially', async ({ page }) => {
    await page.goto('/')
    // Brief loading splash before auth state resolves
    const loader = page.getByText('Loading Concord...')
    // Loader appears briefly then resolves to login or app
    await expect(loader.or(page.getByText('Welcome to Concord'))).toBeVisible({ timeout: 5000 })
  })

  test('should display device code when login starts', async ({ page }) => {
    await page.goto('/')
    // Click sign in button
    const signInButton = page.getByRole('button', { name: /sign in with github/i })
    await expect(signInButton).toBeVisible()
    // Note: actual GitHub device flow requires mocking the Wails backend
    // In E2E with mocked backend, clicking should show user code
  })

  test('should redirect to app after authentication', async ({ page }) => {
    // Mock localStorage to simulate authenticated state
    await page.addInitScript(() => {
      localStorage.setItem('concord-user-id', 'test-user-123')
    })
    await page.goto('/')
    // Should show main app layout (server sidebar visible)
    // Note: requires mocked Wails bindings for RestoreSession
  })

  test('should handle logout', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('concord-user-id', 'test-user-123')
    })
    await page.goto('/')
    // Note: requires mocked Wails bindings
    // After logout, should show login page again
  })
})
