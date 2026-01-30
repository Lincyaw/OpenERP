import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography } from '@douyinfe/semi-ui-19'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Form,
  FormActions,
  FormSection,
  FormRow,
  TextField,
  NumberField,
  TextAreaField,
  SelectField,
  useFormWithValidation,
  validationMessages,
} from '@/components/common/form'
import { Container } from '@/components/common/layout'
import { createProduct, updateProduct } from '@/api/products/products'
import { useListCategories } from '@/api/categories/categories'
import type { HandlerProductResponse } from '@/api/models'
import { ProductAttachmentUploader } from './ProductAttachmentUploader'
import './ProductForm.css'

const { Title } = Typography

type ProductFormData = z.infer<ReturnType<typeof createProductFormSchema>>

interface ProductFormProps {
  /** Product ID for edit mode, undefined for create mode */
  productId?: string
  /** Initial product data for edit mode */
  initialData?: HandlerProductResponse
}

/**
 * Create form validation schema with i18n support
 */
function createProductFormSchema(codeRegexError: string, categoryRequired: string) {
  return z.object({
    code: z
      .string()
      .min(1, validationMessages.required)
      .max(50, validationMessages.maxLength(50))
      .regex(/^[A-Za-z0-9_-]+$/, codeRegexError),
    name: z
      .string()
      .min(1, validationMessages.required)
      .max(200, validationMessages.maxLength(200)),
    category_id: z.string().min(1, categoryRequired),
    unit: z.string().min(1, validationMessages.required).max(20, validationMessages.maxLength(20)),
    barcode: z
      .string()
      .max(50, validationMessages.maxLength(50))
      .optional()
      .transform((val) => val || undefined),
    description: z
      .string()
      .max(2000, validationMessages.maxLength(2000))
      .optional()
      .transform((val) => val || undefined),
    // Use coerce to handle string values from API/form inputs
    purchase_price: z.coerce
      .number()
      .nonnegative(validationMessages.nonNegative)
      .optional()
      .nullable()
      .transform((val) => val ?? undefined),
    selling_price: z.coerce
      .number()
      .nonnegative(validationMessages.nonNegative)
      .optional()
      .nullable()
      .transform((val) => val ?? undefined),
    min_stock: z.coerce
      .number()
      .int(validationMessages.integer)
      .nonnegative(validationMessages.nonNegative)
      .optional()
      .nullable()
      .transform((val) => val ?? undefined),
    sort_order: z.coerce
      .number()
      .int(validationMessages.integer)
      .nonnegative(validationMessages.nonNegative)
      .optional()
      .nullable()
      .transform((val) => val ?? undefined),
  })
}

/**
 * Product form component for creating and editing products
 *
 * Features:
 * - Zod schema validation
 * - Create/edit modes
 * - Form sections for better organization
 * - API integration with error handling
 */
export function ProductForm({ productId, initialData }: ProductFormProps) {
  const navigate = useNavigate()
  const { t } = useTranslation(['catalog', 'common'])
  const isEditMode = Boolean(productId)

  // Create schema with i18n error messages
  const productFormSchema = useMemo(
    () =>
      createProductFormSchema(
        t('products.form.codeRegexError'),
        t('products.form.categoryRequired') as string
      ),
    [t]
  )

  // Fetch categories for select
  const { data: categoriesResponse } = useListCategories({ status: 'active', page_size: 20 })
  const categoryOptions = useMemo(() => {
    const categories =
      (categoriesResponse?.data?.data as Array<{ id?: string; name?: string }>) || []
    return categories.map((cat) => ({
      value: cat.id || '',
      label: cat.name || '',
    }))
  }, [categoriesResponse])

  // Transform API data to form values
  const defaultValues: Partial<ProductFormData> = useMemo(() => {
    if (!initialData) {
      return {
        code: '',
        name: '',
        category_id: '',
        unit: '',
        barcode: '',
        description: '',
        purchase_price: undefined,
        selling_price: undefined,
        min_stock: undefined,
        sort_order: undefined,
      }
    }
    // Convert string values to numbers if needed (API may return strings)
    const toNumber = (val: unknown): number | undefined => {
      if (val === undefined || val === null || val === '') return undefined
      const num = typeof val === 'string' ? parseFloat(val) : val
      return typeof num === 'number' && !isNaN(num) ? num : undefined
    }
    return {
      code: initialData.code || '',
      name: initialData.name || '',
      category_id: initialData.category_id || '',
      unit: initialData.unit || '',
      barcode: initialData.barcode || '',
      description: initialData.description || '',
      purchase_price: toNumber(initialData.purchase_price),
      selling_price: toNumber(initialData.selling_price),
      min_stock: toNumber(initialData.min_stock),
      sort_order: toNumber(initialData.sort_order),
    }
  }, [initialData])

  const { control, handleFormSubmit, isSubmitting, reset } = useFormWithValidation<ProductFormData>(
    {
      schema: productFormSchema,
      defaultValues,
      successMessage: isEditMode
        ? t('products.messages.updateSuccess')
        : t('products.messages.createSuccess'),
      onSuccess: () => {
        navigate('/catalog/products')
      },
    }
  )

  // Reset form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      reset(defaultValues)
    }
  }, [initialData, defaultValues, reset])

  // Handle form submission
  const onSubmit = async (data: ProductFormData) => {
    if (isEditMode && productId) {
      // Update existing product
      const response = await updateProduct(productId, {
        name: data.name,
        barcode: data.barcode,
        description: data.description,
        purchase_price: data.purchase_price,
        selling_price: data.selling_price,
        min_stock: data.min_stock,
        sort_order: data.sort_order,
      })
      if (response.status !== 200 || !response.data.success) {
        const error = response.data.error as { message?: string } | undefined
        throw new Error(error?.message || t('products.messages.updateError'))
      }
    } else {
      // Create new product
      const response = await createProduct({
        code: data.code,
        name: data.name,
        category_id: data.category_id,
        unit: data.unit,
        barcode: data.barcode,
        description: data.description,
        purchase_price: data.purchase_price,
        selling_price: data.selling_price,
        min_stock: data.min_stock,
        sort_order: data.sort_order,
      })
      if (response.status !== 201 || !response.data.success) {
        const error = response.data.error as { message?: string } | undefined
        throw new Error(error?.message || t('products.messages.createError'))
      }
    }
  }

  const handleCancel = () => {
    navigate('/catalog/products')
  }

  return (
    <Container size="md" className="product-form-page">
      <Card className="product-form-card">
        <div className="product-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? t('products.editProduct') : t('products.createProduct')}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection
            title={t('products.form.basicInfo')}
            description={t('products.form.basicInfoDesc')}
          >
            <FormRow cols={2}>
              <SelectField
                name="category_id"
                control={control}
                label={t('products.form.category') as string}
                placeholder={t('products.form.categoryPlaceholder') as string}
                options={categoryOptions}
                required
                showSearch
                disabled={isEditMode}
              />
              <TextField
                name="name"
                control={control}
                label={t('products.form.name')}
                placeholder={t('products.form.namePlaceholder')}
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="code"
                control={control}
                label={t('products.form.code')}
                placeholder={t('products.form.codePlaceholder')}
                required
                disabled={isEditMode}
                helperText={
                  isEditMode
                    ? t('products.form.codeHelperEdit')
                    : t('products.form.codeHelperCreate')
                }
              />
              <TextField
                name="unit"
                control={control}
                label={t('products.form.unit')}
                placeholder={t('products.form.unitPlaceholder')}
                required
                disabled={isEditMode}
                helperText={
                  isEditMode
                    ? t('products.form.unitHelperEdit')
                    : t('products.form.unitHelperCreate')
                }
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="barcode"
                control={control}
                label={t('products.form.barcode')}
                placeholder={t('products.form.barcodePlaceholder')}
              />
              <div /> {/* Empty placeholder for grid alignment */}
            </FormRow>
            <TextAreaField
              name="description"
              control={control}
              label={t('products.form.description')}
              placeholder={t('products.form.descriptionPlaceholder')}
              rows={3}
              maxCount={2000}
            />
          </FormSection>

          <FormSection
            title={t('products.form.priceInfo')}
            description={t('products.form.priceInfoDesc')}
          >
            <FormRow cols={2}>
              <NumberField
                name="purchase_price"
                control={control}
                label={t('products.form.purchasePrice')}
                placeholder={t('products.form.purchasePricePlaceholder')}
                min={0}
                precision={2}
                prefix="¥"
              />
              <NumberField
                name="selling_price"
                control={control}
                label={t('products.form.sellingPrice')}
                placeholder={t('products.form.sellingPricePlaceholder')}
                min={0}
                precision={2}
                prefix="¥"
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('products.form.stockSettings')}
            description={t('products.form.stockSettingsDesc')}
          >
            <FormRow cols={2}>
              <NumberField
                name="min_stock"
                control={control}
                label={t('products.form.minStock')}
                placeholder={t('products.form.minStockPlaceholder')}
                min={0}
                precision={0}
                helperText={t('products.form.minStockHelper')}
              />
              <NumberField
                name="sort_order"
                control={control}
                label={t('products.form.sortOrder')}
                placeholder={t('products.form.sortOrderPlaceholder')}
                min={0}
                precision={0}
                helperText={t('products.form.sortOrderHelper')}
              />
            </FormRow>
          </FormSection>

          {/* Image management section - only shown in edit mode */}
          {isEditMode && productId && (
            <FormSection title={t('attachments.title')} description={t('attachments.description')}>
              <ProductAttachmentUploader productId={productId} disabled={isSubmitting} />
            </FormSection>
          )}

          <FormActions
            submitText={isEditMode ? t('common:actions.save') : t('common:actions.create')}
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}

export default ProductForm
