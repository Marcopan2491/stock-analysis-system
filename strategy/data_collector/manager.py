"""
数据采集模块 - 多数据源适配器
数据源优先级：
1. vn.py内置接口 (免费)
2. MiniQMT (需券商权限)
3. AKShare (免费开源)
4. 网页爬虫 (备选)
"""

from abc import ABC, abstractmethod
from datetime import datetime
from typing import List, Optional, Dict, Any
import pandas as pd
import logging

logger = logging.getLogger(__name__)


class DataSourceAdapter(ABC):
    """数据源适配器基类"""
    
    def __init__(self, name: str):
        self.name = name
        self.is_available = False
    
    @abstractmethod
    def connect(self) -> bool:
        """连接数据源"""
        pass
    
    @abstractmethod
    def get_stock_list(self) -> pd.DataFrame:
        """获取股票列表"""
        pass
    
    @abstractmethod
    def get_daily_bars(self, symbol: str, start: datetime, end: datetime) -> pd.DataFrame:
        """获取日K线数据"""
        pass
    
    @abstractmethod
    def get_minute_bars(self, symbol: str, start: datetime, end: datetime, freq: str = "1min") -> pd.DataFrame:
        """获取分钟K线数据"""
        pass
    
    @abstractmethod
    def get_realtime_quote(self, symbols: List[str]) -> pd.DataFrame:
        """获取实时行情"""
        pass


class VnpyAdapter(DataSourceAdapter):
    """
    vn.py内置数据适配器
    使用新浪财经/腾讯/东方财富免费接口
    """
    
    def __init__(self):
        super().__init__("vnpy_builtin")
        self.database = None
    
    def connect(self) -> bool:
        """初始化vn.py数据管理器"""
        try:
            from vnpy.trader.database import get_database
            from vnpy.trader.setting import SETTINGS
            
            # 配置数据库
            SETTINGS["database.database"] = "stock_market"
            SETTINGS["database.host"] = "localhost"
            SETTINGS["database.port"] = 8086
            SETTINGS["database.driver"] = "influxdb"
            
            self.database = get_database()
            self.is_available = True
            logger.info("vn.py数据适配器连接成功")
            return True
        except Exception as e:
            logger.error(f"vn.py数据适配器连接失败: {e}")
            self.is_available = False
            return False
    
    def get_stock_list(self) -> pd.DataFrame:
        """获取沪深A股列表"""
        try:
            # vn.py本身不提供股票列表接口，需要从其他源获取
            # 这里可以调用akshare获取
            import akshare as ak
            df = ak.stock_zh_a_spot_em()
            return df[["代码", "名称"]].rename(columns={"代码": "symbol", "名称": "name"})
        except Exception as e:
            logger.error(f"获取股票列表失败: {e}")
            return pd.DataFrame()
    
    def get_daily_bars(self, symbol: str, start: datetime, end: datetime) -> pd.DataFrame:
        """从vn.py数据库获取日K线"""
        if not self.is_available:
            return pd.DataFrame()
        
        try:
            from vnpy.trader.constant import Exchange, Interval
            
            # 确定交易所
            exchange = Exchange.SZSE if symbol.startswith(("000", "001", "002", "300")) else Exchange.SSE
            
            bars = self.database.load_bar_data(
                symbol=symbol,
                exchange=exchange,
                interval=Interval.DAILY,
                start=start,
                end=end
            )
            
            if not bars:
                return pd.DataFrame()
            
            data = []
            for bar in bars:
                data.append({
                    "datetime": bar.datetime,
                    "open": bar.open_price,
                    "high": bar.high_price,
                    "low": bar.low_price,
                    "close": bar.close_price,
                    "volume": bar.volume,
                    "turnover": bar.turnover
                })
            
            return pd.DataFrame(data)
        except Exception as e:
            logger.error(f"获取日K线失败 {symbol}: {e}")
            return pd.DataFrame()
    
    def get_minute_bars(self, symbol: str, start: datetime, end: datetime, freq: str = "1min") -> pd.DataFrame:
        """获取分钟K线"""
        if not self.is_available:
            return pd.DataFrame()
        
        try:
            from vnpy.trader.constant import Exchange, Interval
            
            exchange = Exchange.SZSE if symbol.startswith(("000", "001", "002", "300")) else Exchange.SSE
            
            interval_map = {
                "1min": Interval.MINUTE,
                "5min": Interval.MINUTE_5,
                "15min": Interval.MINUTE_15,
                "30min": Interval.MINUTE_30,
                "60min": Interval.HOUR
            }
            
            bars = self.database.load_bar_data(
                symbol=symbol,
                exchange=exchange,
                interval=interval_map.get(freq, Interval.MINUTE),
                start=start,
                end=end
            )
            
            if not bars:
                return pd.DataFrame()
            
            data = []
            for bar in bars:
                data.append({
                    "datetime": bar.datetime,
                    "open": bar.open_price,
                    "high": bar.high_price,
                    "low": bar.low_price,
                    "close": bar.close_price,
                    "volume": bar.volume,
                    "turnover": bar.turnover
                })
            
            return pd.DataFrame(data)
        except Exception as e:
            logger.error(f"获取分钟K线失败 {symbol}: {e}")
            return pd.DataFrame()
    
    def get_realtime_quote(self, symbols: List[str]) -> pd.DataFrame:
        """获取实时行情 - vn.py需通过gateway，这里使用akshare作为补充"""
        try:
            import akshare as ak
            df = ak.stock_zh_a_spot_em()
            df = df[df["代码"].isin(symbols)]
            return df
        except Exception as e:
            logger.error(f"获取实时行情失败: {e}")
            return pd.DataFrame()


class AkshareAdapter(DataSourceAdapter):
    """
    AKShare数据适配器
    完全免费的开源金融数据接口
    """
    
    def __init__(self):
        super().__init__("akshare")
    
    def connect(self) -> bool:
        """AKShare无需连接，直接可用"""
        try:
            import akshare as ak
            # 测试接口
            _ = ak.stock_zh_a_spot_em()
            self.is_available = True
            logger.info("AKShare适配器可用")
            return True
        except Exception as e:
            logger.error(f"AKShare适配器测试失败: {e}")
            self.is_available = False
            return False
    
    def get_stock_list(self) -> pd.DataFrame:
        """获取沪深A股列表"""
        try:
            import akshare as ak
            df = ak.stock_zh_a_spot_em()
            df = df.rename(columns={
                "代码": "symbol",
                "名称": "name",
                "最新价": "price",
                "涨跌幅": "change_pct",
                "成交量": "volume"
            })
            return df[["symbol", "name", "price", "change_pct", "volume"]]
        except Exception as e:
            logger.error(f"AKShare获取股票列表失败: {e}")
            return pd.DataFrame()
    
    def get_daily_bars(self, symbol: str, start: datetime, end: datetime) -> pd.DataFrame:
        """获取日K线数据"""
        try:
            import akshare as ak
            
            start_str = start.strftime("%Y%m%d")
            end_str = end.strftime("%Y%m%d")
            
            df = ak.stock_zh_a_hist(
                symbol=symbol,
                period="daily",
                start_date=start_str,
                end_date=end_str,
                adjust="qfq"  # 前复权
            )
            
            if df.empty:
                return pd.DataFrame()
            
            df = df.rename(columns={
                "日期": "datetime",
                "开盘": "open",
                "收盘": "close",
                "最高": "high",
                "最低": "low",
                "成交量": "volume",
                "成交额": "turnover"
            })
            
            df["datetime"] = pd.to_datetime(df["datetime"])
            return df[["datetime", "open", "high", "low", "close", "volume", "turnover"]]
        except Exception as e:
            logger.error(f"AKShare获取日K线失败 {symbol}: {e}")
            return pd.DataFrame()
    
    def get_minute_bars(self, symbol: str, start: datetime, end: datetime, freq: str = "1min") -> pd.DataFrame:
        """获取分钟K线数据"""
        try:
            import akshare as ak
            
            period_map = {
                "1min": "1",
                "5min": "5",
                "15min": "15",
                "30min": "30",
                "60min": "60"
            }
            
            df = ak.stock_zh_a_hist_min_em(
                symbol=symbol,
                period=period_map.get(freq, "1"),
                adjust="qfq"
            )
            
            if df.empty:
                return pd.DataFrame()
            
            df = df.rename(columns={
                "时间": "datetime",
                "开盘": "open",
                "收盘": "close",
                "最高": "high",
                "最低": "low",
                "成交量": "volume",
                "成交额": "turnover"
            })
            
            df["datetime"] = pd.to_datetime(df["datetime"])
            
            # 过滤时间范围
            df = df[(df["datetime"] >= start) & (df["datetime"] <= end)]
            
            return df[["datetime", "open", "high", "low", "close", "volume", "turnover"]]
        except Exception as e:
            logger.error(f"AKShare获取分钟K线失败 {symbol}: {e}")
            return pd.DataFrame()
    
    def get_realtime_quote(self, symbols: List[str]) -> pd.DataFrame:
        """获取实时行情"""
        try:
            import akshare as ak
            df = ak.stock_zh_a_spot_em()
            df = df[df["代码"].isin(symbols)]
            return df
        except Exception as e:
            logger.error(f"AKShare获取实时行情失败: {e}")
            return pd.DataFrame()


class MiniQMTAdapter(DataSourceAdapter):
    """
    MiniQMT数据适配器
    需要开通QMT权限的券商账户
    """
    
    def __init__(self):
        super().__init__("miniqmt")
        self.connected = False
    
    def connect(self) -> bool:
        """连接MiniQMT"""
        try:
            from xtquant import xtdata
            # 测试连接
            xtdata.get_stock_list_in_sector("沪深A股")
            self.is_available = True
            self.connected = True
            logger.info("MiniQMT连接成功")
            return True
        except ImportError:
            logger.warning("MiniQMT SDK未安装，跳过")
            self.is_available = False
            return False
        except Exception as e:
            logger.error(f"MiniQMT连接失败: {e}")
            self.is_available = False
            return False
    
    def get_stock_list(self) -> pd.DataFrame:
        """获取股票列表"""
        if not self.connected:
            return pd.DataFrame()
        
        try:
            from xtquant import xtdata
            stocks = xtdata.get_stock_list_in_sector("沪深A股")
            # 转换为DataFrame
            data = []
            for stock in stocks:
                symbol = stock.split(".")[0]
                exchange = "SH" if ".SH" in stock else "SZ"
                data.append({"symbol": symbol, "exchange": exchange})
            return pd.DataFrame(data)
        except Exception as e:
            logger.error(f"MiniQMT获取股票列表失败: {e}")
            return pd.DataFrame()
    
    def get_daily_bars(self, symbol: str, start: datetime, end: datetime) -> pd.DataFrame:
        """获取日K线数据"""
        if not self.connected:
            return pd.DataFrame()
        
        try:
            from xtquant import xtdata
            
            exchange = "SH" if symbol.startswith(("600", "601", "603", "688", "689")) else "SZ"
            full_symbol = f"{symbol}.{exchange}"
            
            start_str = start.strftime("%Y%m%d")
            end_str = end.strftime("%Y%m%d")
            
            # 下载历史数据
            xtdata.download_history_data(full_symbol, "1d", start_str, end_str)
            
            # 获取数据
            data = xtdata.get_local_data([full_symbol], "1d", start_str, end_str)
            
            if data.empty:
                return pd.DataFrame()
            
            df = data.reset_index()
            df = df.rename(columns={
                "time": "datetime",
                "open": "open",
                "high": "high",
                "low": "low",
                "close": "close",
                "volume": "volume"
            })
            
            return df[["datetime", "open", "high", "low", "close", "volume"]]
        except Exception as e:
            logger.error(f"MiniQMT获取日K线失败 {symbol}: {e}")
            return pd.DataFrame()
    
    def get_minute_bars(self, symbol: str, start: datetime, end: datetime, freq: str = "1min") -> pd.DataFrame:
        """获取分钟K线数据"""
        if not self.connected:
            return pd.DataFrame()
        
        try:
            from xtquant import xtdata
            
            exchange = "SH" if symbol.startswith(("600", "601", "603", "688", "689")) else "SZ"
            full_symbol = f"{symbol}.{exchange}"
            
            period_map = {
                "1min": "1m",
                "5min": "5m",
                "15min": "15m",
                "30min": "30m",
                "60min": "1h"
            }
            
            start_str = start.strftime("%Y%m%d")
            end_str = end.strftime("%Y%m%d")
            period = period_map.get(freq, "1m")
            
            xtdata.download_history_data(full_symbol, period, start_str, end_str)
            data = xtdata.get_local_data([full_symbol], period, start_str, end_str)
            
            if data.empty:
                return pd.DataFrame()
            
            df = data.reset_index()
            return df
        except Exception as e:
            logger.error(f"MiniQMT获取分钟K线失败 {symbol}: {e}")
            return pd.DataFrame()
    
    def get_realtime_quote(self, symbols: List[str]) -> pd.DataFrame:
        """获取实时行情"""
        if not self.connected:
            return pd.DataFrame()
        
        try:
            from xtquant import xtdata
            
            full_symbols = []
            for s in symbols:
                exchange = "SH" if s.startswith(("600", "601", "603", "688")) else "SZ"
                full_symbols.append(f"{s}.{exchange}")
            
            data = xtdata.get_full_tick(full_symbols)
            
            # 转换为DataFrame
            records = []
            for symbol, tick in data.items():
                records.append({
                    "symbol": symbol.split(".")[0],
                    "price": tick["lastPrice"],
                    "volume": tick["volume"]
                })
            
            return pd.DataFrame(records)
        except Exception as e:
            logger.error(f"MiniQMT获取实时行情失败: {e}")
            return pd.DataFrame()


class DataManager:
    """
    数据管理器 - 统一管理多个数据源
    按优先级自动选择可用数据源
    """
    
    def __init__(self):
        self.adapters: List[DataSourceAdapter] = [
            VnpyAdapter(),      # 优先级1
            MiniQMTAdapter(),   # 优先级2
            AkshareAdapter(),   # 优先级3
        ]
        self._init_adapters()
    
    def _init_adapters(self):
        """初始化所有适配器"""
        for adapter in self.adapters:
            adapter.connect()
    
    def _get_available_adapter(self) -> Optional[DataSourceAdapter]:
        """获取第一个可用的适配器"""
        for adapter in self.adapters:
            if adapter.is_available:
                return adapter
        return None
    
    def get_stock_list(self) -> pd.DataFrame:
        """获取股票列表"""
        adapter = self._get_available_adapter()
        if adapter:
            return adapter.get_stock_list()
        logger.error("没有可用的数据源")
        return pd.DataFrame()
    
    def get_daily_bars(self, symbol: str, start: datetime, end: datetime) -> pd.DataFrame:
        """获取日K线数据"""
        adapter = self._get_available_adapter()
        if adapter:
            return adapter.get_daily_bars(symbol, start, end)
        logger.error("没有可用的数据源")
        return pd.DataFrame()
    
    def get_minute_bars(self, symbol: str, start: datetime, end: datetime, freq: str = "1min") -> pd.DataFrame:
        """获取分钟K线数据"""
        adapter = self._get_available_adapter()
        if adapter:
            return adapter.get_minute_bars(symbol, start, end, freq)
        logger.error("没有可用的数据源")
        return pd.DataFrame()
    
    def get_realtime_quote(self, symbols: List[str]) -> pd.DataFrame:
        """获取实时行情"""
        adapter = self._get_available_adapter()
        if adapter:
            return adapter.get_realtime_quote(symbols)
        logger.error("没有可用的数据源")
        return pd.DataFrame()


# 全局数据管理器实例
data_manager = DataManager()


if __name__ == "__main__":
    # 测试数据管理器
    logging.basicConfig(level=logging.INFO)
    
    dm = DataManager()
    
    # 测试获取股票列表
    print("测试获取股票列表...")
    stocks = dm.get_stock_list()
    print(f"获取到 {len(stocks)} 只股票")
    if not stocks.empty:
        print(stocks.head())
    
    # 测试获取日K线
    print("\n测试获取日K线...")
    from datetime import datetime, timedelta
    end = datetime.now()
    start = end - timedelta(days=30)
    bars = dm.get_daily_bars("000001", start, end)
    print(f"获取到 {len(bars)} 条日K线")
    if not bars.empty:
        print(bars.head())
    
    # 测试获取分钟K线
    print("\n测试获取分钟K线...")
    minute_bars = dm.get_minute_bars("000001", start, end, "5min")
    print(f"获取到 {len(minute_bars)} 条分钟K线")
    if not minute_bars.empty:
        print(minute_bars.head())
