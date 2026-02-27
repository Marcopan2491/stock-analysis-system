-- 初始化数据库脚本
-- 创建股票基础信息表

-- 股票列表
CREATE TABLE IF NOT EXISTS stocks (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    exchange VARCHAR(10) NOT NULL,
    industry VARCHAR(50),
    market VARCHAR(10),  -- 主板/创业板/科创板/北交所
    list_date DATE,
    total_shares BIGINT,
    float_shares BIGINT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_stocks_exchange ON stocks(exchange);
CREATE INDEX idx_stocks_industry ON stocks(industry);

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 自选股表
CREATE TABLE IF NOT EXISTS watchlists (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    symbol VARCHAR(10) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, symbol)
);

-- 策略配置表
CREATE TABLE IF NOT EXISTS strategies (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,  -- trend/mean_reversion/multi_factor
    params JSONB NOT NULL DEFAULT '{}',
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 交易信号表
CREATE TABLE IF NOT EXISTS signals (
    id SERIAL PRIMARY KEY,
    strategy_id INTEGER REFERENCES strategies(id) ON DELETE CASCADE,
    symbol VARCHAR(10) NOT NULL,
    signal_type VARCHAR(20) NOT NULL,  -- buy/sell/hold
    price DECIMAL(10, 4),
    volume INTEGER,
    reason TEXT,
    signal_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_signals_symbol_time ON signals(symbol, signal_time);
CREATE INDEX idx_signals_strategy ON signals(strategy_id);

-- 回测记录表
CREATE TABLE IF NOT EXISTS backtests (
    id SERIAL PRIMARY KEY,
    strategy_id INTEGER REFERENCES strategies(id) ON DELETE SET NULL,
    symbol VARCHAR(10) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    initial_capital DECIMAL(15, 2) DEFAULT 1000000.00,
    final_capital DECIMAL(15, 2),
    total_return DECIMAL(10, 4),  -- 总收益率
    max_drawdown DECIMAL(10, 4),  -- 最大回撤
    sharpe_ratio DECIMAL(10, 4),  -- 夏普比率
    win_rate DECIMAL(10, 4),      -- 胜率
    trade_count INTEGER,
    result_data JSONB,            -- 详细回测结果
    created_at TIMESTAMP DEFAULT NOW()
);

-- 触发器：自动更新 updated_at
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