# vn.py 股票策略开发环境

## 安装依赖

```bash
# 创建虚拟环境
python -m venv venv
venv\Scripts\activate

# 安装核心依赖
pip install vnpy vnpy_ctp vnpy_sqlite

# 安装数据分析依赖  
pip install pandas numpy talib-binary

# 安装免费数据源
pip install akshare
```

## 目录说明

```
strategy/
├── vnpy_strategies/      # vn.py CTA策略
│   ├── __init__.py
│   ├── dual_ma.py       # 双均线策略
│   ├── macd_strategy.py # MACD策略
│   └── rsi_strategy.py  # RSI策略
├── data_analysis/        # 数据分析
│   ├── __init__.py
│   ├── indicators.py    # 技术指标计算
│   └── factor_model.py  # 多因子模型
├── backtest/            # 回测系统
│   ├── __init__.py
│   └── runner.py        # 回测运行器
└── utils/               # 工具函数
    ├── __init__.py
    ├── data_loader.py   # 数据加载器
    └── database.py      # 数据库连接
```

## 快速开始

### 1. 下载股票数据

```python
from utils.data_loader import DataLoader

loader = DataLoader()
# 下载贵州茅台日K数据
loader.download_daily_bars("600519.SH", "20230101", "20240101")
```

### 2. 运行策略回测

```python
from vnpy.app.cta_backtester import BacktesterEngine
from vnpy_strategies.dual_ma import DualMAStrategy

# 创建回测引擎
engine = BacktesterEngine()
engine.set_parameters(
    vt_symbol="600519.SH",
    interval="1d",
    start=datetime(2023, 1, 1),
    end=datetime(2024, 1, 1),
    rate=0.00025,
    slippage=0.001,
    size=100,
    pricetick=0.01,
    capital=1_000_000,
)
engine.add_strategy(DualMAStrategy, {})
engine.run_backtesting()
engine.calculate_result()
engine.calculate_statistics()
engine.show_chart()
```

### 3. 计算技术指标

```python
from data_analysis.indicators import IndicatorEngine

engine = IndicatorEngine()
df = engine.get_kline_data("600519.SH", "20230101", "20240101")
df = engine.add_ma(df, [5, 10, 20, 60])
df = engine.add_macd(df)
df = engine.add_kdj(df)
```