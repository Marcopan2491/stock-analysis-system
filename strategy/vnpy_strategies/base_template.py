"""
vn.py 基础策略模板
所有策略应继承此模板
"""

from vnpy.app.cta_strategy import CtaTemplate, StopOrder
from vnpy.trader.object import TickData, BarData, TradeData, OrderData
from vnpy.trader.constant import Direction, Offset


class BaseStrategy(CtaTemplate):
    """
    基础策略模板
    包含通用的风控和日志功能
    """
    
    author = "StrategyDev"
    
    # 基础参数
    risk_percent = 2.0          # 单笔风险百分比
    max_position = 100          # 最大持仓
    stop_loss_pct = 5.0         # 止损百分比
    take_profit_pct = 10.0      # 止盈百分比
    
    # 内部变量
    current_price = 0.0
    entry_price = 0.0
    highest_price = 0.0         # 持仓期间最高价（移动止损用）
    lowest_price = float('inf') # 持仓期间最低价
    
    parameters = ["risk_percent", "max_position", "stop_loss_pct", "take_profit_pct"]
    variables = ["current_price", "entry_price", "highest_price", "lowest_price"]
    
    def __init__(self, cta_engine, strategy_name, vt_symbol, setting):
        """构造函数"""
        super().__init__(cta_engine, strategy_name, vt_symbol, setting)
        
    def on_init(self):
        """策略初始化回调"""
        self.write_log("策略初始化")
        self.load_bar(10)  # 加载10天历史数据
        
    def on_start(self):
        """策略启动回调"""
        self.write_log("策略启动")
        
    def on_stop(self):
        """策略停止回调"""
        self.write_log("策略停止")
        
    def on_tick(self, tick: TickData):
        """Tick数据回调"""
        self.current_price = tick.last_price
        
        # 更新最高最低价
        if self.pos > 0:
            self.highest_price = max(self.highest_price, tick.high_price)
        elif self.pos < 0:
            self.lowest_price = min(self.lowest_price, tick.low_price)
    
    def on_bar(self, bar: BarData):
        """K线数据回调 - 子类必须实现"""
        self.current_price = bar.close_price
        
        # 更新最高最低价
        if self.pos > 0:
            self.highest_price = max(self.highest_price, bar.high_price)
        elif self.pos < 0:
            self.lowest_price = min(self.lowest_price, bar.low_price)
            
        # 调用风控检查
        self.risk_management(bar)
        
        # 调用交易逻辑（子类实现）
        self.trading_logic(bar)
    
    def trading_logic(self, bar: BarData):
        """交易逻辑 - 子类必须重写"""
        raise NotImplementedError("子类必须实现trading_logic方法")
    
    def risk_management(self, bar: BarData):
        """风控管理"""
        if self.pos == 0:
            return
            
        # 止损检查
        if self.pos > 0:  # 多头持仓
            loss_pct = (self.entry_price - bar.close_price) / self.entry_price * 100
            if loss_pct >= self.stop_loss_pct:
                self.write_log(f"触发止损: 亏损 {loss_pct:.2f}%")
                self.sell(bar.close_price, abs(self.pos))
                return
                
            # 移动止盈（回撤止盈）
            drawdown_pct = (self.highest_price - bar.close_price) / self.highest_price * 100
            if drawdown_pct >= self.take_profit_pct:
                self.write_log(f"触发移动止盈: 回撤 {drawdown_pct:.2f}%")
                self.sell(bar.close_price, abs(self.pos))
                return
                
        elif self.pos < 0:  # 空头持仓
            loss_pct = (bar.close_price - self.entry_price) / self.entry_price * 100
            if loss_pct >= self.stop_loss_pct:
                self.write_log(f"触发止损: 亏损 {loss_pct:.2f}%")
                self.cover(bar.close_price, abs(self.pos))
                return
    
    def on_trade(self, trade: TradeData):
        """成交回调"""
        if trade.offset == Offset.OPEN:
            self.entry_price = trade.price
            self.highest_price = trade.price
            self.lowest_price = trade.price
            self.write_log(f"开仓成交: {trade.direction.value} {trade.volume}@{trade.price}")
        else:
            pnl = (trade.price - self.entry_price) * trade.volume if trade.direction == Direction.SHORT else \
                  (self.entry_price - trade.price) * trade.volume
            self.write_log(f"平仓成交: {trade.direction.value} {trade.volume}@{trade.price}, 盈亏: {pnl:.2f}")
            self.entry_price = 0.0
            
    def on_order(self, order: OrderData):
        """委托回调"""
        pass
    
    def on_stop_order(self, stop_order: StopOrder):
        """停止单回调"""
        pass
    
    def calculate_position(self, capital: float, price: float) -> int:
        """计算仓位大小"""
        risk_amount = capital * self.risk_percent / 100
        stop_loss_amount = price * self.stop_loss_pct / 100
        position = int(risk_amount / stop_loss_amount)
        return min(position, self.max_position)
