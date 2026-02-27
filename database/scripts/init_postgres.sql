-- 股票分析系统 - PostgreSQL 初始化脚本
-- 创建数据库
-- CREATE DATABASE stock_analysis;

-- 切换到数据库
\c stock_analysis;

-- ============================================
-- 1. 股票基础信息表
-- ============================================
CREATE TABLE IF NOT EXISTS stocks (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,              -- 股票代码
    name VARCHAR(100) NOT NULL,               -- 股票名称
    exchange VARCHAR(10) NOT NULL,            -- 交易所 (SH/SZ)
    industry VARCHAR(50),                     -- 所属行业
    full_name VARCHAR(200),                   -- 公司全称
    list_date DATE,                           -- 上市日期
    total_share BIGINT,                       -- 总股本
    float_share BIGINT,                       -- 流通股本
    status VARCHAR(10) DEFAULT 'active',      -- 状态
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(symbol, exchange)
);

CREATE INDEX idx_stocks_symbol ON stocks(symbol);
CREATE INDEX idx_stocks_exchange ON stocks(exchange);
CREATE INDEX idx_stocks_industry ON stocks(industry);

COMMENT ON TABLE stocks IS '股票基础信息表';
COMMENT ON COLUMN stocks.symbol IS '股票代码，如600000';
COMMENT ON COLUMN stocks.exchange IS '交易所代码：SH上交所 SZ深交所';

-- ============================================
-- 2. 用户表
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,     -- 用户名
    email VARCHAR(100) NOT NULL UNIQUE,       -- 邮箱
    password_hash VARCHAR(255) NOT NULL,      -- 密码哈希
    avatar_url VARCHAR(500),                  -- 头像URL
    phone VARCHAR(20),                        -- 手机号
    status VARCHAR(10) DEFAULT 'active',      -- 状态
    last_login_at TIMESTAMP,                  -- 最后登录时间
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

COMMENT ON TABLE users IS '用户信息表';

-- ============================================
-- 3. 策略配置表
-- ============================================
CREATE TABLE IF NOT EXISTS strategies (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,               -- 策略名称
    description TEXT,                         -- 策略描述
    type VARCHAR(50) NOT NULL,                -- 策略类型
    class_name VARCHAR(100) NOT NULL,         -- 策略类名
    params JSONB NOT NULL DEFAULT '{}',       -- 策略参数
    symbols TEXT[],                           -- 交易标的列表
    is_active BOOLEAN DEFAULT true,           -- 是否激活
    is_public BOOLEAN DEFAULT false,          -- 是否公开
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_strategies_user_id ON strategies(user_id);
CREATE INDEX idx_strategies_type ON strategies(type);
CREATE INDEX idx_strategies_is_active ON strategies(is_active);

COMMENT ON TABLE strategies IS '策略配置表';
COMMENT ON COLUMN strategies.type IS '策略类型：trend_following/mean_reversion/multi_factor';

-- ============================================
-- 4. 交易信号表
-- ============================================
CREATE TABLE IF NOT EXISTS trade_signals (
    id SERIAL PRIMARY KEY,
    strategy_id INTEGER REFERENCES strategies(id) ON DELETE CASCADE,
    symbol VARCHAR(10) NOT NULL,              -- 股票代码
    exchange VARCHAR(10) NOT NULL,            -- 交易所
    signal_type VARCHAR(10) NOT NULL,         -- 信号类型：buy/sell/close
    price DECIMAL(12, 4),                     -- 触发价格
    volume INTEGER,                           -- 建议数量
    reason TEXT,                              -- 信号原因
    confidence DECIMAL(3, 2),                 -- 置信度 (0-1)
    is_executed BOOLEAN DEFAULT false,        -- 是否已执行
    executed_at TIMESTAMP,                    -- 执行时间
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_signals_strategy_id ON trade_signals(strategy_id);
CREATE INDEX idx_signals_symbol ON trade_signals(symbol);
CREATE INDEX idx_signals_type ON trade_signals(signal_type);
CREATE INDEX idx_signals_created_at ON trade_signals(created_at);

COMMENT ON TABLE trade_signals IS '交易信号记录表';

-- ============================================
-- 5. 回测记录表
-- ============================================
CREATE TABLE IF NOT EXISTS backtest_records (
    id SERIAL PRIMARY KEY,
    strategy_id INTEGER REFERENCES strategies(id) ON DELETE CASCADE,
    start_date DATE NOT NULL,                 -- 回测开始日期
    end_date DATE NOT NULL,                   -- 回测结束日期
    initial_capital DECIMAL(15, 2) NOT NULL,  -- 初始资金
    final_capital DECIMAL(15, 2),             -- 最终资金
    total_return DECIMAL(8, 4),               -- 总收益率
    annual_return DECIMAL(8, 4),              -- 年化收益率
    max_drawdown DECIMAL(8, 4),               -- 最大回撤
    sharpe_ratio DECIMAL(8, 4),               -- 夏普比率
    win_rate DECIMAL(5, 4),                   -- 胜率
    profit_loss_ratio DECIMAL(8, 4),          -- 盈亏比
    trade_count INTEGER,                      -- 交易次数
    params JSONB,                             -- 回测参数
    result_data JSONB,                        -- 详细结果数据
    status VARCHAR(20) DEFAULT 'running',     -- 状态
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_backtest_strategy_id ON backtest_records(strategy_id);
CREATE INDEX idx_backtest_status ON backtest_records(status);
CREATE INDEX idx_backtest_created_at ON backtest_records(created_at);

COMMENT ON TABLE backtest_records IS '策略回测记录表';

-- ============================================
-- 6. 自选股表
-- ============================================
CREATE TABLE IF NOT EXISTS watchlists (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,                -- 分组名称
    description TEXT,                         -- 分组描述
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS watchlist_items (
    id SERIAL PRIMARY KEY,
    watchlist_id INTEGER REFERENCES watchlists(id) ON DELETE CASCADE,
    symbol VARCHAR(10) NOT NULL,
    exchange VARCHAR(10) NOT NULL,
    added_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(watchlist_id, symbol, exchange)
);

CREATE INDEX idx_watchlist_user_id ON watchlists(user_id);
CREATE INDEX idx_watchlist_item_watchlist_id ON watchlist_items(watchlist_id);

COMMENT ON TABLE watchlists IS '自选股分组表';
COMMENT ON TABLE watchlist_items IS '自选股明细表';

-- ============================================
-- 7. 财务数据表（基础版）
-- ============================================
CREATE TABLE IF NOT EXISTS financial_reports (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    exchange VARCHAR(10) NOT NULL,
    report_type VARCHAR(10) NOT NULL,         -- 报告类型：Q1/Q2/Q3/annual
    report_date DATE NOT NULL,                -- 报告期
    
    -- 利润表主要指标
    total_revenue DECIMAL(15, 2),             -- 营业总收入
    net_profit DECIMAL(15, 2),                -- 净利润
    gross_profit DECIMAL(15, 2),              -- 毛利润
    
    -- 资产负债表主要指标
    total_assets DECIMAL(15, 2),              -- 总资产
    total_liabilities DECIMAL(15, 2),         -- 总负债
    shareholders_equity DECIMAL(15, 2),       -- 股东权益
    
    -- 关键比率（可计算得出，存储方便查询）
    roe DECIMAL(8, 4),                        -- 净资产收益率
    roa DECIMAL(8, 4),                        -- 总资产收益率
    gross_margin DECIMAL(6, 4),               -- 毛利率
    debt_ratio DECIMAL(6, 4),                 -- 资产负债率
    
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(symbol, exchange, report_type, report_date)
);

CREATE INDEX idx_financial_symbol ON financial_reports(symbol);
CREATE INDEX idx_financial_report_date ON financial_reports(report_date);

COMMENT ON TABLE financial_reports IS '财务报告数据表';

-- ============================================
-- 8. 创建更新时间触发器
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_stocks_updated_at BEFORE UPDATE ON stocks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_strategies_updated_at BEFORE UPDATE ON strategies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 9. 初始化测试数据
-- ============================================

-- 插入测试股票（沪深300部分成分股）
INSERT INTO stocks (symbol, name, exchange, industry, list_date) VALUES
('000001', '平安银行', 'SZ', '银行', '1991-04-03'),
('000002', '万科A', 'SZ', '房地产', '1991-01-29'),
('000063', '中兴通讯', 'SZ', '通信设备', '1997-11-18'),
('000100', 'TCL科技', 'SZ', '电子', '2004-01-30'),
('000333', '美的集团', 'SZ', '家电', '2013-09-18'),
('000568', '泸州老窖', 'SZ', '白酒', '1994-05-09'),
('000651', '格力电器', 'SZ', '家电', '1996-11-18'),
('000725', '京东方A', 'SZ', '电子', '2001-01-12'),
('000768', '中航西飞', 'SZ', '航空装备', '1997-06-26'),
('000858', '五粮液', 'SZ', '白酒', '1998-04-27'),
('600000', '浦发银行', 'SH', '银行', '1999-11-10'),
('600009', '上海机场', 'SH', '机场', '1998-02-18'),
('600016', '民生银行', 'SH', '银行', '2000-12-19'),
('600028', '中国石化', 'SH', '石油石化', '2001-08-08'),
('600030', '中信证券', 'SH', '证券', '2003-01-06'),
('600031', '三一重工', 'SH', '工程机械', '2003-07-03'),
('600036', '招商银行', 'SH', '银行', '2002-04-09'),
('600048', '保利发展', 'SH', '房地产', '2006-07-31'),
('600276', '恒瑞医药', 'SH', '医药', '2000-10-18'),
('600309', '万华化学', 'SH', '化工', '2001-01-09'),
('600519', '贵州茅台', 'SH', '白酒', '2001-08-27'),
('600585', '海螺水泥', 'SH', '水泥', '2002-02-07'),
('600690', '海尔智家', 'SH', '家电', '1993-11-19'),
('600837', '海通证券', 'SH', '证券', '1994-02-24'),
('601012', '隆基绿能', 'SH', '光伏', '2012-04-11'),
('601088', '中国神华', 'SH', '煤炭', '2007-10-09'),
('601166', '兴业银行', 'SH', '银行', '2007-02-05'),
('601318', '中国平安', 'SH', '保险', '2007-03-01'),
('601398', '工商银行', 'SH', '银行', '2006-10-27'),
('601888', '中国中免', 'SH', '免税', '2009-10-15')
ON CONFLICT DO NOTHING;

-- 插入测试用户
INSERT INTO users (username, email, password_hash) VALUES
('admin', 'admin@example.com', '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiAYMyzJ/IhO')
ON CONFLICT DO NOTHING;

-- 插入测试策略
INSERT INTO strategies (user_id, name, description, type, class_name, params) VALUES
(1, '双均线策略', '基于5日和20日均线的金叉死叉交易策略', 'trend_following', 'DualMAStrategy', 
 '{"fast_window": 5, "slow_window": 20, "volume_threshold": 1000}'),
(1, 'MACD趋势策略', '基于MACD指标的趋势跟踪策略', 'trend_following', 'MACDStrategy',
 '{"fast_period": 12, "slow_period": 26, "signal_period": 9}'),
(1, 'RSI超买超卖', '基于RSI指标的均值回复策略', 'mean_reversion', 'RSIStrategy',
 '{"rsi_period": 14, "overbought": 70, "oversold": 30}')
ON CONFLICT DO NOTHING;

-- ============================================
-- 10. 创建视图
-- ============================================

-- 股票完整信息视图
CREATE OR REPLACE VIEW v_stock_info AS
SELECT 
    s.symbol,
    s.name,
    s.exchange,
    CASE s.exchange 
        WHEN 'SH' THEN '上海证券交易所'
        WHEN 'SZ' THEN '深圳证券交易所'
    END AS exchange_name,
    s.industry,
    s.list_date,
    s.status
FROM stocks s
WHERE s.status = 'active';

-- 策略绩效概览视图
CREATE OR REPLACE VIEW v_strategy_performance AS
SELECT 
    s.id AS strategy_id,
    s.name AS strategy_name,
    s.type,
    COUNT(DISTINCT sig.id) AS total_signals,
    COUNT(DISTINCT CASE WHEN sig.signal_type = 'buy' THEN sig.id END) AS buy_signals,
    COUNT(DISTINCT CASE WHEN sig.signal_type = 'sell' THEN sig.id END) AS sell_signals,
    COUNT(DISTINCT br.id) AS backtest_count,
    MAX(br.created_at) AS last_backtest_at
FROM strategies s
LEFT JOIN trade_signals sig ON s.id = sig.strategy_id
LEFT JOIN backtest_records br ON s.id = br.strategy_id
GROUP BY s.id, s.name, s.type;

-- ============================================
-- 完成初始化
-- ============================================
SELECT 'Database initialization completed successfully!' AS status;
