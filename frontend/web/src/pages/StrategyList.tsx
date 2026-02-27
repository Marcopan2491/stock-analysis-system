import { useState, useEffect } from 'react'
import { Card, Table, Button, Tag, Modal, Form, Input, Select } from 'antd'
import { getStrategies, createStrategy } from '../api/market'

interface Strategy {
  id: number
  name: string
  type: string
  description: string
  is_active: boolean
  created_at: string
}

function StrategyList() {
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    fetchStrategies()
  }, [])

  const fetchStrategies = async () => {
    setLoading(true)
    try {
      const res = await getStrategies()
      if (res.code === 0) {
        setStrategies(res.data.list || [])
      }
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: any) => {
    const res = await createStrategy(values)
    if (res.code === 0) {
      setModalVisible(false)
      form.resetFields()
      fetchStrategies()
    }
  }

  const columns = [
    { title: '策略名称', dataIndex: 'name', key: 'name' },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => {
        const typeMap: Record<string, string> = {
          trend_following: '趋势跟踪',
          mean_reversion: '均值回归',
          multi_factor: '多因子',
        }
        return <Tag>{typeMap[type] || type}</Tag>
      },
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'red'}>
          {active ? '运行中' : '已停止'}
        </Tag>
      ),
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  ]

  return (
    <Card
      title="策略管理"
      extra={
        <Button type="primary" onClick={() => setModalVisible(true)}>
          新建策略
        </Button>
      }
    >
      <Table
        columns={columns}
        dataSource={strategies}
        loading={loading}
        rowKey="id"
      />
      
      <Modal
        title="新建策略"
        open={modalVisible}
        onOk={() => form.submit()}
        onCancel={() => setModalVisible(false)}
      >
        <Form form={form} onFinish={handleCreate} layout="vertical">
          <Form.Item name="name" label="策略名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="策略类型" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="trend_following">趋势跟踪</Select.Option>
              <Select.Option value="mean_reversion">均值回归</Select.Option>
              <Select.Option value="multi_factor">多因子</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="class_name" label="类名" rules={[{ required: true }]}>
            <Input placeholder="如: DualMAStrategy" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}

export default StrategyList
