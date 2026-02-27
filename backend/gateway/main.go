package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Timeout int    `json:"timeout"`
	Healthy bool   `json:"healthy"`
}

// APIGateway API网关
type APIGateway struct {
	services map[string]*ServiceConfig
	logger   *zap.Logger
	client   *http.Client
}

// NewAPIGateway 创建API网关
func NewAPIGateway() *APIGateway {
	return &APIGateway{
		services: make(map[string]*ServiceConfig),
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// LoadServiceConfig 加载服务配置
func (g *APIGateway) LoadServiceConfig() {
	// 从环境变量或配置文件加载
	g.services["market"] = &ServiceConfig{
		Name:    "market-service",
		URL:     getEnv("MARKET_SERVICE_URL", "http://localhost:8082"),
		Timeout: 30,
		Healthy: true,
	}
	g.services["user"] = &ServiceConfig{
		Name:    "user-service",
		URL:     getEnv("USER_SERVICE_URL", "http://localhost:8083"),
		Timeout: 30,
		Healthy: true,
	}
	g.services["strategy"] = &ServiceConfig{
		Name:    "strategy-service",
		URL:     getEnv("STRATEGY_SERVICE_URL", "http://localhost:8084"),
		Timeout: 30,
		Healthy: true,
	}
	g.services["backtest"] = &ServiceConfig{
		Name:    "backtest-service",
		URL:     getEnv("BACKTEST_SERVICE_URL", "http://localhost:8085"),
		Timeout: 60,
		Healthy: true,
	}
	g.services["data"] = &ServiceConfig{
		Name:    "data-service",
		URL:     getEnv("DATA_SERVICE_URL", "http://localhost:8081"),
		Timeout: 60,
		Healthy: true,
	}
}

// GetServiceProxy 获取服务代理
func (g *APIGateway) GetServiceProxy(serviceName string) *httputil.ReverseProxy {
	service, exists := g.services[serviceName]
	if !exists {
		return nil
	}

	target, _ := url.Parse(service.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// 自定义Director
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1/"+serviceName)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", target.Host)
	}

	// 错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		g.logger.Error("代理请求失败", zap.String("service", serviceName), zap.Error(err))
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(gin.H{
			"code": 503,
			"msg":  "服务暂时不可用",
		})
	}

	return proxy
}

// HealthCheck 服务健康检查
func (g *APIGateway) HealthCheck(serviceName string) bool {
	service, exists := g.services[serviceName]
	if !exists {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", service.URL+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := g.client.Do(req)
	if err != nil {
		service.Healthy = false
		return false
	}
	defer resp.Body.Close()

	service.Healthy = resp.StatusCode == 200
	return service.Healthy
}

// HealthCheckAll 检查所有服务
func (g *APIGateway) HealthCheckAll() map[string]bool {
	results := make(map[string]bool)
	for name := range g.services {
		results[name] = g.HealthCheck(name)
	}
	return results
}

func main() {
	// 初始化配置
	initConfig()

	// 初始化日志
	logger := initLogger()
	defer logger.Sync()

	// 创建网关
	gateway := NewAPIGateway()
	gateway.logger = logger
	gateway.LoadServiceConfig()

	// 设置运行模式
	if viper.GetString("app.mode") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	r.Use(requestLogger(logger))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		results := gateway.HealthCheckAll()
		allHealthy := true
		for _, healthy := range results {
			if !healthy {
				allHealthy = false
				break
			}
		}

		status := "healthy"
		code := http.StatusOK
		if !allHealthy {
			status = "degraded"
			code = http.StatusOK // 仍返回200，但状态显示降级
		}

		c.JSON(code, gin.H{
			"status":    status,
			"services":  results,
			"timestamp": time.Now().Unix(),
		})
	})

	// API路由组 - 服务路由
	api := r.Group("/api/v1")
	{
		// 行情服务路由
		market := api.Group("/market")
		{
			market.Any("/*path", func(c *gin.Context) {
				proxy := gateway.GetServiceProxy("market")
				if proxy == nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "服务不可用"})
					return
				}
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		}

		// 用户服务路由
		user := api.Group("/user")
		{
			user.Any("/*path", func(c *gin.Context) {
				proxy := gateway.GetServiceProxy("user")
				if proxy == nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "服务不可用"})
					return
				}
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		}

		// 认证路由（映射到用户服务）
		auth := api.Group("/auth")
		{
			auth.Any("/*path", func(c *gin.Context) {
				proxy := gateway.GetServiceProxy("user")
				if proxy == nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "服务不可用"})
					return
				}
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		}

		// 策略服务路由
		strategy := api.Group("/strategy")
		{
			strategy.Any("/*path", func(c *gin.Context) {
				proxy := gateway.GetServiceProxy("strategy")
				if proxy == nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "服务不可用"})
					return
				}
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		}

		// 回测服务路由
		backtest := api.Group("/backtest")
		{
			backtest.Any("/*path", func(c *gin.Context) {
				proxy := gateway.GetServiceProxy("backtest")
				if proxy == nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "服务不可用"})
					return
				}
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		}

		// 数据同步服务路由
		data := api.Group("/data")
		{
			data.Any("/*path", func(c *gin.Context) {
				proxy := gateway.GetServiceProxy("data")
				if proxy == nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "服务不可用"})
					return
				}
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		}
	}

	// 启动HTTP服务
	srv := &http.Server{
		Addr:    ":" + viper.GetString("app.port"),
		Handler: r,
	}

	// 优雅关机
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Info("API Gateway started", zap.String("port", viper.GetString("app.port")))

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// 初始化配置
func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 默认值
	viper.SetDefault("app.port", "8080")
	viper.SetDefault("app.mode", "development")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Config file not found, using defaults: %v", err)
	}
}

// 初始化日志
func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout", "./logs/api-gateway.log"}
	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	return logger
}

// CORS中间件
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

// 请求日志中间件
func requestLogger(logger *zap.Logger) gin.HandlerFunc {
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

		logger.Info("HTTP Request",
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
		)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
