import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Spin,
  Tree,
  Empty,
  Collapse,
  Input,
  Space,
  Button,
  Tooltip,
  Banner,
} from '@douyinfe/semi-ui'
import type { TreeNodeData } from '@douyinfe/semi-ui/lib/es/tree/interface'
import { IconRefresh, IconSearch, IconInfoCircle } from '@douyinfe/semi-icons'
import { Container } from '@/components/common/layout'
import { getIdentity } from '@/api/identity'
import type { Role, RoleListQuery } from '@/api/identity'
import './Permissions.css'

const { Title, Text } = Typography

/**
 * Resource name labels (Chinese)
 */
const RESOURCE_LABELS: Record<string, string> = {
  product: '商品',
  category: '分类',
  customer: '客户',
  supplier: '供应商',
  warehouse: '仓库',
  inventory: '库存',
  sales_order: '销售订单',
  purchase_order: '采购订单',
  sales_return: '销售退货',
  purchase_return: '采购退货',
  account_receivable: '应收账款',
  account_payable: '应付账款',
  receipt: '收款单',
  payment: '付款单',
  expense: '费用',
  income: '其他收入',
  report: '报表',
  user: '用户',
  role: '角色',
  tenant: '租户',
}

/**
 * Action name labels (Chinese)
 */
const ACTION_LABELS: Record<string, string> = {
  create: '创建',
  read: '查看',
  update: '修改',
  delete: '删除',
  enable: '启用',
  disable: '停用',
  confirm: '确认',
  cancel: '取消',
  ship: '发货',
  receive: '收货',
  approve: '审批通过',
  reject: '审批拒绝',
  adjust: '调整',
  lock: '锁定',
  unlock: '解锁',
  reconcile: '核销',
  export: '导出',
  import: '导入',
  assign_role: '分配角色',
  view_all: '查看全部',
}

/**
 * Action descriptions
 */
const ACTION_DESCRIPTIONS: Record<string, string> = {
  create: '允许创建新记录',
  read: '允许查看数据',
  update: '允许修改现有记录',
  delete: '允许删除记录',
  enable: '允许启用已禁用的记录',
  disable: '允许禁用记录',
  confirm: '允许确认订单或操作',
  cancel: '允许取消订单或操作',
  ship: '允许执行发货操作',
  receive: '允许执行收货操作',
  approve: '允许审批通过申请',
  reject: '允许拒绝申请',
  adjust: '允许进行数量调整',
  lock: '允许锁定资源',
  unlock: '允许解锁资源',
  reconcile: '允许执行核销操作',
  export: '允许导出数据',
  import: '允许导入数据',
  assign_role: '允许给用户分配角色',
  view_all: '允许查看所有数据（跨部门/跨人员）',
}

/**
 * Group permissions by resource
 */
function groupPermissionsByResource(permissions: string[]): Map<string, string[]> {
  const grouped = new Map<string, string[]>()

  for (const permission of permissions) {
    const parts = permission.split(':')
    if (parts.length === 2) {
      const [resource] = parts
      if (!grouped.has(resource)) {
        grouped.set(resource, [])
      }
      grouped.get(resource)!.push(permission)
    }
  }

  return grouped
}

/**
 * Convert permissions to tree data with additional metadata
 */
function permissionsToTreeData(
  permissions: string[],
  rolePermissionMap: Map<string, string[]>
): TreeNodeData[] {
  const grouped = groupPermissionsByResource(permissions)
  const treeData: TreeNodeData[] = []

  grouped.forEach((perms, resource) => {
    const resourceLabel = RESOURCE_LABELS[resource] || resource

    // Count roles that have any permission in this resource
    const rolesWithResource = new Set<string>()
    perms.forEach((perm) => {
      rolePermissionMap.forEach((rolePerms, roleName) => {
        if (rolePerms.includes(perm)) {
          rolesWithResource.add(roleName)
        }
      })
    })

    treeData.push({
      key: resource,
      label: (
        <div className="permission-resource-node">
          <span className="permission-resource-name">{resourceLabel}</span>
          <Tag size="small" color="blue" className="permission-count-tag">
            {perms.length} 项权限
          </Tag>
          {rolesWithResource.size > 0 && (
            <Tag size="small" color="green" className="role-count-tag">
              {rolesWithResource.size} 个角色
            </Tag>
          )}
        </div>
      ),
      value: resource,
      children: perms.map((perm) => {
        const action = perm.split(':')[1]
        const actionLabel = ACTION_LABELS[action] || action
        const actionDescription = ACTION_DESCRIPTIONS[action] || ''

        // Find roles that have this permission
        const rolesWithPermission: string[] = []
        rolePermissionMap.forEach((rolePerms, roleName) => {
          if (rolePerms.includes(perm)) {
            rolesWithPermission.push(roleName)
          }
        })

        return {
          key: perm,
          label: (
            <div className="permission-action-node">
              <div className="permission-action-info">
                <span className="permission-action-name">{actionLabel}</span>
                <Text type="tertiary" size="small" className="permission-code">
                  {perm}
                </Text>
                {actionDescription && (
                  <Tooltip content={actionDescription}>
                    <IconInfoCircle size="small" className="permission-info-icon" />
                  </Tooltip>
                )}
              </div>
              {rolesWithPermission.length > 0 && (
                <div className="permission-roles">
                  {rolesWithPermission.slice(0, 3).map((role) => (
                    <Tag key={role} size="small" color="light-green">
                      {role}
                    </Tag>
                  ))}
                  {rolesWithPermission.length > 3 && (
                    <Tooltip content={rolesWithPermission.slice(3).join(', ')}>
                      <Tag size="small" color="grey">
                        +{rolesWithPermission.length - 3}
                      </Tag>
                    </Tooltip>
                  )}
                </div>
              )}
            </div>
          ),
          value: perm,
        }
      }),
    })
  })

  // Sort by resource name
  treeData.sort((a, b) => {
    const aLabel = RESOURCE_LABELS[a.value as string] || (a.value as string)
    const bLabel = RESOURCE_LABELS[b.value as string] || (b.value as string)
    return aLabel.localeCompare(bLabel, 'zh-CN')
  })

  return treeData
}

/**
 * Permission Configuration Page
 *
 * Features:
 * - Display all system permissions in a tree view
 * - Show which roles have each permission
 * - Search/filter permissions
 * - Resource grouping with descriptions
 */
export default function PermissionsPage() {
  const api = useMemo(() => getIdentity(), [])

  // State for permissions
  const [allPermissions, setAllPermissions] = useState<string[]>([])
  const [permissionsLoading, setPermissionsLoading] = useState(false)

  // State for roles
  const [roles, setRoles] = useState<Role[]>([])
  const [rolesLoading, setRolesLoading] = useState(false)

  // Search state
  const [searchKeyword, setSearchKeyword] = useState('')

  // Expanded keys
  const [expandedKeys, setExpandedKeys] = useState<string[]>([])

  // Fetch all permissions
  const fetchPermissions = useCallback(async () => {
    setPermissionsLoading(true)
    try {
      const response = await api.getAllPermissions()
      if (response.success && response.data) {
        setAllPermissions(response.data.permissions || [])
        // Initially expand all resource groups
        const resources = new Set(
          (response.data.permissions || []).map((p) => p.split(':')[0])
        )
        setExpandedKeys(Array.from(resources))
      }
    } catch {
      Toast.error('获取权限列表失败')
    } finally {
      setPermissionsLoading(false)
    }
  }, [api])

  // Fetch all roles with their permissions
  const fetchRoles = useCallback(async () => {
    setRolesLoading(true)
    try {
      const query: RoleListQuery = {
        page: 1,
        page_size: 1000, // Get all roles
        is_enabled: true,
      }
      const response = await api.listRoles(query)
      if (response.success && response.data) {
        setRoles(response.data.roles || [])
      }
    } catch {
      Toast.error('获取角色列表失败')
    } finally {
      setRolesLoading(false)
    }
  }, [api])

  // Fetch data on mount
  useEffect(() => {
    fetchPermissions()
    fetchRoles()
  }, [fetchPermissions, fetchRoles])

  // Build role -> permissions map
  const rolePermissionMap = useMemo(() => {
    const map = new Map<string, string[]>()
    roles.forEach((role) => {
      map.set(role.name, role.permissions || [])
    })
    return map
  }, [roles])

  // Filter permissions by search keyword
  const filteredPermissions = useMemo(() => {
    if (!searchKeyword.trim()) {
      return allPermissions
    }
    const keyword = searchKeyword.toLowerCase()
    return allPermissions.filter((perm) => {
      const [resource, action] = perm.split(':')
      const resourceLabel = RESOURCE_LABELS[resource] || resource
      const actionLabel = ACTION_LABELS[action] || action
      return (
        perm.toLowerCase().includes(keyword) ||
        resourceLabel.toLowerCase().includes(keyword) ||
        actionLabel.toLowerCase().includes(keyword)
      )
    })
  }, [allPermissions, searchKeyword])

  // Tree data for display
  const treeData = useMemo(() => {
    return permissionsToTreeData(filteredPermissions, rolePermissionMap)
  }, [filteredPermissions, rolePermissionMap])

  // Statistics
  const stats = useMemo(() => {
    const resources = groupPermissionsByResource(allPermissions)
    return {
      totalPermissions: allPermissions.length,
      totalResources: resources.size,
      totalRoles: roles.length,
    }
  }, [allPermissions, roles])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchPermissions()
    fetchRoles()
  }, [fetchPermissions, fetchRoles])

  // Handle expand all
  const handleExpandAll = useCallback(() => {
    const resources = new Set(allPermissions.map((p) => p.split(':')[0]))
    setExpandedKeys(Array.from(resources))
  }, [allPermissions])

  // Handle collapse all
  const handleCollapseAll = useCallback(() => {
    setExpandedKeys([])
  }, [])

  const loading = permissionsLoading || rolesLoading

  return (
    <Container size="full" className="permissions-page">
      <Card className="permissions-card">
        <div className="permissions-header">
          <div className="permissions-title-section">
            <Title heading={4} style={{ margin: 0 }}>
              权限配置
            </Title>
            <Text type="tertiary">
              查看系统所有权限及其分配情况
            </Text>
          </div>
          <Button
            icon={<IconRefresh />}
            onClick={handleRefresh}
            loading={loading}
          >
            刷新
          </Button>
        </div>

        <Banner
          type="info"
          description="权限配置为只读视图。如需修改角色权限，请前往「角色管理」页面的「配置权限」功能。"
          style={{ marginBottom: 'var(--spacing-4)' }}
        />

        {/* Statistics */}
        <div className="permissions-stats">
          <div className="stat-item">
            <Text type="tertiary">资源模块</Text>
            <Text strong>{stats.totalResources}</Text>
          </div>
          <div className="stat-item">
            <Text type="tertiary">权限总数</Text>
            <Text strong>{stats.totalPermissions}</Text>
          </div>
          <div className="stat-item">
            <Text type="tertiary">已启用角色</Text>
            <Text strong>{stats.totalRoles}</Text>
          </div>
        </div>

        {/* Search and actions */}
        <div className="permissions-toolbar">
          <Input
            prefix={<IconSearch />}
            placeholder="搜索权限..."
            value={searchKeyword}
            onChange={(value) => setSearchKeyword(value)}
            showClear
            style={{ width: 300 }}
          />
          <Space>
            <Button size="small" onClick={handleExpandAll}>
              展开全部
            </Button>
            <Button size="small" onClick={handleCollapseAll}>
              收起全部
            </Button>
          </Space>
        </div>

        {/* Permissions tree */}
        <Spin spinning={loading}>
          {treeData.length > 0 ? (
            <div className="permissions-tree-container">
              <Tree
                treeData={treeData}
                expandedKeys={expandedKeys}
                onExpand={(
                  expandedKeys: string[],
                ) => {
                  setExpandedKeys(expandedKeys)
                }}
                className="permissions-tree"
                blockNode
              />
            </div>
          ) : (
            <Empty
              description={searchKeyword ? '未找到匹配的权限' : '暂无权限数据'}
            />
          )}
        </Spin>

        {/* Role-Permission Summary */}
        <Collapse className="role-summary-collapse">
          <Collapse.Panel header="角色权限摘要" itemKey="summary">
            <div className="role-summary-list">
              {roles.map((role) => (
                <div key={role.id} className="role-summary-item">
                  <div className="role-summary-header">
                    <Text strong>{role.name}</Text>
                    {role.is_system_role && (
                      <Tag size="small" color="blue">
                        系统角色
                      </Tag>
                    )}
                    <Tag size="small" color={role.is_enabled ? 'green' : 'grey'}>
                      {role.is_enabled ? '已启用' : '已禁用'}
                    </Tag>
                  </div>
                  <div className="role-summary-permissions">
                    <Text type="tertiary" size="small">
                      {role.permissions?.length || 0} 项权限
                      {role.permissions && role.permissions.length > 0 && (
                        <>
                          {' - '}
                          {Array.from(
                            new Set(role.permissions.map((p) => p.split(':')[0]))
                          )
                            .slice(0, 5)
                            .map((r) => RESOURCE_LABELS[r] || r)
                            .join('、')}
                          {new Set(role.permissions.map((p) => p.split(':')[0])).size > 5 && ' 等'}
                        </>
                      )}
                    </Text>
                  </div>
                </div>
              ))}
            </div>
          </Collapse.Panel>
        </Collapse>
      </Card>
    </Container>
  )
}
