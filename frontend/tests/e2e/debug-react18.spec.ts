import { test, expect } from '@playwright/test'

test.describe('Debug Frontend', () => {
  test('check for JS errors', async ({ page }) => {
    const pageErrors: string[] = []
    page.on('pageerror', (error) => {
      pageErrors.push(error.message)
      console.log('PAGE ERROR:', error.message)
    })

    const consoleLogs: string[] = []
    page.on('console', (msg) => {
      const text = `[${msg.type()}] ${msg.text()}`
      consoleLogs.push(text)
      if (msg.type() === 'error' || msg.type() === 'warning') {
        console.log('CONSOLE:', text)
      }
    })

    let apiCallCount = 0
    await page.route('**/api/**', async (route) => {
      apiCallCount++
      console.log('API CALL:', route.request().method(), route.request().url())
      await route.continue()
    })

    await page.goto('/catalog/products')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)

    console.log('Page errors:', pageErrors.length)
    console.log('API calls:', apiCallCount)

    if (pageErrors.length > 0) {
      console.log('Errors:')
      pageErrors.forEach(e => console.log('  -', e))
    }

    // Check localStorage
    const localStorage = await page.evaluate(() => {
      return {
        access_token: window.localStorage.getItem('access_token')?.substring(0, 30) || 'NONE',
        refresh_token: window.localStorage.getItem('refresh_token')?.substring(0, 30) || 'NONE',
        user: window.localStorage.getItem('user')?.substring(0, 50) || 'NONE',
        erp_auth: window.localStorage.getItem('erp-auth')?.substring(0, 50) || 'NONE',
      }
    })
    console.log('localStorage:', JSON.stringify(localStorage, null, 2))

    // Check if authenticated
    const isAuth = await page.evaluate(() => {
      const erpAuth = window.localStorage.getItem('erp-auth')
      if (!erpAuth) return false
      try {
        const parsed = JSON.parse(erpAuth)
        return parsed?.state?.isAuthenticated
      } catch {
        return false
      }
    })
    console.log('isAuthenticated:', isAuth)

    // Check if Zustand isLoading
    const zustandState = await page.evaluate(() => {
      const erpAuth = window.localStorage.getItem('erp-auth')
      if (!erpAuth) return { found: false }
      try {
        return JSON.parse(erpAuth)
      } catch {
        return { parseError: true }
      }
    })
    console.log('Zustand full state:', JSON.stringify(zustandState, null, 2).substring(0, 500))

    // Try calling API directly from browser
    const apiResult = await page.evaluate(async () => {
      const token = window.localStorage.getItem('access_token')
      const userStr = window.localStorage.getItem('user')
      let tenantId = '00000000-0000-0000-0000-000000000001'
      if (userStr) {
        try {
          const user = JSON.parse(userStr)
          tenantId = user.tenantId || tenantId
        } catch {}
      }

      try {
        const response = await fetch('/api/v1/catalog/products?page=1&page_size=10', {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId
          }
        })
        const data = await response.json()
        return { status: response.status, dataLength: data.data?.length || 0, success: data.success }
      } catch (e) {
        return { error: (e as Error).message }
      }
    })
    console.log('Direct API call result:', JSON.stringify(apiResult))

    await page.screenshot({ path: 'test-results/debug-react18.png' })

    // Check table rows
    const rows = await page.locator('.semi-table-tbody .semi-table-row').count()
    console.log('Table rows:', rows)

    expect(pageErrors.length).toBe(0)
    expect(apiCallCount).toBeGreaterThan(0)
  })
})
