/**
 * API utilities for Feature Flag E2E tests
 */

/**
 * Get API base URL - handles both local and Docker environments
 *
 * In Docker mode (E2E_BASE_URL contains 'erp-frontend'), we use erp-backend:8080
 * In local mode, we convert the frontend URL to backend URL
 */
export function getApiBaseUrl(): string {
  const frontendUrl = process.env.E2E_BASE_URL || 'http://localhost:3000'
  if (frontendUrl.includes('erp-frontend')) {
    return 'http://erp-backend:8080'
  }
  // Replace port 3000 with 8080, but handle the case where :80 is part of :8080
  return frontendUrl.replace(/:3000$/, ':8080').replace(/:80$/, ':8080')
}

export const API_BASE_URL = getApiBaseUrl()
