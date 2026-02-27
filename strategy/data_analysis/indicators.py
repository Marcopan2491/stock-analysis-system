"""
技术指标计算引擎
"""
import pandas as pd
import numpy as np


class IndicatorEngine:
    """技术指标计算引擎"""
    
    @staticmethod
    def add_ma(df: pd.DataFrame, periods: list = [5, 10, 20, 60]) -> pd.DataFrame:
        """添加移动平均线"""
        for period in periods:
            df[f'ma{period}'] = df['close'].rolling(window=period).mean()
        return df
    
    @staticmethod
    def add_ema(df: pd.DataFrame, periods: list = [12, 26]) -> pd.DataFrame:
        """添加指数移动平均线"""
        for period in periods:
            df[f'ema{period}'] = df['close'].ewm(span=period, adjust=False).mean()
        return df
    
    @staticmethod
    def add_macd(df: pd.DataFrame, fast: int = 12, slow: int = 26, signal: int = 9) -> pd.DataFrame:
        """添加MACD指标"""
        ema_fast = df['close'].ewm(span=fast, adjust=False).mean()
        ema_slow = df['close'].ewm(span=slow, adjust=False).mean()
        df['macd_dif'] = ema_fast - ema_slow
        df['macd_dea'] = df['macd_dif'].ewm(span=signal, adjust=False).mean()
        df['macd_hist'] = 2 * (df['macd_dif'] - df['macd_dea'])
        return df
    
    @staticmethod
    def add_kdj(df: pd.DataFrame, n: int = 9, m1: int = 3, m2: int = 3) -> pd.DataFrame:
        """添加KDJ指标"""
        low_list = df['low'].rolling(window=n, min_periods=n).min()
        high_list = df['high'].rolling(window=n, min_periods=n).max()
        rsv = (df['close'] - low_list) / (high_list - low_list) * 100
        
        df['kdj_k'] = rsv.ewm(alpha=1/m1, adjust=False).mean()
        df['kdj_d'] = df['kdj_k'].ewm(alpha=1/m2, adjust=False).mean()
        df['kdj_j'] = 3 * df['kdj_k'] - 2 * df['kdj_d']
        return df
    
    @staticmethod
    def add_rsi(df: pd.DataFrame, periods: list = [6, 12, 24]) -> pd.DataFrame:
        """添加RSI指标"""
        for period in periods:
            delta = df['close'].diff()
            gain = (delta.where(delta > 0, 0)).rolling(window=period).mean()
            loss = (-delta.where(delta < 0, 0)).rolling(window=period).mean()
            rs = gain / loss
            df[f'rsi{period}'] = 100 - (100 / (1 + rs))
        return df
    
    @staticmethod
    def add_boll(df: pd.DataFrame, period: int = 20, std: int = 2) -> pd.DataFrame:
        """添加布林带"""
        df['boll_mid'] = df['close'].rolling(window=period).mean()
        df['boll_std'] = df['close'].rolling(window=period).std()
        df['boll_up'] = df['boll_mid'] + std * df['boll_std']
        df['boll_down'] = df['boll_mid'] - std * df['boll_std']
        return df
    
    @staticmethod
    def add_atr(df: pd.DataFrame, period: int = 14) -> pd.DataFrame:
        """添加ATR指标"""
        high_low = df['high'] - df['low']
        high_close = np.abs(df['high'] - df['close'].shift())
        low_close = np.abs(df['low'] - df['close'].shift())
        ranges = pd.concat([high_low, high_close, low_close], axis=1)
        true_range = np.max(ranges, axis=1)
        df[f'atr{period}'] = true_range.rolling(period).mean()
        return df
    
    @staticmethod
    def detect_ma_cross(df: pd.DataFrame, fast: int = 5, slow: int = 10) -> pd.DataFrame:
        """检测均线金叉/死叉"""
        df['ma_diff'] = df[f'ma{fast}'] - df[f'ma{slow}']
        df['cross_signal'] = 0
        df.loc[(df['ma_diff'] > 0) & (df['ma_diff'].shift(1) <= 0), 'cross_signal'] = 1  # 金叉
        df.loc[(df['ma_diff'] < 0) & (df['ma_diff'].shift(1) >= 0), 'cross_signal'] = -1  # 死叉
        return df
    
    @staticmethod
    def calculate_all(df: pd.DataFrame) -> pd.DataFrame:
        """计算所有常用指标"""
        df = IndicatorEngine.add_ma(df)
        df = IndicatorEngine.add_ema(df)
        df = IndicatorEngine.add_macd(df)
        df = IndicatorEngine.add_kdj(df)
        df = IndicatorEngine.add_rsi(df)
        df = IndicatorEngine.add_boll(df)
        df = IndicatorEngine.add_atr(df)
        return df


# 测试
if __name__ == "__main__":
    # 创建测试数据
    import numpy as np
    np.random.seed(42)
    
    dates = pd.date_range('2024-01-01', periods=100, freq='D')
    df = pd.DataFrame({
        'datetime': dates,
        'open': np.random.randn(100).cumsum() + 100,
        'high': np.random.randn(100).cumsum() + 101,
        'low': np.random.randn(100).cumsum() + 99,
        'close': np.random.randn(100).cumsum() + 100,
        'volume': np.random.randint(1000000, 10000000, 100)
    })
    
    # 确保价格合理性
    df['high'] = df[['open', 'close', 'high']].max(axis=1)
    df['low'] = df[['open', 'close', 'low']].min(axis=1)
    
    # 计算指标
    df = IndicatorEngine.calculate_all(df)
    print(df.tail())
    print("\n指标列:", df.columns.tolist())