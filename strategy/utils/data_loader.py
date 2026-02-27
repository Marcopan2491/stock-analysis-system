"""
数据加载器 - 免费数据源适配器
优先顺序：vn.py内置 -> MiniQMT -> AKShare -> 爬虫
"""
import os
from datetime import datetime
from typing import List, Optional
import pandas as pd


class DataSourceAdapter:
    """数据源适配器基类"""
    
    def get_stock_list(self) -> pd.DataFrame:
        raise NotImplementedError
    
    def get_daily_bars(self, symbol: str, start: str, end: str) -> pd.DataFrame:
        raise NotImplementedError
    
    def get_minute_bars(self, symbol: str, start: str, end: str, freq: str = "1min") -> pd.DataFrame:
        raise NotImplementedError


class VnpyAdapter(DataSourceAdapter):
    """vn.py内置数据源适配器"""
    
    def __init__(self):
        try:
            from vnpy.trader.database import get_database
            from vnpy.trader.datafeed import get_datafeed
            from vnpy.trader.constant import Exchange, Interval
            
            self.db = get_database()
            self.Exchange = Exchange
            self.Interval = Interval
            self.available = True
        except Exception as e:
            print(f"vn.py数据源不可用: {e}")
            self.available = False
    
    def get_stock_list(self) -> pd.DataFrame:
        """从vn.py获取股票列表"""
        if not self.available:
            return pd.DataFrame()
        
        # vn.py本身不提供股票列表，需要配合其他方式
        # 这里返回空，让上层使用其他数据源
        return pd.DataFrame()
    
    def get_daily_bars(self, symbol: str, start: str, end: str) -> pd.DataFrame:
        """获取日K数据"""
        if not self.available:
            return pd.DataFrame()
        
        try:
            # 解析symbol (格式: 000001.SZ)
            code, exchange_str = symbol.split(".")
            exchange = getattr(self.Exchange, exchange_str)
            
            bars = self.db.load_bar_data(
                symbol=code,
                exchange=exchange,
                interval=self.Interval.DAILY,
                start=datetime.strptime(start, "%Y%m%d"),
                end=datetime.strptime(end, "%Y%m%d")
            )
            
            if bars:
                df = pd.DataFrame([{
                    'datetime': bar.datetime,
                    'open': bar.open_price,
                    'high': bar.high_price,
                    'low': bar.low_price,
                    'close': bar.close_price,
                    'volume': bar.volume,
                } for bar in bars])
                return df
            return pd.DataFrame()
        except Exception as e:
            print(f"vn.py获取数据失败: {e}")
            return pd.DataFrame()


class MiniQMTAdapter(DataSourceAdapter):
    """MiniQMT数据源适配器"""
    
    def __init__(self, qmt_path: Optional[str] = None):
        self.qmt_path = qmt_path or os.getenv("QMT_PATH")
        self.available = self._check_available()
    
    def _check_available(self) -> bool:
        try:
            import sys
            if self.qmt_path:
                sys.path.append(self.qmt_path)
            from xtquant import xtdata
            return True
        except ImportError:
            print("MiniQMT不可用，请检查QMT安装路径")
            return False
    
    def get_stock_list(self) -> pd.DataFrame:
        """获取股票列表"""
        if not self.available:
            return pd.DataFrame()
        
        try:
            from xtquant import xtdata
            stocks = xtdata.get_stock_list_in_sector('沪深A股')
            df = pd.DataFrame({'symbol': stocks})
            return df
        except Exception as e:
            print(f"MiniQMT获取股票列表失败: {e}")
            return pd.DataFrame()
    
    def get_daily_bars(self, symbol: str, start: str, end: str) -> pd.DataFrame:
        """获取日K数据"""
        if not self.available:
            return pd.DataFrame()
        
        try:
            from xtquant import xtdata
            # MiniQMT格式: 000001.SZ
            xtdata.download_history_data(symbol, '1d', start, end)
            data = xtdata.get_local_data(symbol, '1d', start, end)
            
            if data and len(data) > 0:
                df = pd.DataFrame(data)
                df.rename(columns={
                    'time': 'datetime',
                    'open': 'open',
                    'high': 'high',
                    'low': 'low',
                    'close': 'close',
                    'volume': 'volume',
                }, inplace=True)
                return df
            return pd.DataFrame()
        except Exception as e:
            print(f"MiniQMT获取数据失败: {e}")
            return pd.DataFrame()


class AKShareAdapter(DataSourceAdapter):
    """AKShare数据源适配器"""
    
    def __init__(self):
        self.available = self._check_available()
    
    def _check_available(self) -> bool:
        try:
            import akshare as ak
            return True
        except ImportError:
            print("AKShare未安装")
            return False
    
    def get_stock_list(self) -> pd.DataFrame:
        """获取A股列表"""
        if not self.available:
            return pd.DataFrame()
        
        try:
            import akshare as ak
            df = ak.stock_zh_a_spot_em()
            df = df[['代码', '名称']]
            df.columns = ['symbol', 'name']
            return df
        except Exception as e:
            print(f"AKShare获取股票列表失败: {e}")
            return pd.DataFrame()
    
    def get_daily_bars(self, symbol: str, start: str, end: str) -> pd.DataFrame:
        """获取日K数据"""
        if not self.available:
            return pd.DataFrame()
        
        try:
            import akshare as ak
            # AKShare格式: 000001，需要去掉后缀
            code = symbol.split('.')[0]
            df = ak.stock_zh_a_hist(
                symbol=code,
                period="daily",
                start_date=start,
                end_date=end,
                adjust="qfq"  # 前复权
            )
            
            if df is not None and len(df) > 0:
                df.rename(columns={
                    '日期': 'datetime',
                    '开盘': 'open',
                    '最高': 'high',
                    '最低': 'low',
                    '收盘': 'close',
                    '成交量': 'volume',
                    '成交额': 'amount',
                }, inplace=True)
                return df
            return pd.DataFrame()
        except Exception as e:
            print(f"AKShare获取数据失败: {e}")
            return pd.DataFrame()


class DataLoader:
    """数据加载器 - 按优先级选择数据源"""
    
    def __init__(self, qmt_path: Optional[str] = None):
        self.adapters = {
            'vnpy': VnpyAdapter(),
            'miniqmt': MiniQMTAdapter(qmt_path),
            'akshare': AKShareAdapter(),
        }
        self.priority = ['vnpy', 'miniqmt', 'akshare']
    
    def get_stock_list(self) -> pd.DataFrame:
        """获取股票列表"""
        for source in self.priority:
            adapter = self.adapters[source]
            if adapter.available:
                df = adapter.get_stock_list()
                if not df.empty:
                    print(f"使用 {source} 获取股票列表")
                    return df
        
        print("警告：所有数据源均不可用")
        return pd.DataFrame()
    
    def get_daily_bars(self, symbol: str, start: str, end: str) -> pd.DataFrame:
        """获取日K数据"""
        for source in self.priority:
            adapter = self.adapters[source]
            if adapter.available:
                df = adapter.get_daily_bars(symbol, start, end)
                if not df.empty:
                    print(f"使用 {source} 获取 {symbol} 日K数据")
                    return df
        
        print(f"警告：无法获取 {symbol} 数据")
        return pd.DataFrame()
    
    def download_daily_bars(self, symbol: str, start: str, end: str) -> bool:
        """下载日K数据并保存到数据库"""
        df = self.get_daily_bars(symbol, start, end)
        if df.empty:
            return False
        
        # TODO: 保存到InfluxDB
        print(f"已获取 {symbol} {len(df)} 条日K数据")
        return True


# 测试代码
if __name__ == "__main__":
    loader = DataLoader()
    
    # 测试获取股票列表
    stocks = loader.get_stock_list()
    print(f"股票数量: {len(stocks)}")
    if not stocks.empty:
        print(stocks.head())
    
    # 测试获取K线数据
    df = loader.get_daily_bars("000001.SZ", "20240101", "20241231")
    print(f"\nK线数据条数: {len(df)}")
    if not df.empty:
        print(df.head())