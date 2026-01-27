import { test, expect } from '@playwright/test'

test.describe('Debug Navigation', () => {
  test('should navigate to sales returns and log page content', async ({ page }) => {
    // Collect network requests
    const requests: string[] = []
    const failedRequests: string[] = []
    const responses: Array<{ url: string; status: number }> = []

    page.on('request', (request) => {
      requests.push(request.url())
    })

    page.on('response', (response) => {
      responses.push({ url: response.url(), status: response.status() })
    })

    page.on('requestfailed', (request) => {
      failedRequests.push(`${request.url()} - ${request.failure()?.errorText || 'unknown'}`)
    })

    // Collect console messages
    const consoleMessages: string[] = []
    page.on('console', (msg) => {
      consoleMessages.push(`[${msg.type()}] ${msg.text()}`)
    })

    // Collect page errors
    const pageErrors: string[] = []
    page.on('pageerror', (error) => {
      pageErrors.push(error.message)
    })

    // Go to the sales returns page
    console.log('Navigating to /trade/sales-returns...')
    const response = await page.goto('/trade/sales-returns')
    console.log('Navigation response status:', response?.status())

    // Wait for JS to execute
    await page.waitForTimeout(10000)

    // Check if JS executed by looking for document changes
    const documentReady = await page.evaluate(() => {
      return {
        readyState: document.readyState,
        rootChildren: document.getElementById('root')?.children.length || 0,
        scripts: Array.from(document.scripts).map((s) => ({ src: s.src, type: s.type })),
        hasReact: typeof (window as any).React !== 'undefined',
        hasReactDom: typeof (window as any).ReactDOM !== 'undefined',
      }
    })
    console.log('Document ready info:', documentReady)

    // Log response statuses
    console.log('Response statuses:', responses)

    // Log console messages
    console.log('Console messages count:', consoleMessages.length)
    consoleMessages.forEach((msg, i) => console.log(`Console ${i}:`, msg))

    // Log page errors
    console.log('Page errors:', pageErrors)

    // Log failed requests
    console.log('Failed requests:', failedRequests)

    // Take screenshot
    await page.screenshot({ path: 'test-results/debug-screenshot.png', fullPage: true })

    // Simple assertion
    expect(responses.length).toBeGreaterThan(0)
  })
})
