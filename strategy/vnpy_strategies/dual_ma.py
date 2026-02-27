"""
双均线策略 - vn.py CTA策略模板
"""
from vnpy.app.cta_strategy import (
    CtaTemplate,
    StopOrder,
    TickData,
    BarData,
    TradeData,
    OrderData,
    BarGenerator,
    ArrayManager,
)


class DualMAStrategy(CtaTemplate):
    """
    双均线策略
    快线（短期）上穿慢线（长期）时买入
    快线下穿慢线时卖出
    """
    author = "StrategyDev"
    
    # 策略参数
    fast_window = 10      # 快线周期
    slow_window = 30      # 慢线周期
    fixed_size = 100      # 固定交易数量
    
    # 策略变量
    fast_ma = 0.0         # 快线数值
    slow_ma = 0.0         # 慢线数值
    ma_diff = 0.0         # 均线差值
    
    parameters = ["fast_window", "slow_window", "fixed_size"]
    variables = ["fast_ma", "slow_ma", "ma_diff"]
    
    def __init__(self, cta_engine, strategy_name, vt_symbol, setting):
        """构造函数"""
        super().__init__(cta_engine, strategy_name, vt_symbol, setting)
        
        self.bg = BarGenerator(self.on_bar)
        self.am = ArrayManager()
    
    def on_init(self):
        """策略初始化"""
        self.write_log("策略初始化")
        self.load_bar(10)   # 加载10天历史数据
    
    def on_start(self):
        """策略启动"""
        self.write_log("策略启动")
        self.put_event()
    
    def on_stop(self):
        """策略停止"""
        self.write_log("策略停止")
        self.put_event()
    
    def on_tick(self, tick: TickData):
        """Tick数据回调"""
        self.bg.update_tick(tick)
    
    def on_bar(self, bar: BarData):
        """K线数据回调"""
        # 更新K线到ArrayManager
        self.am.update_bar(bar)
        
        # 确保有足够的数据计算均线
        if not self.am.inited:
            return
        
        # 计算均线
        self.fast_ma = self.am.sma(self.fast_window, array=False)
        self.slow_ma = self.am.sma(self.slow_window, array=False)
        self.ma_diff = self.fast_ma - self.slow_ma
        
        # 交易逻辑
        if self.ma_diff > 0 and self.pos == 0:
            # 金叉，买入开仓
            self.buy(bar.close_price, self.fixed_size)
            self.write_log(f"金叉买入 {bar.close_price}")
        
        elif self.ma_diff < 0 and self.pos > 0:
            # 死叉，卖出平仓
            self.sell(bar.close_price, abs(self.pos))
            self.write_log(f"死叉卖出 {bar.close_price}")
        
        # 更新界面
        self.put_event()
    
    def on_order(self, order: OrderData):
        """委托回调"""
        pass
    
    def on_trade(self, trade: TradeData):
        """成交回调"""
        self.write_log(f"成交: {trade.direction.value} {trade.volume}@{trade.price}")
        self.put_event()
    
    def on_stop_order(self, stop_order: StopOrder):
        """停止单回调"""
        pass


class MACDStrategy(CtaTemplate):
    """
    MACD趋势策略
    DIF上穿DEA（金叉）时买入
    DIF下穿DEA（死叉）时卖出
    """
    author = "StrategyDev"
    
    # 策略参数
    fast_period = 12
    slow_period = 26
    signal_period = 9
    fixed_size = 100
    
    # 策略变量
    macd_dif = 0.0
    macd_dea = 0.0
    macd_hist = 0.0
    
    parameters = ["fast_period", "slow_period", "signal_period", "fixed_size"]
    variables = ["macd_dif", "macd_dea", "macd_hist"]
    
    def __init__(self, cta_engine, strategy_name, vt_symbol, setting):
        super().__init__(cta_engine, strategy_name, vt_symbol, setting)
        self.bg = BarGenerator(self.on_bar)
        self.am = ArrayManager()
    
    def on_bar(self, bar: BarData):
        self.am.update_bar(bar)
        
        if not self.am.inited:
            return
        
        # 计算MACD
        dif, dea, hist = self.am.macd(
            self.fast_period,
            self.slow_period,
            self.signal_period,
            array=False
        )
        
        self.macd_dif = dif
        self.macd_dea = dea
        self.macd_hist = hist
        
        # 交易逻辑
        if hist > 0 and self.pos == 0:
            self.buy(bar.close_price, self.fixed_size)
        elif hist < 0 and self.pos > 0:
            self.sell(bar.close_price, abs(self.pos))
        
        self.put_event()


class RSIStrategy(CtaTemplate):
    """
    RSI均值回复策略
    RSI < 30（超卖）时买入
    RSI > 70（超买）时卖出
    """
    author = "StrategyDev"
    
    # 策略参数
    rsi_period = 14
    rsi_oversold = 30
    rsi_overbought = 70
    fixed_size = 100
    
    # 策略变量
    rsi_value = 0.0
    
    parameters = ["rsi_period", "rsi_oversold", "rsi_overbought", "fixed_size"]
    variables = ["rsi_value"]
    
    def __init__(self, cta_engine, strategy_name, vt_symbol, setting):
        super().__init__(cta_engine, strategy_name, vt_symbol, setting)
        self.bg = BarGenerator(self.on_bar)
        self.am = ArrayManager()
    
    def on_bar(self, bar: BarData):
        self.am.update_bar(bar)
        
        if not self.am.inited:
            return
        
        # 计算RSI
        self.rsi_value = self.am.rsi(self.rsi_period, array=False)
        
        # 交易逻辑
        if self.rsi_value < self.rsi_oversold and self.pos == 0:
            # 超卖，买入
            self.buy(bar.close_price, self.fixed_size)
            self.write_log(f"RSI超卖({self.rsi_value:.2f})买入")
        
        elif self.rsi_value > self.rsi_overbought and self.pos > 0:
            # 超买，卖出
            self.sell(bar.close_price, abs(self.pos))
            self.write_log(f"RSI超买({self.rsi_value:.2f})卖出")
        
        self.put_event()


# 策略注册表
STRATEGY_REGISTRY = {
    "DualMA": DualMAStrategy,
    "MACD": MACDStrategy,
    "RSI": RSIStrategy,
}


def get_strategy(name: str):
    """获取策略类"""
    return STRATEGY_REGISTRY.get(name)