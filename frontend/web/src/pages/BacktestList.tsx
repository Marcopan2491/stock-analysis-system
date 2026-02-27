import { useState, useEffect } from 'react'
import { Card, Table, Button, Tag, Progress } from 'antd'
import { getBacktests, runBacktest } from '../api/market'

interface Backtest {
  id: number
  strategy_id: number
  start_date: string
  end_date: string
  initial_capital: number
  final_capital: number
  total_return: number
  max_drawdown: number
  sharpe_ratio: number
  status: string
  created_at: string
}

function BacktestList() {
  const [backtests, setBacktests] = useState<Backtest[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    fetchBacktests()
  }, [])

  const fetchBacktests = async () => {
    setLoading(true)
    try {
      const res = await getBacktests()
      if (res.code === 0) {
        setBacktests(res.data.list || [])
      }
    } finally {
      setLoading(false)
    }
  }

  const handleRunBacktest = async () => {
    // 简化示例，实际应从策略列表选择
    await runBacktest({
      strategy_id: 1,
      start_date: '2024-01-01',
      end_date: '2024-12-31',
      initial_capital: 100000,
    })
    fetchBacktests()
  }

  const columns = [
    { title: '策略ID', dataIndex: 'strategy_id', key: 'strategy_id' },
    { title: '回测区间', key: 'range', render: (r: Backtest) => `${r.start_date} ~ ${r.end_date}` },
    { title: '初始资金', dataIndex: 'initial_capital', key: 'initial_capital' },
    { title: '最终资金', dataIndex: 'final_capital', key: 'final_capital' },
    {
      title: '总收益',
      dataIndex: 'total_return',
      key: 'total_return',
      render: (v: number) => <span style={{ color: v >= 0 ? 'red' : 'green' }}>{(v * 100).toFixed(2)}%</span>,
    },
    {
      title: '最大回撤',
      dataIndex: 'max_drawdown',
      key: 'max_drawdown',
      render: (v: number) => `${(v * 100).toFixed(2)}%`,
    },
    { title: '夏普比率', dataIndex: 'sharpe_ratio', key: 'sharpe_ratio' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const statusMap: Record<string, { color: string; text: string }> = {
          running: { color: 'blue', text: '运行中' },
          completed: { color: 'green', text: '已完成' },
          failed: { color: 'red', text: '失败' },
        }
        const s = statusMap[status] || { color: 'default', text: status }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
  ]

  return (
    <Card
      title="回测记录"
      extra={
        <Button type="primary" onClick={handleRunBacktest}>
          运行回测
        </Button>
      }
    >
      <Table
        columns={columns}
        dataSource={backtests}
        loading={loading}
        rowKey="id"
      />
    </Card>
  )
}

export default BacktestList
