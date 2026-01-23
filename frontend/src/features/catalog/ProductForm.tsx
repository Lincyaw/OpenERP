import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography } from '@douyinfe/semi-ui'
import { useNavigate } from 'react-router-dom'
import {
  Form,
  FormActions,
  FormSection,
  FormRow,
  TextField,
  NumberField,
  TextAreaField,
  useFormWithValidation,
  validationMessages,
} from '@/components/common/form'
import { Container } from '@/components/common/layout'
import { getProducts } from '@/api/products/products'
import type { HandlerProductResponse } from '@/api/models'
import './ProductForm.css'

const { Title } = Typography

// Form validation schema
const productFormSchema = z.object({
  code: z
    .string()
    .min(1, validationMessages.required)
    .max(50, validationMessages.maxLength(50))
    .regex(/^[A-Za-z0-9_-]+$/, '编码只能包含字母、数字、下划线和横线'),
  name: z.string().min(1, validationMessages.required).max(200, validationMessages.maxLength(200)),
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
  purchase_price: z
    .number()
    .nonnegative(validationMessages.nonNegative)
    .optional()
    .nullable()
    .transform((val) => val ?? undefined),
  selling_price: z
    .number()
    .nonnegative(validationMessages.nonNegative)
    .optional()
    .nullable()
    .transform((val) => val ?? undefined),
  min_stock: z
    .number()
    .int(validationMessages.integer)
    .nonnegative(validationMessages.nonNegative)
    .optional()
    .nullable()
    .transform((val) => val ?? undefined),
  sort_order: z
    .number()
    .int(validationMessages.integer)
    .nonnegative(validationMessages.nonNegative)
    .optional()
    .nullable()
    .transform((val) => val ?? undefined),
})

type ProductFormData = z.infer<typeof productFormSchema>

interface ProductFormProps {
  /** Product ID for edit mode, undefined for create mode */
  productId?: string
  /** Initial product data for edit mode */
  initialData?: HandlerProductResponse
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
  const api = useMemo(() => getProducts(), [])
  const isEditMode = Boolean(productId)

  // Transform API data to form values
  const defaultValues: Partial<ProductFormData> = useMemo(() => {
    if (!initialData) {
      return {
        code: '',
        name: '',
        unit: '个',
        barcode: '',
        description: '',
        purchase_price: undefined,
        selling_price: undefined,
        min_stock: undefined,
        sort_order: undefined,
      }
    }
    return {
      code: initialData.code || '',
      name: initialData.name || '',
      unit: initialData.unit || '个',
      barcode: initialData.barcode || '',
      description: initialData.description || '',
      purchase_price: initialData.purchase_price ?? undefined,
      selling_price: initialData.selling_price ?? undefined,
      min_stock: initialData.min_stock ?? undefined,
      sort_order: initialData.sort_order ?? undefined,
    }
  }, [initialData])

  const { control, handleFormSubmit, isSubmitting, reset } = useFormWithValidation<ProductFormData>(
    {
      schema: productFormSchema,
      defaultValues,
      successMessage: isEditMode ? '商品更新成功' : '商品创建成功',
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
      const response = await api.putCatalogProductsId(productId, {
        name: data.name,
        barcode: data.barcode,
        description: data.description,
        purchase_price: data.purchase_price,
        selling_price: data.selling_price,
        min_stock: data.min_stock,
        sort_order: data.sort_order,
      })
      if (!response.success) {
        throw new Error(response.error?.message || '更新失败')
      }
    } else {
      // Create new product
      const response = await api.postCatalogProducts({
        code: data.code,
        name: data.name,
        unit: data.unit,
        barcode: data.barcode,
        description: data.description,
        purchase_price: data.purchase_price,
        selling_price: data.selling_price,
        min_stock: data.min_stock,
        sort_order: data.sort_order,
      })
      if (!response.success) {
        throw new Error(response.error?.message || '创建失败')
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
            {isEditMode ? '编辑商品' : '新增商品'}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection title="基本信息" description="商品的基本属性和标识信息">
            <FormRow cols={2}>
              <TextField
                name="code"
                control={control}
                label="商品编码"
                placeholder="请输入商品编码 (SKU)"
                required
                disabled={isEditMode}
                helperText={isEditMode ? '编码创建后不可修改' : '支持字母、数字、下划线和横线'}
              />
              <TextField
                name="name"
                control={control}
                label="商品名称"
                placeholder="请输入商品名称"
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="unit"
                control={control}
                label="单位"
                placeholder="请输入计量单位"
                required
                disabled={isEditMode}
                helperText={isEditMode ? '单位创建后不可修改' : '例如：个、件、箱、千克'}
              />
              <TextField
                name="barcode"
                control={control}
                label="条形码"
                placeholder="请输入条形码 (可选)"
              />
            </FormRow>
            <TextAreaField
              name="description"
              control={control}
              label="商品描述"
              placeholder="请输入商品描述 (可选)"
              rows={3}
              maxCount={2000}
            />
          </FormSection>

          <FormSection title="价格信息" description="商品的进货价和销售价">
            <FormRow cols={2}>
              <NumberField
                name="purchase_price"
                control={control}
                label="进货价"
                placeholder="请输入进货价"
                min={0}
                precision={2}
                prefix="¥"
              />
              <NumberField
                name="selling_price"
                control={control}
                label="销售价"
                placeholder="请输入销售价"
                min={0}
                precision={2}
                prefix="¥"
              />
            </FormRow>
          </FormSection>

          <FormSection title="库存设置" description="库存相关的配置">
            <FormRow cols={2}>
              <NumberField
                name="min_stock"
                control={control}
                label="最低库存"
                placeholder="请输入最低库存预警值"
                min={0}
                precision={0}
                helperText="库存低于此值时会触发预警"
              />
              <NumberField
                name="sort_order"
                control={control}
                label="排序"
                placeholder="请输入排序值"
                min={0}
                precision={0}
                helperText="数值越小越靠前"
              />
            </FormRow>
          </FormSection>

          <FormActions
            submitText={isEditMode ? '保存' : '创建'}
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
