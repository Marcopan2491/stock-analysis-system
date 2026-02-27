package models

import (
	"time"

	"gorm.io/gorm"
)

// Stock 股票基础信息模型
type Stock struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Symbol       string    `gorm:"size:10;not null;index;uniqueIndex:idx_symbol_exchange" json:"symbol"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Exchange     string    `gorm:"size:10;not null;index;uniqueIndex:idx_symbol_exchange" json:"exchange"`
	Industry     string    `gorm:"size:50;index" json:"industry"`
	FullName     string    `gorm:"size:200" json:"full_name"`
	ListDate     *time.Time `json:"list_date"`
	TotalShare   int64     `json:"total_share"`
	FloatShare   int64     `json:"float_share"`
	Status       string    `gorm:"size:10;default:'active'" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Stock) TableName() string {
	return "stocks"
}

// IsActive 检查股票是否活跃
func (s *Stock) IsActive() bool {
	return s.Status == "active"
}

// GetFullCode 获取完整代码 (symbol.exchange)
func (s *Stock) GetFullCode() string {
	return s.Symbol + "." + s.Exchange
}

// DailyBar 日K线数据模型 (用于InfluxDB)
type DailyBar struct {
	Symbol   string    `json:"symbol"`
	Exchange string    `json:"exchange"`
	Date     time.Time `json:"date"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   int64     `json:"volume"`
	Amount   float64   `json:"amount"`
}

// MinuteBar 分钟K线数据模型 (用于InfluxDB)
type MinuteBar struct {
	Symbol   string    `json:"symbol"`
	Exchange string    `json:"exchange"`
	Interval string    `json:"interval"` // 1m, 5m, 15m, 30m, 60m
	Time     time.Time `json:"time"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   int64     `json:"volume"`
	Amount   float64   `json:"amount"`
}

// Indicator 技术指标模型 (用于InfluxDB)
type Indicator struct {
	Symbol        string    `json:"symbol"`
	Exchange      string    `json:"exchange"`
	Date          time.Time `json:"date"`
	IndicatorType string    `json:"indicator_type"` // ma, macd, rsi, kdj, boll
	// MA指标
	MA5   float64 `json:"ma5,omitempty"`
	MA10  float64 `json:"ma10,omitempty"`
	MA20  float64 `json:"ma20,omitempty"`
	MA30  float64 `json:"ma30,omitempty"`
	MA60  float64 `json:"ma60,omitempty"`
	MA120 float64 `json:"ma120,omitempty"`
	MA250 float64 `json:"ma250,omitempty"`
	// MACD指标
	MACD      float64 `json:"macd,omitempty"`
	MACDSignal float64 `json:"macd_signal,omitempty"`
	MACDHist  float64 `json:"macd_hist,omitempty"`
	// RSI指标
	RSI6  float64 `json:"rsi6,omitempty"`
	RSI12 float64 `json:"rsi12,omitempty"`
	RSI24 float64 `json:"rsi24,omitempty"`
	// KDJ指标
	K float64 `json:"k,omitempty"`
	D float64 `json:"d,omitempty"`
	J float64 `json:"j,omitempty"`
	// BOLL指标
	BollUpper float64 `json:"boll_upper,omitempty"`
	BollMid   float64 `json:"boll_mid,omitempty"`
	BollLower float64 `json:"boll_lower,omitempty"`
}

// User 用户模型
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Email        string     `gorm:"size:100;not null;uniqueIndex" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	AvatarURL    string     `gorm:"size:500" json:"avatar_url"`
	Phone        string     `gorm:"size:20" json:"phone"`
	Status       string     `gorm:"size:10;default:'active'" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// Strategy 策略模型
type Strategy struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `json:"description"`
	Type        string         `gorm:"size:50;not null;index" json:"type"`
	ClassName   string         `gorm:"size:100;not null" json:"class_name"`
	Params      string         `gorm:"type:jsonb" json:"params"`
	Symbols     string         `gorm:"type:text[]" json:"symbols"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	IsPublic    bool           `gorm:"default:false" json:"is_public"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TableName 指定表名
func (Strategy) TableName() string {
	return "strategies"
}

// TradeSignal 交易信号模型
type TradeSignal struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	StrategyID uint      `gorm:"not null;index" json:"strategy_id"`
	Symbol     string    `gorm:"size:10;not null;index" json:"symbol"`
	Exchange   string    `gorm:"size:10;not null" json:"exchange"`
	SignalType string    `gorm:"size:10;not null;index" json:"signal_type"` // buy, sell, close
	Price      float64   `json:"price"`
	Volume     int       `json:"volume"`
	Reason     string    `json:"reason"`
	Confidence float64   `json:"confidence"`
	IsExecuted bool      `gorm:"default:false" json:"is_executed"`
	ExecutedAt *time.Time `json:"executed_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName 指定表名
func (TradeSignal) TableName() string {
	return "trade_signals"
}

// BacktestRecord 回测记录模型
type BacktestRecord struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	StrategyID     uint       `gorm:"not null;index" json:"strategy_id"`
	StartDate      time.Time  `json:"start_date"`
	EndDate        time.Time  `json:"end_date"`
	InitialCapital float64    `json:"initial_capital"`
	FinalCapital   float64    `json:"final_capital"`
	TotalReturn    float64    `json:"total_return"`
	AnnualReturn   float64    `json:"annual_return"`
	MaxDrawdown    float64    `json:"max_drawdown"`
	SharpeRatio    float64    `json:"sharpe_ratio"`
	WinRate        float64    `json:"win_rate"`
	ProfitLossRatio float64   `json:"profit_loss_ratio"`
	TradeCount     int        `json:"trade_count"`
	Params         string     `gorm:"type:jsonb" json:"params"`
	ResultData     string     `gorm:"type:jsonb" json:"result_data"`
	Status         string     `gorm:"size:20;default:'running'" json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

// TableName 指定表名
func (BacktestRecord) TableName() string {
	return "backtest_records"
}

// Watchlist 自选股分组模型
type Watchlist struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	UserID      uint            `gorm:"not null;index" json:"user_id"`
	Name        string          `gorm:"size:50;not null" json:"name"`
	Description string          `json:"description"`
	Items       []*WatchlistItem `json:"items,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// TableName 指定表名
func (Watchlist) TableName() string {
	return "watchlists"
}

// WatchlistItem 自选股明细模型
type WatchlistItem struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WatchlistID uint      `gorm:"not null;index" json:"watchlist_id"`
	Symbol      string    `gorm:"size:10;not null" json:"symbol"`
	Exchange    string    `gorm:"size:10;not null" json:"exchange"`
	AddedAt     time.Time `json:"added_at"`
}

// TableName 指定表名
func (WatchlistItem) TableName() string {
	return "watchlist_items"
}
