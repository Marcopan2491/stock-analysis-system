package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"stock-analysis-system/backend/pkg/config"
	"stock-analysis-system/backend/pkg/database"
	"stock-analysis-system/backend/pkg/models"
	"stock-analysis-system/backend/pkg/repository"
)

// MarketService 行情服务
type MarketService struct {
	cfg        *config.Config
	dbManager  *database.Manager
	stockRepo  repository.StockRepository
	marketRepo repository.MarketRepository
}

// NewMarketService 创建行情服务
func NewMarketService(cfg *config.Config) (*MarketService, error) {
	// 创建数据库管理器
	dbManager, err := database.NewManager(&cfg.Database)
	if err != nil {
		return nil, err
	}

	// 创建仓库
	stockRepo := repository.NewStockRepository(dbManager.Postgres.DB)
	marketRepo := repository.NewMarketRepository(dbManager.Influx)

	return &MarketService{
		cfg:        cfg,
		dbManager:  dbManager,
		stockRepo:  stockRepo,
		marketRepo: marketRepo,
	}, nil
}

// Close 关闭服务
func (s *MarketService) Close() {
	if s.dbManager != nil {
		s.dbManager.Close()
	}
}

// ============ 股票列表接口 ============

// StockListRequest 股票列表请求
type StockListRequest struct {
	Exchange string `form:"exchange"` // 交易所筛选
	Industry string `form:"industry"` // 行业筛选
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

// StockListResponse 股票列表响应
type StockListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg,omitempty"`
	Data struct {
		List       []*models.Stock `json:"list"`
		Total      int64           `json:"total"`
		Page       int             `json:"page"`
		PageSize   int             `json:"page_size"`
		TotalPages int             `json:"total_pages"`
	} `json:"data"`
}

// GetStockList 获取股票列表
func (s *MarketService) GetStockList(c *gin.Context) {
	var req StockListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	ctx := c.Request.Context()
	var stocks []*models.Stock
	var total int64
	var err error

	// 根据筛选条件查询
	if req.Exchange != "" {
		stocks, total, err = s.stockRepo.GetByExchange(ctx, req.Exchange, offset, req.PageSize)
	} else if req.Industry != "" {
		stocks, total, err = s.stockRepo.GetByIndustry(ctx, req.Industry, offset, req.PageSize)
	} else {
		stocks, total, err = s.stockRepo.GetAll(ctx, offset, req.PageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败: " + err.Error()})
		return
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	resp := StockListResponse{Code: 0}
	resp.Data.List = stocks
	resp.Data.Total = total
	resp.Data.Page = req.Page
	resp.Data.PageSize = req.PageSize
	resp.Data.TotalPages = totalPages

	c.JSON(http.StatusOK, resp)
}

// ============ 实时行情接口 ============

// QuoteRequest 实时行情请求
type QuoteRequest struct {
	Symbol   string `uri:"symbol" binding:"required"`
	Exchange string `form:"exchange,default=SZ"`
}

// QuoteResponse 实时行情响应
type QuoteResponse struct {
	Symbol      string  `json:"symbol"`
	Exchange    string  `json:"exchange"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Change      float64 `json:"change"`
	ChangePct   float64 `json:"change_pct"`
	Volume      int64   `json:"volume"`
	Amount      float64 `json:"amount"`
	Open        float64 `json:"open"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	PreClose    float64 `json:"pre_close"`
	BidPrice    float64 `json:"bid_price"`
	BidVolume   int64   `json:"bid_volume"`
	AskPrice    float64 `json:"ask_price"`
	AskVolume   int64   `json:"ask_volume"`
	Timestamp   int64   `json:"timestamp"`
	UpdateTime  string  `json:"update_time"`
}

// GetRealtimeQuote 获取实时行情
func (s *MarketService) GetRealtimeQuote(c *gin.Context) {
	var req QuoteRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 查询股票信息
	ctx := c.Request.Context()
	stock, err := s.stockRepo.GetBySymbol(ctx, req.Symbol, req.Exchange)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "股票不存在"})
		return
	}

	// 查询最新K线数据
	latestBar, err := s.marketRepo.GetLatestDailyBar(ctx, req.Symbol, req.Exchange)
	if err != nil {
		log.Printf("查询最新K线失败: %v", err)
	}

	// 获取昨收（前一天收盘价）
	var preClose float64
	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayBars, err := s.marketRepo.GetDailyBars(ctx, req.Symbol, req.Exchange, yesterday.AddDate(0, 0, -5), yesterday)
	if err == nil && len(yesterdayBars) > 0 {
		preClose = yesterdayBars[len(yesterdayBars)-1].Close
	}

	// 构建响应
	quote := QuoteResponse{
		Symbol:     req.Symbol,
		Exchange:   req.Exchange,
		Name:       stock.Name,
		Timestamp:  time.Now().Unix(),
		UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	if latestBar != nil {
		quote.Price = latestBar.Close
		quote.Open = latestBar.Open
		quote.High = latestBar.High
		quote.Low = latestBar.Low
		quote.Volume = latestBar.Volume
		quote.Amount = latestBar.Amount
	}

	if preClose > 0 {
		quote.PreClose = preClose
		quote.Change = quote.Price - preClose
		quote.ChangePct = (quote.Change / preClose) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": quote,
	})
}

// ============ K线数据接口 ============

// KlineRequest K线数据请求
type KlineRequest struct {
	Symbol   string `uri:"symbol" binding:"required"`
	Exchange string `form:"exchange,default=SZ"`
	Period   string `form:"period,default=1d"` // 1d, 1m, 5m, 15m, 30m, 60m
	Start    string `form:"start" binding:"required"` // YYYY-MM-DD
	End      string `form:"end" binding:"required"`
}

// KlineData K线数据点
type KlineData struct {
	Time   string  `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
	Amount float64 `json:"amount"`
}

// GetKlineData 获取K线数据
func (s *MarketService) GetKlineData(c *gin.Context) {
	var req KlineRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 解析时间
	start, err := time.Parse("2006-01-02", req.Start)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "开始日期格式错误"})
		return
	}
	end, err := time.Parse("2006-01-02", req.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "结束日期格式错误"})
		return
	}

	// 调整结束时间到当天结束
	end = end.Add(24 * time.Hour).Add(-time.Second)

	ctx := c.Request.Context()
	var klines []KlineData

	switch req.Period {
	case "1d":
		bars, err := s.marketRepo.GetDailyBars(ctx, req.Symbol, req.Exchange, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败: " + err.Error()})
			return
		}
		klines = convertDailyBarsToKline(bars)

	case "1m", "5m", "15m", "30m", "60m":
		bars, err := s.marketRepo.GetMinuteBars(ctx, req.Symbol, req.Exchange, req.Period, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败: " + err.Error()})
			return
		}
		klines = convertMinuteBarsToKline(bars)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "不支持的周期"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"symbol":   req.Symbol,
			"exchange": req.Exchange,
			"period":   req.Period,
			"start":    req.Start,
			"end":      req.End,
			"bars":     klines,
			"count":    len(klines),
		},
	})
}

func convertDailyBarsToKline(bars []*models.DailyBar) []KlineData {
	klines := make([]KlineData, len(bars))
	for i, bar := range bars {
		klines[i] = KlineData{
			Time:   bar.Date.Format("2006-01-02"),
			Open:   bar.Open,
			High:   bar.High,
			Low:    bar.Low,
			Close:  bar.Close,
			Volume: bar.Volume,
			Amount: bar.Amount,
		}
	}
	return klines
}

func convertMinuteBarsToKline(bars []*models.MinuteBar) []KlineData {
	klines := make([]KlineData, len(bars))
	for i, bar := range bars {
		klines[i] = KlineData{
			Time:   bar.Time.Format("2006-01-02 15:04"),
			Open:   bar.Open,
			High:   bar.High,
			Low:    bar.Low,
			Close:  bar.Close,
			Volume: bar.Volume,
			Amount: bar.Amount,
		}
	}
	return klines
}

// ============ 技术指标接口 ============

// IndicatorRequest 技术指标请求
type IndicatorRequest struct {
	Symbol       string `uri:"symbol" binding:"required"`
	Exchange     string `form:"exchange,default=SZ"`
	IndicatorType string `form:"type,default=ma"` // ma, macd, rsi, kdj, boll
	Period       int    `form:"period,default=20"` // 计算周期
	Start        string `form:"start"`
	End          string `form:"end"`
}

// IndicatorData 指标数据点
type IndicatorData struct {
	Time string  `json:"time"`
	Value float64 `json:"value,omitempty"`
	MA5  float64 `json:"ma5,omitempty"`
	MA10 float64 `json:"ma10,omitempty"`
	MA20 float64 `json:"ma20,omitempty"`
	MA60 float64 `json:"ma60,omitempty"`
}

// GetIndicators 获取技术指标
func (s *MarketService) GetIndicators(c *gin.Context) {
	var req IndicatorRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 解析时间
	start, _ := time.Parse("2006-01-02", req.Start)
	end, _ := time.Parse("2006-01-02", req.End)

	if start.IsZero() {
		start = time.Now().AddDate(0, 0, -req.Period)
	}
	if end.IsZero() {
		end = time.Now()
	}
	end = end.Add(24 * time.Hour).Add(-time.Second)

	ctx := c.Request.Context()

	// 查询指标数据
	indicators, err := s.marketRepo.GetIndicators(ctx, req.Symbol, req.Exchange, req.IndicatorType, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败: " + err.Error()})
		return
	}

	// 转换数据格式
	data := make([]IndicatorData, len(indicators))
	for i, ind := range indicators {
		d := IndicatorData{Time: ind.Date.Format("2006-01-02")}
		
		switch req.IndicatorType {
		case "ma":
			d.MA5 = ind.MA5
			d.MA10 = ind.MA10
			d.MA20 = ind.MA20
			d.MA60 = ind.MA60
		case "macd":
			d.Value = ind.MACD
		case "rsi":
			d.Value = ind.RSI6
		case "kdj":
			d.Value = ind.K
		case "boll":
			d.Value = ind.BollMid
		}
		
		data[i] = d
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"symbol":    req.Symbol,
			"exchange":  req.Exchange,
			"type":      req.IndicatorType,
			"indicators": data,
			"count":     len(data),
		},
	})
}

// ============ 搜索接口 ============

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword string `form:"q" binding:"required,min=1,max=20"`
}

// SearchStocks 搜索股票
func (s *MarketService) SearchStocks(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	ctx := c.Request.Context()
	stocks, err := s.stockRepo.Search(ctx, req.Keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "搜索失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"keyword": req.Keyword,
			"results": stocks,
			"count":   len(stocks),
		},
	})
}

// ============ 主函数 ============

func main() {
	// 加载配置
	cfg := config.LoadFromEnv()

	// 创建服务
	service, err := NewMarketService(cfg)
	if err != nil {
		log.Fatalf("创建行情服务失败: %v", err)
	}
	defer service.Close()

	// 设置Gin模式
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	r.Use(requestLogger())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		ctx := c.Request.Context()
		healthy := service.dbManager.IsHealthy(ctx)
		
		status := "healthy"
		code := 200
		if !healthy {
			status = "unhealthy"
			code = 503
		}

		c.JSON(code, gin.H{
			"status":    status,
			"service":   "market-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// API路由组
	api := r.Group("/api/v1")
	{
		// 行情接口
		market := api.Group("/market")
		{
			market.GET("/stocks", service.GetStockList)
			market.GET("/stocks/search", service.SearchStocks)
			market.GET("/quote/:symbol", service.GetRealtimeQuote)
			market.GET("/kline/:symbol", service.GetKlineData)
			market.GET("/indicators/:symbol", service.GetIndicators)
		}
	}

	// 获取端口
	port := os.Getenv("MARKET_SERVICE_PORT")
	if port == "" {
		port = "8082"
	}

	// 启动服务
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// 优雅退出
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("正在关闭服务...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("服务关闭失败: %v", err)
		}
	}()

	log.Printf("行情服务启动在端口 %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// requestLogger 请求日志中间件
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[%s] %s %s %d %v", clientIP, method, path, statusCode, latency)
	}
}
