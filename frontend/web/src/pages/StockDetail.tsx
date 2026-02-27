import { useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { Card, Row, Col, Statistic, Tabs } from 'antd'
import ReactECharts from 'echarts-for-react'
import { getQuote, getKline } from '../api/market'

interface QuoteData {
  symbol: string
  name: string
  price: number
  change: number
  change_pct: number
  volume: number
  amount: number
  open: number
  high: number
  low: number
  pre_close: number
}

interface KlineData {
  time: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

function StockDetail() {
  const { symbol } = useParams()
  const [searchParams] = useSearchParams()
  const exchange = searchParams.get('exchange') || 'SZ'
  
  const [quote, setQuote] = useState<QuoteData | null>(null)
  const [klineData, setKlineData] = useState<KlineData[]>([])

  useEffect(() => {
    if (symbol) {
      fetchQuote()
      fetchKline()
    }
  }, [symbol])

  const fetchQuote = async () => {
    const res = await getQuote(symbol!, exchange)
    if (res.code === 0) {
      setQuote(res.data)
    }
  }

  const fetchKline = async () => {
    const end = new Date().toISOString().split('T')[0]
    const start = new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
    const res = await getKline(symbol!, exchange, '1d', start, end)
    if (res.code === 0) {
      setKlineData(res.data.bars || [])
    }
  }

  const getKlineOption = () => {
    const dates = klineData.map(d => d.time)
    const data = klineData.map(d => [d.open, d.close, d.low, d.high])
    
    return {
      title: { text: '日K线图', left: 'center' },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: dates },
      yAxis: { type: 'value', scale: true },
      dataZoom: [{ type: 'inside' }, { type: 'slider' }],
      series: [{
        type: 'candlestick',
        data: data,
        itemStyle: {
          color: '#ef232a',
          color0: '#14b143',
          borderColor: '#ef232a',
          borderColor0: '#14b143',
        }
      }]
    }
  }

  if (!quote) return <div>加载中...</div>

  const isUp = quote.change >= 0

  return (
    <div>
      <h2>{quote.name} ({quote.symbol})</h2>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="当前价格"
              value={quote.price}
              precision={2}
              valueStyle={{ color: isUp ? '#cf1322' : '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="涨跌额"
              value={quote.change}
              precision={2}
              valueStyle={{ color: isUp ? '#cf1322' : '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="涨跌幅"
              value={quote.change_pct}
              precision={2}
              suffix="%"
              valueStyle={{ color: isUp ? '#cf1322' : '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="成交量" value={quote.volume} />
          </Card>
        </Col>
      </Row>

      <Card>
        <Tabs defaultActiveKey="kline">
          <Tabs.TabPane tab="K线图" key="kline">
            <ReactECharts option={getKlineOption()} style={{ height: 400 }} />
          </Tabs.TabPane>
        </Tabs>
      </Card>
    </div>
  )
}

export default StockDetail
