import { useState } from 'react'
import { Layout, Menu, theme } from 'antd'
import {
  DashboardOutlined,
  LineChartOutlined,
  SettingOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { Routes, Route, Link, useLocation } from 'react-router-dom'
import StockList from './pages/StockList'
import StockDetail from './pages/StockDetail'
import StrategyList from './pages/StrategyList'
import BacktestList from './pages/BacktestList'
import './App.css'

const { Header, Sider, Content } = Layout

function App() {
  const [collapsed, setCollapsed] = useState(false)
  const {
    token: { colorBgContainer },
  } = theme.useToken()

  const location = useLocation()

  const menuItems = [
    {
      key: '/',
      icon: <DashboardOutlined />,
      label: <Link to="/">股票列表</Link>,
    },
    {
      key: '/strategies',
      icon: <LineChartOutlined />,
      label: <Link to="/strategies">策略管理</Link>,
    },
    {
      key: '/backtests',
      icon: <SettingOutlined />,
      label: <Link to="/backtests">回测记录</Link>,
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div className="logo">
          <h3 style={{ color: 'white', textAlign: 'center', padding: '16px 0' }}>
            {collapsed ? 'SAS' : '股票分析系统'}
          </h3>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
        />
      </Sider>
      <Layout>
        <Header style={{ padding: 0, background: colorBgContainer }}>
          <div style={{ float: 'right', marginRight: 24 }}>
            <UserOutlined /> 管理员
          </div>
        </Header>
        <Content
          style={{
            margin: '24px 16px',
            padding: 24,
            minHeight: 280,
            background: colorBgContainer,
          }}
        >
          <Routes>
            <Route path="/" element={<StockList />} />
            <Route path="/stock/:symbol" element={<StockDetail />} />
            <Route path="/strategies" element={<StrategyList />} />
            <Route path="/backtests" element={<BacktestList />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  )
}

export default App
