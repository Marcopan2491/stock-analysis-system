package quality

import (
	"context"
	"fmt"
	"time"

	"stock-analysis-system/backend/pkg/models"
	"stock-analysis-system/backend/pkg/repository"
)

// DataQualityChecker 数据质量检查器
type DataQualityChecker struct {
	stockRepo  repository.StockRepository
	marketRepo repository.MarketRepository
}

// NewDataQualityChecker 创建数据质量检查器
func NewDataQualityChecker(stockRepo repository.StockRepository, marketRepo repository.MarketRepository) *DataQualityChecker {
	return &DataQualityChecker{
		stockRepo:  stockRepo,
		marketRepo: marketRepo,
	}
}

// CheckResult 检查结果
type CheckResult struct {
	Symbol      string    `json:"symbol"`
	Exchange    string    `json:"exchange"`
	CheckType   string    `json:"check_type"`
	Status      string    `json:"status"` // pass, warning, error
	Message     string    `json:"message"`
	Details     map[string]interface{} `json:"details"`
	CheckedAt   time.Time `json:"checked_at"`
}

// DataQualityReport 数据质量报告
type DataQualityReport struct {
	GeneratedAt time.Time     `json:"generated_at"`
	TotalStocks int           `json:"total_stocks"`
	Checks      []CheckResult `json:"checks"`
	Summary     struct {
		PassCount    int `json:"pass_count"`
		WarningCount int `json:"warning_count"`
		ErrorCount   int `json:"error_count"`
	} `json:"summary"`
}

// ============ 完整性检查 ============

// CheckCompleteness 检查数据完整性
func (c *DataQualityChecker) CheckCompleteness(ctx context.Context, symbol, exchange string, start, end time.Time) (*CheckResult, error) {
	integrity, err := c.marketRepo.CheckDataIntegrity(ctx, symbol, exchange, start, end)
	if err != nil {
		return nil, err
	}

	status := integrity["status"].(string)
	integrityRatio := integrity["integrity"].(float64)

	result := &CheckResult{
		Symbol:    symbol,
		Exchange:  exchange,
		CheckType: "completeness",
		Status:    status,
		CheckedAt: time.Now(),
		Details:   integrity,
	}

	switch status {
	case "complete":
		result.Message = fmt.Sprintf("数据完整，完整度 %.1f%%", integrityRatio*100)
	case "partial":
		result.Message = fmt.Sprintf("数据部分缺失，完整度 %.1f%%", integrityRatio*100)
	case "incomplete":
		result.Message = fmt.Sprintf("数据严重缺失，完整度 %.1f%%", integrityRatio*100)
	}

	return result, nil
}

// ============ 连续性检查 ============

// CheckContinuity 检查数据连续性
func (c *DataQualityChecker) CheckContinuity(ctx context.Context, symbol, exchange string, days int) (*CheckResult, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -days)

	// 获取K线数据
	bars, err := c.marketRepo.GetDailyBars(ctx, symbol, exchange, start, end)
	if err != nil {
		return nil, err
	}

	if len(bars) == 0 {
		return &CheckResult{
			Symbol:    symbol,
			Exchange:  exchange,
			CheckType: "continuity",
			Status:    "error",
			Message:   "没有数据",
			CheckedAt: time.Now(),
			Details: map[string]interface{}{
				"expected_days": days,
				"actual_days":   0,
			},
		}, nil
	}

	// 检查数据点之间的间隔
	gaps := []map[string]string{}
	expectedInterval := 24 * time.Hour // 预期日期间隔

	for i := 1; i < len(bars); i++ {
		interval := bars[i].Date.Sub(bars[i-1].Date)
		// 允许周末跳过（间隔大于1天但小于等于3天）
		if interval > expectedInterval*3 {
			gaps = append(gaps, map[string]string{
				"from": bars[i-1].Date.Format("2006-01-02"),
				"to":   bars[i].Date.Format("2006-01-02"),
				"days": fmt.Sprintf("%.0f", interval.Hours()/24),
			})
		}
	}

	result := &CheckResult{
		Symbol:    symbol,
		Exchange:  exchange,
		CheckType: "continuity",
		CheckedAt: time.Now(),
		Details: map[string]interface{}{
			"expected_days": days,
			"actual_days":   len(bars),
			"gaps":          gaps,
			"gap_count":     len(gaps),
		},
	}

	if len(gaps) == 0 {
		result.Status = "pass"
		result.Message = fmt.Sprintf("数据连续，共 %d 个交易日", len(bars))
	} else if len(gaps) <= 2 {
		result.Status = "warning"
		result.Message = fmt.Sprintf("数据基本连续，发现 %d 个间隔", len(gaps))
	} else {
		result.Status = "error"
		result.Message = fmt.Sprintf("数据不连续，发现 %d 个间隔", len(gaps))
	}

	return result, nil
}

// ============ 异常值检查 ============

// CheckAnomalies 检查数据异常
func (c *DataQualityChecker) CheckAnomalies(ctx context.Context, symbol, exchange string, days int) (*CheckResult, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -days)

	bars, err := c.marketRepo.GetDailyBars(ctx, symbol, exchange, start, end)
	if err != nil {
		return nil, err
	}

	if len(bars) < 2 {
		return &CheckResult{
			Symbol:    symbol,
			Exchange:  exchange,
			CheckType: "anomalies",
			Status:    "warning",
			Message:   "数据不足，无法检查异常",
			CheckedAt: time.Now(),
		}, nil
	}

	anomalies := []map[string]interface{}{}

	for i, bar := range bars {
		// 检查价格是否为0或负数
		if bar.Open <= 0 || bar.High <= 0 || bar.Low <= 0 || bar.Close <= 0 {
			anomalies = append(anomalies, map[string]interface{}{
				"date":   bar.Date.Format("2006-01-02"),
				"type":   "invalid_price",
				"values": map[string]float64{"open": bar.Open, "high": bar.High, "low": bar.Low, "close": bar.Close},
			})
			continue
		}

		// 检查高低价逻辑
		if bar.Low > bar.High || bar.Open > bar.High || bar.Open < bar.Low ||
			bar.Close > bar.High || bar.Close < bar.Low {
			anomalies = append(anomalies, map[string]interface{}{
				"date": bar.Date.Format("2006-01-02"),
				"type": "price_logic_error",
				"values": map[string]float64{
					"open":  bar.Open,
					"high":  bar.High,
					"low":   bar.Low,
					"close": bar.Close,
				},
			})
			continue
		}

		// 检查涨跌幅异常（单日涨跌超过20%）
		if i > 0 {
			prevClose := bars[i-1].Close
			if prevClose > 0 {
				changePct := (bar.Close - prevClose) / prevClose * 100
				if changePct > 20 || changePct < -20 {
					anomalies = append(anomalies, map[string]interface{}{
						"date":        bar.Date.Format("2006-01-02"),
						"type":        "extreme_change",
						"change_pct":  changePct,
						"prev_close":  prevClose,
						"close":       bar.Close,
					})
				}
			}
		}

		// 检查成交量异常（为0或异常大）
		if bar.Volume == 0 {
			anomalies = append(anomalies, map[string]interface{}{
				"date":   bar.Date.Format("2006-01-02"),
				"type":   "zero_volume",
				"volume": bar.Volume,
			})
		}
	}

	result := &CheckResult{
		Symbol:    symbol,
		Exchange:  exchange,
		CheckType: "anomalies",
		CheckedAt: time.Now(),
		Details: map[string]interface{}{
			"total_bars":     len(bars),
			"anomaly_count":  len(anomalies),
			"anomalies":      anomalies,
		},
	}

	if len(anomalies) == 0 {
		result.Status = "pass"
		result.Message = "未发现数据异常"
	} else if len(anomalies) <= 2 {
		result.Status = "warning"
		result.Message = fmt.Sprintf("发现 %d 个潜在异常", len(anomalies))
	} else {
		result.Status = "error"
		result.Message = fmt.Sprintf("发现 %d 个数据异常", len(anomalies))
	}

	return result, nil
}

// ============ 全量检查 ============

// CheckStock 对单只股票进行全面检查
func (c *DataQualityChecker) CheckStock(ctx context.Context, symbol, exchange string) ([]CheckResult, error) {
	results := []CheckResult{}

	// 完整性检查（最近30天）
	end := time.Now()
	start := end.AddDate(0, 0, -30)
	
	if result, err := c.CheckCompleteness(ctx, symbol, exchange, start, end); err == nil {
		results = append(results, *result)
	}

	// 连续性检查
	if result, err := c.CheckContinuity(ctx, symbol, exchange, 30); err == nil {
		results = append(results, *result)
	}

	// 异常值检查
	if result, err := c.CheckAnomalies(ctx, symbol, exchange, 30); err == nil {
		results = append(results, *result)
	}

	return results, nil
}

// GenerateReport 生成全市场数据质量报告
func (c *DataQualityChecker) GenerateReport(ctx context.Context) (*DataQualityReport, error) {
	report := &DataQualityReport{
		GeneratedAt: time.Now(),
		Checks:      []CheckResult{},
	}

	// 获取所有活跃股票
	stocks, err := c.stockRepo.GetActiveStocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取股票列表失败: %w", err)
	}

	report.TotalStocks = len(stocks)

	// 对每只股票进行检查
	for _, stock := range stocks {
		results, err := c.CheckStock(ctx, stock.Symbol, stock.Exchange)
		if err != nil {
			continue
		}
		report.Checks = append(report.Checks, results...)

		// 统计
		for _, r := range results {
			switch r.Status {
			case "pass":
				report.Summary.PassCount++
			case "warning":
				report.Summary.WarningCount++
			case "error":
				report.Summary.ErrorCount++
			}
		}
	}

	return report, nil
}

// CheckDataFreshness 检查数据新鲜度
func (c *DataQualityChecker) CheckDataFreshness(ctx context.Context, symbol, exchange string) (*CheckResult, error) {
	latestBar, err := c.marketRepo.GetLatestDailyBar(ctx, symbol, exchange)
	if err != nil {
		return nil, err
	}

	if latestBar == nil {
		return &CheckResult{
			Symbol:    symbol,
			Exchange:  exchange,
			CheckType: "freshness",
			Status:    "error",
			Message:   "没有数据",
			CheckedAt: time.Now(),
		}, nil
	}

	// 计算数据延迟
	delay := time.Since(latestBar.Date)
	result := &CheckResult{
		Symbol:    symbol,
		Exchange:  exchange,
		CheckType: "freshness",
		CheckedAt: time.Now(),
		Details: map[string]interface{}{
			"latest_date": latestBar.Date.Format("2006-01-02"),
			"delay_hours": delay.Hours(),
			"delay_days":  delay.Hours() / 24,
		},
	}

	// A股数据延迟判断
	days := int(delay.Hours() / 24)
	switch {
	case days <= 1:
		result.Status = "pass"
		result.Message = fmt.Sprintf("数据最新，最后更新: %s", latestBar.Date.Format("2006-01-02"))
	case days <= 3:
		result.Status = "warning"
		result.Message = fmt.Sprintf("数据稍有延迟，最后更新: %s", latestBar.Date.Format("2006-01-02"))
	default:
		result.Status = "error"
		result.Message = fmt.Sprintf("数据严重滞后，最后更新: %s", latestBar.Date.Format("2006-01-02"))
	}

	return result, nil
}

// ValidateBarData 验证K线数据有效性
func ValidateBarData(bar *models.DailyBar) error {
	if bar == nil {
		return fmt.Errorf("数据为空")
	}

	// 检查必需字段
	if bar.Symbol == "" || bar.Exchange == "" {
		return fmt.Errorf("股票代码或交易所为空")
	}

	if bar.Date.IsZero() {
		return fmt.Errorf("日期为空")
	}

	// 检查价格
	if bar.Open <= 0 || bar.High <= 0 || bar.Low <= 0 || bar.Close <= 0 {
		return fmt.Errorf("价格必须大于0: open=%.2f, high=%.2f, low=%.2f, close=%.2f",
			bar.Open, bar.High, bar.Low, bar.Close)
	}

	// 检查价格逻辑
	if bar.Low > bar.High {
		return fmt.Errorf("最低价不能高于最高价: low=%.2f, high=%.2f", bar.Low, bar.High)
	}
	if bar.Open < bar.Low || bar.Open > bar.High {
		return fmt.Errorf("开盘价必须在高低价范围内: open=%.2f, low=%.2f, high=%.2f",
			bar.Open, bar.Low, bar.High)
	}
	if bar.Close < bar.Low || bar.Close > bar.High {
		return fmt.Errorf("收盘价必须在高低价范围内: close=%.2f, low=%.2f, high=%.2f",
			bar.Close, bar.Low, bar.High)
	}

	// 检查成交量
	if bar.Volume < 0 {
		return fmt.Errorf("成交量不能为负数: volume=%d", bar.Volume)
	}

	return nil
}
