import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * ImportPage - Page Object for Bulk Import functionality
 *
 * Covers:
 * - Import wizard modal across different entity types
 * - File upload step
 * - Validation step
 * - Import execution step
 * - Results step
 */
export class ImportPage extends BasePage {
  // Import wizard modal
  readonly importModal: Locator
  readonly importModalTitle: Locator

  // Steps indicator
  readonly stepsIndicator: Locator
  readonly currentStep: Locator

  // File upload step
  readonly fileUploadZone: Locator
  readonly fileInput: Locator
  readonly downloadTemplateLink: Locator
  readonly selectedFileName: Locator

  // Validation step
  readonly validationResult: Locator
  readonly validationSummary: Locator
  readonly validationErrors: Locator
  readonly validRowCount: Locator
  readonly errorRowCount: Locator
  readonly retryButton: Locator
  readonly proceedButton: Locator
  readonly validationLoading: Locator

  // Import step
  readonly conflictModeSelect: Locator
  readonly importButton: Locator
  readonly importLoading: Locator

  // Results step
  readonly resultsSuccess: Locator
  readonly resultsSummary: Locator
  readonly importedCount: Locator
  readonly updatedCount: Locator
  readonly skippedCount: Locator
  readonly errorCount: Locator
  readonly importMoreButton: Locator
  readonly closeButton: Locator

  // Toast messages
  readonly successToast: Locator
  readonly errorToast: Locator

  constructor(page: Page) {
    super(page)

    // Import wizard modal
    this.importModal = page.locator('.import-wizard-modal')
    this.importModalTitle = page.locator('.import-wizard-modal .semi-modal-title')

    // Steps indicator
    this.stepsIndicator = page.locator('.import-wizard-steps')
    this.currentStep = page.locator('.semi-steps-item-active')

    // File upload step
    this.fileUploadZone = page.locator('.file-upload-zone, .semi-upload-drag-area')
    this.fileInput = page.locator('input[type="file"]')
    this.downloadTemplateLink = page.locator('a[href*="template"], .template-download-link')
    this.selectedFileName = page.locator('.selected-file-name, .semi-upload-file-name')

    // Validation step
    this.validationResult = page.locator('.validation-result')
    this.validationSummary = page.locator('.validation-summary')
    this.validationErrors = page.locator('.validation-errors, .error-list')
    this.validRowCount = page.locator('.valid-row-count, [data-testid="valid-rows"]')
    this.errorRowCount = page.locator('.error-row-count, [data-testid="error-rows"]')
    this.retryButton = page.getByRole('button', { name: /重试|retry/i })
    this.proceedButton = page.getByRole('button', { name: /继续导入|proceed|下一步/i })
    this.validationLoading = page.locator('.validation-loading, .semi-spin-spinning')

    // Import step
    this.conflictModeSelect = page.locator(
      '.conflict-mode-select, [data-testid="conflict-mode-select"]'
    )
    this.importButton = page.getByRole('button', { name: /开始导入|start import|执行导入/i })
    this.importLoading = page.locator('.import-loading, .semi-spin-spinning')

    // Results step
    this.resultsSuccess = page.locator('.import-results-success, .results-success')
    this.resultsSummary = page.locator('.import-results-summary, .results-summary')
    this.importedCount = page.locator('.imported-count, [data-testid="imported-count"]')
    this.updatedCount = page.locator('.updated-count, [data-testid="updated-count"]')
    this.skippedCount = page.locator('.skipped-count, [data-testid="skipped-count"]')
    this.errorCount = page.locator('.error-count, [data-testid="error-count"]')
    this.importMoreButton = page.getByRole('button', { name: /继续导入|import more/i })
    this.closeButton = page.getByRole('button', { name: /关闭|close|完成/i })

    // Toast messages
    this.successToast = page.locator('.semi-toast-content').filter({ hasText: /成功|success/i })
    this.errorToast = page.locator('.semi-toast-content').filter({ hasText: /失败|error|错误/i })
  }

  /**
   * Navigate to products page and open import modal
   */
  async openProductImport(): Promise<void> {
    await this.goto('/catalog/products')
    await this.waitForTableLoad()
    await this.clickImportButton()
  }

  /**
   * Navigate to customers page and open import modal
   */
  async openCustomerImport(): Promise<void> {
    await this.goto('/partner/customers')
    await this.waitForTableLoad()
    await this.clickImportButton()
  }

  /**
   * Navigate to suppliers page and open import modal
   */
  async openSupplierImport(): Promise<void> {
    await this.goto('/partner/suppliers')
    await this.waitForTableLoad()
    await this.clickImportButton()
  }

  /**
   * Navigate to inventory page and open import modal
   */
  async openInventoryImport(): Promise<void> {
    await this.goto('/inventory/stock')
    await this.waitForTableLoad()
    await this.clickImportButton()
  }

  /**
   * Click the import button on a list page to open the import wizard
   */
  async clickImportButton(): Promise<void> {
    // Wait for the toolbar to be visible first
    await this.page.locator('.table-toolbar-right').waitFor({ state: 'visible', timeout: 15000 })

    // Try multiple selectors for import button - button text varies by language
    // The import button is in the secondary actions of the toolbar
    const importButtonSelectors = [
      // By role with exact text (Chinese)
      this.page.getByRole('button', { name: '导入' }),
      // By role with exact text (English)
      this.page.getByRole('button', { name: 'Import' }),
      // By role with regex pattern
      this.page.getByRole('button', { name: /^导入$|^Import$|批量导入/i }),
      // By icon - IconUpload creates an SVG with specific path
      this.page.locator('button:has(.semi-icon-upload)'),
      // By class and icon
      this.page.locator('.table-toolbar-right button:has(svg)').filter({ hasText: /导入|Import/i }),
    ]

    // Try each selector until one works
    for (const selector of importButtonSelectors) {
      try {
        const isVisible = await selector.isVisible().catch(() => false)
        if (isVisible) {
          await selector.click()
          await this.waitForImportModal()
          return
        }
      } catch {
        // Try next selector
      }
    }

    // Fallback: click the button that is next to refresh button
    // Import is typically before refresh in secondary actions
    const buttons = this.page.locator('.table-toolbar-right button')
    const count = await buttons.count()

    // Look for button with Upload icon or "导入" text
    for (let i = 0; i < count; i++) {
      const btn = buttons.nth(i)
      const text = await btn.textContent()
      const hasUploadIcon = await btn
        .locator('svg')
        .isVisible()
        .catch(() => false)

      if (text?.includes('导入') || text?.includes('Import') || hasUploadIcon) {
        await btn.click()
        await this.waitForImportModal()
        return
      }
    }

    // Last resort: try direct click with timeout
    const importButton = this.page.getByRole('button', { name: /批量导入|import|导入/i })
    await importButton.click({ timeout: 10000 })
    await this.waitForImportModal()
  }

  /**
   * Wait for import modal to be visible
   */
  async waitForImportModal(): Promise<void> {
    await this.page.locator('.semi-modal').waitFor({ state: 'visible', timeout: 10000 })
  }

  /**
   * Upload a CSV file to the import wizard
   */
  async uploadFile(filePath: string): Promise<void> {
    // Find file input (may be hidden)
    const fileInput = this.page.locator('input[type="file"]')
    await fileInput.setInputFiles(filePath)
  }

  /**
   * Upload a file using buffer (for dynamically created test files)
   */
  async uploadFileFromBuffer(
    filename: string,
    content: string,
    mimeType: string = 'text/csv'
  ): Promise<void> {
    // Click upload zone to trigger file chooser
    const uploadZone = this.page.locator('.semi-upload-drag-area, .file-upload-zone').first()
    const [fileChooser] = await Promise.all([
      this.page.waitForEvent('filechooser'),
      uploadZone.click(),
    ])

    await fileChooser.setFiles([
      {
        name: filename,
        mimeType: mimeType,
        buffer: Buffer.from(content, 'utf-8'),
      },
    ])
  }

  /**
   * Wait for validation to complete
   */
  async waitForValidation(): Promise<void> {
    // Wait for loading to start (may be very quick)
    await this.page.waitForTimeout(500)

    // Wait for loading to finish
    await this.page
      .locator('.semi-spin-spinning')
      .waitFor({ state: 'hidden', timeout: 30000 })
      .catch(() => {
        // Spinner may not appear if validation is instant
      })

    // Wait for validation result to appear
    await this.page.waitForTimeout(500)
  }

  /**
   * Get validation summary data
   */
  async getValidationSummary(): Promise<{
    totalRows: number
    validRows: number
    errorRows: number
  }> {
    // Wait for validation to complete
    await this.waitForValidation()

    // Find summary stats in the validation step
    const summaryText = await this.page.locator('.import-wizard-content').textContent()

    // Extract numbers from summary
    const totalMatch = summaryText?.match(/总计[：:]?\s*(\d+)/i)
    const validMatch = summaryText?.match(/有效[：:]?\s*(\d+)/i)
    const errorMatch = summaryText?.match(/错误[：:]?\s*(\d+)/i)

    return {
      totalRows: totalMatch ? parseInt(totalMatch[1], 10) : 0,
      validRows: validMatch ? parseInt(validMatch[1], 10) : 0,
      errorRows: errorMatch ? parseInt(errorMatch[1], 10) : 0,
    }
  }

  /**
   * Check if validation step shows errors
   */
  async hasValidationErrors(): Promise<boolean> {
    await this.waitForValidation()
    const errorIndicator = this.page.locator('.validation-errors, .error-list, .semi-tag-red')
    return errorIndicator.isVisible().catch(() => false)
  }

  /**
   * Get validation error messages
   */
  async getValidationErrors(): Promise<string[]> {
    await this.waitForValidation()
    const errors = await this.page.locator('.error-item, .validation-error-row').all()
    const messages: string[] = []
    for (const error of errors) {
      const text = await error.textContent()
      if (text) messages.push(text.trim())
    }
    return messages
  }

  /**
   * Click retry button to go back to upload step
   */
  async clickRetry(): Promise<void> {
    await this.retryButton.click()
  }

  /**
   * Click proceed button to go to import step
   */
  async clickProceed(): Promise<void> {
    await this.proceedButton.click()
    await this.page.waitForTimeout(500)
  }

  /**
   * Select conflict mode
   */
  async selectConflictMode(mode: 'skip' | 'update' | 'error'): Promise<void> {
    const modeLabels: Record<string, string> = {
      skip: '跳过',
      update: '更新',
      error: '报错',
    }

    // Click the radio button or select option
    const modeOption = this.page.getByRole('radio', { name: new RegExp(modeLabels[mode], 'i') })
    const isRadio = await modeOption.isVisible().catch(() => false)

    if (isRadio) {
      await modeOption.click()
    } else {
      // Fallback to select dropdown
      await this.conflictModeSelect.click()
      await this.page.locator('.semi-select-option').filter({ hasText: modeLabels[mode] }).click()
    }
  }

  /**
   * Click import button to start import
   */
  async clickImport(): Promise<void> {
    await this.importButton.click()
  }

  /**
   * Wait for import to complete
   */
  async waitForImportComplete(): Promise<void> {
    // Wait for loading to start
    await this.page.waitForTimeout(500)

    // Wait for loading to finish
    await this.page
      .locator('.semi-spin-spinning')
      .waitFor({ state: 'hidden', timeout: 60000 })
      .catch(() => {
        // Spinner may not appear
      })

    // Wait for results step to appear
    await this.page.waitForTimeout(500)
  }

  /**
   * Get import results
   */
  async getImportResults(): Promise<{
    imported: number
    updated: number
    skipped: number
    errors: number
  }> {
    await this.waitForImportComplete()

    const resultsText = await this.page.locator('.import-wizard-content').textContent()

    const importedMatch = resultsText?.match(/导入[：:]?\s*(\d+)/i)
    const updatedMatch = resultsText?.match(/更新[：:]?\s*(\d+)/i)
    const skippedMatch = resultsText?.match(/跳过[：:]?\s*(\d+)/i)
    const errorsMatch = resultsText?.match(/错误[：:]?\s*(\d+)/i)

    return {
      imported: importedMatch ? parseInt(importedMatch[1], 10) : 0,
      updated: updatedMatch ? parseInt(updatedMatch[1], 10) : 0,
      skipped: skippedMatch ? parseInt(skippedMatch[1], 10) : 0,
      errors: errorsMatch ? parseInt(errorsMatch[1], 10) : 0,
    }
  }

  /**
   * Close import modal
   */
  async closeImportModal(): Promise<void> {
    const closeBtn =
      this.page.locator('.semi-modal-close').first() ||
      this.page.getByRole('button', { name: /关闭|close/i })
    await closeBtn.click()
    await this.page
      .locator('.semi-modal')
      .waitFor({ state: 'hidden', timeout: 5000 })
      .catch(() => {
        // Modal may already be hidden
      })
  }

  /**
   * Complete full import flow
   */
  async completeImportFlow(
    fileContent: string,
    filename: string,
    conflictMode: 'skip' | 'update' | 'error' = 'skip'
  ): Promise<{
    imported: number
    updated: number
    skipped: number
    errors: number
  }> {
    // Step 1: Upload file
    await this.uploadFileFromBuffer(filename, fileContent)

    // Step 2: Wait for validation
    await this.waitForValidation()

    // Step 3: Proceed to import step (if valid rows exist)
    const summary = await this.getValidationSummary()
    if (summary.validRows > 0) {
      await this.clickProceed()

      // Step 4: Select conflict mode
      await this.selectConflictMode(conflictMode)

      // Step 5: Execute import
      await this.clickImport()

      // Step 6: Get results
      return await this.getImportResults()
    }

    return { imported: 0, updated: 0, skipped: 0, errors: summary.errorRows }
  }

  /**
   * Assert current step
   */
  async assertCurrentStep(
    step: 'upload' | 'validate' | 'import' | 'results',
    stepIndex: number
  ): Promise<void> {
    const stepIndicator = this.page.locator('.semi-steps-item').nth(stepIndex)
    await expect(stepIndicator).toHaveClass(/semi-steps-item-active/)
  }

  /**
   * Assert validation success (all rows valid)
   */
  async assertValidationSuccess(): Promise<void> {
    await this.waitForValidation()
    // Proceed button should be enabled
    await expect(this.proceedButton).toBeEnabled()
  }

  /**
   * Assert validation has errors
   */
  async assertValidationHasErrors(): Promise<void> {
    await this.waitForValidation()
    // Error count should be visible
    const hasErrors = await this.hasValidationErrors()
    expect(hasErrors).toBe(true)
  }

  /**
   * Assert import completed successfully
   */
  async assertImportSuccess(): Promise<void> {
    await this.waitForImportComplete()
    // Check for success message or positive import count
    const hasSuccess =
      (await this.successToast.isVisible().catch(() => false)) ||
      (await this.page
        .locator('.import-success, .results-success')
        .isVisible()
        .catch(() => false))
    expect(hasSuccess).toBe(true)
  }

  /**
   * Take screenshot of current import state
   */
  async screenshotImport(name: string): Promise<void> {
    await this.screenshot(`import-${name}`)
  }
}
