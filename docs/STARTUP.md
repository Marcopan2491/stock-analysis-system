# 股票分析系统 - 启动指南

## 项目概述

基于多Agent协作的股票行情分析系统，包含：
- Go 微服务后端 (Gin + GORM + InfluxDB)
- React + TypeScript 前端
- Python 策略引擎 (vn.py)

## 快速启动

### 方式一：Docker Compose 一键启动（推荐）

```bash
# 1. 进入部署目录
cd deploy/docker

# 2. 启动所有服务
docker-compose up -d

# 3. 查看服务状态
docker-compose ps

# 4. 停止所有服务
docker-compose down
```

访问地址：
- 前端 Web: http://localhost:3000
- API Gateway: http://localhost:8080
- PostgreSQL: localhost:5432
- InfluxDB: http://localhost:8086

### 方式二：本地开发环境启动

#### 1. 启动数据库

```bash
# 使用 Docker 启动数据库
docker run -d \
  --name stock-postgres \
  -e POSTGRES_USER=stock_user \
  -e POSTGRES_PASSWORD=stock_pass \
  -e POSTGRES_DB=stock_analysis \
  -p 5432:5432 \
  postgres:15-alpine

docker run -d \
  --name stock-influxdb \
  -e DOCKER_INFLUXDB_INIT_MODE=setup \
  -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
  -e DOCKER_INFLUXDB_INIT_PASSWORD=admin123 \
  -e DOCKER_INFLUXDB_INIT_ORG=stock_org \
  -e DOCKER_INFLUXDB_INIT_BUCKET=stock_market \
  -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=stock-token-12345 \
  -p 8086:8086 \
  influxdb:2.7-alpine
```

#### 2. 初始化数据库

```bash
# 初始化 PostgreSQL
cd database/scripts
psql -h localhost -U stock_user -d stock_analysis -f init_postgres.sql
```

#### 3. 启动后端服务

```bash
# 安装 Go 依赖
cd backend
go mod tidy

# 启动数据同步服务 (端口 8081)
cd services/data-service
go run main.go

# 启动行情服务 (端口 8082)
cd services/market-service
go run main.go

# 启动用户服务 (端口 8083)
cd services/user-service
go run main.go

# 启动策略服务 (端口 8084)
cd services/strategy-service
go run main.go

# 启动回测服务 (端口 8085)
cd services/backtest-service
go run main.go

# 启动 API Gateway (端口 8080)
cd gateway
go run main.go
```

#### 4. 启动前端

```bash
cd frontend/web
npm install
npm run dev
```

访问 http://localhost:3000

## API 接口列表

### 认证接口
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/auth/register | 用户注册 |
| POST | /api/v1/auth/login | 用户登录 |

### 行情接口
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/market/stocks | 股票列表 |
| GET | /api/v1/market/stocks/search?q={keyword} | 搜索股票 |
| GET | /api/v1/market/quote/{symbol} | 实时行情 |
| GET | /api/v1/market/kline/{symbol} | K线数据 |
| GET | /api/v1/market/indicators/{symbol} | 技术指标 |

### 用户接口
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/user/profile | 用户信息 |
| PUT | /api/v1/user/profile | 更新信息 |
| GET | /api/v1/watchlist | 自选股列表 |
| POST | /api/v1/watchlist | 创建分组 |
| POST | /api/v1/watchlist/{id}/items | 添加自选股 |

### 策略接口
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/strategy | 策略列表 |
| POST | /api/v1/strategy | 创建策略 |
| GET | /api/v1/strategy/{id} | 策略详情 |
| PUT | /api/v1/strategy/{id} | 更新策略 |
| DELETE | /api/v1/strategy/{id} | 删除策略 |
| GET | /api/v1/signals | 交易信号 |

### 回测接口
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/backtest | 回测列表 |
| POST | /api/v1/backtest/run | 运行回测 |
| GET | /api/v1/backtest/status/{id} | 回测状态 |
| GET | /api/v1/backtest/result/{id} | 回测结果 |

## 环境变量配置

### 后端服务

```bash
# 数据库配置
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=stock_user
POSTGRES_PASSWORD=stock_pass
POSTGRES_DB=stock_analysis

# InfluxDB配置
INFLUXDB_URL=http://localhost:8086
INFLUXDB_TOKEN=stock-token-12345
INFLUXDB_ORG=stock_org
INFLUXDB_BUCKET=stock_market

# JWT密钥
JWT_SECRET=your-secret-key-here

# 服务端口
DATA_SERVICE_PORT=8081
MARKET_SERVICE_PORT=8082
USER_SERVICE_PORT=8083
STRATEGY_SERVICE_PORT=8084
BACKTEST_SERVICE_PORT=8085
SERVER_PORT=8080
```

### 前端

```bash
# API基础地址
VITE_API_BASE=http://localhost:8080/api/v1
```

## 测试示例

```bash
# 1. 用户注册
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"123456"}'

# 2. 用户登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456"}'

# 3. 获取股票列表
curl http://localhost:8080/api/v1/market/stocks

# 4. 查询行情 (需要登录后获取的token)
curl http://localhost:8080/api/v1/market/quote/000001?exchange=SZ \
  -H "Authorization: Bearer YOUR_TOKEN"

# 5. 查询K线
curl "http://localhost:8080/api/v1/market/kline/000001?exchange=SZ&period=1d&start=2024-01-01&end=2024-01-31" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 项目结构

```
stock-analysis-system/
├── backend/                 # Go 后端
│   ├── gateway/            # API 网关
│   ├── pkg/                # 公共库
│   │   ├── config/        # 配置
│   │   ├── database/      # 数据库连接
│   │   ├── models/        # 数据模型
│   │   ├── quality/       # 数据质量
│   │   └── repository/    # 数据仓库
│   └── services/          # 微服务
│       ├── data-service/   # 数据同步
│       ├── market-service/ # 行情服务
│       ├── user-service/   # 用户服务
│       ├── strategy-service/ # 策略服务
│       └── backtest-service/ # 回测服务
├── frontend/               # 前端
│   └── web/               # React Web应用
├── strategy/               # Python策略层
│   ├── data_collector/    # 数据采集
│   ├── data_analysis/     # 数据分析
│   └── vnpy_strategies/   # 策略实现
├── database/               # 数据库脚本
│   └── scripts/
└── deploy/                 # 部署配置
    └── docker/
```

## 注意事项

1. **首次启动**：需要先启动数据库，再初始化数据库脚本
2. **服务依赖**：Gateway 依赖其他服务，其他服务依赖数据库
3. **数据同步**：数据同步服务负责从 Python 采集器同步数据到数据库
4. **JWT 认证**：除登录注册外，其他接口都需要携带 Authorization Header

---

**维护者**: OpenClaw Agent  
**更新日期**: 2026-02-22
