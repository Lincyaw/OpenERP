import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
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
} from '@douyinfe/semi-ui-19'
import type { TreeNodeData } from '@douyinfe/semi-ui-19/lib/es/tree/interface'
import { IconRefresh, IconSearch, IconInfoCircle } from '@douyinfe/semi-icons'
import { Container } from '@/components/common/layout'
import { listRoles, getRolePermissions } from '@/api/roles/roles'
import type { HandlerRoleResponse, ListRolesParams } from '@/api/models'

// Type aliases for backward compatibility
type Role = HandlerRoleResponse
type RoleListQuery = ListRolesParams
import './Permissions.css'

const { Title, Text } = Typography

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
 * Permission Configuration Page
 *
 * Features:
 * - Display all system permissions in a tree view
 * - Show which roles have each permission
 * - Search/filter permissions
 * - Resource grouping with descriptions
 */
export default function PermissionsPage() {
  const { t, i18n } = useTranslation('system')

  /**
   * Convert permissions to tree data with additional metadata
   */
  const permissionsToTreeData = useCallback(
    (permissions: string[], rolePermissionMap: Map<string, string[]>): TreeNodeData[] => {
      const grouped = groupPermissionsByResource(permissions)
      const treeData: TreeNodeData[] = []

      grouped.forEach((perms, resource) => {
        const resourceLabel = String(t(`permissions.resources.${resource}`)) || resource

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
                {t('permissions.permissionCount', { count: perms.length })}
              </Tag>
              {rolesWithResource.size > 0 && (
                <Tag size="small" color="green" className="role-count-tag">
                  {t('permissions.roleCount', { count: rolesWithResource.size })}
                </Tag>
              )}
            </div>
          ),
          value: resource,
          children: perms.map((perm) => {
            const action = perm.split(':')[1]
            const actionLabel = String(t(`permissions.actions.${action}`)) || action
            const actionDescriptionKey = `permissions.actionDescriptions.${action}`
            const actionDescription =
              t(actionDescriptionKey) !== actionDescriptionKey
                ? String(t(actionDescriptionKey))
                : ''

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
                        <span style={{ display: 'inline-flex' }}>
                          <IconInfoCircle size="small" className="permission-info-icon" />
                        </span>
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
                          <span style={{ display: 'inline-flex' }}>
                            <Tag size="small" color="grey">
                              +{rolesWithPermission.length - 3}
                            </Tag>
                          </span>
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
        const aLabel =
          String(t(`permissions.resources.${a.value as string}`)) || (a.value as string)
        const bLabel =
          String(t(`permissions.resources.${b.value as string}`)) || (b.value as string)
        return aLabel.localeCompare(bLabel, i18n.language)
      })

      return treeData
    },
    [t, i18n.language]
  )

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
      const response = await getRolePermissions()
      if (response.status === 200 && response.data.success && response.data.data) {
        setAllPermissions(response.data.data.permissions || [])
        // Initially expand all resource groups
        const resources = new Set(
          (response.data.data.permissions || []).map((p: string) => p.split(':')[0])
        )
        setExpandedKeys(Array.from(resources))
      }
    } catch {
      Toast.error(t('permissions.messages.fetchPermissionsError'))
    } finally {
      setPermissionsLoading(false)
    }
  }, [t])

  // Fetch all roles with their permissions
  const fetchRoles = useCallback(async () => {
    setRolesLoading(true)
    try {
      const query: RoleListQuery = {
        page: 1,
        page_size: 100, // Backend max is 100
        is_enabled: true,
      }
      const response = await listRoles(query)
      if (response.status === 200 && response.data.success && response.data.data) {
        setRoles(response.data.data.roles || [])
      }
    } catch {
      Toast.error(t('permissions.messages.fetchRolesError'))
    } finally {
      setRolesLoading(false)
    }
  }, [t])

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
      const resourceLabel = String(t(`permissions.resources.${resource}`)) || resource
      const actionLabel = String(t(`permissions.actions.${action}`)) || action
      return (
        perm.toLowerCase().includes(keyword) ||
        resourceLabel.toLowerCase().includes(keyword) ||
        actionLabel.toLowerCase().includes(keyword)
      )
    })
  }, [allPermissions, searchKeyword, t])

  // Tree data for display
  const treeData = useMemo(() => {
    return permissionsToTreeData(filteredPermissions, rolePermissionMap)
  }, [filteredPermissions, rolePermissionMap, permissionsToTreeData])

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
              {t('permissions.title')}
            </Title>
            <Text type="tertiary">{t('permissions.subtitle')}</Text>
          </div>
          <Button icon={<IconRefresh />} onClick={handleRefresh} loading={loading}>
            {t('common.refresh')}
          </Button>
        </div>

        <Banner
          type="info"
          description={t('permissions.readOnlyBanner')}
          style={{ marginBottom: 'var(--spacing-4)' }}
        />

        {/* Statistics */}
        <div className="permissions-stats">
          <div className="stat-item">
            <Text type="tertiary">{t('permissions.stats.resources')}</Text>
            <Text strong>{stats.totalResources}</Text>
          </div>
          <div className="stat-item">
            <Text type="tertiary">{t('permissions.stats.permissions')}</Text>
            <Text strong>{stats.totalPermissions}</Text>
          </div>
          <div className="stat-item">
            <Text type="tertiary">{t('permissions.stats.roles')}</Text>
            <Text strong>{stats.totalRoles}</Text>
          </div>
        </div>

        {/* Search and actions */}
        <div className="permissions-toolbar">
          <Input
            prefix={<IconSearch />}
            placeholder={t('permissions.searchPlaceholder')}
            value={searchKeyword}
            onChange={(value) => setSearchKeyword(value)}
            showClear
            style={{ width: 300 }}
          />
          <Space>
            <Button size="small" onClick={handleExpandAll}>
              {t('permissions.expandAll')}
            </Button>
            <Button size="small" onClick={handleCollapseAll}>
              {t('permissions.collapseAll')}
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
                onExpand={(expandedKeys: string[]) => {
                  setExpandedKeys(expandedKeys)
                }}
                className="permissions-tree"
                blockNode
              />
            </div>
          ) : (
            <Empty
              description={
                searchKeyword
                  ? t('permissions.noMatchingPermissions')
                  : t('permissions.noPermissions')
              }
            />
          )}
        </Spin>

        {/* Role-Permission Summary */}
        <Collapse className="role-summary-collapse">
          <Collapse.Panel header={t('permissions.roleSummary.title')} itemKey="summary">
            <div className="role-summary-list">
              {roles.map((role) => (
                <div key={role.id} className="role-summary-item">
                  <div className="role-summary-header">
                    <Text strong>{role.name}</Text>
                    {role.is_system_role && (
                      <Tag size="small" color="blue">
                        {t('roles.systemRole')}
                      </Tag>
                    )}
                    <Tag size="small" color={role.is_enabled ? 'green' : 'grey'}>
                      {role.is_enabled ? t('roles.enabled') : t('roles.disabled')}
                    </Tag>
                  </div>
                  <div className="role-summary-permissions">
                    <Text type="tertiary" size="small">
                      {t('permissions.roleSummary.permissionCount', {
                        count: role.permissions?.length || 0,
                      })}
                      {role.permissions && role.permissions.length > 0 && (
                        <>
                          {' - '}
                          {Array.from(new Set(role.permissions.map((p) => p.split(':')[0])))
                            .slice(0, 5)
                            .map((r) => t(`permissions.resources.${r}`, r))
                            .join(i18n.language === 'zh-CN' ? '、' : ', ')}
                          {new Set(role.permissions.map((p) => p.split(':')[0])).size > 5 &&
                            (i18n.language === 'zh-CN' ? ' 等' : ' ...')}
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
