import { z } from 'zod'

/**
 * Common validation messages (Chinese)
 */
export const validationMessages = {
  required: '此字段为必填项',
  email: '请输入有效的邮箱地址',
  minLength: (min: number) => `至少需要 ${min} 个字符`,
  maxLength: (max: number) => `最多 ${max} 个字符`,
  min: (min: number) => `不能小于 ${min}`,
  max: (max: number) => `不能大于 ${max}`,
  pattern: '格式不正确',
  phone: '请输入有效的手机号码',
  url: '请输入有效的URL',
  number: '请输入有效的数字',
  integer: '请输入整数',
  positive: '必须为正数',
  nonNegative: '不能为负数',
  date: '请输入有效的日期',
  dateRange: '结束日期不能早于开始日期',
  passwordMatch: '两次输入的密码不一致',
  unique: '该值已存在',
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
 */
export const schemas = {
  /** Required string */
  requiredString: z.string().min(1, validationMessages.required),

  /** Optional string (empty string becomes undefined) */
  optionalString: z
    .string()
    .optional()
    .transform((val) => val || undefined),

  /** Email field */
  email: z.string().min(1, validationMessages.required).email(validationMessages.email),

  /** Optional email */
  optionalEmail: z.string().email(validationMessages.email).optional().or(z.literal('')),

  /** Phone number (Chinese mobile) */
  phone: z
    .string()
    .min(1, validationMessages.required)
    .regex(patterns.phone, validationMessages.phone),

  /** Optional phone */
  optionalPhone: z
    .string()
    .regex(patterns.phone, validationMessages.phone)
    .optional()
    .or(z.literal('')),

  /** Positive number */
  positiveNumber: z
    .number({ message: validationMessages.number })
    .positive(validationMessages.positive),

  /** Non-negative number */
  nonNegativeNumber: z
    .number({ message: validationMessages.number })
    .nonnegative(validationMessages.nonNegative),

  /** Positive integer */
  positiveInteger: z
    .number({ message: validationMessages.number })
    .int(validationMessages.integer)
    .positive(validationMessages.positive),

  /** Money value (positive, 2 decimal places) */
  money: z
    .number({ message: validationMessages.number })
    .nonnegative(validationMessages.nonNegative)
    .multipleOf(0.01, '金额最多两位小数'),

  /** Quantity (positive number) */
  quantity: z.number({ message: validationMessages.number }).positive(validationMessages.positive),

  /** SKU */
  sku: z
    .string()
    .min(1, validationMessages.required)
    .regex(patterns.sku, 'SKU只能包含字母、数字和横线'),

  /** Barcode */
  barcode: z.string().regex(patterns.barcode, '请输入有效的条形码').optional().or(z.literal('')),

  /** URL */
  url: z.string().url(validationMessages.url).optional().or(z.literal('')),

  /** Date string (ISO format) */
  dateString: z.string().datetime({ message: validationMessages.date }),

  /** Optional date string */
  optionalDateString: z.string().datetime({ message: validationMessages.date }).optional(),

  /** ID (UUID format) */
  id: z.string().uuid('无效的ID格式'),

  /** Optional ID */
  optionalId: z.string().uuid('无效的ID格式').optional().or(z.literal('')),
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
    schema = schema.min(1, validationMessages.required)
  }

  if (options.min) {
    schema = schema.min(options.min, validationMessages.minLength(options.min))
  }

  if (options.max) {
    schema = schema.max(options.max, validationMessages.maxLength(options.max))
  }

  if (options.pattern) {
    schema = schema.regex(options.pattern, options.patternMessage || validationMessages.pattern)
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
    schema = schema.int(validationMessages.integer)
  }

  if (options.positive) {
    schema = schema.positive(validationMessages.positive)
  } else if (options.min !== undefined) {
    schema = schema.min(options.min, validationMessages.min(options.min))
  }

  if (options.max !== undefined) {
    schema = schema.max(options.max, validationMessages.max(options.max))
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
export const coercePositiveNumber = coerceNumber.positive(validationMessages.positive)

/**
 * Coerce string to non-negative number
 */
export const coerceNonNegativeNumber = coerceNumber.nonnegative(validationMessages.nonNegative)
