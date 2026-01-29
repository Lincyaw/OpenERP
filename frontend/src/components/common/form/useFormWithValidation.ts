import { useCallback, useState } from 'react'
import {
  useForm,
  type UseFormProps,
  type FieldValues,
  type SubmitHandler,
  type Path,
} from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { z } from 'zod'
import { Toast } from '@douyinfe/semi-ui-19'
import type { ApiError } from '@/types/api'
import { handleError } from '@/services/error-handler'

interface UseFormWithValidationOptions<TFieldValues extends FieldValues> extends Omit<
  UseFormProps<TFieldValues>,
  'resolver'
> {
  /** Zod schema for validation */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  schema: z.ZodType<TFieldValues, any, any>
  /** Success message to show on submit */
  successMessage?: string
  /** Error message to show on submit failure */
  errorMessage?: string
  /** Reset form on successful submit */
  resetOnSuccess?: boolean
  /** Callback on successful submit */
  onSuccess?: (data: TFieldValues) => void | Promise<void>
  /** Callback on submit error */
  onError?: (error: Error | ApiError) => void
}

interface UseFormWithValidationReturn<TFieldValues extends FieldValues> {
  // Form methods from react-hook-form
  register: ReturnType<typeof useForm<TFieldValues>>['register']
  control: ReturnType<typeof useForm<TFieldValues>>['control']
  handleSubmit: ReturnType<typeof useForm<TFieldValues>>['handleSubmit']
  watch: ReturnType<typeof useForm<TFieldValues>>['watch']
  formState: ReturnType<typeof useForm<TFieldValues>>['formState']
  setValue: ReturnType<typeof useForm<TFieldValues>>['setValue']
  getValues: ReturnType<typeof useForm<TFieldValues>>['getValues']
  reset: ReturnType<typeof useForm<TFieldValues>>['reset']
  setError: ReturnType<typeof useForm<TFieldValues>>['setError']
  clearErrors: ReturnType<typeof useForm<TFieldValues>>['clearErrors']
  trigger: ReturnType<typeof useForm<TFieldValues>>['trigger']
  // Custom additions
  isSubmitting: boolean
  submitError: Error | ApiError | null
  handleFormSubmit: (
    onSubmit: SubmitHandler<TFieldValues>
  ) => (e?: React.BaseSyntheticEvent) => Promise<void>
  setServerErrors: (errors: Array<{ field: string; message: string }>) => void
}

/**
 * Custom hook that wraps react-hook-form with Zod validation and common functionality
 */
export function useFormWithValidation<TFieldValues extends FieldValues>({
  schema,
  successMessage,
  errorMessage = '操作失败，请重试',
  resetOnSuccess = false,
  onSuccess,
  onError,
  ...formOptions
}: UseFormWithValidationOptions<TFieldValues>): UseFormWithValidationReturn<TFieldValues> {
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<Error | ApiError | null>(null)

  const form = useForm<TFieldValues>({
    ...formOptions,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(schema as any),
  })

  const handleFormSubmit = useCallback(
    (onSubmit: SubmitHandler<TFieldValues>) => {
      return form.handleSubmit(async (data) => {
        setIsSubmitting(true)
        setSubmitError(null)

        try {
          await onSubmit(data)

          if (successMessage) {
            Toast.success(successMessage)
          }

          if (resetOnSuccess) {
            form.reset()
          }

          await onSuccess?.(data)
        } catch (error) {
          const err = error as Error | ApiError
          setSubmitError(err)

          // Handle API errors with field-level details
          if ('details' in err && Array.isArray(err.details)) {
            err.details.forEach(({ field, message }) => {
              form.setError(field as Path<TFieldValues>, {
                type: 'server',
                message,
              })
            })
          } else {
            // Use unified error handler with custom fallback message
            handleError(error, { showToast: true, fallbackMessage: errorMessage })
          }

          onError?.(err)
        } finally {
          setIsSubmitting(false)
        }
      })
    },
    [form, successMessage, resetOnSuccess, onSuccess, onError]
  )

  const setServerErrors = useCallback(
    (errors: Array<{ field: string; message: string }>) => {
      errors.forEach(({ field, message }) => {
        form.setError(field as Path<TFieldValues>, {
          type: 'server',
          message,
        })
      })
    },
    [form]
  )

  return {
    register: form.register,
    control: form.control,
    handleSubmit: form.handleSubmit,
    watch: form.watch,
    formState: form.formState,
    setValue: form.setValue,
    getValues: form.getValues,
    reset: form.reset,
    setError: form.setError,
    clearErrors: form.clearErrors,
    trigger: form.trigger,
    isSubmitting,
    submitError,
    handleFormSubmit,
    setServerErrors,
  }
}

export default useFormWithValidation
