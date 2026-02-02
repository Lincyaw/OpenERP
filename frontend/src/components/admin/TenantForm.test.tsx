/**
 * TenantForm Component Tests (P3-ADMIN-021)
 *
 * Tests for create and edit tenant form with Zod validation.
 * Covers: schema validation, form rendering, mode differences, loading state.
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { createTenantSchema, editTenantSchema } from './TenantForm'
import TenantForm from './TenantForm'
import { createRef } from 'react'
import type { TenantFormHandle } from './TenantForm'

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback: string) => fallback,
    i18n: { language: 'en' },
  }),
}))

describe('TenantForm', () => {
  describe('createTenantSchema', () => {
    it('should validate a valid create tenant form', () => {
      const validData = {
        name: 'Acme Corp',
        code: 'acme-corp',
        short_name: 'Acme',
        contact_email: 'admin@acme.com',
        contact_name: 'John Doe',
        contact_phone: '123-456-7890',
        address: '123 Main St',
        domain: 'acme.example.com',
        plan: 'basic' as const,
        trial_days: 14,
        notes: 'Test tenant',
      }

      const result = createTenantSchema.safeParse(validData)
      expect(result.success).toBe(true)
    })

    it('should require name', () => {
      const result = createTenantSchema.safeParse({
        name: '',
        code: 'test',
      })
      expect(result.success).toBe(false)
      if (!result.success) {
        expect(result.error.issues.some((i) => i.path.includes('name'))).toBe(true)
      }
    })

    it('should enforce name max length of 200', () => {
      const result = createTenantSchema.safeParse({
        name: 'a'.repeat(201),
        code: 'test',
      })
      expect(result.success).toBe(false)
    })

    it('should require code to start with lowercase letter', () => {
      const invalidCodes = ['123abc', 'Abc', '-test', '_test']
      for (const code of invalidCodes) {
        const result = createTenantSchema.safeParse({ name: 'Test', code })
        expect(result.success).toBe(false)
      }
    })

    it('should accept valid codes', () => {
      const validCodes = ['abc', 'a1', 'test-code', 'my_tenant', 'a123-test_org']
      for (const code of validCodes) {
        const result = createTenantSchema.safeParse({ name: 'Test', code })
        expect(result.success).toBe(true)
      }
    })

    it('should enforce code min length of 2', () => {
      const result = createTenantSchema.safeParse({ name: 'Test', code: 'a' })
      expect(result.success).toBe(false)
    })

    it('should validate email format', () => {
      const result = createTenantSchema.safeParse({
        name: 'Test',
        code: 'test',
        contact_email: 'not-an-email',
      })
      expect(result.success).toBe(false)
    })

    it('should allow empty optional email', () => {
      const result = createTenantSchema.safeParse({
        name: 'Test',
        code: 'test',
        contact_email: '',
      })
      expect(result.success).toBe(true)
    })

    it('should validate plan enum values', () => {
      const validPlans = ['free', 'basic', 'pro', 'enterprise'] as const
      for (const plan of validPlans) {
        const result = createTenantSchema.safeParse({
          name: 'Test',
          code: 'test',
          plan,
        })
        expect(result.success).toBe(true)
      }

      const invalidResult = createTenantSchema.safeParse({
        name: 'Test',
        code: 'test',
        plan: 'invalid',
      })
      expect(invalidResult.success).toBe(false)
    })

    it('should validate trial_days range (1-365)', () => {
      const result0 = createTenantSchema.safeParse({
        name: 'Test',
        code: 'test',
        trial_days: 0,
      })
      expect(result0.success).toBe(false)

      const result366 = createTenantSchema.safeParse({
        name: 'Test',
        code: 'test',
        trial_days: 366,
      })
      expect(result366.success).toBe(false)

      const resultValid = createTenantSchema.safeParse({
        name: 'Test',
        code: 'test',
        trial_days: 30,
      })
      expect(resultValid.success).toBe(true)
    })

    it('should require trial_days to be an integer', () => {
      const result = createTenantSchema.safeParse({
        name: 'Test',
        code: 'test',
        trial_days: 14.5,
      })
      expect(result.success).toBe(false)
    })
  })

  describe('editTenantSchema', () => {
    it('should validate a valid edit tenant form', () => {
      const result = editTenantSchema.safeParse({
        name: 'Updated Corp',
        short_name: 'Updated',
        contact_email: 'new@corp.com',
        contact_name: 'Jane',
        contact_phone: '999-999-9999',
        address: '456 Other St',
        domain: 'updated.example.com',
        notes: 'Updated notes',
      })
      expect(result.success).toBe(true)
    })

    it('should not have code field (read-only after creation)', () => {
      const result = editTenantSchema.safeParse({
        name: 'Test',
        code: 'test-code',
      })
      // code should be stripped/ignored in edit schema
      expect(result.success).toBe(true)
      if (result.success) {
        expect('code' in result.data).toBe(false)
      }
    })

    it('should not have plan field (separate modal)', () => {
      const result = editTenantSchema.safeParse({
        name: 'Test',
        plan: 'pro',
      })
      expect(result.success).toBe(true)
      if (result.success) {
        expect('plan' in result.data).toBe(false)
      }
    })

    it('should not have trial_days field', () => {
      const result = editTenantSchema.safeParse({
        name: 'Test',
        trial_days: 14,
      })
      expect(result.success).toBe(true)
      if (result.success) {
        expect('trial_days' in result.data).toBe(false)
      }
    })

    it('should require name', () => {
      const result = editTenantSchema.safeParse({
        name: '',
      })
      expect(result.success).toBe(false)
    })

    it('should enforce address max length of 500', () => {
      const result = editTenantSchema.safeParse({
        name: 'Test',
        address: 'a'.repeat(501),
      })
      expect(result.success).toBe(false)
    })
  })

  describe('TenantForm Component - Create Mode', () => {
    it('should render in create mode with all fields', () => {
      const onSubmit = vi.fn()
      render(<TenantForm mode="create" onSubmit={onSubmit} />)

      // Check for required and optional fields
      expect(screen.getByPlaceholderText('Enter tenant name')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('Enter tenant code (e.g., acme-corp)')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('Enter short name (optional)')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('Enter contact email')).toBeInTheDocument()
    })

    it('should show loading spinner when isLoading', () => {
      const onSubmit = vi.fn()
      render(<TenantForm mode="create" onSubmit={onSubmit} isLoading={true} />)

      expect(screen.queryByPlaceholderText('Enter tenant name')).not.toBeInTheDocument()
    })

    it('should render plan select in create mode', () => {
      const onSubmit = vi.fn()
      render(<TenantForm mode="create" onSubmit={onSubmit} />)

      expect(screen.getByText('Initial Plan')).toBeInTheDocument()
    })

    it('should render trial days input in create mode', () => {
      const onSubmit = vi.fn()
      render(<TenantForm mode="create" onSubmit={onSubmit} />)

      expect(screen.getByText('Trial Days')).toBeInTheDocument()
    })
  })

  describe('TenantForm Component - Edit Mode', () => {
    it('should render in edit mode without code field', () => {
      const onSubmit = vi.fn()
      render(
        <TenantForm
          mode="edit"
          onSubmit={onSubmit}
          initialData={{
            id: '123',
            name: 'Existing Tenant',
            code: 'existing-code',
          }}
        />
      )

      // Name field should be present
      expect(screen.getByPlaceholderText('Enter tenant name')).toBeInTheDocument()
      // Code field should not be present in edit mode
      expect(
        screen.queryByPlaceholderText('Enter tenant code (e.g., acme-corp)')
      ).not.toBeInTheDocument()
    })

    it('should not show plan field in edit mode', () => {
      const onSubmit = vi.fn()
      render(
        <TenantForm
          mode="edit"
          onSubmit={onSubmit}
          initialData={{
            id: '123',
            name: 'Test',
          }}
        />
      )

      // Plan is changed via separate modal, not in edit form
      expect(screen.queryByText('Initial Plan')).not.toBeInTheDocument()
    })

    it('should not show trial days in edit mode', () => {
      const onSubmit = vi.fn()
      render(
        <TenantForm
          mode="edit"
          onSubmit={onSubmit}
          initialData={{
            id: '123',
            name: 'Test',
          }}
        />
      )

      expect(screen.queryByText('Trial Days')).not.toBeInTheDocument()
    })

    it('should expose submit and reset via ref', () => {
      const onSubmit = vi.fn()
      const ref = createRef<TenantFormHandle>()

      render(
        <TenantForm
          ref={ref}
          mode="edit"
          onSubmit={onSubmit}
          initialData={{
            id: '123',
            name: 'Test',
          }}
        />
      )

      // Verify ref methods exist
      expect(ref.current).toBeDefined()
      expect(typeof ref.current?.submit).toBe('function')
      expect(typeof ref.current?.reset).toBe('function')
    })
  })
})
