import { type Page, expect } from '@playwright/test'

/**
 * Fill Semi Design Form field
 */
export async function fillFormField(page: Page, fieldLabel: string, value: string): Promise<void> {
  // Find field by label
  const field = page.locator(`.semi-form-field:has-text("${fieldLabel}")`)
  const input = field.locator('input, textarea').first()
  await input.fill(value)
}

/**
 * Select option in Semi Design Select
 */
export async function selectOption(
  page: Page,
  fieldLabel: string,
  optionText: string
): Promise<void> {
  // Find select by label
  const field = page.locator(`.semi-form-field:has-text("${fieldLabel}")`)
  const select = field.locator('.semi-select')
  await select.click()

  // Select option
  await page.locator(`.semi-select-option:has-text("${optionText}")`).click()
}

/**
 * Select multiple options in Semi Design Select
 */
export async function selectMultipleOptions(
  page: Page,
  fieldLabel: string,
  optionTexts: string[]
): Promise<void> {
  const field = page.locator(`.semi-form-field:has-text("${fieldLabel}")`)
  const select = field.locator('.semi-select')
  await select.click()

  for (const text of optionTexts) {
    await page.locator(`.semi-select-option:has-text("${text}")`).click()
  }

  // Close dropdown
  await page.keyboard.press('Escape')
}

/**
 * Check/uncheck Semi Design Checkbox
 */
export async function toggleCheckbox(
  page: Page,
  labelText: string,
  checked: boolean
): Promise<void> {
  const checkbox = page.locator(`.semi-checkbox:has-text("${labelText}")`)
  const input = checkbox.locator('input[type="checkbox"]')
  const isChecked = await input.isChecked()

  if (isChecked !== checked) {
    await checkbox.click()
  }
}

/**
 * Toggle Semi Design Switch
 */
export async function toggleSwitch(page: Page, labelText: string, enabled: boolean): Promise<void> {
  const switchElement = page.locator(`.semi-form-field:has-text("${labelText}") .semi-switch`)
  const isChecked = await switchElement.getAttribute('aria-checked')
  const currentlyEnabled = isChecked === 'true'

  if (currentlyEnabled !== enabled) {
    await switchElement.click()
  }
}

/**
 * Select date in Semi Design DatePicker
 */
export async function selectDate(
  page: Page,
  fieldLabel: string,
  date: string // Format: YYYY-MM-DD
): Promise<void> {
  const field = page.locator(`.semi-form-field:has-text("${fieldLabel}")`)
  const datePicker = field.locator('.semi-datepicker')
  await datePicker.click()

  // Clear and type date
  const input = datePicker.locator('input')
  await input.fill(date)
  await input.press('Enter')
}

/**
 * Submit form
 */
export async function submitForm(page: Page): Promise<void> {
  await page.click('button[type="submit"], button:has-text("提交"), button:has-text("保存")')
}

/**
 * Cancel form
 */
export async function cancelForm(page: Page): Promise<void> {
  await page.click('button:has-text("取消"), button:has-text("返回")')
}

/**
 * Assert form field has error
 */
export async function assertFieldError(
  page: Page,
  fieldLabel: string,
  errorMessage?: string
): Promise<void> {
  const field = page.locator(`.semi-form-field:has-text("${fieldLabel}")`)
  const error = field.locator('.semi-form-field-error-message')
  await expect(error).toBeVisible()

  if (errorMessage) {
    await expect(error).toContainText(errorMessage)
  }
}

/**
 * Assert form field has no error
 */
export async function assertNoFieldError(page: Page, fieldLabel: string): Promise<void> {
  const field = page.locator(`.semi-form-field:has-text("${fieldLabel}")`)
  const error = field.locator('.semi-form-field-error-message')
  await expect(error).not.toBeVisible()
}

/**
 * Get form field value
 */
export async function getFormFieldValue(page: Page, fieldLabel: string): Promise<string> {
  const field = page.locator(`.semi-form-field:has-text("${fieldLabel}")`)
  const input = field.locator('input, textarea').first()
  return (await input.inputValue()) || ''
}
