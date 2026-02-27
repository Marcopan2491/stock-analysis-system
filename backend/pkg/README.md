# 数据层使用文档

## 概述

数据层是股票分析系统的核心基础设施，负责管理所有数据的存储、查询和同步。采用双数据库架构：

- **PostgreSQL**: 存储关系型数据（股票信息、用户信息、策略配置等）
- **InfluxDB**: 存储时序数据（K线、技术指标等）

## 目录结构

```
backend/pkg/
├── config/           # 配置管理
│   └── config.go
├── database/         # 数据库连接
│   ├── postgres.go   # PostgreSQL客户端
│   ├── influxdb.go   # InfluxDB客户端
│   └── manager.go    # 数据库管理器
├── models/           # 数据模型
│   └── models.go
├── repository/       # 数据仓库
│   ├── stock_repository.go   # 股票数据仓库
│   └── market_repository.go  # 行情数据仓库
└── quality/          # 数据质量监控
    └── monitor.go
```

## 快速开始

### 1. 配置数据库连接

通过环境变量配置：

```bash
# PostgreSQL
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=stock_user
export POSTGRES_PASSWORD=your_password
export POSTGRES_DB=stock_analysis

# InfluxDB
export INFLUXDB_URL=http://localhost:8086
export INFLUXDB_TOKEN=your_token
export INFLUXDB_ORG=stock_org
export INFLUXDB_BUCKET=stock_market
```

或通过配置文件 `config.yaml`：

```yaml
database:
  postgres:
    host: localhost
    port: 5432
    user: stock_user
    password: your_password
    database: stock_analysis
    max_conns: 20
    min_conns: 5
  influxdb:
    url: http://localhost:8086
    token: your_token
    org: stock_org
    bucket: stock_market
    batch_size: 100
```

### 2. 初始化数据库连接

```go
package main

import (
    "context"
    "log"
    
    "stock-analysis-system/backend/pkg/config"
    "stock-analysis-system/backend/pkg/database"
)

func main() {
    // 加载配置
    cfg := config.LoadFromEnv()
    
    // 创建数据库管理器
    dbManager, err := database.NewManager(&cfg.Database)
    if err != nil {
        log.Fatalf("初始化数据库失败: %v", err)
    }
    defer dbManager.Close()
    
    // 健康检查
    ctx := context.Background()
    results := dbManager.HealthCheck(ctx)
    for db, err := range results {
        if err != nil {
            log.Printf("%s 健康检查失败: %v", db, err)
        } else {
            log.Printf("%s 连接正常", db)
        }
    }
}
```

### 3. 使用数据仓库

#### 股票数据操作（PostgreSQL）

```go
import (
    "context"
    "stock-analysis-system/backend/pkg/models"
    "stock-analysis-system/backend/pkg/repository"
)

// 创建仓库
stockRepo := repository.NewStockRepository(dbManager.Postgres.DB)

// 创建股票
ctx := context.Background()
stock := &models.Stock{
    Symbol:   "000001",
    Name:     "平安银行",
    Exchange: "SZ",
    Industry: "银行",
}
if err := stockRepo.Create(ctx, stock); err != nil {
    log.Printf("创建失败: %v", err)
}

// 查询股票
stock, err := stockRepo.GetBySymbol(ctx, "000001", "SZ")
if err != nil {
    log.Printf("查询失败: %v", err)
}

// 搜索股票
stocks, err := stockRepo.Search(ctx, "银行")
if err != nil {
    log.Printf("搜索失败: %v", err)
}
```

#### 行情数据操作（InfluxDB）

```go
// 创建仓库
marketRepo := repository.NewMarketRepository(dbManager.Influx)

// 保存日K线
bar := &models.DailyBar{
    Symbol:   "000001",
    Exchange: "SZ",
    Date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
    Open:     12.35,
    High:     12.50,
    Low:      12.20,
    Close:    12.45,
    Volume:   1000000,
    Amount:   12450000,
}
if err := marketRepo.SaveDailyBar(ctx, bar); err != nil {
    log.Printf("保存失败: %v", err)
}

// 查询日K线
start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
bars, err := marketRepo.GetDailyBars(ctx, "000001", "SZ", start, end)
if err != nil {
    log.Printf("查询失败: %v", err)
}

// 查询技术指标
indicators, err := marketRepo.GetIndicators(ctx, "000001", "SZ", "ma", start, end)
if err != nil {
    log.Printf("查询指标失败: %v", err)
}
```

## 数据同步服务

### 启动数据同步服务

```bash
cd backend/services/data-service
go run main.go
```

服务默认监听端口 8081，提供以下 API：

- `POST /api/v1/sync/stocks` - 同步股票列表
- `POST /api/v1/sync/bars` - 同步单只股票K线
- `POST /api/v1/sync/incremental` - 执行增量更新
- `GET /health` - 健康检查

### 手动触发同步

```bash
# 同步股票列表
curl -X POST http://localhost:8081/api/v1/sync/stocks

# 同步单只股票K线
curl -X POST http://localhost:8081/api/v1/sync/bars \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "000001",
    "exchange": "SZ",
    "start": "2024-01-01",
    "end": "2024-01-31"
  }'

# 执行增量更新
curl -X POST http://localhost:8081/api/v1/sync/incremental
```

## 数据质量监控

### 使用数据质量检查器

```go
import "stock-analysis-system/backend/pkg/quality"

// 创建检查器
checker := quality.NewDataQualityChecker(stockRepo, marketRepo)

// 检查单只股票
checks, err := checker.CheckStock(ctx, "000001", "SZ")
if err != nil {
    log.Printf("检查失败: %v", err)
}

for _, check := range checks {
    log.Printf("%s: %s - %s", check.CheckType, check.Status, check.Message)
}

// 生成全市场报告
report, err := checker.GenerateReport(ctx)
if err != nil {
    log.Printf("生成报告失败: %v", err)
}

log.Printf("总股票数: %d", report.TotalStocks)
log.Printf("通过: %d, 警告: %d, 错误: %d", 
    report.Summary.PassCount,
    report.Summary.WarningCount,
    report.Summary.ErrorCount)
```

### 检查类型

1. **完整性检查 (completeness)** - 检查数据是否完整
2. **连续性检查 (continuity)** - 检查数据是否连续无断档
3. **异常值检查 (anomalies)** - 检查价格、成交量是否异常
4. **新鲜度检查 (freshness)** - 检查数据是否最新

## 数据库 Schema

### PostgreSQL

详见 `database/scripts/init_postgres.sql`

主要表：
- `stocks` - 股票基础信息
- `users` - 用户信息
- `strategies` - 策略配置
- `trade_signals` - 交易信号
- `backtest_records` - 回测记录
- `watchlists` - 自选股

### InfluxDB

- `daily_bars` - 日K线数据
- `minute_bars` - 分钟K线数据
- `indicators` - 技术指标

## 性能优化

### PostgreSQL

- 连接池配置（MaxConns: 20, MinConns: 5）
- 关键字段索引（symbol, exchange, industry）

### InfluxDB

- 批量写入（batch_size: 100）
- 异步写入 API
- 数据保留策略（原始数据2年，聚合数据5年）

## 故障排查

### 连接失败

1. 检查数据库服务是否启动
2. 检查连接配置（主机、端口、认证信息）
3. 查看防火墙设置

### 数据查询慢

1. 检查索引是否正确创建
2. 优化查询时间范围
3. 考虑使用聚合查询

### 数据不一致

1. 运行数据质量检查
2. 执行增量更新
3. 必要时重新同步历史数据

---

**维护者**: OpenClaw Agent  
**更新时间**: 2026-02-22
