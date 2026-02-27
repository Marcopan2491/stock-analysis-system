package database

import (
	"context"
	"fmt"

	"stock-analysis-system/backend/pkg/config"
)

// Manager 数据库管理器
type Manager struct {
	Postgres *PostgresClient
	Influx   *InfluxClient
	config   *config.DatabaseConfig
}

// NewManager 创建数据库管理器
func NewManager(cfg *config.DatabaseConfig) (*Manager, error) {
	manager := &Manager{
		config: cfg,
	}

	// 连接PostgreSQL
	if cfg.Postgres.Host != "" {
		postgresClient, err := NewPostgresClient(&cfg.Postgres)
		if err != nil {
			return nil, fmt.Errorf("初始化PostgreSQL失败: %w", err)
		}
		manager.Postgres = postgresClient
	}

	// 连接InfluxDB
	if cfg.InfluxDB.URL != "" {
		influxClient, err := NewInfluxClient(&cfg.InfluxDB)
		if err != nil {
			return nil, fmt.Errorf("初始化InfluxDB失败: %w", err)
		}
		manager.Influx = influxClient
	}

	return manager, nil
}

// Close 关闭所有数据库连接
func (m *Manager) Close() error {
	var errs []error

	if m.Postgres != nil {
		if err := m.Postgres.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭PostgreSQL失败: %w", err))
		}
	}

	if m.Influx != nil {
		m.Influx.Close()
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// HealthCheck 健康检查所有数据库
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)

	if m.Postgres != nil {
		if err := m.Postgres.HealthCheck(ctx); err != nil {
			results["postgres"] = err
		} else {
			results["postgres"] = nil
		}
	}

	if m.Influx != nil {
		if err := m.Influx.HealthCheck(ctx); err != nil {
			results["influxdb"] = err
		} else {
			results["influxdb"] = nil
		}
	}

	return results
}

// IsHealthy 检查是否所有数据库都健康
func (m *Manager) IsHealthy(ctx context.Context) bool {
	results := m.HealthCheck(ctx)
	for _, err := range results {
		if err != nil {
			return false
		}
	}
	return true
}
