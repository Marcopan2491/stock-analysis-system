package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"stock-analysis-system/backend/pkg/config"
	"stock-analysis-system/backend/pkg/database"
	"stock-analysis-system/backend/pkg/models"
	"stock-analysis-system/backend/pkg/repository"
)

// BacktestService 回测服务
type BacktestService struct {
	cfg            *config.Config
	dbManager      *database.Manager
	backtestRepo   repository.BacktestRepository
	strategyRepo   repository.StrategyRepository
	jwtSecret      []byte
	runningJobs    map[string]*BacktestJob
}

// BacktestJob 回测任务
type BacktestJob struct {
	ID         string    `json:"id"`
	StrategyID uint      `json:"strategy_id"`
	UserID     uint      `json:"user_id"`
	Status     string    `json:"status"` // pending, running, completed, failed
	Progress   float64   `json:"progress"`
	Result     *models.BacktestRecord `json:"result,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewBacktestService 创建回测服务
func NewBacktestService(cfg *config.Config) (*BacktestService, error) {
	dbManager, err := database.NewManager(&cfg.Database)
	if err != nil {
		return nil, err
	}

	backtestRepo := repository.NewBacktestRepository(dbManager.Postgres.DB)
	strategyRepo := repository.NewStrategyRepository(dbManager.Postgres.DB)
	jwtSecret := []byte(getEnv("JWT_SECRET", "your-secret-key"))

	return &BacktestService{
		cfg:          cfg,
		dbManager:    dbManager,
		backtestRepo: backtestRepo,
		strategyRepo: strategyRepo,
		jwtSecret:    jwtSecret,
		runningJobs:  make(map[string]*BacktestJob),
	}, nil
}

// Close 关闭服务
func (s *BacktestService) Close() {
	if s.dbManager != nil {
		s.dbManager.Close()
	}
}

// AuthMiddleware JWT认证中间件
func (s *BacktestService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "缺少认证信息"})
			c.Abort()
			return
		}

		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			tokenString = authHeader
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "无效的认证信息"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, ok := claims["user_id"].(float64); ok {
				c.Set("user_id", uint(userID))
			}
		}

		c.Next()
	}
}

// ============ 回测任务接口 ============

// RunBacktestRequest 运行回测请求
type RunBacktestRequest struct {
	StrategyID    uint     `json:"strategy_id" binding:"required"`
	StartDate     string   `json:"start_date" binding:"required"` // YYYY-MM-DD
	EndDate       string   `json:"end_date" binding:"required"`
	Symbols       []string `json:"symbols"`
	InitialCapital float64 `json:"initial_capital"` // 默认 100000
}

// RunBacktest 运行回测
func (s *BacktestService) RunBacktest(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	var req RunBacktestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 验证策略存在且属于当前用户
	ctx := c.Request.Context()
	strategy, err := s.strategyRepo.GetByID(ctx, req.StrategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "策略不存在"})
		return
	}
	if strategy.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权使用该策略"})
		return
	}

	// 解析日期
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "开始日期格式错误"})
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "结束日期格式错误"})
		return
	}

	// 设置默认初始资金
	initialCapital := req.InitialCapital
	if initialCapital <= 0 {
		initialCapital = 100000
	}

	// 生成任务ID
	jobID := uuid.New().String()

	// 创建回测记录
	record := &models.BacktestRecord{
		StrategyID:     req.StrategyID,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: initialCapital,
		Status:         "running",
	}

	if err := s.backtestRepo.Create(ctx, record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建回测记录失败"})
		return
	}

	// 创建任务
	job := &BacktestJob{
		ID:         jobID,
		StrategyID: req.StrategyID,
		UserID:     uid,
		Status:     "running",
		Progress:   0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	s.runningJobs[jobID] = job

	// 异步执行回测
	go s.executeBacktest(job, record, strategy)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "回测任务已提交",
		"data": gin.H{
			"job_id":      jobID,
			"backtest_id": record.ID,
			"status":      "running",
			"created_at":  job.CreatedAt.Format(time.RFC3339),
		},
	})
}

// executeBacktest 执行回测（模拟）
func (s *BacktestService) executeBacktest(job *BacktestJob, record *models.BacktestRecord, strategy *models.Strategy) {
	ctx := context.Background()

	// 模拟回测过程
	time.Sleep(2 * time.Second)

	// 模拟回测结果
	totalReturn := 0.15 + (float64(time.Now().Unix()%100) / 1000) // 随机收益率 15-25%
	tradeCount := 50 + int(time.Now().Unix()%50)

	record.FinalCapital = record.InitialCapital * (1 + totalReturn)
	record.TotalReturn = totalReturn
	record.AnnualReturn = totalReturn / float64(record.EndDate.Sub(record.StartDate).Days()/365+1)
	record.MaxDrawdown = 0.08
	record.SharpeRatio = 1.2
	record.WinRate = 0.55
	record.ProfitLossRatio = 1.8
	record.TradeCount = tradeCount
	record.Status = "completed"
	now := time.Now()
	record.CompletedAt = &now

	// 更新数据库
	if err := s.backtestRepo.Update(ctx, record); err != nil {
		job.Status = "failed"
		return
	}

	// 更新任务状态
	job.Status = "completed"
	job.Progress = 100
	job.Result = record
	job.UpdatedAt = time.Now()
}

// GetBacktestStatus 获取回测状态
func (s *BacktestService) GetBacktestStatus(c *gin.Context) {
	jobID := c.Param("id")

	job, exists := s.runningJobs[jobID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"job_id":    job.ID,
			"status":    job.Status,
			"progress":  job.Progress,
			"created_at": job.CreatedAt.Format(time.RFC3339),
			"updated_at": job.UpdatedAt.Format(time.RFC3339),
		},
	})
}

// GetBacktestResult 获取回测结果
func (s *BacktestService) GetBacktestResult(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	backtestID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "回测ID错误"})
		return
	}

	ctx := c.Request.Context()
	record, err := s.backtestRepo.GetByID(ctx, uint(backtestID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "回测记录不存在"})
		return
	}

	// 验证权限
	strategy, _ := s.strategyRepo.GetByID(ctx, record.StrategyID)
	if strategy == nil || strategy.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权查看"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": record,
	})
}

// GetBacktestList 获取回测列表
func (s *BacktestService) GetBacktestList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	strategyID := c.Query("strategy_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	ctx := c.Request.Context()

	var records []*models.BacktestRecord
	var total int64
	var err error

	if strategyID != "" {
		sid, _ := strconv.ParseUint(strategyID, 10, 32)
		// 验证策略权限
		strategy, _ := s.strategyRepo.GetByID(ctx, uint(sid))
		if strategy == nil || strategy.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权查看"})
			return
		}
		records, total, err = s.backtestRepo.GetByStrategyID(ctx, uint(sid), page, pageSize)
	} else {
		// 获取用户所有策略的回测记录
		records, total, err = s.backtestRepo.GetByUserID(ctx, uid, page, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":        records,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		},
	})
}

// ============ 主函数 ============

func main() {
	cfg := config.LoadFromEnv()

	service, err := NewBacktestService(cfg)
	if err != nil {
		panic(err)
	}
	defer service.Close()

	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "backtest-service",
		})
	})

	// API路由
	api := r.Group("/api/v1")
	{
		// 回测接口（需要认证）
		backtest := api.Group("/backtest")
		backtest.Use(service.AuthMiddleware())
		{
			backtest.GET("", service.GetBacktestList)
			backtest.POST("/run", service.RunBacktest)
			backtest.GET("/status/:id", service.GetBacktestStatus)
			backtest.GET("/result/:id", service.GetBacktestResult)
		}
	}

	port := getEnv("BACKTEST_SERVICE_PORT", "8085")

	// 优雅退出
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
	}()

	r.Run(":" + port)
}

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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
