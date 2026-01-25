import { test, expect } from '@playwright/test'

test.describe('Debug Customer Page', () => {
  test('check customer page loading', async ({ page }) => {
    const pageErrors: string[] = []
    page.on('pageerror', (error) => {
      pageErrors.push(error.message)
      console.log('PAGE ERROR:', error.message)
    })

    const consoleLogs: string[] = []
    page.on('console', (msg) => {
      const text = `[${msg.type()}] ${msg.text()}`
      consoleLogs.push(text)
      console.log('CONSOLE:', text)
    })

    // Log ALL network requests
    page.on('request', (request) => {
      if (request.url().includes('/api/')) {
        console.log('REQUEST:', request.method(), request.url())
        console.log('  Headers:', JSON.stringify(request.headers()))
      }
    })

    page.on('response', async (response) => {
      if (response.url().includes('/api/')) {
        try {
          const body = await response.text()
          console.log('RESPONSE:', response.status(), response.url())
          console.log('  Body:', body.substring(0, 500))
        } catch {
          console.log('RESPONSE:', response.status(), response.url(), '(could not read body)')
        }
      }
    })

    // Navigate to customers page
    console.log('Navigating to /partner/customers...')
    await page.goto('/partner/customers')
    await page.waitForLoadState('networkidle')

    // Wait a bit for any async rendering
    await page.waitForTimeout(3000)

    // Check what's visible on page
    const pageContent = await page.locator('body').innerText()
    console.log('Page body text (first 500 chars):')
    console.log(pageContent.substring(0, 500))

    // Check for table
    const tableExists = await page.locator('.semi-table').count()
    console.log('Table count:', tableExists)

    // Check table body visibility
    const tbodyVisible = await page.locator('.semi-table-tbody').isVisible()
    console.log('Table body visible:', tbodyVisible)

    // Get table body HTML
    if (tableExists > 0) {
      const tbodyHtml = await page.locator('.semi-table-tbody').innerHTML().catch(() => 'N/A')
      console.log('Table body HTML (first 300 chars):', tbodyHtml.substring(0, 300))
    }

    // Check for loading spinner
    const spinnerVisible = await page.locator('.semi-spin-spinning').isVisible().catch(() => false)
    console.log('Loading spinner visible:', spinnerVisible)

    // Check for empty state
    const emptyStateVisible = await page.locator('.semi-table-empty').isVisible().catch(() => false)
    console.log('Empty state visible:', emptyStateVisible)

    // Check table rows
    const rows = await page.locator('.semi-table-tbody .semi-table-row').count()
    console.log('Table rows:', rows)

    // Try API call directly
    const apiResult = await page.evaluate(async () => {
      const token = window.localStorage.getItem('access_token')
      try {
        const response = await fetch('/api/v1/partner/customers?page=1&page_size=10', {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        })
        const data = await response.json()
        return {
          status: response.status,
          success: data.success,
          dataLength: data.data?.length || 0,
          firstCustomer: data.data?.[0]?.name || 'N/A'
        }
      } catch (e) {
        return { error: (e as Error).message }
      }
    })
    console.log('Direct API call result:', JSON.stringify(apiResult))

    await page.screenshot({ path: 'test-results/debug-customer.png' })

    // Test should pass if no page errors
    expect(pageErrors.length).toBe(0)
  })
})
