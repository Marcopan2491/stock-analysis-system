import { useState, useEffect } from 'react'
import { Table, Card, Input, Tag } from 'antd'
import { useNavigate } from 'react-router-dom'
import { searchStocks } from '../api/market'

interface Stock {
  id: number
  symbol: string
  name: string
  exchange: string
  industry: string
}

function StockList() {
  const [stocks, setStocks] = useState<Stock[]>([])
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')
  const navigate = useNavigate()

  useEffect(() => {
    fetchStocks()
  }, [])

  const fetchStocks = async () => {
    setLoading(true)
    try {
      const res = await searchStocks('')
      if (res.code === 0) {
        setStocks(res.data.results || [])
      }
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = async (value: string) => {
    setKeyword(value)
    setLoading(true)
    try {
      const res = await searchStocks(value)
      if (res.code === 0) {
        setStocks(res.data.results || [])
      }
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    { title: '代码', dataIndex: 'symbol', key: 'symbol' },
    { title: '名称', dataIndex: 'name', key: 'name' },
    {
      title: '交易所',
      dataIndex: 'exchange',
      key: 'exchange',
      render: (text: string) => (
        <Tag color={text === 'SH' ? 'blue' : 'green'}>
          {text === 'SH' ? '上海' : '深圳'}
        </Tag>
      ),
    },
    { title: '行业', dataIndex: 'industry', key: 'industry' },
  ]

  return (
    <Card title="股票列表">
      <Input.Search
        placeholder="搜索股票代码或名称"
        value={keyword}
        onChange={(e) => setKeyword(e.target.value)}
        onSearch={handleSearch}
        style={{ width: 300, marginBottom: 16 }}
      />
      <Table
        columns={columns}
        dataSource={stocks}
        loading={loading}
        rowKey="id"
        onRow={(record) => ({
          onClick: () => navigate(`/stock/${record.symbol}?exchange=${record.exchange}`),
          style: { cursor: 'pointer' },
        })}
      />
    </Card>
  )
}

export default StockList
