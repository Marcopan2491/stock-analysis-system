# 数据库 Schema 文档

## PostgreSQL - 关系型数据库

### 表结构说明

| 表名 | 用途 | 主要字段 |
|------|------|---------|
| stocks | 股票基础信息 | symbol, name, exchange, industry |
| users | 用户信息 | username, email, password_hash |
| strategies | 策略配置 | name, type, params(JSONB), symbols |
| trade_signals | 交易信号 | strategy_id, symbol, signal_type, price |
| backtest_records | 回测记录 | strategy_id, total_return, max_drawdown, sharpe_ratio |
| watchlists | 自选股分组 | user_id, name |
| watchlist_items | 自选股明细 | watchlist_id, symbol |
| financial_reports | 财务数据 | symbol, report_date, revenue, profit, roe |

## InfluxDB - 时序数据库

### Measurement 说明

| Measurement | Fields | Tags |
|-------------|--------|------|
| daily_bars | open, high, low, close, volume, amount | symbol, exchange |
| minute_bars | open, high, low, close, volume, amount | symbol, exchange, interval |
| indicators | ma5, ma10, ma20, macd, rsi, kdj_k, kdj_d | symbol, indicator_type |

### 数据示例

```sql
-- 查询某日K线数据
SELECT * FROM daily_bars 
WHERE symbol = '000001' AND exchange = 'SZ' 
AND time >= '2024-01-01' AND time <= '2024-12-31'

-- 查询技术指标
SELECT mean("rsi") FROM "indicators" 
WHERE symbol = '000001' 
GROUP BY time(1d)
```

## 连接配置

### PostgreSQL
```yaml
host: localhost
port: 5432
database: stock_analysis
user: postgres
password: your_password
```

### InfluxDB
```yaml
host: localhost
port: 8086
database: stock_market
username: admin
password: your_password
```
