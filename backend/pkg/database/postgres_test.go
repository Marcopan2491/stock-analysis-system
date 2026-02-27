package database

import (
	"testing"
	"time"

	"stock-analysis-system/backend/pkg/config"
)

func TestNewPostgresClient(t *testing.T) {
	cfg := &config.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "test_user",
		Password: "test_pass",
		Database: "test_db",
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}

	// 注意：这需要真实的数据库连接
	// 在CI环境中应使用测试数据库或mock
	t.Skip("需要真实数据库连接，跳过")

	client, err := NewPostgresClient(cfg)
	if err != nil {
		t.Errorf("创建PostgreSQL客户端失败: %v", err)
		return
	}
	defer client.Close()

	// 测试健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		t.Errorf("健康检查失败: %v", err)
	}
}

func TestPostgresConfig_DSN(t *testing.T) {
	cfg := &config.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "stock_user",
		Password: "stock_pass",
		Database: "stock_analysis",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()
	expected := "host=localhost port=5432 user=stock_user password=stock_pass dbname=stock_analysis sslmode=disable"
	
	if dsn != expected {
		t.Errorf("DSN不正确，期望: %s, 实际: %s", expected, dsn)
	}
}
