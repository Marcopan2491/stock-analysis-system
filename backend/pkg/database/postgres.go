package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"stock-analysis-system/backend/pkg/config"
)

// PostgresClient PostgreSQL客户端
type PostgresClient struct {
	DB     *gorm.DB
	config *config.PostgresConfig
}

// NewPostgresClient 创建PostgreSQL客户端
func NewPostgresClient(cfg *config.PostgresConfig) (*PostgresClient, error) {
	client := &PostgresClient{
		config: cfg,
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}

// Connect 连接到PostgreSQL
func (c *PostgresClient) Connect() error {
	// GORM配置
	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",    // 表名前缀
			SingularTable: false, // 使用复数表名
		},
		Logger: logger.Default.LogMode(logger.Info),
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(c.config.DSN()), gormConfig)
	if err != nil {
		return fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	// 获取底层SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取SQL DB失败: %w", err)
	}

	// 设置连接池
	sqlDB.SetMaxOpenConns(c.config.MaxConns)
	sqlDB.SetMaxIdleConns(c.config.MinConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("Ping PostgreSQL失败: %w", err)
	}

	c.DB = db
	return nil
}

// Close 关闭连接
func (c *PostgresClient) Close() error {
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// HealthCheck 健康检查
func (c *PostgresClient) HealthCheck(ctx context.Context) error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Transaction 执行事务
func (c *PostgresClient) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return c.DB.WithContext(ctx).Transaction(fn)
}

// AutoMigrate 自动迁移表结构
func (c *PostgresClient) AutoMigrate(models ...interface{}) error {
	return c.DB.AutoMigrate(models...)
}
