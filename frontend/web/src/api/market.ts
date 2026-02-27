import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'

const api = axios.create({
  baseURL: API_BASE,
  timeout: 30000,
})

// 请求拦截器添加token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 市场数据API
export const searchStocks = (keyword: string) =>
  api.get(`/market/stocks/search?q=${keyword}`).then(r => r.data)

export const getQuote = (symbol: string, exchange: string) =>
  api.get(`/market/quote/${symbol}?exchange=${exchange}`).then(r => r.data)

export const getKline = (symbol: string, exchange: string, period: string, start: string, end: string) =>
  api.get(`/market/kline/${symbol}?exchange=${exchange}&period=${period}&start=${start}&end=${end}`).then(r => r.data)

// 认证API
export const login = (username: string, password: string) =>
  api.post('/auth/login', { username, password }).then(r => r.data)

export const register = (username: string, email: string, password: string) =>
  api.post('/auth/register', { username, email, password }).then(r => r.data)

// 策略API
export const getStrategies = () =>
  api.get('/strategy').then(r => r.data)

export const createStrategy = (data: any) =>
  api.post('/strategy', data).then(r => r.data)

// 回测API
export const getBacktests = () =>
  api.get('/backtest').then(r => r.data)

export const runBacktest = (data: any) =>
  api.post('/backtest/run', data).then(r => r.data)

export default api
