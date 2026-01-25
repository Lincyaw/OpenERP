import { useState } from 'react'
import {
  Button,
  Card,
  Typography,
  Space,
  Tag,
  Input,
  Toast,
  Table,
  Avatar,
  Descriptions,
} from '@douyinfe/semi-ui-19'
import { IconHome, IconSearch, IconUser, IconSetting, IconGithubLogo } from '@douyinfe/semi-icons'

const { Title, Text, Paragraph } = Typography

function App() {
  const [searchValue, setSearchValue] = useState('')

  const handleButtonClick = () => {
    Toast.success({
      content: 'Semi Design components are working!',
      duration: 3,
    })
  }

  const columns = [
    {
      title: 'Name',
      dataIndex: 'name',
      render: (text: string, record: { avatar: string }) => (
        <Space>
          <Avatar size="small" src={record.avatar} />
          <Text>{text}</Text>
        </Space>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      render: (status: string) => (
        <Tag color={status === 'Active' ? 'green' : 'grey'}>{status}</Tag>
      ),
    },
    {
      title: 'Role',
      dataIndex: 'role',
    },
  ]

  const data = [
    {
      key: '1',
      name: 'Admin User',
      avatar: '',
      status: 'Active',
      role: 'Administrator',
    },
    {
      key: '2',
      name: 'John Doe',
      avatar: '',
      status: 'Active',
      role: 'Manager',
    },
    {
      key: '3',
      name: 'Jane Smith',
      avatar: '',
      status: 'Inactive',
      role: 'Staff',
    },
  ]

  const descData = [
    { key: 'Framework', value: 'React 19 + TypeScript' },
    { key: 'UI Library', value: 'Semi Design' },
    { key: 'Build Tool', value: 'Vite' },
    { key: 'Status', value: <Tag color="green">Ready</Tag> },
  ]

  return (
    <div style={{ padding: '24px', maxWidth: '1200px', margin: '0 auto' }}>
      <Space vertical align="start" spacing="loose" style={{ width: '100%' }}>
        {/* Header */}
        <Card style={{ width: '100%' }}>
          <Space vertical align="start" spacing="tight">
            <Space>
              <IconHome size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
              <Title heading={2} style={{ margin: 0 }}>
                ERP System
              </Title>
            </Space>
            <Paragraph type="secondary">
              Inventory Management System - Frontend scaffolding with Semi Design
            </Paragraph>
          </Space>
        </Card>

        {/* Component Demo Section */}
        <Card
          title="Component Verification"
          style={{ width: '100%' }}
          headerExtraContent={
            <Tag color="blue" prefixIcon={<IconGithubLogo />}>
              v1.0.0
            </Tag>
          }
        >
          <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
            {/* Buttons */}
            <Space>
              <Button type="primary" icon={<IconHome />} onClick={handleButtonClick}>
                Primary Button
              </Button>
              <Button type="secondary" icon={<IconUser />}>
                Secondary
              </Button>
              <Button type="tertiary" icon={<IconSetting />}>
                Tertiary
              </Button>
              <Button type="danger">Danger</Button>
            </Space>

            {/* Input */}
            <Input
              prefix={<IconSearch />}
              placeholder="Search..."
              value={searchValue}
              onChange={(value) => setSearchValue(value)}
              style={{ width: 300 }}
              showClear
            />

            {/* Tags */}
            <Space>
              <Tag color="blue">Blue Tag</Tag>
              <Tag color="green">Green Tag</Tag>
              <Tag color="orange">Orange Tag</Tag>
              <Tag color="red">Red Tag</Tag>
              <Tag color="purple">Purple Tag</Tag>
            </Space>
          </Space>
        </Card>

        {/* Project Info */}
        <Card title="Project Information" style={{ width: '100%' }}>
          <Descriptions data={descData} />
        </Card>

        {/* Sample Table */}
        <Card title="Sample Data Table" style={{ width: '100%' }}>
          <Table columns={columns} dataSource={data} pagination={false} size="small" />
        </Card>

        {/* Footer */}
        <Card style={{ width: '100%' }}>
          <Space style={{ width: '100%', justifyContent: 'center' }}>
            <Text type="tertiary">Semi Design UI components integrated successfully</Text>
          </Space>
        </Card>
      </Space>
    </div>
  )
}

export default App
