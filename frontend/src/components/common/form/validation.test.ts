/**
 * Validation Utilities Tests
 *
 * Tests for Zod schemas and validation patterns
 */

import { describe, it, expect } from 'vitest'
import { schemas, patterns, validationMessages } from './validation'

describe('validation patterns', () => {
  describe('phone pattern', () => {
    it('should match valid Chinese phone numbers', () => {
      expect(patterns.phone.test('13800138000')).toBe(true)
      expect(patterns.phone.test('15912345678')).toBe(true)
      expect(patterns.phone.test('18600000000')).toBe(true)
    })

    it('should reject invalid phone numbers', () => {
      expect(patterns.phone.test('12345678901')).toBe(false) // starts with 1 but not 13-19
      expect(patterns.phone.test('1380013800')).toBe(false) // 10 digits
      expect(patterns.phone.test('138001380000')).toBe(false) // 12 digits
      expect(patterns.phone.test('23800138000')).toBe(false) // starts with 2
    })
  })

  describe('sku pattern', () => {
    it('should match valid SKU formats', () => {
      expect(patterns.sku.test('ABC123')).toBe(true)
      expect(patterns.sku.test('SKU-001-A')).toBe(true)
      expect(patterns.sku.test('12345')).toBe(true)
    })

    it('should reject invalid SKU formats', () => {
      expect(patterns.sku.test('SKU@001')).toBe(false)
      expect(patterns.sku.test('SKU 001')).toBe(false) // spaces
      expect(patterns.sku.test('')).toBe(false)
    })
  })

  describe('barcode pattern', () => {
    it('should match valid barcode formats', () => {
      expect(patterns.barcode.test('69012345')).toBe(true) // 8 digits (EAN-8)
      expect(patterns.barcode.test('690123456789')).toBe(true) // 12 digits (UPC-A)
      expect(patterns.barcode.test('6901234567890')).toBe(true) // 13 digits (EAN-13)
      expect(patterns.barcode.test('69012345678901')).toBe(true) // 14 digits
    })

    it('should reject invalid barcode formats', () => {
      expect(patterns.barcode.test('1234567')).toBe(false) // 7 digits
      expect(patterns.barcode.test('123456789')).toBe(false) // 9 digits
      expect(patterns.barcode.test('ABC12345678')).toBe(false) // letters
    })
  })
})

describe('validation schemas', () => {
  describe('requiredString', () => {
    it('should accept non-empty strings', () => {
      expect(schemas.requiredString.safeParse('hello').success).toBe(true)
      expect(schemas.requiredString.safeParse('a').success).toBe(true)
    })

    it('should reject empty strings', () => {
      const result = schemas.requiredString.safeParse('')
      expect(result.success).toBe(false)
      if (!result.success) {
        // Zod 4 uses 'issues' array
        const issues = result.error.issues
        expect(issues[0].message).toBe(validationMessages.required)
      }
    })
  })

  describe('email', () => {
    it('should accept valid emails', () => {
      expect(schemas.email.safeParse('test@example.com').success).toBe(true)
      expect(schemas.email.safeParse('user.name+tag@domain.co.uk').success).toBe(true)
    })

    it('should reject invalid emails', () => {
      expect(schemas.email.safeParse('notanemail').success).toBe(false)
      expect(schemas.email.safeParse('missing@').success).toBe(false)
      expect(schemas.email.safeParse('@nodomain.com').success).toBe(false)
    })

    it('should reject empty emails', () => {
      expect(schemas.email.safeParse('').success).toBe(false)
    })
  })

  describe('phone', () => {
    it('should accept valid Chinese phone numbers', () => {
      expect(schemas.phone.safeParse('13800138000').success).toBe(true)
      expect(schemas.phone.safeParse('15912345678').success).toBe(true)
    })

    it('should reject invalid phone numbers', () => {
      expect(schemas.phone.safeParse('12345678901').success).toBe(false)
      expect(schemas.phone.safeParse('abc').success).toBe(false)
    })
  })

  describe('positiveNumber', () => {
    it('should accept positive numbers', () => {
      expect(schemas.positiveNumber.safeParse(1).success).toBe(true)
      expect(schemas.positiveNumber.safeParse(0.01).success).toBe(true)
      expect(schemas.positiveNumber.safeParse(1000000).success).toBe(true)
    })

    it('should reject zero and negative numbers', () => {
      expect(schemas.positiveNumber.safeParse(0).success).toBe(false)
      expect(schemas.positiveNumber.safeParse(-1).success).toBe(false)
      expect(schemas.positiveNumber.safeParse(-0.01).success).toBe(false)
    })

    it('should reject non-numbers', () => {
      expect(schemas.positiveNumber.safeParse('123').success).toBe(false)
      expect(schemas.positiveNumber.safeParse(null).success).toBe(false)
    })
  })

  describe('nonNegativeNumber', () => {
    it('should accept zero and positive numbers', () => {
      expect(schemas.nonNegativeNumber.safeParse(0).success).toBe(true)
      expect(schemas.nonNegativeNumber.safeParse(1).success).toBe(true)
      expect(schemas.nonNegativeNumber.safeParse(0.01).success).toBe(true)
    })

    it('should reject negative numbers', () => {
      expect(schemas.nonNegativeNumber.safeParse(-1).success).toBe(false)
      expect(schemas.nonNegativeNumber.safeParse(-0.01).success).toBe(false)
    })
  })

  describe('positiveInteger', () => {
    it('should accept positive integers', () => {
      expect(schemas.positiveInteger.safeParse(1).success).toBe(true)
      expect(schemas.positiveInteger.safeParse(100).success).toBe(true)
    })

    it('should reject decimals', () => {
      expect(schemas.positiveInteger.safeParse(1.5).success).toBe(false)
    })

    it('should reject zero and negative integers', () => {
      expect(schemas.positiveInteger.safeParse(0).success).toBe(false)
      expect(schemas.positiveInteger.safeParse(-1).success).toBe(false)
    })
  })

  describe('money', () => {
    it('should accept valid money values', () => {
      expect(schemas.money.safeParse(0).success).toBe(true)
      expect(schemas.money.safeParse(99.99).success).toBe(true)
      expect(schemas.money.safeParse(1000.0).success).toBe(true)
    })

    it('should reject negative values', () => {
      expect(schemas.money.safeParse(-1).success).toBe(false)
    })

    it('should reject values with more than 2 decimal places', () => {
      expect(schemas.money.safeParse(1.001).success).toBe(false)
      expect(schemas.money.safeParse(1.999).success).toBe(false)
    })
  })
})

describe('validation messages', () => {
  it('should have required message', () => {
    expect(validationMessages.required).toBe('此字段为必填项')
  })

  it('should generate min/max length messages', () => {
    expect(validationMessages.minLength(5)).toBe('至少需要 5 个字符')
    expect(validationMessages.maxLength(100)).toBe('最多 100 个字符')
  })

  it('should generate min/max value messages', () => {
    expect(validationMessages.min(0)).toBe('不能小于 0')
    expect(validationMessages.max(100)).toBe('不能大于 100')
  })
})
