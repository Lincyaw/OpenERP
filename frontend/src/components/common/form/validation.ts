import { z } from 'zod'
import i18n from '@/i18n'

/**
 * Get validation message with i18n support
 *
 * Uses the 'validation' namespace from i18n.
 * Falls back to Chinese if translation is not found.
 */
function t(key: string, defaultValue: string, options?: Record<string, unknown>): string {
  return i18n.t(`validation:${key}`, { defaultValue, ...options })
}

/**
 * Common validation messages (with i18n support)
 *
 * Note: These are evaluated at module load time. For language changes to take effect,
 * the page needs to be reloaded. This is a limitation of Zod's static message requirement.
 *
 * For dynamic i18n in forms, consider using the schemas export or the create*Schema helpers
 * which evaluate messages at schema creation time.
 */
export const validationMessages = {
  required: t('required', '此字段为必填项'),
  email: t('email', '请输入有效的邮箱地址'),
  minLength: (min: number) => t('minLength', `至少需要 ${min} 个字符`, { min }),
  maxLength: (max: number) => t('maxLength', `最多 ${max} 个字符`, { max }),
  min: (min: number) => t('min', `不能小于 ${min}`, { min }),
  max: (max: number) => t('max', `不能大于 ${max}`, { max }),
  pattern: t('pattern', '格式不正确'),
  phone: t('phone', '请输入有效的手机号码'),
  url: t('url', '请输入有效的URL'),
  number: t('number', '请输入有效的数字'),
  integer: t('integer', '请输入整数'),
  positive: t('positive', '必须为正数'),
  nonNegative: t('nonNegative', '不能为负数'),
  date: t('date', '请输入有效的日期'),
  dateRange: t('dateRange', '结束日期不能早于开始日期'),
  passwordMatch: t('passwordMatch', '两次输入的密码不一致'),
  unique: t('unique', '该值已存在'),
  invalidSku: t('invalidSku', 'SKU只能包含字母、数字和横线'),
  invalidBarcode: t('invalidBarcode', '请输入有效的条形码'),
  invalidId: t('invalidId', '无效的ID格式'),
  moneyPrecision: t('moneyPrecision', '金额最多两位小数'),
}

/**
 * Common validation patterns
 */
export const patterns = {
  /** Chinese mobile phone number */
  phone: /^1[3-9]\d{9}$/,
  /** Chinese ID card */
  idCard: /(^\d{15}$)|(^\d{18}$)|(^\d{17}(\d|X|x)$)/,
  /** Postal code */
  postalCode: /^\d{6}$/,
  /** SKU format (alphanumeric with dashes) */
  sku: /^[A-Za-z0-9-]+$/,
  /** Barcode (EAN-13, UPC-A) */
  barcode: /^(\d{8}|\d{12}|\d{13}|\d{14})$/,
  /** Chinese characters only */
  chinese: /^[\u4e00-\u9fa5]+$/,
  /** Alphanumeric only */
  alphanumeric: /^[A-Za-z0-9]+$/,
  /** No special characters */
  noSpecialChars: /^[A-Za-z0-9\u4e00-\u9fa5\s]+$/,
}

/**
 * Pre-built Zod schemas for common fields
 *
 * Note: These use the i18n-aware validationMessages.
 * Messages are evaluated at module load time.
 */
export const schemas = {
  /** Required string */
  requiredString: z.string().min(1, { message: validationMessages.required }),

  /** Optional string (empty string becomes undefined) */
  optionalString: z
    .string()
    .optional()
    .transform((val) => val || undefined),

  /** Email field */
  email: z
    .string()
    .min(1, { message: validationMessages.required })
    .email({ message: validationMessages.email }),

  /** Optional email */
  optionalEmail: z
    .string()
    .email({ message: validationMessages.email })
    .optional()
    .or(z.literal('')),

  /** Phone number (Chinese mobile) */
  phone: z
    .string()
    .min(1, { message: validationMessages.required })
    .regex(patterns.phone, { message: validationMessages.phone }),

  /** Optional phone */
  optionalPhone: z
    .string()
    .regex(patterns.phone, { message: validationMessages.phone })
    .optional()
    .or(z.literal('')),

  /** Positive number */
  positiveNumber: z
    .number({ message: validationMessages.number })
    .positive({ message: validationMessages.positive }),

  /** Non-negative number */
  nonNegativeNumber: z
    .number({ message: validationMessages.number })
    .nonnegative({ message: validationMessages.nonNegative }),

  /** Positive integer */
  positiveInteger: z
    .number({ message: validationMessages.number })
    .int({ message: validationMessages.integer })
    .positive({ message: validationMessages.positive }),

  /** Money value (positive, 2 decimal places) */
  money: z
    .number({ message: validationMessages.number })
    .nonnegative({ message: validationMessages.nonNegative })
    .multipleOf(0.01, { message: validationMessages.moneyPrecision }),

  /** Quantity (positive number) */
  quantity: z
    .number({ message: validationMessages.number })
    .positive({ message: validationMessages.positive }),

  /** SKU */
  sku: z
    .string()
    .min(1, { message: validationMessages.required })
    .regex(patterns.sku, { message: validationMessages.invalidSku }),

  /** Barcode */
  barcode: z
    .string()
    .regex(patterns.barcode, { message: validationMessages.invalidBarcode })
    .optional()
    .or(z.literal('')),

  /** URL */
  url: z.string().url({ message: validationMessages.url }).optional().or(z.literal('')),

  /** Date string (ISO format) */
  dateString: z.string().datetime({ message: validationMessages.date }),

  /** Optional date string */
  optionalDateString: z.string().datetime({ message: validationMessages.date }).optional(),

  /** ID (UUID format) */
  id: z.string().uuid({ message: validationMessages.invalidId }),

  /** Optional ID */
  optionalId: z
    .string()
    .uuid({ message: validationMessages.invalidId })
    .optional()
    .or(z.literal('')),
}

/**
 * Create a required string schema with custom min/max length
 */
export function createStringSchema(options: {
  required?: boolean
  min?: number
  max?: number
  pattern?: RegExp
  patternMessage?: string
}) {
  let schema = z.string()

  if (options.required !== false) {
    schema = schema.min(1, { message: validationMessages.required })
  }

  if (options.min) {
    schema = schema.min(options.min, { message: validationMessages.minLength(options.min) })
  }

  if (options.max) {
    schema = schema.max(options.max, { message: validationMessages.maxLength(options.max) })
  }

  if (options.pattern) {
    schema = schema.regex(options.pattern, {
      message: options.patternMessage || validationMessages.pattern,
    })
  }

  return schema
}

/**
 * Create a number schema with custom range
 */
export function createNumberSchema(options: {
  required?: boolean
  min?: number
  max?: number
  integer?: boolean
  positive?: boolean
}) {
  let schema = z.number({ message: validationMessages.number })

  if (options.integer) {
    schema = schema.int({ message: validationMessages.integer })
  }

  if (options.positive) {
    schema = schema.positive({ message: validationMessages.positive })
  } else if (options.min !== undefined) {
    schema = schema.min(options.min, { message: validationMessages.min(options.min) })
  }

  if (options.max !== undefined) {
    schema = schema.max(options.max, { message: validationMessages.max(options.max) })
  }

  if (options.required === false) {
    return schema.optional()
  }

  return schema
}

/**
 * Create a select/enum schema
 */
export function createEnumSchema<T extends string>(values: readonly T[], required = true) {
  const schema = z.enum(values as [T, ...T[]])
  return required ? schema : schema.optional()
}

/**
 * Utility to transform empty strings to undefined
 */
export const emptyToUndefined = z.literal('').transform(() => undefined)

/**
 * Coerce string to number (for form inputs)
 */
export const coerceNumber = z.coerce.number({ message: validationMessages.number })

/**
 * Coerce string to positive number
 */
export const coercePositiveNumber = coerceNumber.positive({ message: validationMessages.positive })

/**
 * Coerce string to non-negative number
 */
export const coerceNonNegativeNumber = coerceNumber.nonnegative({
  message: validationMessages.nonNegative,
})
