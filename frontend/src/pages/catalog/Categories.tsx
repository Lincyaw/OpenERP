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
      Toast.error('获取分类树失败')
    } finally {
      setLoading(false)
    }
  }, [api])

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
          Toast.success('分类创建成功')
          setModalVisible(false)
          fetchCategoryTree()
        } else {
          Toast.error(response.error?.message || '创建分类失败')
        }
      } else if (editingCategory) {
        const request: HandlerUpdateCategoryRequest = {
          name: values.name,
          description: values.description || undefined,
          sort_order: values.sort_order ? Number(values.sort_order) : undefined,
        }
        const response = await api.putCatalogCategoriesId(editingCategory.id, request)
        if (response.success) {
          Toast.success('分类更新成功')
          setModalVisible(false)
          fetchCategoryTree()
        } else {
          Toast.error(response.error?.message || '更新分类失败')
        }
      }
    } catch {
      // Validation failed
    } finally {
      setModalLoading(false)
    }
  }, [modalMode, editingCategory, parentCategory, api, fetchCategoryTree])

  // Handle delete category
  const handleDelete = useCallback(
    async (category: CategoryNode) => {
      if (category.children && category.children.length > 0) {
        Toast.warning('该分类有子分类，请先删除子分类')
        return
      }

      Modal.confirm({
        title: '确认删除',
        content: `确定要删除分类 "${category.name}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.deleteCatalogCategoriesId(category.id)
            Toast.success(`分类 "${category.name}" 已删除`)
            fetchCategoryTree()
          } catch {
            Toast.error('删除分类失败')
          }
        },
      })
    },
    [api, fetchCategoryTree]
  )

  // Handle activate category
  const handleActivate = useCallback(
    async (category: CategoryNode) => {
      try {
        const response = await api.postCatalogCategoriesIdActivate(category.id)
        if (response.success) {
          Toast.success(`分类 "${category.name}" 已启用`)
          fetchCategoryTree()
        } else {
          Toast.error(response.error?.message || '启用分类失败')
        }
      } catch {
        Toast.error('启用分类失败')
      }
    },
    [api, fetchCategoryTree]
  )

  // Handle deactivate category
  const handleDeactivate = useCallback(
    async (category: CategoryNode) => {
      Modal.confirm({
        title: '确认停用',
        content: `确定要停用分类 "${category.name}" 吗？停用后该分类下的商品将无法正常展示。`,
        okText: '确认停用',
        cancelText: '取消',
        okButtonProps: { type: 'warning' },
        onOk: async () => {
          try {
            const response = await api.postCatalogCategoriesIdDeactivate(category.id)
            if (response.success) {
              Toast.success(`分类 "${category.name}" 已停用`)
              fetchCategoryTree()
            } else {
              Toast.error(response.error?.message || '停用分类失败')
            }
          } catch {
            Toast.error('停用分类失败')
          }
        },
      })
    },
    [api, fetchCategoryTree]
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
            Toast.success('分类移动成功')
            fetchCategoryTree()
          } else {
            Toast.error(response.error?.message || '移动分类失败')
          }
        } catch {
          Toast.error('移动分类失败')
        }
      }
    },
    [api, fetchCategoryTree, nodeMap]
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
                已停用
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
                  <Dropdown.Item onClick={() => handleViewDetail(category)}>查看详情</Dropdown.Item>
                  <Dropdown.Item onClick={() => handleCreateChild(category)}>
                    添加子分类
                  </Dropdown.Item>
                  <Dropdown.Item onClick={() => handleEdit(category)}>编辑</Dropdown.Item>
                  <Dropdown.Divider />
                  {isInactive ? (
                    <Dropdown.Item onClick={() => handleActivate(category)}>启用</Dropdown.Item>
                  ) : (
                    <Dropdown.Item onClick={() => handleDeactivate(category)}>停用</Dropdown.Item>
                  )}
                  <Dropdown.Item type="danger" onClick={() => handleDelete(category)}>
                    删除
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

  return (
    <Container size="full" className="categories-page">
      <Card className="categories-card">
        <div className="categories-header">
          <div className="categories-header-left">
            <IconTreeTriangleDown size="large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={4} style={{ margin: 0 }}>
              商品分类
            </Title>
          </div>
          <Space>
            <Button icon={<IconRefresh />} onClick={handleRefresh}>
              刷新
            </Button>
            <Button icon={<IconPlus />} type="primary" onClick={handleCreateRoot}>
              新增根分类
            </Button>
          </Space>
        </div>

        <div className="categories-toolbar">
          <Input
            prefix={<IconSearch />}
            placeholder="搜索分类名称或编码..."
            value={searchKeyword}
            onChange={handleSearch}
            showClear
            style={{ width: 280 }}
          />
          <Space>
            <Button size="small" theme="borderless" onClick={handleExpandAll}>
              全部展开
            </Button>
            <Button size="small" theme="borderless" onClick={handleCollapseAll}>
              全部收起
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
                title={searchKeyword ? '未找到匹配的分类' : '暂无分类数据'}
                description={
                  searchKeyword ? '请尝试其他搜索关键词' : '点击"新增根分类"按钮创建第一个分类'
                }
              >
                {!searchKeyword && (
                  <Button type="primary" icon={<IconPlus />} onClick={handleCreateRoot}>
                    新增根分类
                  </Button>
                )}
              </Empty>
            )}
          </div>
        </Spin>
      </Card>

      {/* Create/Edit Category Modal */}
      <Modal
        title={
          modalMode === 'create'
            ? parentCategory
              ? `新增子分类 - ${parentCategory.name}`
              : '新增根分类'
            : '编辑分类'
        }
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleModalSubmit}
        confirmLoading={modalLoading}
        width={500}
        okText={modalMode === 'create' ? '创建' : '保存'}
        cancelText="取消"
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
            label="分类编码"
            placeholder="请输入分类编码，如 electronics、clothing"
            rules={[
              { required: true, message: '请输入分类编码' },
              { min: 1, message: '分类编码至少1个字符' },
              { max: 50, message: '分类编码最多50个字符' },
            ]}
            disabled={modalMode === 'edit'}
          />
          <Form.Input
            field="name"
            label="分类名称"
            placeholder="请输入分类名称"
            rules={[
              { required: true, message: '请输入分类名称' },
              { min: 1, message: '分类名称至少1个字符' },
              { max: 100, message: '分类名称最多100个字符' },
            ]}
          />
          <Form.TextArea
            field="description"
            label="描述"
            placeholder="请输入分类描述"
            rows={3}
            maxLength={2000}
          />
          <Form.InputNumber
            field="sort_order"
            label="排序值"
            placeholder="数值越小越靠前"
            min={0}
            max={9999}
            style={{ width: '100%' }}
          />
          {parentCategory && (
            <Form.Slot label="父分类">
              <Tag color="blue">{parentCategory.name}</Tag>
            </Form.Slot>
          )}
        </Form>
      </Modal>

      {/* Category Detail Modal */}
      <Modal
        title="分类详情"
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
                  编辑
                </Button>
                <Button
                  type="primary"
                  onClick={() => {
                    setDetailModalVisible(false)
                    handleCreateChild(detailCategory)
                  }}
                >
                  添加子分类
                </Button>
              </>
            )}
            <Button onClick={() => setDetailModalVisible(false)}>关闭</Button>
          </Space>
        }
        width={600}
      >
        {detailCategory && (
          <div className="category-detail">
            <Descriptions
              data={[
                { key: '分类编码', value: detailCategory.code },
                { key: '分类名称', value: detailCategory.name },
                { key: '描述', value: detailCategory.description || '-' },
                {
                  key: '状态',
                  value: detailCategory.status === 'active' ? '已启用' : '已停用',
                },
                { key: '层级', value: String(detailCategory.level) },
                { key: '排序值', value: String(detailCategory.sort_order || 0) },
                {
                  key: '子分类数',
                  value: String(detailCategory.children?.length || 0),
                },
                {
                  key: '父分类',
                  value: detailCategory.parent_id
                    ? nodeMap.get(detailCategory.parent_id)?.name || '-'
                    : '(根分类)',
                },
              ]}
            />

            {detailCategory.children && detailCategory.children.length > 0 && (
              <div className="category-children-section">
                <Title heading={6}>子分类 ({detailCategory.children.length})</Title>
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
