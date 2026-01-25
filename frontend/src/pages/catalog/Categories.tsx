import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Space,
  Modal,
  Spin,
  Form,
  Button,
  Tree,
  Empty,
  Descriptions,
  Input,
  Dropdown,
} from '@douyinfe/semi-ui'
import type { FormApi } from '@douyinfe/semi-ui/lib/es/form/interface'
import type { TreeNodeData, OnDragProps } from '@douyinfe/semi-ui/lib/es/tree/interface'
import {
  IconPlus,
  IconRefresh,
  IconEdit,
  IconMore,
  IconSearch,
  IconTreeTriangleDown,
  IconFolder,
  IconFolderOpen,
} from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import { getCategories } from '@/api/categories/categories'
import type {
  HandlerCategoryTreeNode,
  HandlerCreateCategoryRequest,
  HandlerUpdateCategoryRequest,
} from '@/api/models'
import './Categories.css'

const { Title } = Typography

// Category type for internal use
interface CategoryNode extends HandlerCategoryTreeNode {
  id: string
  code: string
  name: string
  description?: string
  parent_id?: string
  level: number
  sort_order: number
  status: string
  children: CategoryNode[]
}

/**
 * Convert API tree node to Semi Tree node data
 */
function convertToTreeData(nodes: CategoryNode[]): TreeNodeData[] {
  return nodes.map((node) => ({
    key: node.id,
    label: node.name,
    value: node.id,
    icon: node.children && node.children.length > 0 ? <IconFolderOpen /> : <IconFolder />,
    children: node.children ? convertToTreeData(node.children) : [],
    // Store original data for actions
    data: node,
  }))
}

/**
 * Filter tree data by search keyword
 */
function filterTreeData(
  nodes: TreeNodeData[],
  keyword: string,
  originalData: Map<string, CategoryNode>
): TreeNodeData[] {
  if (!keyword) return nodes

  const loweredKeyword = keyword.toLowerCase()
  const matchedKeys = new Set<string>()

  // Find all matching nodes and their ancestors
  const findMatches = (nodeList: TreeNodeData[]): void => {
    for (const node of nodeList) {
      const data = originalData.get(node.key as string)
      if (data) {
        const nameMatch = data.name.toLowerCase().includes(loweredKeyword)
        const codeMatch = data.code.toLowerCase().includes(loweredKeyword)
        if (nameMatch || codeMatch) {
          // Mark this node and all ancestors
          let currentKey = node.key as string
          while (currentKey) {
            matchedKeys.add(currentKey)
            const currentData = originalData.get(currentKey)
            currentKey = currentData?.parent_id || ''
          }
        }
      }
      if (node.children) {
        findMatches(node.children)
      }
    }
  }

  findMatches(nodes)

  // Filter tree to only show matched nodes
  const filterNodes = (nodeList: TreeNodeData[]): TreeNodeData[] => {
    return nodeList
      .filter((node) => matchedKeys.has(node.key as string))
      .map((node) => ({
        ...node,
        children: node.children ? filterNodes(node.children) : [],
      }))
  }

  return filterNodes(nodes)
}

/**
 * Build a map of all nodes by ID for quick lookup
 */
function buildNodeMap(nodes: CategoryNode[]): Map<string, CategoryNode> {
  const map = new Map<string, CategoryNode>()

  const traverse = (nodeList: CategoryNode[]): void => {
    for (const node of nodeList) {
      map.set(node.id, node)
      if (node.children) {
        traverse(node.children)
      }
    }
  }

  traverse(nodes)
  return map
}

/**
 * Categories management page
 *
 * Features:
 * - Tree structure display
 * - Search by name or code
 * - Create/edit/delete categories
 * - Drag and drop to reorder/move
 * - Activate/deactivate categories
 */
export default function CategoriesPage() {
  const { t } = useTranslation(['catalog', 'common'])
  const api = useMemo(() => getCategories(), [])

  // State for data
  const [categoryTree, setCategoryTree] = useState<CategoryNode[]>([])
  const [loading, setLoading] = useState(false)
  const [nodeMap, setNodeMap] = useState<Map<string, CategoryNode>>(new Map())

  // Search state
  const [searchKeyword, setSearchKeyword] = useState('')

  // Tree state
  const [expandedKeys, setExpandedKeys] = useState<string[]>([])

  // Modal state
  const [modalVisible, setModalVisible] = useState(false)
  const [modalMode, setModalMode] = useState<'create' | 'edit'>('create')
  const [editingCategory, setEditingCategory] = useState<CategoryNode | null>(null)
  const [parentCategory, setParentCategory] = useState<CategoryNode | null>(null)
  const [modalLoading, setModalLoading] = useState(false)

  // Detail modal state
  const [detailModalVisible, setDetailModalVisible] = useState(false)
  const [detailCategory, setDetailCategory] = useState<CategoryNode | null>(null)

  // Form ref
  const formApiRef = useRef<FormApi | null>(null)

  // Fetch category tree
  const fetchCategoryTree = useCallback(async () => {
    setLoading(true)
    try {
      const response = await api.getCatalogCategoriesTree()

      if (response.success && response.data) {
        const tree = (response.data as CategoryNode[]) || []
        setCategoryTree(tree)
        setNodeMap(buildNodeMap(tree))

        // Auto-expand first level
        const rootKeys = tree.map((node) => node.id)
        setExpandedKeys(rootKeys)
      }
    } catch {
      Toast.error(t('categories.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [api, t])

  // Fetch on mount
  useEffect(() => {
    fetchCategoryTree()
  }, [fetchCategoryTree])

  // Convert to tree data for Semi Tree component
  const treeData = useMemo(() => {
    const data = convertToTreeData(categoryTree)
    return filterTreeData(data, searchKeyword, nodeMap)
  }, [categoryTree, searchKeyword, nodeMap])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      // Expand all nodes when searching
      if (value) {
        const allKeys: string[] = []
        const collectKeys = (nodes: CategoryNode[]): void => {
          for (const node of nodes) {
            allKeys.push(node.id)
            if (node.children) {
              collectKeys(node.children)
            }
          }
        }
        collectKeys(categoryTree)
        setExpandedKeys(allKeys)
      }
    },
    [categoryTree]
  )

  // Handle create root category
  const handleCreateRoot = useCallback(() => {
    setModalMode('create')
    setEditingCategory(null)
    setParentCategory(null)
    setModalVisible(true)
  }, [])

  // Handle create child category
  const handleCreateChild = useCallback((parent: CategoryNode) => {
    setModalMode('create')
    setEditingCategory(null)
    setParentCategory(parent)
    setModalVisible(true)
  }, [])

  // Handle edit category
  const handleEdit = useCallback(
    (category: CategoryNode) => {
      setModalMode('edit')
      setEditingCategory(category)
      setParentCategory(category.parent_id ? nodeMap.get(category.parent_id) || null : null)
      setModalVisible(true)
    },
    [nodeMap]
  )

  // Handle view detail
  const handleViewDetail = useCallback((category: CategoryNode) => {
    setDetailCategory(category)
    setDetailModalVisible(true)
  }, [])

  // Handle modal submit
  const handleModalSubmit = useCallback(async () => {
    if (!formApiRef.current) return

    try {
      await formApiRef.current.validate()
      const values = formApiRef.current.getValues()
      setModalLoading(true)

      if (modalMode === 'create') {
        const request: HandlerCreateCategoryRequest = {
          code: values.code,
          name: values.name,
          description: values.description || undefined,
          parent_id: parentCategory?.id || undefined,
          sort_order: values.sort_order ? Number(values.sort_order) : 0,
        }
        const response = await api.postCatalogCategories(request)
        if (response.success) {
          Toast.success(t('categories.messages.createSuccess'))
          setModalVisible(false)
          fetchCategoryTree()
        } else {
          Toast.error(response.error?.message || t('categories.messages.createError'))
        }
      } else if (editingCategory) {
        const request: HandlerUpdateCategoryRequest = {
          name: values.name,
          description: values.description || undefined,
          sort_order: values.sort_order ? Number(values.sort_order) : undefined,
        }
        const response = await api.putCatalogCategoriesId(editingCategory.id, request)
        if (response.success) {
          Toast.success(t('categories.messages.updateSuccess'))
          setModalVisible(false)
          fetchCategoryTree()
        } else {
          Toast.error(response.error?.message || t('categories.messages.updateError'))
        }
      }
    } catch {
      // Validation failed
    } finally {
      setModalLoading(false)
    }
  }, [modalMode, editingCategory, parentCategory, api, fetchCategoryTree, t])

  // Handle delete category
  const handleDelete = useCallback(
    async (category: CategoryNode) => {
      if (category.children && category.children.length > 0) {
        Toast.warning(t('categories.messages.hasChildren'))
        return
      }

      Modal.confirm({
        title: t('categories.confirm.deleteTitle'),
        content: t('categories.confirm.deleteContent', { name: category.name }),
        okText: t('categories.confirm.deleteOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.deleteCatalogCategoriesId(category.id)
            Toast.success(t('categories.messages.deleteSuccess', { name: category.name }))
            fetchCategoryTree()
          } catch {
            Toast.error(t('categories.messages.deleteError'))
          }
        },
      })
    },
    [api, fetchCategoryTree, t]
  )

  // Handle activate category
  const handleActivate = useCallback(
    async (category: CategoryNode) => {
      try {
        const response = await api.postCatalogCategoriesIdActivate(category.id)
        if (response.success) {
          Toast.success(t('categories.messages.activateSuccess', { name: category.name }))
          fetchCategoryTree()
        } else {
          Toast.error(response.error?.message || t('categories.messages.activateError'))
        }
      } catch {
        Toast.error(t('categories.messages.activateError'))
      }
    },
    [api, fetchCategoryTree, t]
  )

  // Handle deactivate category
  const handleDeactivate = useCallback(
    async (category: CategoryNode) => {
      Modal.confirm({
        title: t('categories.confirm.deactivateTitle'),
        content: t('categories.confirm.deactivateContent', { name: category.name }),
        okText: t('categories.confirm.deactivateOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'warning' },
        onOk: async () => {
          try {
            const response = await api.postCatalogCategoriesIdDeactivate(category.id)
            if (response.success) {
              Toast.success(t('categories.messages.deactivateSuccess', { name: category.name }))
              fetchCategoryTree()
            } else {
              Toast.error(response.error?.message || t('categories.messages.deactivateError'))
            }
          } catch {
            Toast.error(t('categories.messages.deactivateError'))
          }
        },
      })
    },
    [api, fetchCategoryTree, t]
  )

  // Handle drag and drop
  const handleDrop = useCallback(
    async (info: OnDragProps) => {
      const { dragNode, node, dropPosition } = info
      const dragKey = dragNode.key as string
      const dropKey = node.key as string
      const dragData = nodeMap.get(dragKey)

      if (!dragData) return

      let newParentId: string | undefined

      // dropPosition: -1 = before, 0 = inside, 1 = after
      if (dropPosition === 0) {
        // Drop inside - new parent is drop target
        newParentId = dropKey
      } else {
        // Drop before/after - new parent is same as drop target's parent
        const dropData = nodeMap.get(dropKey)
        newParentId = dropData?.parent_id || undefined
      }

      // Only call API if parent changed
      if (dragData.parent_id !== newParentId) {
        try {
          const response = await api.postCatalogCategoriesIdMove(dragKey, {
            parent_id: newParentId,
          })
          if (response.success) {
            Toast.success(t('categories.messages.moveSuccess'))
            fetchCategoryTree()
          } else {
            Toast.error(response.error?.message || t('categories.messages.moveError'))
          }
        } catch {
          Toast.error(t('categories.messages.moveError'))
        }
      }
    },
    [api, fetchCategoryTree, nodeMap, t]
  )

  // Handle expand change
  const handleExpand = useCallback((keys: string[]) => {
    setExpandedKeys(keys)
  }, [])

  // Render tree node label with actions
  const renderLabel = useCallback(
    (label?: React.ReactNode, data?: TreeNodeData) => {
      if (!data) return label
      const category = data.data as CategoryNode
      if (!category) return label

      const isInactive = category.status === 'inactive'

      return (
        <div className="category-tree-node">
          <div className="category-tree-node-content">
            <span className={`category-name ${isInactive ? 'inactive' : ''}`}>{label}</span>
            <Tag size="small" className="category-code">
              {category.code}
            </Tag>
            {isInactive && (
              <Tag size="small" color="grey">
                {t('categories.deactivated')}
              </Tag>
            )}
          </div>
          <div className="category-tree-node-actions">
            <Button
              icon={<IconEdit />}
              size="small"
              theme="borderless"
              type="tertiary"
              onClick={(e) => {
                e.stopPropagation()
                handleEdit(category)
              }}
            />
            <Dropdown
              trigger="click"
              position="bottomRight"
              render={
                <Dropdown.Menu>
                  <Dropdown.Item onClick={() => handleViewDetail(category)}>
                    {t('categories.actions.viewDetail')}
                  </Dropdown.Item>
                  <Dropdown.Item onClick={() => handleCreateChild(category)}>
                    {t('categories.actions.addChild')}
                  </Dropdown.Item>
                  <Dropdown.Item onClick={() => handleEdit(category)}>
                    {t('categories.actions.edit')}
                  </Dropdown.Item>
                  <Dropdown.Divider />
                  {isInactive ? (
                    <Dropdown.Item onClick={() => handleActivate(category)}>
                      {t('categories.actions.activate')}
                    </Dropdown.Item>
                  ) : (
                    <Dropdown.Item onClick={() => handleDeactivate(category)}>
                      {t('categories.actions.deactivate')}
                    </Dropdown.Item>
                  )}
                  <Dropdown.Item type="danger" onClick={() => handleDelete(category)}>
                    {t('categories.actions.delete')}
                  </Dropdown.Item>
                </Dropdown.Menu>
              }
            >
              <Button
                icon={<IconMore />}
                size="small"
                theme="borderless"
                type="tertiary"
                onClick={(e) => e.stopPropagation()}
              />
            </Dropdown>
          </div>
        </div>
      )
    },
    [
      t,
      handleViewDetail,
      handleCreateChild,
      handleEdit,
      handleActivate,
      handleDeactivate,
      handleDelete,
    ]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchCategoryTree()
  }, [fetchCategoryTree])

  // Expand/collapse all
  const handleExpandAll = useCallback(() => {
    const allKeys: string[] = []
    const collectKeys = (nodes: CategoryNode[]): void => {
      for (const node of nodes) {
        allKeys.push(node.id)
        if (node.children) {
          collectKeys(node.children)
        }
      }
    }
    collectKeys(categoryTree)
    setExpandedKeys(allKeys)
  }, [categoryTree])

  const handleCollapseAll = useCallback(() => {
    setExpandedKeys([])
  }, [])

  // Get modal title
  const getModalTitle = useCallback(() => {
    if (modalMode === 'create') {
      return parentCategory
        ? t('categories.createChildTitle', { name: parentCategory.name })
        : t('categories.createRootTitle')
    }
    return t('categories.editCategory')
  }, [modalMode, parentCategory, t])

  return (
    <Container size="full" className="categories-page">
      <Card className="categories-card">
        <div className="categories-header">
          <div className="categories-header-left">
            <IconTreeTriangleDown size="large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={4} style={{ margin: 0 }}>
              {t('categories.title')}
            </Title>
          </div>
          <Space>
            <Button icon={<IconRefresh />} onClick={handleRefresh}>
              {t('common:actions.refresh')}
            </Button>
            <Button icon={<IconPlus />} type="primary" onClick={handleCreateRoot}>
              {t('categories.addRootCategory')}
            </Button>
          </Space>
        </div>

        <div className="categories-toolbar">
          <Input
            prefix={<IconSearch />}
            placeholder={t('categories.searchPlaceholder')}
            value={searchKeyword}
            onChange={handleSearch}
            showClear
            style={{ width: 280 }}
          />
          <Space>
            <Button size="small" theme="borderless" onClick={handleExpandAll}>
              {t('categories.expandAll')}
            </Button>
            <Button size="small" theme="borderless" onClick={handleCollapseAll}>
              {t('categories.collapseAll')}
            </Button>
          </Space>
        </div>

        <Spin spinning={loading}>
          <div className="categories-tree-container">
            {treeData.length > 0 ? (
              <Tree
                treeData={treeData}
                expandedKeys={expandedKeys}
                onExpand={(expandedKeys) => handleExpand(expandedKeys as string[])}
                renderLabel={renderLabel}
                draggable
                onDrop={handleDrop}
                className="categories-tree"
                blockNode
              />
            ) : (
              <Empty
                image={<IconTreeTriangleDown size="extra-large" />}
                title={searchKeyword ? t('categories.empty.titleSearch') : t('categories.empty.title')}
                description={
                  searchKeyword
                    ? t('categories.empty.descriptionSearch')
                    : t('categories.empty.description')
                }
              >
                {!searchKeyword && (
                  <Button type="primary" icon={<IconPlus />} onClick={handleCreateRoot}>
                    {t('categories.addRootCategory')}
                  </Button>
                )}
              </Empty>
            )}
          </div>
        </Spin>
      </Card>

      {/* Create/Edit Category Modal */}
      <Modal
        title={getModalTitle()}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleModalSubmit}
        confirmLoading={modalLoading}
        width={500}
        okText={modalMode === 'create' ? t('common:actions.create') : t('common:actions.save')}
        cancelText={t('common:actions.cancel')}
      >
        <Form
          getFormApi={(api) => {
            formApiRef.current = api
          }}
          initValues={
            editingCategory
              ? {
                  code: editingCategory.code,
                  name: editingCategory.name,
                  description: editingCategory.description,
                  sort_order: editingCategory.sort_order,
                }
              : { sort_order: 0 }
          }
          labelPosition="left"
          labelWidth={80}
        >
          <Form.Input
            field="code"
            label={t('categories.form.code')}
            placeholder={t('categories.form.codePlaceholder')}
            rules={[
              { required: true, message: t('categories.form.codeRequired') },
              { min: 1, message: t('categories.form.codeMinLength') },
              { max: 50, message: t('categories.form.codeMaxLength') },
            ]}
            disabled={modalMode === 'edit'}
          />
          <Form.Input
            field="name"
            label={t('categories.form.name')}
            placeholder={t('categories.form.namePlaceholder')}
            rules={[
              { required: true, message: t('categories.form.nameRequired') },
              { min: 1, message: t('categories.form.nameMinLength') },
              { max: 100, message: t('categories.form.nameMaxLength') },
            ]}
          />
          <Form.TextArea
            field="description"
            label={t('categories.form.description')}
            placeholder={t('categories.form.descriptionPlaceholder')}
            rows={3}
            maxLength={2000}
          />
          <Form.InputNumber
            field="sort_order"
            label={t('categories.form.sortOrder')}
            placeholder={t('categories.form.sortOrderPlaceholder')}
            min={0}
            max={9999}
            style={{ width: '100%' }}
          />
          {parentCategory && (
            <Form.Slot label={t('categories.form.parentCategory')}>
              <Tag color="blue">{parentCategory.name}</Tag>
            </Form.Slot>
          )}
        </Form>
      </Modal>

      {/* Category Detail Modal */}
      <Modal
        title={t('categories.detail.title')}
        visible={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={
          <Space>
            {detailCategory && (
              <>
                <Button
                  onClick={() => {
                    setDetailModalVisible(false)
                    handleEdit(detailCategory)
                  }}
                >
                  {t('common:actions.edit')}
                </Button>
                <Button
                  type="primary"
                  onClick={() => {
                    setDetailModalVisible(false)
                    handleCreateChild(detailCategory)
                  }}
                >
                  {t('categories.addChildCategory')}
                </Button>
              </>
            )}
            <Button onClick={() => setDetailModalVisible(false)}>{t('common:actions.close')}</Button>
          </Space>
        }
        width={600}
      >
        {detailCategory && (
          <div className="category-detail">
            <Descriptions
              data={[
                { key: t('categories.detail.code'), value: detailCategory.code },
                { key: t('categories.detail.name'), value: detailCategory.name },
                { key: t('categories.detail.description'), value: detailCategory.description || '-' },
                {
                  key: t('categories.detail.status'),
                  value:
                    detailCategory.status === 'active'
                      ? t('categories.status.active')
                      : t('categories.status.inactive'),
                },
                { key: t('categories.detail.level'), value: String(detailCategory.level) },
                { key: t('categories.detail.sortOrder'), value: String(detailCategory.sort_order || 0) },
                {
                  key: t('categories.detail.childCount'),
                  value: String(detailCategory.children?.length || 0),
                },
                {
                  key: t('categories.detail.parentCategory'),
                  value: detailCategory.parent_id
                    ? nodeMap.get(detailCategory.parent_id)?.name || '-'
                    : t('categories.rootCategory'),
                },
              ]}
            />

            {detailCategory.children && detailCategory.children.length > 0 && (
              <div className="category-children-section">
                <Title heading={6}>
                  {t('categories.detail.childCategories', { count: detailCategory.children.length })}
                </Title>
                <div className="category-children-list">
                  {detailCategory.children.map((child) => (
                    <Tag
                      key={child.id}
                      className="category-child-tag"
                      onClick={() => {
                        setDetailCategory(child)
                      }}
                      style={{ cursor: 'pointer' }}
                    >
                      {child.name}
                    </Tag>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </Container>
  )
}
