package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"

	"stock-analysis-system/backend/pkg/database"
	"stock-analysis-system/backend/pkg/models"
)

// MarketRepository 行情数据仓库接口
type MarketRepository interface {
	// 日K线数据操作
	SaveDailyBar(ctx context.Context, bar *models.DailyBar) error
	SaveDailyBars(ctx context.Context, bars []*models.DailyBar) error
	GetDailyBars(ctx context.Context, symbol, exchange string, start, end time.Time) ([]*models.DailyBar, error)
	GetLatestDailyBar(ctx context.Context, symbol, exchange string) (*models.DailyBar, error)
	
	// 分钟K线数据操作
	SaveMinuteBar(ctx context.Context, bar *models.MinuteBar) error
	SaveMinuteBars(ctx context.Context, bars []*models.MinuteBar) error
	GetMinuteBars(ctx context.Context, symbol, exchange, interval string, start, end time.Time) ([]*models.MinuteBar, error)
	
	// 技术指标操作
	SaveIndicator(ctx context.Context, indicator *models.Indicator) error
	SaveIndicators(ctx context.Context, indicators []*models.Indicator) error
	GetIndicators(ctx context.Context, symbol, exchange string, indicatorType string, start, end time.Time) ([]*models.Indicator, error)
	GetLatestIndicator(ctx context.Context, symbol, exchange string, indicatorType string) (*models.Indicator, error)
	
	// 数据完整性检查
	CheckDataIntegrity(ctx context.Context, symbol, exchange string, start, end time.Time) (map[string]interface{}, error)
}

// marketRepository 行情数据仓库实现
type marketRepository struct {
	influx *database.InfluxClient
}

// NewMarketRepository 创建行情数据仓库
func NewMarketRepository(influx *database.InfluxClient) MarketRepository {
	return &marketRepository{influx: influx}
}

// ============ 日K线数据操作 ============

// SaveDailyBar 保存单条日K线
func (r *marketRepository) SaveDailyBar(ctx context.Context, bar *models.DailyBar) error {
	point := write.NewPoint(
		"daily_bars",
		map[string]string{
			"symbol":   bar.Symbol,
			"exchange": bar.Exchange,
		},
		map[string]interface{}{
			"open":   bar.Open,
			"high":   bar.High,
			"low":    bar.Low,
			"close":  bar.Close,
			"volume": bar.Volume,
			"amount": bar.Amount,
		},
		bar.Date,
	)
	
	r.influx.WritePoint(point)
	r.influx.Flush()
	return nil
}

// SaveDailyBars 批量保存日K线
func (r *marketRepository) SaveDailyBars(ctx context.Context, bars []*models.DailyBar) error {
	points := make([]*write.Point, 0, len(bars))
	
	for _, bar := range bars {
		point := write.NewPoint(
			"daily_bars",
			map[string]string{
				"symbol":   bar.Symbol,
				"exchange": bar.Exchange,
			},
			map[string]interface{}{
				"open":   bar.Open,
				"high":   bar.High,
				"low":    bar.Low,
				"close":  bar.Close,
				"volume": bar.Volume,
				"amount": bar.Amount,
			},
			bar.Date,
		)
		points = append(points, point)
	}
	
	r.influx.WritePoints(points)
	r.influx.Flush()
	return nil
}

// GetDailyBars 查询日K线数据
func (r *marketRepository) GetDailyBars(ctx context.Context, symbol, exchange string, start, end time.Time) ([]*models.DailyBar, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "daily_bars")
		|> filter(fn: (r) => r.symbol == "%s")
		|> filter(fn: (r) => r.exchange == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> sort(columns: ["_time"])
	`, r.influx.GetBucket(), start.Format(time.RFC3339), end.Format(time.RFC3339), symbol, exchange)

	result, err := r.influx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询日K线失败: %w", err)
	}
	defer result.Close()

	var bars []*models.DailyBar
	for result.Next() {
		record := result.Record()
		bar := &models.DailyBar{
			Symbol:   symbol,
			Exchange: exchange,
			Date:     record.Time(),
		}
		
		if v, ok := record.ValueByKey("open").(float64); ok {
			bar.Open = v
		}
		if v, ok := record.ValueByKey("high").(float64); ok {
			bar.High = v
		}
		if v, ok := record.ValueByKey("low").(float64); ok {
			bar.Low = v
		}
		if v, ok := record.ValueByKey("close").(float64); ok {
			bar.Close = v
		}
		if v, ok := record.ValueByKey("volume").(int64); ok {
			bar.Volume = v
		}
		if v, ok := record.ValueByKey("amount").(float64); ok {
			bar.Amount = v
		}
		
		bars = append(bars, bar)
	}

	if result.Err() != nil {
		return nil, result.Err()
	}

	return bars, nil
}

// GetLatestDailyBar 获取最新日K线
func (r *marketRepository) GetLatestDailyBar(ctx context.Context, symbol, exchange string) (*models.DailyBar, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -30d)
		|> filter(fn: (r) => r._measurement == "daily_bars")
		|> filter(fn: (r) => r.symbol == "%s")
		|> filter(fn: (r) => r.exchange == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> sort(columns: ["_time"], desc: true)
		|> limit(n: 1)
	`, r.influx.GetBucket(), symbol, exchange)

	result, err := r.influx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询最新日K线失败: %w", err)
	}
	defer result.Close()

	if result.Next() {
		record := result.Record()
		bar := &models.DailyBar{
			Symbol:   symbol,
			Exchange: exchange,
			Date:     record.Time(),
		}
		
		if v, ok := record.ValueByKey("open").(float64); ok {
			bar.Open = v
		}
		if v, ok := record.ValueByKey("high").(float64); ok {
			bar.High = v
		}
		if v, ok := record.ValueByKey("low").(float64); ok {
			bar.Low = v
		}
		if v, ok := record.ValueByKey("close").(float64); ok {
			bar.Close = v
		}
		if v, ok := record.ValueByKey("volume").(int64); ok {
			bar.Volume = v
		}
		if v, ok := record.ValueByKey("amount").(float64); ok {
			bar.Amount = v
		}
		
		return bar, nil
	}

	return nil, nil
}

// ============ 分钟K线数据操作 ============

// SaveMinuteBar 保存单条分钟K线
func (r *marketRepository) SaveMinuteBar(ctx context.Context, bar *models.MinuteBar) error {
	point := write.NewPoint(
		"minute_bars",
		map[string]string{
			"symbol":   bar.Symbol,
			"exchange": bar.Exchange,
			"interval": bar.Interval,
		},
		map[string]interface{}{
			"open":   bar.Open,
			"high":   bar.High,
			"low":    bar.Low,
			"close":  bar.Close,
			"volume": bar.Volume,
			"amount": bar.Amount,
		},
		bar.Time,
	)
	
	r.influx.WritePoint(point)
	r.influx.Flush()
	return nil
}

// SaveMinuteBars 批量保存分钟K线
func (r *marketRepository) SaveMinuteBars(ctx context.Context, bars []*models.MinuteBar) error {
	points := make([]*write.Point, 0, len(bars))
	
	for _, bar := range bars {
		point := write.NewPoint(
			"minute_bars",
			map[string]string{
				"symbol":   bar.Symbol,
				"exchange": bar.Exchange,
				"interval": bar.Interval,
			},
			map[string]interface{}{
				"open":   bar.Open,
				"high":   bar.High,
				"low":    bar.Low,
				"close":  bar.Close,
				"volume": bar.Volume,
				"amount": bar.Amount,
			},
			bar.Time,
		)
		points = append(points, point)
	}
	
	r.influx.WritePoints(points)
	r.influx.Flush()
	return nil
}

// GetMinuteBars 查询分钟K线数据
func (r *marketRepository) GetMinuteBars(ctx context.Context, symbol, exchange, interval string, start, end time.Time) ([]*models.MinuteBar, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "minute_bars")
		|> filter(fn: (r) => r.symbol == "%s")
		|> filter(fn: (r) => r.exchange == "%s")
		|> filter(fn: (r) => r.interval == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> sort(columns: ["_time"])
	`, r.influx.GetBucket(), start.Format(time.RFC3339), end.Format(time.RFC3339), symbol, exchange, interval)

	result, err := r.influx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询分钟K线失败: %w", err)
	}
	defer result.Close()

	var bars []*models.MinuteBar
	for result.Next() {
		record := result.Record()
		bar := &models.MinuteBar{
			Symbol:   symbol,
			Exchange: exchange,
			Interval: interval,
			Time:     record.Time(),
		}
		
		if v, ok := record.ValueByKey("open").(float64); ok {
			bar.Open = v
		}
		if v, ok := record.ValueByKey("high").(float64); ok {
			bar.High = v
		}
		if v, ok := record.ValueByKey("low").(float64); ok {
			bar.Low = v
		}
		if v, ok := record.ValueByKey("close").(float64); ok {
			bar.Close = v
		}
		if v, ok := record.ValueByKey("volume").(int64); ok {
			bar.Volume = v
		}
		if v, ok := record.ValueByKey("amount").(float64); ok {
			bar.Amount = v
		}
		
		bars = append(bars, bar)
	}

	if result.Err() != nil {
		return nil, result.Err()
	}

	return bars, nil
}

// ============ 技术指标操作 ============

// SaveIndicator 保存技术指标
func (r *marketRepository) SaveIndicator(ctx context.Context, indicator *models.Indicator) error {
	fields := make(map[string]interface{})
	
	// 根据指标类型存储不同字段
	switch indicator.IndicatorType {
	case "ma":
		if indicator.MA5 != 0 {
			fields["ma5"] = indicator.MA5
		}
		if indicator.MA10 != 0 {
			fields["ma10"] = indicator.MA10
		}
		if indicator.MA20 != 0 {
			fields["ma20"] = indicator.MA20
		}
		if indicator.MA60 != 0 {
			fields["ma60"] = indicator.MA60
		}
	case "macd":
		fields["macd"] = indicator.MACD
		fields["macd_signal"] = indicator.MACDSignal
		fields["macd_hist"] = indicator.MACDHist
	case "rsi":
		fields["rsi6"] = indicator.RSI6
		fields["rsi12"] = indicator.RSI12
		fields["rsi24"] = indicator.RSI24
	case "kdj":
		fields["k"] = indicator.K
		fields["d"] = indicator.D
		fields["j"] = indicator.J
	case "boll":
		fields["boll_upper"] = indicator.BollUpper
		fields["boll_mid"] = indicator.BollMid
		fields["boll_lower"] = indicator.BollLower
	}
	
	point := write.NewPoint(
		"indicators",
		map[string]string{
			"symbol":         indicator.Symbol,
			"exchange":       indicator.Exchange,
			"indicator_type": indicator.IndicatorType,
		},
		fields,
		indicator.Date,
	)
	
	r.influx.WritePoint(point)
	r.influx.Flush()
	return nil
}

// SaveIndicators 批量保存技术指标
func (r *marketRepository) SaveIndicators(ctx context.Context, indicators []*models.Indicator) error {
	for _, indicator := range indicators {
		if err := r.SaveIndicator(ctx, indicator); err != nil {
			return err
		}
	}
	return nil
}

// GetIndicators 查询技术指标
func (r *marketRepository) GetIndicators(ctx context.Context, symbol, exchange string, indicatorType string, start, end time.Time) ([]*models.Indicator, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "indicators")
		|> filter(fn: (r) => r.symbol == "%s")
		|> filter(fn: (r) => r.exchange == "%s")
		|> filter(fn: (r) => r.indicator_type == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> sort(columns: ["_time"])
	`, r.influx.GetBucket(), start.Format(time.RFC3339), end.Format(time.RFC3339), symbol, exchange, indicatorType)

	result, err := r.influx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询技术指标失败: %w", err)
	}
	defer result.Close()

	var indicators []*models.Indicator
	for result.Next() {
		record := result.Record()
		indicator := &models.Indicator{
			Symbol:        symbol,
			Exchange:      exchange,
			Date:          record.Time(),
			IndicatorType: indicatorType,
		}
		
		// 根据指标类型解析字段
		switch indicatorType {
		case "ma":
			if v, ok := record.ValueByKey("ma5").(float64); ok {
				indicator.MA5 = v
			}
			if v, ok := record.ValueByKey("ma10").(float64); ok {
				indicator.MA10 = v
			}
			if v, ok := record.ValueByKey("ma20").(float64); ok {
				indicator.MA20 = v
			}
			if v, ok := record.ValueByKey("ma60").(float64); ok {
				indicator.MA60 = v
			}
		case "macd":
			if v, ok := record.ValueByKey("macd").(float64); ok {
				indicator.MACD = v
			}
			if v, ok := record.ValueByKey("macd_signal").(float64); ok {
				indicator.MACDSignal = v
			}
			if v, ok := record.ValueByKey("macd_hist").(float64); ok {
				indicator.MACDHist = v
			}
		case "rsi":
			if v, ok := record.ValueByKey("rsi6").(float64); ok {
				indicator.RSI6 = v
			}
			if v, ok := record.ValueByKey("rsi12").(float64); ok {
				indicator.RSI12 = v
			}
			if v, ok := record.ValueByKey("rsi24").(float64); ok {
				indicator.RSI24 = v
			}
		case "kdj":
			if v, ok := record.ValueByKey("k").(float64); ok {
				indicator.K = v
			}
			if v, ok := record.ValueByKey("d").(float64); ok {
				indicator.D = v
			}
			if v, ok := record.ValueByKey("j").(float64); ok {
				indicator.J = v
			}
		case "boll":
			if v, ok := record.ValueByKey("boll_upper").(float64); ok {
				indicator.BollUpper = v
			}
			if v, ok := record.ValueByKey("boll_mid").(float64); ok {
				indicator.BollMid = v
			}
			if v, ok := record.ValueByKey("boll_lower").(float64); ok {
				indicator.BollLower = v
			}
		}
		
		indicators = append(indicators, indicator)
	}

	if result.Err() != nil {
		return nil, result.Err()
	}

	return indicators, nil
}

// GetLatestIndicator 获取最新技术指标
func (r *marketRepository) GetLatestIndicator(ctx context.Context, symbol, exchange string, indicatorType string) (*models.Indicator, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -30d)
		|> filter(fn: (r) => r._measurement == "indicators")
		|> filter(fn: (r) => r.symbol == "%s")
		|> filter(fn: (r) => r.exchange == "%s")
		|> filter(fn: (r) => r.indicator_type == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> sort(columns: ["_time"], desc: true)
		|> limit(n: 1)
	`, r.influx.GetBucket(), symbol, exchange, indicatorType)

	result, err := r.influx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询最新技术指标失败: %w", err)
	}
	defer result.Close()

	if result.Next() {
		record := result.Record()
		indicator := &models.Indicator{
			Symbol:        symbol,
			Exchange:      exchange,
			Date:          record.Time(),
			IndicatorType: indicatorType,
		}
		
		// 解析字段（简化版）
		switch indicatorType {
		case "ma":
			if v, ok := record.ValueByKey("ma5").(float64); ok {
				indicator.MA5 = v
			}
		case "macd":
			if v, ok := record.ValueByKey("macd").(float64); ok {
				indicator.MACD = v
			}
		case "rsi":
			if v, ok := record.ValueByKey("rsi6").(float64); ok {
				indicator.RSI6 = v
			}
		case "kdj":
			if v, ok := record.ValueByKey("k").(float64); ok {
				indicator.K = v
			}
		case "boll":
			if v, ok := record.ValueByKey("boll_upper").(float64); ok {
				indicator.BollUpper = v
			}
		}
		
		return indicator, nil
	}

	return nil, nil
}

// ============ 数据完整性检查 ============

// CheckDataIntegrity 检查数据完整性
func (r *marketRepository) CheckDataIntegrity(ctx context.Context, symbol, exchange string, start, end time.Time) (map[string]interface{}, error) {
	// 查询时间范围内的数据点数量
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "daily_bars")
		|> filter(fn: (r) => r.symbol == "%s")
		|> filter(fn: (r) => r.exchange == "%s")
		|> count()
	`, r.influx.GetBucket(), start.Format(time.RFC3339), end.Format(time.RFC3339), symbol, exchange)

	result, err := r.influx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("数据完整性检查失败: %w", err)
	}
	defer result.Close()

	var count int64
	if result.Next() {
		if v, ok := result.Record().Value().(int64); ok {
			count = v
		}
	}

	// 计算预期交易日数量（简化计算，实际应考虑节假日）
	expectedDays := int(end.Sub(start).Hours() / 24 * 5 / 7) // 约5/7是交易日
	
	integrity := float64(count) / float64(expectedDays)
	status := "complete"
	if integrity < 0.9 {
		status = "incomplete"
	} else if integrity < 1.0 {
		status = "partial"
	}

	return map[string]interface{}{
		"symbol":        symbol,
		"exchange":      exchange,
		"start_date":    start.Format("2006-01-02"),
		"end_date":      end.Format("2006-01-02"),
		"actual_count":  count,
		"expected_days": expectedDays,
		"integrity":     integrity,
		"status":        status,
	}, nil
}
