package database

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	"stock-analysis-system/backend/pkg/config"
)

// InfluxClient InfluxDB客户端
type InfluxClient struct {
	client    influxdb2.Client
	writeAPI  api.WriteAPI
	queryAPI  api.QueryAPI
	deleteAPI api.DeleteAPI
	org       string
	bucket    string
	batchSize int
}

// NewInfluxClient 创建InfluxDB客户端
func NewInfluxClient(cfg *config.InfluxDBConfig) (*InfluxClient, error) {
	// 创建客户端
	client := influxdb2.NewClient(cfg.URL, cfg.Token)
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if _, err := client.Health(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("连接InfluxDB失败: %w", err)
	}

	// 创建写入API（异步批量写入）
	writeAPI := client.WriteAPI(cfg.Org, cfg.Bucket)
	
	// 创建查询和删除API
	queryAPI := client.QueryAPI(cfg.Org)
	deleteAPI := client.DeleteAPI()

	return &InfluxClient{
		client:    client,
		writeAPI:  writeAPI,
		queryAPI:  queryAPI,
		deleteAPI: deleteAPI,
		org:       cfg.Org,
		bucket:    cfg.Bucket,
		batchSize: cfg.BatchSize,
	}, nil
}

// Close 关闭客户端
func (c *InfluxClient) Close() {
	if c.writeAPI != nil {
		c.writeAPI.Flush()
	}
	if c.client != nil {
		c.client.Close()
	}
}

// HealthCheck 健康检查
func (c *InfluxClient) HealthCheck(ctx context.Context) error {
	_, err := c.client.Health(ctx)
	return err
}

// WritePoint 写入单条数据点
func (c *InfluxClient) WritePoint(point *write.Point) {
	c.writeAPI.WritePoint(point)
}

// WritePoints 批量写入数据点
func (c *InfluxClient) WritePoints(points []*write.Point) {
	for _, point := range points {
		c.writeAPI.WritePoint(point)
	}
}

// Flush 刷新缓冲区
func (c *InfluxClient) Flush() {
	c.writeAPI.Flush()
}

// Query 执行Flux查询
func (c *InfluxClient) Query(ctx context.Context, query string) (*api.QueryTableResult, error) {
	return c.queryAPI.Query(ctx, query)
}

// QueryRaw 执行原始Flux查询
func (c *InfluxClient) QueryRaw(ctx context.Context, query string) (string, error) {
	return c.queryAPI.QueryRaw(ctx, query, influxdb2.DefaultDialect())
}

// Delete 删除数据
func (c *InfluxClient) Delete(ctx context.Context, start, stop time.Time, predicate string) error {
	return c.deleteAPI.DeleteWithName(ctx, c.org, c.bucket, start, stop, predicate)
}

// GetOrg 获取组织名
func (c *InfluxClient) GetOrg() string {
	return c.org
}

// GetBucket 获取Bucket名
func (c *InfluxClient) GetBucket() string {
	return c.bucket
}

// GetBatchSize 获取批量大小
func (c *InfluxClient) GetBatchSize() int {
	return c.batchSize
}

// GetQueryAPI 获取查询API
func (c *InfluxClient) GetQueryAPI() api.QueryAPI {
	return c.queryAPI
}

// GetWriteAPI 获取写入API
func (c *InfluxClient) GetWriteAPI() api.WriteAPI {
	return c.writeAPI
}
