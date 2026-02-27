# 股票分析系统 - 测试清单

## 📋 测试前准备

### 环境要求
- **Go**: 1.21+ (后端服务)
- **Node.js**: 18+ (前端)
- **Python**: 3.10+ (策略层)
- **Docker**: 20.10+ (可选，用于容器化部署)
- **Git**: 2.30+

### 克隆项目
```bash
git clone <your-repo-url>
cd stock-analysis-system
```

---

## 🖥️ 一、本地启动测试

### 1.1 方式一：Docker Compose 一键启动（推荐）

#### 步骤
```bash
# 1. 进入部署目录
cd deploy/docker

# 2. 启动所有服务
docker-compose up -d

# 3. 查看服务状态
docker-compose ps

# 4. 查看日志
docker-compose logs -f

# 5. 停止所有服务
docker-compose down
```

#### 测试点
| 检查项 | 命令/方法 | 预期结果 |
|--------|----------|----------|
| 数据库启动 | `docker ps` | postgres 和 influxdb 运行中 |
| API Gateway | `curl http://localhost:8080/health` | 返回 healthy |
| 前端访问 | 浏览器访问 http://localhost:3000 | 看到登录页面 |
| 服务连通性 | `docker-compose ps` | 所有服务状态为 Up |

#### 端口检查
```bash
# 检查端口占用
netstat -an | findstr "8080"
netstat -an | findstr "8081"
netstat -an | findstr "8082"
netstat -an | findstr "3000"
```

---

### 1.2 方式二：手动启动各服务

#### 步骤1：启动数据库
```bash
# 启动 PostgreSQL
docker run -d --name stock-postgres \
  -e POSTGRES_USER=stock_user \
  -e POSTGRES_PASSWORD=stock_pass \
  -e POSTGRES_DB=stock_analysis \
  -p 5432:5432 \
  postgres:15-alpine

# 启动 InfluxDB
docker run -d --name stock-influxdb \
  -e DOCKER_INFLUXDB_INIT_MODE=setup \
  -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
  -e DOCKER_INFLUXDB_INIT_PASSWORD=admin123 \
  -e DOCKER_INFLUXDB_INIT_ORG=stock_org \
  -e DOCKER_INFLUXDB_INIT_BUCKET=stock_market \
  -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=stock-token-12345 \
  -p 8086:8086 \
  influxdb:2.7-alpine
```

#### 步骤2：初始化数据库
```bash
# 等待数据库启动 (约10秒)
sleep 10

# 初始化 PostgreSQL
cd database/scripts
psql -h localhost -U stock_user -d stock_analysis -f init_postgres.sql
# 密码: stock_pass
```

#### 步骤3：安装 Go 依赖
```bash
cd backend
go mod tidy
go mod download
```

#### 步骤4：启动后端服务（每个服务一个终端）

**终端1 - 数据同步服务 (端口 8081)**
```bash
cd backend/services/data-service
set POSTGRES_HOST=localhost
set POSTGRES_PORT=5432
set POSTGRES_USER=stock_user
set POSTGRES_PASSWORD=stock_pass
set POSTGRES_DB=stock_analysis
set INFLUXDB_URL=http://localhost:8086
set INFLUXDB_TOKEN=stock-token-12345
set INFLUXDB_ORG=stock_org
set INFLUXDB_BUCKET=stock_market
set DATA_SERVICE_PORT=8081
go run main.go
```

**终端2 - 行情服务 (端口 8082)**
```bash
cd backend/services/market-service
set POSTGRES_HOST=localhost
set INFLUXDB_URL=http://localhost:8086
set INFLUXDB_TOKEN=stock-token-12345
set MARKET_SERVICE_PORT=8082
go run main.go
```

**终端3 - 用户服务 (端口 8083)**
```bash
cd backend/services/user-service
set POSTGRES_HOST=localhost
set JWT_SECRET=your-secret-key-here
set USER_SERVICE_PORT=8083
go run main.go
```

**终端4 - 策略服务 (端口 8084)**
```bash
cd backend/services/strategy-service
set POSTGRES_HOST=localhost
set JWT_SECRET=your-secret-key-here
set STRATEGY_SERVICE_PORT=8084
go run main.go
```

**终端5 - 回测服务 (端口 8085)**
```bash
cd backend/services/backtest-service
set POSTGRES_HOST=localhost
set JWT_SECRET=your-secret-key-here
set BACKTEST_SERVICE_PORT=8085
go run main.go
```

**终端6 - API Gateway (端口 8080)**
```bash
cd backend/gateway
set MARKET_SERVICE_URL=http://localhost:8082
set USER_SERVICE_URL=http://localhost:8083
set STRATEGY_SERVICE_URL=http://localhost:8084
set BACKTEST_SERVICE_URL=http://localhost:8085
set DATA_SERVICE_URL=http://localhost:8081
set SERVER_PORT=8080
go run main.go
```

#### 步骤5：启动前端
```bash
cd frontend/web
npm install
npm run dev
```

---

## 🌐 二、GitHub 启动方式

### 2.1 GitHub Codespaces（推荐）

#### 步骤
```bash
# 1. 在 GitHub 仓库页面点击 "<> Code" -> "Codespaces" -> "Create codespace"

# 2. 等待环境初始化

# 3. 在 Codespace 终端中启动项目
cd deploy/docker
docker-compose up -d

# 4. 转发端口
# 点击 "PORTS" 标签，转发 8080 和 3000 端口
```

#### 配置 `.devcontainer/devcontainer.json`
```json
{
  "name": "Stock Analysis System",
  "image": "mcr.microsoft.com/devcontainers/go:1.21",
  "features": {
    "ghcr.io/devcontainers/features/node:1": {
      "version": "20"
    },
    "ghcr.io/devcontainers/features/docker-in-docker:2": {}
  },
  "forwardPorts": [8080, 8081, 8082, 8083, 8084, 8085, 3000, 5432, 8086],
  "postCreateCommand": "cd backend && go mod tidy && cd ../frontend/web && npm install",
  "customizations": {
    "vscode": {
      "extensions": ["golang.Go", "bradlc.vscode-tailwindcss"]
    }
  }
}
```

### 2.2 GitHub Actions CI/CD 测试

#### 创建 `.github/workflows/test.yml`
```yaml
name: Test

on: [push, pull_request]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: stock_user
          POSTGRES_PASSWORD: stock_pass
          POSTGRES_DB: stock_analysis
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Test Backend
      run: |
        cd backend
        go mod tidy
        go test ./pkg/... -v
      env:
        POSTGRES_HOST: localhost
        POSTGRES_PORT: 5432
        POSTGRES_USER: stock_user
        POSTGRES_PASSWORD: stock_pass
        POSTGRES_DB: stock_analysis

  frontend-test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Node
      uses: actions/setup-node@v3
      with:
        node-version: '20'
    
    - name: Test Frontend
      run: |
        cd frontend/web
        npm install
        npm run build
```

---

## 🧪 三、功能测试指南

### 3.1 数据库连接测试

#### PostgreSQL 连接测试
```bash
# 测试连接
psql -h localhost -U stock_user -d stock_analysis -c "SELECT version();"

# 查看表结构
psql -h localhost -U stock_user -d stock_analysis -c "\dt"

# 检查股票数据
psql -h localhost -U stock_user -d stock_analysis -c "SELECT COUNT(*) FROM stocks;"
```

#### InfluxDB 连接测试
```bash
# 测试连接
curl http://localhost:8086/health

# 查看 bucket
influx bucket list --org stock_org --token stock-token-12345
```

---

### 3.2 认证功能测试

#### 测试 1: 用户注册
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "123456"
  }'
```
**预期结果**: 返回 code: 0, 包含 user_id

#### 测试 2: 用户登录
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "123456"
  }'
```
**预期结果**: 返回 access_token

#### 测试 3: 获取用户信息
```bash
# 使用上一步获取的 token
curl http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回用户信息

#### 测试 4: 错误处理
```bash
# 错误密码
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"wrongpassword"}'
```
**预期结果**: 返回 code: 401

---

### 3.3 行情功能测试

#### 测试 1: 股票列表查询
```bash
# 需要登录获取 token 后测试
curl http://localhost:8080/api/v1/market/stocks \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回股票列表，包含 symbol, name, exchange

#### 测试 2: 股票搜索
```bash
curl "http://localhost:8080/api/v1/market/stocks/search?q=平安" \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回包含"平安"的股票

#### 测试 3: 实时行情
```bash
curl "http://localhost:8080/api/v1/market/quote/000001?exchange=SZ" \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回价格、涨跌幅、成交量等

#### 测试 4: K线数据
```bash
curl "http://localhost:8080/api/v1/market/kline/000001?exchange=SZ&period=1d&start=2024-01-01&end=2024-01-31" \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回日K线数据数组

---

### 3.4 策略功能测试

#### 测试 1: 创建策略
```bash
curl -X POST http://localhost:8080/api/v1/strategy \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "双均线策略",
    "description": "MA5上穿MA20买入",
    "type": "trend_following",
    "class_name": "DualMAStrategy",
    "params": "{\"fast\":5,\"slow\":20}",
    "symbols": ["000001","000002"],
    "is_public": false
  }'
```
**预期结果**: 返回创建的策略，包含 id

#### 测试 2: 获取策略列表
```bash
curl "http://localhost:8080/api/v1/strategy?page=1&page_size=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回策略列表

#### 测试 3: 更新策略
```bash
curl -X PUT http://localhost:8080/api/v1/strategy/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "双均线策略(修改)",
    "is_active": false
  }'
```
**预期结果**: 返回更新后的策略

#### 测试 4: 删除策略
```bash
curl -X DELETE http://localhost:8080/api/v1/strategy/1 \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回删除成功

---

### 3.5 回测功能测试

#### 测试 1: 提交回测任务
```bash
curl -X POST http://localhost:8080/api/v1/backtest/run \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "strategy_id": 1,
    "start_date": "2024-01-01",
    "end_date": "2024-06-30",
    "symbols": ["000001"],
    "initial_capital": 100000
  }'
```
**预期结果**: 返回 job_id 和 backtest_id

#### 测试 2: 查询回测状态
```bash
# 使用上一步返回的 job_id
curl http://localhost:8080/api/v1/backtest/status/YOUR_JOB_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回 status (running/completed/failed)

#### 测试 3: 查询回测结果
```bash
curl http://localhost:8080/api/v1/backtest/result/YOUR_BACKTEST_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回收益率、最大回撤、夏普比率等

#### 测试 4: 回测列表
```bash
curl "http://localhost:8080/api/v1/backtest?page=1&page_size=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回回测记录列表

---

### 3.6 自选股功能测试

#### 测试 1: 创建自选股分组
```bash
curl -X POST http://localhost:8080/api/v1/watchlist \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "我的自选股",
    "description": "关注的股票"
  }'
```
**预期结果**: 返回创建的分组

#### 测试 2: 获取自选股列表
```bash
curl http://localhost:8080/api/v1/watchlist \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回分组列表

#### 测试 3: 添加自选股
```bash
curl -X POST http://localhost:8080/api/v1/watchlist/1/items \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "symbol": "000001",
    "exchange": "SZ"
  }'
```
**预期结果**: 返回添加成功

#### 测试 4: 移除自选股
```bash
curl -X DELETE "http://localhost:8080/api/v1/watchlist/1/items/000001?exchange=SZ" \
  -H "Authorization: Bearer YOUR_TOKEN"
```
**预期结果**: 返回移除成功

---

### 3.7 前端功能测试

#### 测试 1: 页面访问
```bash
# 启动前端后访问
curl http://localhost:3000
```
**预期结果**: 返回 HTML 页面

#### 测试 2: 登录页面
- 访问 http://localhost:3000
- 输入用户名密码
- 点击登录

**预期结果**: 跳转到股票列表页

#### 测试 3: 股票列表
- 查看股票列表是否加载
- 测试搜索功能
- 点击股票进入详情

#### 测试 4: K线图
- 进入股票详情页
- 查看 K 线图是否显示
- 检查数据是否正确

#### 浏览器开发者工具测试
```javascript
// 打开浏览器控制台 (F12)，测试 API 连通性
fetch('/api/v1/market/stocks', {
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN'
  }
})
.then(r => r.json())
.then(console.log)
```

---

## 🔧 四、故障排查

### 常见问题

#### 1. 数据库连接失败
```bash
# 检查数据库是否运行
docker ps | grep postgres

# 检查端口
telnet localhost 5432

# 检查环境变量
echo %POSTGRES_HOST%
echo %POSTGRES_PORT%
```

#### 2. 服务启动失败
```bash
# 检查端口占用
netstat -ano | findstr "8080"

# 检查日志
cd backend/services/market-service
go run main.go 2>&1
```

#### 3. JWT 认证失败
```bash
# 检查 token 是否过期
# 重新登录获取新 token
```

#### 4. 前端 API 调用失败
```bash
# 检查代理配置
cat frontend/web/vite.config.ts

# 检查网关是否运行
curl http://localhost:8080/health
```

---

## 📊 五、性能测试

### 负载测试
```bash
# 使用 Apache Bench (需要安装)
ab -n 1000 -c 10 -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/market/stocks

# 使用 wrk
wrk -t12 -c400 -d30s -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/market/stocks
```

### 数据库性能
```sql
-- 检查慢查询
SELECT query, calls, mean_time, rows 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;
```

---

## ✅ 六、测试检查清单

### 部署前检查
- [ ] 所有服务可以正常启动
- [ ] 数据库连接正常
- [ ] API Gateway 路由正常
- [ ] JWT 认证工作正常
- [ ] 前端可以访问后端 API

### 功能检查
- [ ] 用户注册/登录
- [ ] 股票列表查询
- [ ] K线数据查询
- [ ] 策略 CRUD
- [ ] 回测任务提交
- [ ] 自选股管理

### 性能检查
- [ ] API 响应时间 < 500ms
- [ ] 数据库查询优化
- [ ] 并发请求处理

---

## 📞 七、获取帮助

### 查看日志
```bash
# Docker 日志
docker-compose logs -f [service-name]

# Go 服务日志
journalctl -u stock-market-service

# 前端日志
npm run dev  # 在控制台查看
```

### 调试模式
```bash
# Go 调试
go run -race main.go  # 检测竞态条件

# 前端调试
npm run dev -- --debug
```

---

**测试日期**: 2026-02-22  
**维护者**: OpenClaw Agent  
**版本**: v1.0
