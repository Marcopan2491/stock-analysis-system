package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
	Log      LogConfig      `yaml:"log"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres"`
	InfluxDB InfluxDBConfig `yaml:"influxdb"`
	Redis    RedisConfig    `yaml:"redis"`
}

// PostgresConfig PostgreSQL配置
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
	MaxConns int    `yaml:"max_conns"`
	MinConns int    `yaml:"min_conns"`
}

// InfluxDBConfig InfluxDB配置
type InfluxDBConfig struct {
	URL       string `yaml:"url"`
	Token     string `yaml:"token"`
	Org       string `yaml:"org"`
	Bucket    string `yaml:"bucket"`
	BatchSize int    `yaml:"batch_size"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Port         int    `yaml:"port"`
	Mode         string `yaml:"mode"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// DSN 生成PostgreSQL连接字符串
func (p *PostgresConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Database, p.SSLMode)
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	cfg.setDefaults()

	return &cfg, nil
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() *Config {
	cfg := &Config{}
	
	// PostgreSQL
	cfg.Database.Postgres.Host = getEnv("POSTGRES_HOST", "localhost")
	cfg.Database.Postgres.Port = getEnvInt("POSTGRES_PORT", 5432)
	cfg.Database.Postgres.User = getEnv("POSTGRES_USER", "stock_user")
	cfg.Database.Postgres.Password = getEnv("POSTGRES_PASSWORD", "stock_pass")
	cfg.Database.Postgres.Database = getEnv("POSTGRES_DB", "stock_analysis")
	cfg.Database.Postgres.SSLMode = getEnv("POSTGRES_SSLMODE", "disable")
	cfg.Database.Postgres.MaxConns = getEnvInt("POSTGRES_MAX_CONNS", 20)
	cfg.Database.Postgres.MinConns = getEnvInt("POSTGRES_MIN_CONNS", 5)
	
	// InfluxDB
	cfg.Database.InfluxDB.URL = getEnv("INFLUXDB_URL", "http://localhost:8086")
	cfg.Database.InfluxDB.Token = getEnv("INFLUXDB_TOKEN", "")
	cfg.Database.InfluxDB.Org = getEnv("INFLUXDB_ORG", "stock_org")
	cfg.Database.InfluxDB.Bucket = getEnv("INFLUXDB_BUCKET", "stock_market")
	cfg.Database.InfluxDB.BatchSize = getEnvInt("INFLUXDB_BATCH_SIZE", 100)
	
	// Redis
	cfg.Database.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Database.Redis.Port = getEnvInt("REDIS_PORT", 6379)
	cfg.Database.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Database.Redis.DB = getEnvInt("REDIS_DB", 0)
	
	// Server
	cfg.Server.Port = getEnvInt("SERVER_PORT", 8080)
	cfg.Server.Mode = getEnv("SERVER_MODE", "release")
	cfg.Server.ReadTimeout = getEnvInt("SERVER_READ_TIMEOUT", 30)
	cfg.Server.WriteTimeout = getEnvInt("SERVER_WRITE_TIMEOUT", 30)
	
	// Log
	cfg.Log.Level = getEnv("LOG_LEVEL", "info")
	cfg.Log.Format = getEnv("LOG_FORMAT", "json")
	cfg.Log.Output = getEnv("LOG_OUTPUT", "stdout")
	
	cfg.setDefaults()
	return cfg
}

// setDefaults 设置默认值
func (c *Config) setDefaults() {
	if c.Database.Postgres.MaxConns == 0 {
		c.Database.Postgres.MaxConns = 20
	}
	if c.Database.Postgres.MinConns == 0 {
		c.Database.Postgres.MinConns = 5
	}
	if c.Database.InfluxDB.BatchSize == 0 {
		c.Database.InfluxDB.BatchSize = 100
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
