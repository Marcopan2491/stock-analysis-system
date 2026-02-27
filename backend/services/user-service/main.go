package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"stock-analysis-system/backend/pkg/config"
	"stock-analysis-system/backend/pkg/database"
	"stock-analysis-system/backend/pkg/models"
	"stock-analysis-system/backend/pkg/repository"
)

// UserService 用户服务
type UserService struct {
	cfg       *config.Config
	dbManager *database.Manager
	userRepo  repository.UserRepository
	jwtSecret []byte
}

// NewUserService 创建用户服务
func NewUserService(cfg *config.Config) (*UserService, error) {
	dbManager, err := database.NewManager(&cfg.Database)
	if err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepository(dbManager.Postgres.DB)

	jwtSecret := []byte(getEnv("JWT_SECRET", "your-secret-key"))

	return &UserService{
		cfg:       cfg,
		dbManager: dbManager,
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}, nil
}

// Close 关闭服务
func (s *UserService) Close() {
	if s.dbManager != nil {
		s.dbManager.Close()
	}
}

// ============ JWT 相关 ============

// Claims JWT声明
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT Token
func (s *UserService) GenerateToken(user *models.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "stock-analysis-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ParseToken 解析JWT Token
func (s *UserService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// AuthMiddleware JWT认证中间件
func (s *UserService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "缺少认证信息"})
			c.Abort()
			return
		}

		// Bearer Token
		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			tokenString = authHeader
		}

		claims, err := s.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "无效的认证信息"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// ============ 认证接口 ============

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register 用户注册
func (s *UserService) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	// 检查用户名是否已存在
	if _, err := s.userRepo.GetByUsername(ctx, req.Username); err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": "用户名已存在"})
		return
	}

	// 检查邮箱是否已存在
	if _, err := s.userRepo.GetByEmail(ctx, req.Email); err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": "邮箱已被注册"})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "密码加密失败"})
		return
	}

	// 创建用户
	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "注册失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "注册成功",
		"data": gin.H{
			"user_id":  user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Login 用户登录
func (s *UserService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	// 查询用户
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户名或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户名或密码错误"})
		return
	}

	// 检查用户状态
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "账号已被禁用"})
		return
	}

	// 生成Token
	token, err := s.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Token生成失败"})
		return
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	s.userRepo.Update(ctx, user)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "登录成功",
		"data": LoginResponse{
			UserID:      user.ID,
			Username:    user.Username,
			Email:       user.Email,
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   86400,
		},
	})
}

// ============ 用户信息接口 ============

// GetUserProfile 获取用户信息
func (s *UserService) GetUserProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	ctx := c.Request.Context()
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user_id":     user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"avatar_url":  user.AvatarURL,
			"phone":       user.Phone,
			"status":      user.Status,
			"created_at":  user.CreatedAt.Format("2006-01-02 15:04:05"),
			"last_login":  func() string { if user.LastLoginAt != nil { return user.LastLoginAt.Format("2006-01-02 15:04:05") } return "" }(),
		},
	})
}

// UpdateUserProfileRequest 更新用户信息请求
type UpdateUserProfileRequest struct {
	AvatarURL string `json:"avatar_url"`
	Phone     string `json:"phone"`
}

// UpdateUserProfile 更新用户信息
func (s *UserService) UpdateUserProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	var req UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	ctx := c.Request.Context()
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	user.AvatarURL = req.AvatarURL
	user.Phone = req.Phone

	if err := s.userRepo.Update(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "更新成功",
	})
}

// ============ 自选股接口 ============

// GetWatchlists 获取自选股列表
func (s *UserService) GetWatchlists(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	ctx := c.Request.Context()
	watchlists, err := s.userRepo.GetWatchlists(ctx, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": watchlists,
	})
}

// CreateWatchlistRequest 创建自选股分组请求
type CreateWatchlistRequest struct {
	Name        string `json:"name" binding:"required,max=50"`
	Description string `json:"description"`
}

// CreateWatchlist 创建自选股分组
func (s *UserService) CreateWatchlist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	var req CreateWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	ctx := c.Request.Context()
	watchlist := &models.Watchlist{
		UserID:      uid,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.userRepo.CreateWatchlist(ctx, watchlist); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "创建成功",
		"data": watchlist,
	})
}

// AddToWatchlistRequest 添加自选股请求
type AddToWatchlistRequest struct {
	Symbol   string `json:"symbol" binding:"required"`
	Exchange string `json:"exchange" binding:"required"`
}

// AddToWatchlist 添加自选股
func (s *UserService) AddToWatchlist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	watchlistID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "分组ID错误"})
		return
	}

	var req AddToWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	ctx := c.Request.Context()

	// 验证分组属于当前用户
	watchlist, err := s.userRepo.GetWatchlistByID(ctx, uint(watchlistID))
	if err != nil || watchlist.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权访问该分组"})
		return
	}

	item := &models.WatchlistItem{
		WatchlistID: uint(watchlistID),
		Symbol:      req.Symbol,
		Exchange:    req.Exchange,
	}

	if err := s.userRepo.AddToWatchlist(ctx, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "添加失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "添加成功",
	})
}

// RemoveFromWatchlist 移除自选股
func (s *UserService) RemoveFromWatchlist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	watchlistID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "分组ID错误"})
		return
	}

	symbol := c.Param("symbol")
	exchange := c.Query("exchange")

	ctx := c.Request.Context()

	// 验证分组属于当前用户
	watchlist, err := s.userRepo.GetWatchlistByID(ctx, uint(watchlistID))
	if err != nil || watchlist.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权访问该分组"})
		return
	}

	if err := s.userRepo.RemoveFromWatchlist(ctx, uint(watchlistID), symbol, exchange); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "移除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "移除成功",
	})
}

// ============ 主函数 ============

func main() {
	cfg := config.LoadFromEnv()

	service, err := NewUserService(cfg)
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
			"service": "user-service",
		})
	})

	// API路由
	api := r.Group("/api/v1")
	{
		// 认证接口（公开）
		auth := api.Group("/auth")
		{
			auth.POST("/register", service.Register)
			auth.POST("/login", service.Login)
		}

		// 用户接口（需要认证）
		user := api.Group("/user")
		user.Use(service.AuthMiddleware())
		{
			user.GET("/profile", service.GetUserProfile)
			user.PUT("/profile", service.UpdateUserProfile)
		}

		// 自选股接口（需要认证）
		watchlist := api.Group("/watchlist")
		watchlist.Use(service.AuthMiddleware())
		{
			watchlist.GET("", service.GetWatchlists)
			watchlist.POST("", service.CreateWatchlist)
			watchlist.POST("/:id/items", service.AddToWatchlist)
			watchlist.DELETE("/:id/items/:symbol", service.RemoveFromWatchlist)
		}
	}

	port := getEnv("USER_SERVICE_PORT", "8083")

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
