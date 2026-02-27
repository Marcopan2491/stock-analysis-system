"""
双均线策略 - 趋势跟踪
当短期均线上穿长期均线时买入，下穿时卖出
"""

from vnpy.app.cta_strategy import CtaTemplate, StopOrder
from vnpy.trader.object import TickData, BarData, TradeData, OrderData

from .base_template import BaseStrategy


class DualMAStrategy(BaseStrategy):
    """
    双均线策略
    
    策略逻辑：
    1. 计算快速均线（如5日）和慢速均线（如20日）
    2. 当快线从下向上穿越慢线时（金叉），产生买入信号
    3. 当快线从上向下穿越慢线时（死叉），产生卖出信号
    
    参数：
    - fast_window: 快线周期
    - slow_window: 慢线周期
    - volume_threshold: 成交量阈值（过滤低流动性股票）
    """
    
    author = "StrategyDev"
    
    # 策略参数
    fast_window = 5
    slow_window = 20
    volume_threshold = 100000  # 最小成交量
    
    # 内部变量
    fast_ma = 0.0
    slow_ma = 0.0
    ma_diff = 0.0             # 均线差值，用于判断交叉
    last_ma_diff = 0.0        # 上一次的差值
    
    parameters = ["fast_window", "slow_window", "volume_threshold", "risk_percent", "stop_loss_pct"]
    variables = ["fast_ma", "slow_ma", "ma_diff", "last_ma_diff"]
    
    def __init__(self, cta_engine, strategy_name, vt_symbol, setting):
        """构造函数"""
        super().__init__(cta_engine, strategy_name, vt_symbol, setting)
        
        # 确保快线周期小于慢线周期
        if self.fast_window >= self.slow_window:
            self.write_log("警告: fast_window 应小于 slow_window")
    
    def on_init(self):
        """策略初始化"""
        self.write_log(f"策略初始化 - 快线:{self.fast_window}, 慢线:{self.slow_window}")
        # 加载足够的历史数据用于计算均线
        self.load_bar(self.slow_window + 10)
    
    def trading_logic(self, bar: BarData):
        """交易逻辑实现"""
        # 过滤低成交量
        if bar.volume < self.volume_threshold:
            return
        
        # 检查是否有足够的数据计算均线
        if not self.am.inited:
            return
        
        # 计算均线
        self.fast_ma = self.am.sma(self.fast_window, array=True)[-1]
        self.slow_ma = self.am.sma(self.slow_window, array=True)[-1]
        
        # 计算差值
        self.last_ma_diff = self.ma_diff
        self.ma_diff = self.fast_ma - self.slow_ma
        
        # 判断交叉
        cross_over = self.ma_diff > 0 and self.last_ma_diff <= 0  # 金叉
        cross_below = self.ma_diff < 0 and self.last_ma_diff >= 0  # 死叉
        
        # 交易信号
        if cross_over:
            # 金叉买入
            if self.pos == 0:
                self.write_log(f"金叉信号 - 快线:{self.fast_ma:.2f}, 慢线:{self.slow_ma:.2f}")
                self.buy(bar.close_price, 100)
            elif self.pos < 0:
                # 平空仓
                self.cover(bar.close_price, abs(self.pos))
                # 开多仓
                self.buy(bar.close_price, 100)
                
        elif cross_below:
            # 死叉卖出
            if self.pos > 0:
                self.write_log(f"死叉信号 - 快线:{self.fast_ma:.2f}, 慢线:{self.slow_ma:.2f}")
                self.sell(bar.close_price, abs(self.pos))
    
    def on_trade(self, trade: TradeData):
        """成交回调"""
        super().on_trade(trade)
        
        # 发送通知（可以集成飞书）
        msg = f"【双均线策略】{self.vt_symbol} {trade.direction.value} {trade.volume}手 @ {trade.price}"
        self.write_log(msg)


class TripleMAStrategy(BaseStrategy):
    """
    三均线策略（均线系统）
    
    使用短、中、长三条均线：
    - 短期均线（如5日）：快速反应价格变化
    - 中期均线（如20日）：中期趋势
    - 长期均线（如60日）：长期趋势
    
    交易逻辑：
    1. 短期 > 中期 > 长期，多头排列，买入
    2. 短期 < 中期 < 长期，空头排列，卖出
    3. 其他情况，观望
    """
    
    author = "StrategyDev"
    
    short_window = 5
    medium_window = 20
    long_window = 60
    
    short_ma = 0.0
    medium_ma = 0.0
    long_ma = 0.0
    
    parameters = ["short_window", "medium_window", "long_window", "risk_percent", "stop_loss_pct"]
    variables = ["short_ma", "medium_ma", "long_ma"]
    
    def trading_logic(self, bar: BarData):
        """交易逻辑"""
        if not self.am.inited:
            return
        
        # 计算三条均线
        self.short_ma = self.am.sma(self.short_window, array=True)[-1]
        self.medium_ma = self.am.sma(self.medium_window, array=True)[-1]
        self.long_ma = self.am.sma(self.long_window, array=True)[-1]
        
        # 判断趋势
        bull_trend = self.short_ma > self.medium_ma > self.long_ma  # 多头排列
        bear_trend = self.short_ma < self.medium_ma < self.long_ma  # 空头排列
        
        # 交易逻辑
        if bull_trend and self.pos <= 0:
            # 多头趋势，买入
            if self.pos < 0:
                self.cover(bar.close_price, abs(self.pos))
            self.buy(bar.close_price, 100)
            self.write_log(f"多头排列买入 - 短:{self.short_ma:.2f}, 中:{self.medium_ma:.2f}, 长:{self.long_ma:.2f}")
            
        elif bear_trend and self.pos >= 0:
            # 空头趋势，卖出
            if self.pos > 0:
                self.sell(bar.close_price, abs(self.pos))
            self.write_log(f"空头排列卖出 - 短:{self.short_ma:.2f}, 中:{self.medium_ma:.2f}, 长:{self.long_ma:.2f}")
