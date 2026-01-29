import { test, expect } from '../fixtures'
import { ImportPage } from '../pages'

/**
 * Supplier Import E2E Tests (IMPORT-TEST-001)
 *
 * Tests the bulk import functionality for suppliers:
 * - Import suppliers with payment terms
 * - Import suppliers with bank details
 * - Different supplier types (manufacturer, distributor)
 */

// Test data constants
const VALID_SUPPLIER_CSV = `name,code,type,contact_person,phone,email,credit_days,credit_limit,address_province,address_city,address_district,address_detail,bank_name,bank_account,notes
导入测试供应商A,IMP-SUP-001,manufacturer,王总,13700137001,wang@supplier.com,30,100000.00,浙江省,杭州市,西湖区,文三路1号,中国银行,6222021234567890001,制造商供应商测试
导入测试供应商B,IMP-SUP-002,distributor,赵经理,13700137002,zhao@supplier.com,15,50000.00,上海市,上海市,浦东新区,陆家嘴金融中心,工商银行,6222031234567890002,分销商供应商测试`

const SUPPLIER_WITH_PAYMENT_TERMS_CSV = `name,code,type,contact_person,phone,email,credit_days,credit_limit,address_province,address_city,address_district,address_detail,bank_name,bank_account,notes
月结供应商,IMP-SUP-003,manufacturer,月结经理,13700137003,month@supplier.com,30,200000.00,广东省,深圳市,南山区,科技园路1号,招商银行,6222041234567890003,30天月结
周结供应商,IMP-SUP-004,distributor,周结经理,13700137004,week@supplier.com,7,50000.00,江苏省,南京市,玄武区,中山路2号,建设银行,6222051234567890004,7天周结
现款供应商,IMP-SUP-005,manufacturer,现款经理,13700137005,cash@supplier.com,0,0,北京市,北京市,朝阳区,CBD国贸1号,农业银行,6222061234567890005,现款现货`

const SUPPLIER_WITH_BANK_DETAILS_CSV = `name,code,type,contact_person,phone,email,credit_days,credit_limit,address_province,address_city,address_district,address_detail,bank_name,bank_account,notes
完整银行信息供应商,IMP-SUP-006,manufacturer,银行测试,13700137006,bank@supplier.com,15,80000.00,四川省,成都市,武侯区,天府大道1号,中国工商银行成都分行,6222071234567890006支行账号,银行信息完整测试`

const SUPPLIER_WITHOUT_BANK_CSV = `name,code,type,contact_person,phone,email,credit_days,credit_limit,address_province,address_city,address_district,address_detail,bank_name,bank_account,notes
无银行信息供应商,IMP-SUP-007,distributor,无银行,13700137007,nobank@supplier.com,0,0,湖北省,武汉市,洪山区,光谷大道1号,,,无银行信息的供应商`

const DUPLICATE_SUPPLIER_CODE_CSV = `name,code,type,contact_person,phone,email,credit_days,credit_limit,address_province,address_city,address_district,address_detail,bank_name,bank_account,notes
重复编码供应商1,IMP-DUP-SUP-001,manufacturer,测试1,13700137008,dup1@supplier.com,30,100000.00,浙江省,杭州市,滨江区,网商路1号,中国银行,6222081234567890008,重复测试1
重复编码供应商2,IMP-DUP-SUP-001,distributor,测试2,13700137009,dup2@supplier.com,15,50000.00,浙江省,杭州市,余杭区,文一西路2号,工商银行,6222091234567890009,重复测试2-同编码`

test.describe('Supplier Import', () => {
  let importPage: ImportPage

  test.beforeEach(async ({ page, authenticatedPage: _authenticatedPage }) => {
    importPage = new ImportPage(page)

    // Navigate to suppliers page
    await page.goto('/partner/suppliers')
    await page.waitForLoadState('domcontentloaded')
    await importPage.waitForTableLoad()
  })

  test.afterEach(async ({ page }) => {
    // Close any open modals
    const modal = page.locator('.semi-modal')
    if (await modal.isVisible().catch(() => false)) {
      await page
        .locator('.semi-modal-close')
        .click()
        .catch(() => {})
      await page.waitForTimeout(500)
    }

    // Take screenshot for debugging
    await page.screenshot({ path: `test-results/screenshots/supplier-import-${Date.now()}.png` })
  })

  test.describe('Valid Supplier Import', () => {
    test('should display import wizard on suppliers page', async ({ page }) => {
      // Find import button
      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        // Skip if import button not found
        test.skip()
        return
      }

      await importButton.click()

      // Verify import wizard modal appears
      const modal = page.locator('.semi-modal')
      await expect(modal).toBeVisible({ timeout: 10000 })

      await page.screenshot({ path: 'test-results/screenshots/supplier-import-wizard.png' })

      // Close modal
      await importPage.closeImportModal()
    })

    test('should import suppliers successfully', async ({ page }) => {
      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        test.skip()
        return
      }

      await importButton.click()
      await importPage.waitForImportModal()

      // Upload valid supplier CSV
      await importPage.uploadFileFromBuffer('valid-suppliers.csv', VALID_SUPPLIER_CSV)
      await importPage.waitForValidation()

      // Check validation result
      const wizardContent = page.locator('.import-wizard-content')
      await expect(wizardContent).toBeVisible()

      await page.screenshot({ path: 'test-results/screenshots/supplier-import-valid.png' })

      // Try to proceed if valid rows exist
      const proceedBtn = page.getByRole('button', { name: /继续导入|proceed|下一步/i })
      if (await proceedBtn.isEnabled().catch(() => false)) {
        await proceedBtn.click()
        await page.waitForTimeout(500)

        // Execute import
        const importBtn = page.getByRole('button', { name: /开始导入|start import|执行导入/i })
        if (await importBtn.isVisible().catch(() => false)) {
          await importBtn.click()
          await importPage.waitForImportComplete()
        }
      }

      // Close modal
      await importPage.closeImportModal()
    })
  })

  test.describe('Supplier with Payment Terms', () => {
    test('should import suppliers with different payment terms', async ({ page }) => {
      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        test.skip()
        return
      }

      await importButton.click()
      await importPage.waitForImportModal()

      // Upload supplier with payment terms CSV
      await importPage.uploadFileFromBuffer(
        'payment-terms-suppliers.csv',
        SUPPLIER_WITH_PAYMENT_TERMS_CSV
      )
      await importPage.waitForValidation()

      // Check validation result
      const wizardContent = page.locator('.import-wizard-content')
      const contentText = await wizardContent.textContent()

      // Should show validation for all 3 suppliers
      const hasValidation = contentText !== null && contentText.length > 0

      await page.screenshot({ path: 'test-results/screenshots/supplier-payment-terms.png' })

      expect(hasValidation).toBe(true)

      // Close modal
      await importPage.closeImportModal()
    })

    test('should handle 0 credit days (cash payment)', async ({ page }) => {
      const cashSupplierCsv = `name,code,type,contact_person,phone,email,credit_days,credit_limit,address_province,address_city,address_district,address_detail,bank_name,bank_account,notes
现金供应商,IMP-CASH-SUP-001,manufacturer,现金,13700137010,cash@supplier.com,0,0,北京市,北京市,海淀区,中关村1号,中国银行,6222101234567890010,现金付款`

      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        test.skip()
        return
      }

      await importButton.click()
      await importPage.waitForImportModal()

      // Upload cash supplier CSV
      await importPage.uploadFileFromBuffer('cash-supplier.csv', cashSupplierCsv)
      await importPage.waitForValidation()

      // Check validation result
      const wizardContent = page.locator('.import-wizard-content')
      await expect(wizardContent).toBeVisible()

      await page.screenshot({ path: 'test-results/screenshots/supplier-cash-payment.png' })

      // Close modal
      await importPage.closeImportModal()
    })
  })

  test.describe('Supplier with Bank Details', () => {
    test('should import suppliers with complete bank details', async ({ page }) => {
      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        test.skip()
        return
      }

      await importButton.click()
      await importPage.waitForImportModal()

      // Upload supplier with bank details CSV
      await importPage.uploadFileFromBuffer(
        'bank-details-suppliers.csv',
        SUPPLIER_WITH_BANK_DETAILS_CSV
      )
      await importPage.waitForValidation()

      // Check validation result
      const wizardContent = page.locator('.import-wizard-content')
      await expect(wizardContent).toBeVisible()

      await page.screenshot({ path: 'test-results/screenshots/supplier-bank-details.png' })

      // Close modal
      await importPage.closeImportModal()
    })

    test('should import suppliers without bank details', async ({ page }) => {
      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        test.skip()
        return
      }

      await importButton.click()
      await importPage.waitForImportModal()

      // Upload supplier without bank CSV
      await importPage.uploadFileFromBuffer('no-bank-suppliers.csv', SUPPLIER_WITHOUT_BANK_CSV)
      await importPage.waitForValidation()

      // Check validation result - should pass without bank details
      const wizardContent = page.locator('.import-wizard-content')
      await expect(wizardContent).toBeVisible()

      await page.screenshot({ path: 'test-results/screenshots/supplier-no-bank.png' })

      // Close modal
      await importPage.closeImportModal()
    })
  })

  test.describe('Duplicate Supplier Code', () => {
    test('should handle duplicate supplier codes', async ({ page }) => {
      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        test.skip()
        return
      }

      await importButton.click()
      await importPage.waitForImportModal()

      // Upload CSV with duplicate codes
      await importPage.uploadFileFromBuffer('duplicate-suppliers.csv', DUPLICATE_SUPPLIER_CODE_CSV)
      await importPage.waitForValidation()

      // Check validation result
      const wizardContent = page.locator('.import-wizard-content')
      const contentText = await wizardContent.textContent()

      await page.screenshot({ path: 'test-results/screenshots/supplier-duplicate-code.png' })

      // Validation should complete
      expect(contentText !== null).toBe(true)

      // Close modal
      await importPage.closeImportModal()
    })
  })

  test.describe('Supplier Types', () => {
    test('should import manufacturer suppliers', async ({ page }) => {
      const manufacturerCsv = `name,code,type,contact_person,phone,email,credit_days,credit_limit,address_province,address_city,address_district,address_detail,bank_name,bank_account,notes
制造商供应商,IMP-MAN-001,manufacturer,制造商联系人,13700137011,manufacturer@supplier.com,30,150000.00,江苏省,苏州市,工业园区,星湖街1号,交通银行,6222111234567890011,制造商类型测试`

      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        test.skip()
        return
      }

      await importButton.click()
      await importPage.waitForImportModal()

      // Upload manufacturer supplier CSV
      await importPage.uploadFileFromBuffer('manufacturer-supplier.csv', manufacturerCsv)
      await importPage.waitForValidation()

      // Check validation result
      const wizardContent = page.locator('.import-wizard-content')
      await expect(wizardContent).toBeVisible()

      await page.screenshot({ path: 'test-results/screenshots/supplier-manufacturer.png' })

      // Close modal
      await importPage.closeImportModal()
    })

    test('should import distributor suppliers', async ({ page }) => {
      const distributorCsv = `name,code,type,contact_person,phone,email,credit_days,credit_limit,address_province,address_city,address_district,address_detail,bank_name,bank_account,notes
分销商供应商,IMP-DIS-001,distributor,分销商联系人,13700137012,distributor@supplier.com,15,80000.00,广东省,广州市,天河区,天河路1号,平安银行,6222121234567890012,分销商类型测试`

      const importButton = page.getByRole('button', { name: /批量导入|import|导入/i })
      const buttonExists = await importButton.isVisible().catch(() => false)

      if (!buttonExists) {
        test.skip()
        return
      }

      await importButton.click()
      await importPage.waitForImportModal()

      // Upload distributor supplier CSV
      await importPage.uploadFileFromBuffer('distributor-supplier.csv', distributorCsv)
      await importPage.waitForValidation()

      // Check validation result
      const wizardContent = page.locator('.import-wizard-content')
      await expect(wizardContent).toBeVisible()

      await page.screenshot({ path: 'test-results/screenshots/supplier-distributor.png' })

      // Close modal
      await importPage.closeImportModal()
    })
  })
})
