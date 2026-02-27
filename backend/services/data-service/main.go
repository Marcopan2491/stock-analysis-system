package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stock-analysis-system/backend/pkg/config"
	"stock-analysis-system/backend/pkg/database"
	"stock-analysis-system/backend/pkg/models"
	"stock-analysis-system/backend/pkg/repository"
)

// DataSyncService 数据同步服务
type DataSyncService struct {
	cfg            *config.Config
	dbManager      *database.Manager
	stockRepo      repository.StockRepository
	marketRepo     repository.MarketRepository
	httpClient     *http.Client
	pythonAPIURL   string
}

// NewDataSyncService 创建数据同步服务
func NewDataSyncService(cfg *config.Config) (*DataSyncService, error) {
	// 创建数据库管理器
	dbManager, err := database.NewManager(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库管理器失败: %w", err)
	}

	// 创建仓库
	stockRepo := repository.NewStockRepository(dbManager.Postgres.DB)
	marketRepo := repository.NewMarketRepository(dbManager.Influx)

	return &DataSyncService{
		cfg:          cfg,
		dbManager:    dbManager,
		stockRepo:    stockRepo,
		marketRepo:   marketRepo,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		pythonAPIURL: getEnv("PYTHON_API_URL", "http://localhost:5000"),
	}, nil
}

// Close 关闭服务
func (s *DataSyncService) Close() {
	if s.dbManager != nil {
		s.dbManager.Close()
	}
}

// ============ 股票列表同步 ============

// SyncStockList 同步股票列表
func (s *DataSyncService) SyncStockList(ctx context.Context) error {
	log.Println("开始同步股票列表...")

	// 调用 Python 数据采集服务获取股票列表
	stocks, err := s.fetchStockListFromPython(ctx)
	if err != nil {
		return fmt.Errorf("从 Python 服务获取股票列表失败: %w", err)
	}

	log.Printf("从 Python 服务获取到 %d 只股票", len(stocks))

	// 批量保存到 PostgreSQL
	batchSize := 100
	for i := 0; i < len(stocks); i += batchSize {
		end := i + batchSize
		if end > len(stocks) {
			end = len(stocks)
		}

		batch := stocks[i:end]
		if err := s.stockRepo.CreateBatch(ctx, batch); err != nil {
			log.Printf("批量保存股票失败: %v", err)
			continue
		}
	}

	log.Printf("股票列表同步完成，共 %d 只", len(stocks))
	return nil
}

// fetchStockListFromPython 从 Python 服务获取股票列表
func (s *DataSyncService) fetchStockListFromPython(ctx context.Context) ([]*models.Stock, error) {
	// 调用 Python 数据采集服务的 HTTP 接口
	// 或者读取 Python 服务写入的文件/数据库
	
	// 简化实现：直接执行 Python 脚本获取数据
	// 实际生产环境应通过消息队列或 HTTP API
	
	// 这里使用模拟数据作为示例
	// 实际应从 strategy/data_collector/manager.py 获取
	
	mockStocks := []*models.Stock{
		{Symbol: "000001", Name: "平安银行", Exchange: "SZ", Industry: "银行"},
		{Symbol: "000002", Name: "万科A", Exchange: "SZ", Industry: "房地产"},
		{Symbol: "000063", Name: "中兴通讯", Exchange: "SZ", Industry: "通信设备"},
		{Symbol: "000333", Name: "美的集团", Exchange: "SZ", Industry: "家电"},
		{Symbol: "000568", Name: "泸州老窖", Exchange: "SZ", Industry: "白酒"},
		{Symbol: "000651", Name: "格力电器", Exchange: "SZ", Industry: "家电"},
		{Symbol: "000725", Name: "京东方A", Exchange: "SZ", Industry: "电子"},
		{Symbol: "000858", Name: "五粮液", Exchange: "SZ", Industry: "白酒"},
		{Symbol: "600000", Name: "浦发银行", Exchange: "SH", Industry: "银行"},
		{Symbol: "600009", Name: "上海机场", Exchange: "SH", Industry: "机场"},
		{Symbol: "600016", Name: "民生银行", Exchange: "SH", Industry: "银行"},
		{Symbol: "600028", Name: "中国石化", Exchange: "SH", Industry: "石油石化"},
		{Symbol: "600030", Name: "中信证券", Exchange: "SH", Industry: "证券"},
		{Symbol: "600036", Name: "招商银行", Exchange: "SH", Industry: "银行"},
		{Symbol: "600276", Name: "恒瑞医药", Exchange: "SH", Industry: "医药"},
		{Symbol: "600519", Name: "贵州茅台", Exchange: "SH", Industry: "白酒"},
		{Symbol: "601012", Name: "隆基绿能", Exchange: "SH", Industry: "光伏"},
		{Symbol: "601088", Name: "中国神华", Exchange: "SH", Industry: "煤炭"},
		{Symbol: "601166", Name: "兴业银行", Exchange: "SH", Industry: "银行"},
		{Symbol: "601318", Name: "中国平安", Exchange: "SH", Industry: "保险"},
	}

	return mockStocks, nil
}

// ============ K线数据同步 ============

// SyncDailyBars 同步日K线数据
func (s *DataSyncService) SyncDailyBars(ctx context.Context, symbol, exchange string, start, end time.Time) error {
	log.Printf("开始同步 %s.%s 的日K线数据 (%s ~ %s)", symbol, exchange, start.Format("2006-01-02"), end.Format("2006-01-02"))

	// 从 Python 服务获取K线数据
	bars, err := s.fetchDailyBarsFromPython(ctx, symbol, exchange, start, end)
	if err != nil {
		return fmt.Errorf("从 Python 服务获取K线数据失败: %w", err)
	}

	if len(bars) == 0 {
		log.Printf("未获取到 %s.%s 的K线数据", symbol, exchange)
		return nil
	}

	log.Printf("获取到 %d 条K线数据", len(bars))

	// 保存到 InfluxDB
	if err := s.marketRepo.SaveDailyBars(ctx, bars); err != nil {
		return fmt.Errorf("保存K线数据失败: %w", err)
	}

	log.Printf("%s.%s 的日K线数据同步完成", symbol, exchange)
	return nil
}

// SyncDailyBarsForAllStocks 为所有股票同步日K线数据
func (s *DataSyncService) SyncDailyBarsForAllStocks(ctx context.Context, start, end time.Time) error {
	// 获取所有活跃股票
	stocks, err := s.stockRepo.GetActiveStocks(ctx)
	if err != nil {
		return fmt.Errorf("获取股票列表失败: %w", err)
	}

	log.Printf("开始为 %d 只股票同步日K线数据", len(stocks))

	for i, stock := range stocks {
		log.Printf("[%d/%d] 同步 %s.%s...", i+1, len(stocks), stock.Symbol, stock.Exchange)
		
		if err := s.SyncDailyBars(ctx, stock.Symbol, stock.Exchange, start, end); err != nil {
			log.Printf("同步 %s.%s 失败: %v", stock.Symbol, stock.Exchange, err)
			continue
		}

		// 避免请求过快
		time.Sleep(500 * time.Millisecond)
	}

	log.Println("所有股票日K线数据同步完成")
	return nil
}

// fetchDailyBarsFromPython 从 Python 服务获取日K线数据
func (s *DataSyncService) fetchDailyBarsFromPython(ctx context.Context, symbol, exchange string, start, end time.Time) ([]*models.DailyBar, error) {
	// 构建请求 URL
	url := fmt.Sprintf("%s/api/v1/market/daily_bars?symbol=%s&exchange=%s&start=%s&end=%s",
		s.pythonAPIURL,
		symbol,
		exchange,
		start.Format("20060102"),
		end.Format("20060102"),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result struct {
		Code int                 `json:"code"`
		Data []*models.DailyBar `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// ============ 增量更新 ============

// IncrementalUpdate 执行增量更新
func (s *DataSyncService) IncrementalUpdate(ctx context.Context) error {
	log.Println("开始执行增量更新...")

	// 获取所有活跃股票
	stocks, err := s.stockRepo.GetActiveStocks(ctx)
	if err != nil {
		return fmt.Errorf("获取股票列表失败: %w", err)
	}

	end := time.Now()
	start := end.AddDate(0, 0, -7) // 更新最近7天的数据

	for _, stock := range stocks {
		// 查询该股票最新的数据日期
		latestBar, err := s.marketRepo.GetLatestDailyBar(ctx, stock.Symbol, stock.Exchange)
		if err != nil {
			log.Printf("获取 %s.%s 最新数据失败: %v", stock.Symbol, stock.Exchange, err)
			continue
		}

		if latestBar != nil {
			// 从最新数据日期的下一天开始更新
			updateStart := latestBar.Date.AddDate(0, 0, 1)
			if updateStart.Before(end) {
				if err := s.SyncDailyBars(ctx, stock.Symbol, stock.Exchange, updateStart, end); err != nil {
					log.Printf("增量更新 %s.%s 失败: %v", stock.Symbol, stock.Exchange, err)
				}
			}
		} else {
			// 没有历史数据，同步最近30天
			updateStart := end.AddDate(0, 0, -30)
			if err := s.SyncDailyBars(ctx, stock.Symbol, stock.Exchange, updateStart, end); err != nil {
				log.Printf("同步 %s.%s 历史数据失败: %v", stock.Symbol, stock.Exchange, err)
			}
		}
	}

	log.Println("增量更新完成")
	return nil
}

// ============ 定时任务 ============

// StartScheduler 启动定时任务
func (s *DataSyncService) StartScheduler(ctx context.Context) {
	log.Println("启动数据同步定时任务...")

	// 每天凌晨 2:00 执行增量更新
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				// 检查是否是凌晨 2:00
				if now.Hour() == 2 {
					if err := s.IncrementalUpdate(ctx); err != nil {
						log.Printf("定时增量更新失败: %v", err)
					}
				}
			}
		}
	}()
}

// ============ HTTP API ============

// StartHTTPServer 启动 HTTP 服务
func (s *DataSyncService) StartHTTPServer(port string) error {
	mux := http.NewServeMux()
	
	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 同步股票列表
	mux.HandleFunc("/api/v1/sync/stocks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		if err := s.SyncStockList(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    0,
			"message": "Stock list synced successfully",
		})
	})

	// 同步单只股票K线
	mux.HandleFunc("/api/v1/sync/bars", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Symbol   string `json:"symbol"`
			Exchange string `json:"exchange"`
			Start    string `json:"start"`
			End      string `json:"end"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		start, _ := time.Parse("2006-01-02", req.Start)
		end, _ := time.Parse("2006-01-02", req.End)

		ctx := r.Context()
		if err := s.SyncDailyBars(ctx, req.Symbol, req.Exchange, start, end); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    0,
			"message": "Bars synced successfully",
		})
	})

	// 执行增量更新
	mux.HandleFunc("/api/v1/sync/incremental", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		if err := s.IncrementalUpdate(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    0,
			"message": "Incremental update completed",
		})
	})

	log.Printf("数据同步服务启动在端口 %s", port)
	return http.ListenAndServe(":"+port, mux)
}

// ============ 主函数 ============

func main() {
	// 加载配置
	cfg := config.LoadFromEnv()

	// 创建服务
	service, err := NewDataSyncService(cfg)
	if err != nil {
		log.Fatalf("创建数据同步服务失败: %v", err)
	}
	defer service.Close()

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动定时任务
	service.StartScheduler(ctx)

	// 启动 HTTP 服务
	port := getEnv("DATA_SERVICE_PORT", "8081")
	
	// 优雅退出
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("正在关闭服务...")
		cancel()
	}()

	if err := service.StartHTTPServer(port); err != nil {
		log.Fatalf("HTTP服务启动失败: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
