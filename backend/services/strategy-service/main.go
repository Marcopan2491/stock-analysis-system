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

	"stock-analysis-system/backend/pkg/config"
	"stock-analysis-system/backend/pkg/database"
	"stock-analysis-system/backend/pkg/models"
	"stock-analysis-system/backend/pkg/repository"
)

// StrategyService 策略服务
type StrategyService struct {
	cfg          *config.Config
	dbManager    *database.Manager
	strategyRepo repository.StrategyRepository
	jwtSecret    []byte
}

// NewStrategyService 创建策略服务
func NewStrategyService(cfg *config.Config) (*StrategyService, error) {
	dbManager, err := database.NewManager(&cfg.Database)
	if err != nil {
		return nil, err
	}

	strategyRepo := repository.NewStrategyRepository(dbManager.Postgres.DB)
	jwtSecret := []byte(getEnv("JWT_SECRET", "your-secret-key"))

	return &StrategyService{
		cfg:          cfg,
		dbManager:    dbManager,
		strategyRepo: strategyRepo,
		jwtSecret:    jwtSecret,
	}, nil
}

// Close 关闭服务
func (s *StrategyService) Close() {
	if s.dbManager != nil {
		s.dbManager.Close()
	}
}

// AuthMiddleware JWT认证中间件
func (s *StrategyService) AuthMiddleware() gin.HandlerFunc {
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

// ============ 策略 CRUD ============

// CreateStrategyRequest 创建策略请求
type CreateStrategyRequest struct {
	Name        string   `json:"name" binding:"required,max=100"`
	Description string   `json:"description"`
	Type        string   `json:"type" binding:"required,oneof=trend_following mean_reversion multi_factor"`
	ClassName   string   `json:"class_name" binding:"required"`
	Params      string   `json:"params"` // JSON string
	Symbols     []string `json:"symbols"`
	IsPublic    bool     `json:"is_public"`
}

// CreateStrategy 创建策略
func (s *StrategyService) CreateStrategy(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	var req CreateStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	strategy := &models.Strategy{
		UserID:      uid,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		ClassName:   req.ClassName,
		Params:      req.Params,
		IsPublic:    req.IsPublic,
		IsActive:    true,
	}

	// 转换 symbols
	if len(req.Symbols) > 0 {
		symbolsStr := "{"
		for i, s := range req.Symbols {
			if i > 0 {
				symbolsStr += ","
			}
			symbolsStr += s
		}
		symbolsStr += "}"
		strategy.Symbols = symbolsStr
	}

	if err := s.strategyRepo.Create(ctx, strategy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "创建成功",
		"data": strategy,
	})
}

// GetStrategies 获取策略列表
func (s *StrategyService) GetStrategies(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	strategyType := c.Query("type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	ctx := c.Request.Context()

	strategies, total, err := s.strategyRepo.GetByUserID(ctx, uid, strategyType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":        strategies,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		},
	})
}

// GetStrategy 获取策略详情
func (s *StrategyService) GetStrategy(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	strategyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "策略ID错误"})
		return
	}

	ctx := c.Request.Context()
	strategy, err := s.strategyRepo.GetByID(ctx, uint(strategyID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "策略不存在"})
		return
	}

	// 检查权限（只能查看自己的或公开的策略）
	if strategy.UserID != uid && !strategy.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权访问"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": strategy,
	})
}

// UpdateStrategyRequest 更新策略请求
type UpdateStrategyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Params      string `json:"params"`
	IsActive    *bool  `json:"is_active,omitempty"`
	IsPublic    *bool  `json:"is_public,omitempty"`
}

// UpdateStrategy 更新策略
func (s *StrategyService) UpdateStrategy(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	strategyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "策略ID错误"})
		return
	}

	var req UpdateStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	ctx := c.Request.Context()
	strategy, err := s.strategyRepo.GetByID(ctx, uint(strategyID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "策略不存在"})
		return
	}

	// 检查权限
	if strategy.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权修改"})
		return
	}

	// 更新字段
	if req.Name != "" {
		strategy.Name = req.Name
	}
	if req.Description != "" {
		strategy.Description = req.Description
	}
	if req.Params != "" {
		strategy.Params = req.Params
	}
	if req.IsActive != nil {
		strategy.IsActive = *req.IsActive
	}
	if req.IsPublic != nil {
		strategy.IsPublic = *req.IsPublic
	}

	if err := s.strategyRepo.Update(ctx, strategy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "更新成功",
		"data": strategy,
	})
}

// DeleteStrategy 删除策略
func (s *StrategyService) DeleteStrategy(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	strategyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "策略ID错误"})
		return
	}

	ctx := c.Request.Context()
	strategy, err := s.strategyRepo.GetByID(ctx, uint(strategyID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "策略不存在"})
		return
	}

	// 检查权限
	if strategy.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权删除"})
		return
	}

	if err := s.strategyRepo.Delete(ctx, uint(strategyID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "删除成功",
	})
}

// ============ 交易信号接口 ============

// GetTradeSignals 获取交易信号
func (s *StrategyService) GetTradeSignals(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	strategyID := c.Query("strategy_id")
	symbol := c.Query("symbol")
	signalType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	ctx := c.Request.Context()

	var signals []*models.TradeSignal
	var total int64
	var err error

	if strategyID != "" {
		sid, _ := strconv.ParseUint(strategyID, 10, 32)
		// 检查策略是否属于当前用户
		strategy, err := s.strategyRepo.GetByID(ctx, uint(sid))
		if err != nil || (strategy.UserID != uid && !strategy.IsPublic) {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权访问"})
			return
		}
		signals, total, err = s.strategyRepo.GetSignalsByStrategyID(ctx, uint(sid), page, pageSize)
	} else {
		signals, total, err = s.strategyRepo.GetSignalsByUserID(ctx, uid, symbol, signalType, page, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":      signals,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// ============ 主函数 ============

func main() {
	cfg := config.LoadFromEnv()

	service, err := NewStrategyService(cfg)
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
			"service": "strategy-service",
		})
	})

	// API路由
	api := r.Group("/api/v1")
	{
		// 策略接口（需要认证）
		strategy := api.Group("/strategy")
		strategy.Use(service.AuthMiddleware())
		{
			strategy.GET("", service.GetStrategies)
			strategy.POST("", service.CreateStrategy)
			strategy.GET("/:id", service.GetStrategy)
			strategy.PUT("/:id", service.UpdateStrategy)
			strategy.DELETE("/:id", service.DeleteStrategy)
		}

		// 交易信号接口（需要认证）
		signals := api.Group("/signals")
		signals.Use(service.AuthMiddleware())
		{
			signals.GET("", service.GetTradeSignals)
		}
	}

	port := getEnv("STRATEGY_SERVICE_PORT", "8084")

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
