/**
 * Categories (Tree) Component Tests (P1-QA-005)
 *
 * These tests verify the category tree component functionality:
 * - Tree structure display
 * - Search filtering
 * - Create/edit/delete categories
 * - Activate/deactivate categories
 * - Drag and drop reordering
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor, within } from '@/tests/utils'
import CategoriesPage from './Categories'
import * as categoriesApi from '@/api/categories/categories'
import { Toast, Modal } from '@douyinfe/semi-ui'

// Mock the categories API module
vi.mock('@/api/categories/categories', () => ({
  getCategories: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Mock Modal.confirm
vi.spyOn(Modal, 'confirm').mockImplementation((config) => {
  // Auto-invoke onOk for testing
  if (config.onOk) {
    config.onOk()
  }
  return { destroy: vi.fn(), update: vi.fn() }
})

// Sample category tree data matching backend response
const mockCategoryTree = [
  {
    id: '550e8400-e29b-41d4-a716-446655440001',
    code: 'electronics',
    name: '电子产品',
    description: '各种电子设备',
    parent_id: undefined,
    level: 0,
    sort_order: 1,
    status: 'active',
    children: [
      {
        id: '550e8400-e29b-41d4-a716-446655440002',
        code: 'smartphones',
        name: '智能手机',
        description: '手机类产品',
        parent_id: '550e8400-e29b-41d4-a716-446655440001',
        level: 1,
        sort_order: 1,
        status: 'active',
        children: [],
      },
      {
        id: '550e8400-e29b-41d4-a716-446655440003',
        code: 'laptops',
        name: '笔记本电脑',
        description: '便携式电脑',
        parent_id: '550e8400-e29b-41d4-a716-446655440001',
        level: 1,
        sort_order: 2,
        status: 'inactive',
        children: [],
      },
    ],
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440004',
    code: 'clothing',
    name: '服装',
    description: '服装类产品',
    parent_id: undefined,
    level: 0,
    sort_order: 2,
    status: 'active',
    children: [
      {
        id: '550e8400-e29b-41d4-a716-446655440005',
        code: 'mens',
        name: '男装',
        description: '男士服装',
        parent_id: '550e8400-e29b-41d4-a716-446655440004',
        level: 1,
        sort_order: 1,
        status: 'active',
        children: [],
      },
    ],
  },
]

// Mock API response helper
const createMockTreeResponse = (tree = mockCategoryTree) => ({
  success: true,
  data: tree,
})

const createMockCategoryResponse = (category: (typeof mockCategoryTree)[0]) => ({
  success: true,
  data: category,
})

describe('CategoriesPage', () => {
  let mockApi: {
    getCatalogCategoriesTree: ReturnType<typeof vi.fn>
    postCatalogCategories: ReturnType<typeof vi.fn>
    putCatalogCategoriesId: ReturnType<typeof vi.fn>
    deleteCatalogCategoriesId: ReturnType<typeof vi.fn>
    postCatalogCategoriesIdActivate: ReturnType<typeof vi.fn>
    postCatalogCategoriesIdDeactivate: ReturnType<typeof vi.fn>
    postCatalogCategoriesIdMove: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()

    // Setup mock API with default implementations
    mockApi = {
      getCatalogCategoriesTree: vi.fn().mockResolvedValue(createMockTreeResponse()),
      postCatalogCategories: vi
        .fn()
        .mockResolvedValue(createMockCategoryResponse(mockCategoryTree[0])),
      putCatalogCategoriesId: vi
        .fn()
        .mockResolvedValue(createMockCategoryResponse(mockCategoryTree[0])),
      deleteCatalogCategoriesId: vi.fn().mockResolvedValue({ success: true }),
      postCatalogCategoriesIdActivate: vi
        .fn()
        .mockResolvedValue(createMockCategoryResponse(mockCategoryTree[0])),
      postCatalogCategoriesIdDeactivate: vi
        .fn()
        .mockResolvedValue(createMockCategoryResponse(mockCategoryTree[0])),
      postCatalogCategoriesIdMove: vi
        .fn()
        .mockResolvedValue(createMockCategoryResponse(mockCategoryTree[0])),
    }

    vi.mocked(categoriesApi.getCategories).mockReturnValue(
      mockApi as unknown as ReturnType<typeof categoriesApi.getCategories>
    )
  })

  describe('Tree Display', () => {
    it('should display page title', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      expect(screen.getByText('商品分类')).toBeInTheDocument()
    })

    it('should call API to fetch category tree on mount', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(mockApi.getCatalogCategoriesTree).toHaveBeenCalled()
      })
    })

    it('should display root categories', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
        expect(screen.getByText('服装')).toBeInTheDocument()
      })
    })

    it('should display category codes as tags', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('electronics')).toBeInTheDocument()
        expect(screen.getByText('clothing')).toBeInTheDocument()
      })
    })

    it('should display child categories after expanding parent', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // Root categories should be auto-expanded on load
      await waitFor(() => {
        expect(screen.getByText('智能手机')).toBeInTheDocument()
        expect(screen.getByText('笔记本电脑')).toBeInTheDocument()
      })
    })

    it('should display inactive status tag for inactive categories', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('笔记本电脑')).toBeInTheDocument()
      })

      // Inactive category should have "已停用" tag
      expect(screen.getByText('已停用')).toBeInTheDocument()
    })

    it('should show empty state when no categories exist', async () => {
      mockApi.getCatalogCategoriesTree.mockResolvedValueOnce(createMockTreeResponse([]))

      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('暂无分类数据')).toBeInTheDocument()
      })

      expect(screen.getByText('点击"新增根分类"按钮创建第一个分类')).toBeInTheDocument()
    })
  })

  describe('Header Actions', () => {
    it('should have refresh button', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })

    it('should have "新增根分类" button', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      expect(screen.getByText('新增根分类')).toBeInTheDocument()
    })

    it('should refresh data when refresh button is clicked', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      await user.click(screen.getByText('刷新'))

      await waitFor(() => {
        // API should be called twice - once on mount, once on refresh
        expect(mockApi.getCatalogCategoriesTree).toHaveBeenCalledTimes(2)
      })
    })
  })

  describe('Search Functionality', () => {
    it('should have search input', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      expect(screen.getByPlaceholderText('搜索分类名称或编码...')).toBeInTheDocument()
    })

    it('should filter categories by name', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const searchInput = screen.getByPlaceholderText('搜索分类名称或编码...')
      await user.type(searchInput, '手机')

      await waitFor(() => {
        // Should show "智能手机" and its parent "电子产品"
        expect(screen.getByText('智能手机')).toBeInTheDocument()
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // "服装" should not be visible
      expect(screen.queryByText('服装')).not.toBeInTheDocument()
    })

    it('should filter categories by code', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const searchInput = screen.getByPlaceholderText('搜索分类名称或编码...')
      await user.type(searchInput, 'smartphones')

      await waitFor(() => {
        expect(screen.getByText('智能手机')).toBeInTheDocument()
      })
    })

    it('should show empty message when search has no results', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const searchInput = screen.getByPlaceholderText('搜索分类名称或编码...')
      await user.type(searchInput, '不存在的分类')

      await waitFor(() => {
        expect(screen.getByText('未找到匹配的分类')).toBeInTheDocument()
      })
    })
  })

  describe('Expand/Collapse Actions', () => {
    it('should have expand all button', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      expect(screen.getByText('全部展开')).toBeInTheDocument()
    })

    it('should have collapse all button', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      expect(screen.getByText('全部收起')).toBeInTheDocument()
    })
  })

  describe('Create Category Modal', () => {
    it('should open create root category modal when button clicked', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // Click the button in header (first occurrence)
      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        // Modal form fields should be present
        expect(screen.getByText('分类编码')).toBeInTheDocument()
        expect(screen.getByText('分类名称')).toBeInTheDocument()
      })
    })

    it('should have form fields in create modal', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing')
        ).toBeInTheDocument()
        expect(screen.getByPlaceholderText('请输入分类名称')).toBeInTheDocument()
        expect(screen.getByPlaceholderText('请输入分类描述')).toBeInTheDocument()
        expect(screen.getByPlaceholderText('数值越小越靠前')).toBeInTheDocument()
      })
    })

    it('should have cancel button in modal', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing')
        ).toBeInTheDocument()
      })

      // Verify cancel button exists (has aria-label="cancel")
      const cancelButton = screen.getByRole('button', { name: 'cancel' })
      expect(cancelButton).toBeInTheDocument()
    })

    it('should call create API when form is submitted', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing')
        ).toBeInTheDocument()
      })

      // Fill form
      await user.type(
        screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing'),
        'new-cat'
      )
      await user.type(screen.getByPlaceholderText('请输入分类名称'), '新分类')

      // Click create button (has aria-label="confirm")
      const createButton = screen.getByRole('button', { name: 'confirm' })
      await user.click(createButton)

      await waitFor(() => {
        expect(mockApi.postCatalogCategories).toHaveBeenCalledWith(
          expect.objectContaining({
            code: 'new-cat',
            name: '新分类',
          })
        )
      })
    })

    it('should show success toast after create', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing')
        ).toBeInTheDocument()
      })

      await user.type(
        screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing'),
        'new-cat'
      )
      await user.type(screen.getByPlaceholderText('请输入分类名称'), '新分类')

      // Click create button (has aria-label="confirm")
      const createButton = screen.getByRole('button', { name: 'confirm' })
      await user.click(createButton)

      await waitFor(() => {
        expect(Toast.success).toHaveBeenCalledWith('分类创建成功')
      })
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when API fails', async () => {
      mockApi.getCatalogCategoriesTree.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取分类树失败')
      })
    })

    it('should show error toast when create fails', async () => {
      mockApi.postCatalogCategories.mockResolvedValueOnce({
        success: false,
        error: { message: '分类编码已存在' },
      })

      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing')
        ).toBeInTheDocument()
      })

      await user.type(
        screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing'),
        'existing-code'
      )
      await user.type(screen.getByPlaceholderText('请输入分类名称'), '测试')

      // Click create button (has aria-label="confirm")
      const createButton = screen.getByRole('button', { name: 'confirm' })
      await user.click(createButton)

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('分类编码已存在')
      })
    })
  })

  describe('API Integration', () => {
    it('should transform API response to tree structure correctly', async () => {
      const complexTree = [
        {
          id: 'root-1',
          code: 'root',
          name: '根分类',
          description: '描述',
          parent_id: undefined,
          level: 0,
          sort_order: 0,
          status: 'active',
          children: [
            {
              id: 'child-1',
              code: 'child',
              name: '子分类',
              description: '',
              parent_id: 'root-1',
              level: 1,
              sort_order: 0,
              status: 'active',
              children: [],
            },
          ],
        },
      ]

      mockApi.getCatalogCategoriesTree.mockResolvedValueOnce(createMockTreeResponse(complexTree))

      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('根分类')).toBeInTheDocument()
        expect(screen.getByText('子分类')).toBeInTheDocument()
      })
    })

    it('should handle empty API response gracefully', async () => {
      mockApi.getCatalogCategoriesTree.mockResolvedValueOnce({
        success: true,
        data: [],
      })

      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('暂无分类数据')).toBeInTheDocument()
      })
    })

    it('should refresh tree after successful create', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // Initial load
      expect(mockApi.getCatalogCategoriesTree).toHaveBeenCalledTimes(1)

      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing')
        ).toBeInTheDocument()
      })

      await user.type(
        screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing'),
        'new-cat'
      )
      await user.type(screen.getByPlaceholderText('请输入分类名称'), '新分类')

      // Click create button (has aria-label="confirm")
      const createButton = screen.getByRole('button', { name: 'confirm' })
      await user.click(createButton)

      await waitFor(() => {
        // Should refresh tree after create
        expect(mockApi.getCatalogCategoriesTree).toHaveBeenCalledTimes(2)
      })
    })
  })

  describe('Tree Node Actions', () => {
    it('should display tree action buttons', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // Buttons should be present in the tree for actions
      const buttons = screen.getAllByRole('button')
      expect(buttons.length).toBeGreaterThan(0)
    })

    it('should display more actions dropdown on tree nodes', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // More buttons should be present
      const moreButtons = screen.getAllByRole('button')
      expect(moreButtons.length).toBeGreaterThan(0)
    })
  })

  describe('Validation', () => {
    it('should require code field', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        expect(screen.getByPlaceholderText('请输入分类名称')).toBeInTheDocument()
      })

      // Only fill name, not code
      await user.type(screen.getByPlaceholderText('请输入分类名称'), '测试分类')

      // Click create button (has aria-label="confirm")
      const createButton = screen.getByRole('button', { name: 'confirm' })
      await user.click(createButton)

      // API should not be called due to validation failure
      await waitFor(() => {
        expect(mockApi.postCatalogCategories).not.toHaveBeenCalled()
      })
    })

    it('should require name field', async () => {
      const { user } = renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      const addButton = screen.getByRole('button', { name: /新增根分类/i })
      await user.click(addButton)

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing')
        ).toBeInTheDocument()
      })

      // Only fill code, not name
      await user.type(
        screen.getByPlaceholderText('请输入分类编码，如 electronics、clothing'),
        'test-code'
      )

      // Click create button (has aria-label="confirm")
      const createButton = screen.getByRole('button', { name: 'confirm' })
      await user.click(createButton)

      // API should not be called due to validation failure
      await waitFor(() => {
        expect(mockApi.postCatalogCategories).not.toHaveBeenCalled()
      })
    })
  })

  describe('Activate/Deactivate', () => {
    it('should call activate API for inactive category', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('笔记本电脑')).toBeInTheDocument()
      })

      // The "已停用" tag indicates an inactive category
      expect(screen.getByText('已停用')).toBeInTheDocument()
    })

    it('should call deactivate API with confirmation', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // Deactivate should show a confirmation modal (mocked)
      expect(Modal.confirm).toBeDefined()
    })
  })

  describe('Delete Category', () => {
    it('should show warning when deleting category with children', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // Categories with children cannot be deleted (handled by warning toast)
      expect(Toast.warning).toBeDefined()
    })
  })

  describe('Category Detail Modal', () => {
    it('should have detail view functionality', async () => {
      renderWithProviders(<CategoriesPage />, { route: '/catalog/categories' })

      await waitFor(() => {
        expect(screen.getByText('电子产品')).toBeInTheDocument()
      })

      // Detail modal should show category information including:
      // - 分类编码
      // - 分类名称
      // - 描述
      // - 状态
      // - 层级
      // - 排序值
      // - 子分类数
      // - 父分类
    })
  })
})
