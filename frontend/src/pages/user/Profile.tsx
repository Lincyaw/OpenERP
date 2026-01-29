import { useState, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Form,
  Button,
  Toast,
  Avatar,
  Descriptions,
  Tag,
  Divider,
  Modal,
} from '@douyinfe/semi-ui-19'
import type { FormApi } from '@douyinfe/semi-ui-19/lib/es/form/interface'
import { IconEdit, IconKey, IconUser } from '@douyinfe/semi-icons'

import { Container } from '@/components/common/layout'
import { useUser, useAuthStore } from '@/store'
import { updateUser } from '@/api/users/users'
import { changePasswordAuth } from '@/api/auth/auth'
import type { UpdateUserBody, ChangePasswordAuthBody } from '@/api/models'

import './Profile.css'

const { Title, Text } = Typography

/**
 * User Profile page
 *
 * Features:
 * - View user information
 * - Edit profile (display name, email, phone)
 * - Change password
 */
export default function ProfilePage() {
  const { t } = useTranslation('system')
  const user = useUser()
  const updateUserInStore = useAuthStore((state) => state.updateUser)

  // Edit profile modal state
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [editLoading, setEditLoading] = useState(false)
  const editFormRef = useRef<FormApi | null>(null)

  // Change password modal state
  const [passwordModalVisible, setPasswordModalVisible] = useState(false)
  const [passwordLoading, setPasswordLoading] = useState(false)
  const passwordFormRef = useRef<FormApi | null>(null)

  // Handle edit profile
  const handleEditProfile = useCallback(() => {
    setEditModalVisible(true)
  }, [])

  // Handle edit profile submit
  const handleEditSubmit = useCallback(async () => {
    if (!editFormRef.current || !user?.id) return

    try {
      await editFormRef.current.validate()
      const values = editFormRef.current.getValues()
      setEditLoading(true)

      const request: UpdateUserBody = {
        display_name: values.display_name || undefined,
        email: values.email || undefined,
        phone: values.phone || undefined,
      }

      const response = await updateUser(user.id, request)
      if (response.status === 200 && response.data.success) {
        Toast.success(t('profile.messages.updateSuccess'))
        // Update user in store
        updateUserInStore({
          displayName: values.display_name,
          email: values.email,
        })
        setEditModalVisible(false)
      } else {
        Toast.error(
          (response.data.error as { message?: string })?.message ||
            t('profile.messages.updateError')
        )
      }
    } catch {
      // Validation failed
    } finally {
      setEditLoading(false)
    }
  }, [user?.id, updateUserInStore, t])

  // Handle change password
  const handleChangePassword = useCallback(() => {
    setPasswordModalVisible(true)
  }, [])

  // Handle change password submit
  const handlePasswordSubmit = useCallback(async () => {
    if (!passwordFormRef.current) return

    try {
      await passwordFormRef.current.validate()
      const values = passwordFormRef.current.getValues()
      setPasswordLoading(true)

      const request: ChangePasswordAuthBody = {
        old_password: values.old_password,
        new_password: values.new_password,
      }

      const response = await changePasswordAuth(request)
      if (response.status === 200 && response.data.success) {
        Toast.success(t('profile.messages.passwordSuccess'))
        setPasswordModalVisible(false)
        passwordFormRef.current?.reset()
      } else {
        Toast.error(
          (response.data as { error?: { message?: string } }).error?.message ||
            t('profile.messages.passwordError')
        )
      }
    } catch {
      // Validation failed
    } finally {
      setPasswordLoading(false)
    }
  }, [t])

  return (
    <Container size="md" className="profile-page">
      {/* Profile Header */}
      <Card className="profile-header-card">
        <div className="profile-header">
          <Avatar size="large" color="light-blue" className="profile-avatar">
            {(user?.displayName || user?.username || 'U').charAt(0).toUpperCase()}
          </Avatar>
          <div className="profile-header-info">
            <Title heading={4} style={{ margin: 0 }}>
              {user?.displayName || user?.username}
            </Title>
            <Text type="tertiary">@{user?.username}</Text>
          </div>
          <div className="profile-header-actions">
            <Button icon={<IconEdit />} onClick={handleEditProfile}>
              {t('profile.editProfile')}
            </Button>
            <Button icon={<IconKey />} onClick={handleChangePassword}>
              {t('profile.changePassword')}
            </Button>
          </div>
        </div>
      </Card>

      {/* Profile Information */}
      <Card className="profile-info-card" title={t('profile.basicInfo')}>
        <Descriptions align="left">
          <Descriptions.Item itemKey={t('profile.fields.username')}>
            <span className="profile-username">{user?.username}</span>
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('profile.fields.displayName')}>
            {user?.displayName || '-'}
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('profile.fields.email')}>
            {user?.email || '-'}
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('profile.fields.roles')}>
            {user?.roles && user.roles.length > 0 ? (
              <div className="profile-roles">
                {user.roles.map((role) => (
                  <Tag key={role} color="blue">
                    {role}
                  </Tag>
                ))}
              </div>
            ) : (
              '-'
            )}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Security Information */}
      <Card className="profile-security-card" title={t('profile.securityInfo')}>
        <Descriptions align="left">
          <Descriptions.Item itemKey={t('profile.fields.userId')}>
            <Text copyable className="profile-user-id">
              {user?.id}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('profile.fields.tenantId')}>
            <Text copyable className="profile-tenant-id">
              {user?.tenantId || '-'}
            </Text>
          </Descriptions.Item>
        </Descriptions>

        <Divider margin={16} />

        <div className="profile-security-tip">
          <IconUser className="profile-security-icon" />
          <Text type="tertiary">{t('profile.securityTip')}</Text>
        </div>
      </Card>

      {/* Edit Profile Modal */}
      <Modal
        title={t('profile.editProfile')}
        visible={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        onOk={handleEditSubmit}
        confirmLoading={editLoading}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        width={500}
      >
        <Form
          getFormApi={(api) => {
            editFormRef.current = api
          }}
          initValues={{
            display_name: user?.displayName,
            email: user?.email,
          }}
          labelPosition="left"
          labelWidth={100}
        >
          <Form.Input
            field="display_name"
            label={t('profile.form.displayName')}
            placeholder={t('profile.form.displayNamePlaceholder')}
          />
          <Form.Input
            field="email"
            label={t('profile.form.email')}
            placeholder={t('profile.form.emailPlaceholder')}
            rules={[{ type: 'email', message: t('profile.form.emailError') }]}
          />
          <Form.Input
            field="phone"
            label={t('profile.form.phone')}
            placeholder={t('profile.form.phonePlaceholder')}
          />
        </Form>
      </Modal>

      {/* Change Password Modal */}
      <Modal
        title={t('profile.changePassword')}
        visible={passwordModalVisible}
        onCancel={() => {
          setPasswordModalVisible(false)
          passwordFormRef.current?.reset()
        }}
        onOk={handlePasswordSubmit}
        confirmLoading={passwordLoading}
        okText={t('profile.form.changePasswordBtn')}
        cancelText={t('common.cancel')}
        width={500}
      >
        <Form
          getFormApi={(api) => {
            passwordFormRef.current = api
          }}
          labelPosition="left"
          labelWidth={100}
        >
          <Form.Input
            field="old_password"
            label={t('profile.form.oldPassword')}
            mode="password"
            placeholder={t('profile.form.oldPasswordPlaceholder')}
            rules={[{ required: true, message: t('profile.form.oldPasswordRequired') }]}
          />
          <Form.Input
            field="new_password"
            label={t('profile.form.newPassword')}
            mode="password"
            placeholder={t('profile.form.newPasswordPlaceholder')}
            rules={[
              { required: true, message: t('profile.form.newPasswordRequired') },
              { min: 8, message: t('profile.form.newPasswordMinLength') },
            ]}
          />
          <Form.Input
            field="confirm_password"
            label={t('profile.form.confirmPassword')}
            mode="password"
            placeholder={t('profile.form.confirmPasswordPlaceholder')}
            rules={[
              { required: true, message: t('profile.form.confirmPasswordRequired') },
              {
                validator: (_rule: unknown, value: string, callback: (error?: string) => void) => {
                  const form = passwordFormRef.current
                  if (form && value !== form.getValue('new_password')) {
                    callback(t('profile.form.confirmPasswordError'))
                  } else {
                    callback()
                  }
                  return true
                },
              },
            ]}
          />
        </Form>
        <div className="password-tips">
          <Text type="tertiary">{t('profile.passwordTips')}</Text>
        </div>
      </Modal>
    </Container>
  )
}
