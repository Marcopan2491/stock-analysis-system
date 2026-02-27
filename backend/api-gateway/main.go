package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	// 数据库连接
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "host=localhost user=stock_user password=stock_password dbname=stock_db port=5432 sslmode=disable"
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 创建路由
	r := gin.Default()

	// API v1 路由组
	api := r.Group("/api/v1")
	{
		// 行情接口
		market := api.Group("/market")
		{
			market.GET("/stocks", getStockList)
			market.GET("/quote/:symbol", getRealtimeQuote)
			market.GET("/kline/:symbol", getKlineData)
		}

		// 策略接口
		strategy := api.Group("/strategy")
		{
			strategy.POST("/backtest", runBacktest)
			strategy.GET("/signals", getSignals)
			strategy.GET("/performance", getPerformance)
		}

		// 用户接口
		user := api.Group("/user")
		{
			user.GET("/watchlist", getWatchlist)
			user.POST("/watchlist", addToWatchlist)
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Println("API Gateway starting on :8080")
	r.Run(":8080")
}

// 获取股票列表
func getStockList(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Stock list endpoint",
		"status": "not implemented yet",
	})
}

// 获取实时行情
func getRealtimeQuote(c *gin.Context) {
	symbol := c.Param("symbol")
	c.JSON(200, gin.H{
		"symbol": symbol,
		"message": "Realtime quote endpoint",
		"status": "not implemented yet",
	})
}

// 获取K线数据
func getKlineData(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1d")
	c.JSON(200, gin.H{
		"symbol":   symbol,
		"interval": interval,
		"message":  "Kline data endpoint",
		"status":   "not implemented yet",
	})
}

// 运行回测
func runBacktest(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Backtest endpoint",
		"status": "not implemented yet",
	})
}

// 获取交易信号
func getSignals(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Signals endpoint",
		"status": "not implemented yet",
	})
}

// 获取策略绩效
func getPerformance(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Performance endpoint",
		"status": "not implemented yet",
	})
}

// 获取自选股
func getWatchlist(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Watchlist endpoint",
		"status": "not implemented yet",
	})
}

// 添加自选股
func addToWatchlist(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Add to watchlist endpoint",
		"status": "not implemented yet",
	})
}