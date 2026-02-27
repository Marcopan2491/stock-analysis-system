package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"stock-analysis-system/backend/pkg/models"
)

// StockRepository 股票数据仓库接口
type StockRepository interface {
	Create(ctx context.Context, stock *models.Stock) error
	CreateBatch(ctx context.Context, stocks []*models.Stock) error
	Update(ctx context.Context, stock *models.Stock) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*models.Stock, error)
	GetBySymbol(ctx context.Context, symbol, exchange string) (*models.Stock, error)
	GetAll(ctx context.Context, offset, limit int) ([]*models.Stock, int64, error)
	GetByExchange(ctx context.Context, exchange string, offset, limit int) ([]*models.Stock, int64, error)
	GetByIndustry(ctx context.Context, industry string, offset, limit int) ([]*models.Stock, int64, error)
	Search(ctx context.Context, keyword string) ([]*models.Stock, error)
	GetActiveStocks(ctx context.Context) ([]*models.Stock, error)
	SymbolExists(ctx context.Context, symbol, exchange string) (bool, error)
}

// stockRepository 股票数据仓库实现
type stockRepository struct {
	db *gorm.DB
}

// NewStockRepository 创建股票数据仓库
func NewStockRepository(db *gorm.DB) StockRepository {
	return &stockRepository{db: db}
}

// Create 创建股票
func (r *stockRepository) Create(ctx context.Context, stock *models.Stock) error {
	return r.db.WithContext(ctx).Create(stock).Error
}

// CreateBatch 批量创建股票
func (r *stockRepository) CreateBatch(ctx context.Context, stocks []*models.Stock) error {
	if len(stocks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(stocks, 100).Error
}

// Update 更新股票
func (r *stockRepository) Update(ctx context.Context, stock *models.Stock) error {
	return r.db.WithContext(ctx).Save(stock).Error
}

// Delete 删除股票
func (r *stockRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Stock{}, id).Error
}

// GetByID 根据ID获取股票
func (r *stockRepository) GetByID(ctx context.Context, id uint) (*models.Stock, error) {
	var stock models.Stock
	if err := r.db.WithContext(ctx).First(&stock, id).Error; err != nil {
		return nil, err
	}
	return &stock, nil
}

// GetBySymbol 根据代码和交易所获取股票
func (r *stockRepository) GetBySymbol(ctx context.Context, symbol, exchange string) (*models.Stock, error) {
	var stock models.Stock
	if err := r.db.WithContext(ctx).
		Where("symbol = ? AND exchange = ?", symbol, exchange).
		First(&stock).Error; err != nil {
		return nil, err
	}
	return &stock, nil
}

// GetAll 获取所有股票
func (r *stockRepository) GetAll(ctx context.Context, offset, limit int) ([]*models.Stock, int64, error) {
	var stocks []*models.Stock
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Stock{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Offset(offset).Limit(limit).
		Order("symbol ASC").
		Find(&stocks).Error; err != nil {
		return nil, 0, err
	}

	return stocks, total, nil
}

// GetByExchange 根据交易所获取股票
func (r *stockRepository) GetByExchange(ctx context.Context, exchange string, offset, limit int) ([]*models.Stock, int64, error) {
	var stocks []*models.Stock
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&models.Stock{}).
		Where("exchange = ?", exchange).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Where("exchange = ?", exchange).
		Offset(offset).Limit(limit).
		Order("symbol ASC").
		Find(&stocks).Error; err != nil {
		return nil, 0, err
	}

	return stocks, total, nil
}

// GetByIndustry 根据行业获取股票
func (r *stockRepository) GetByIndustry(ctx context.Context, industry string, offset, limit int) ([]*models.Stock, int64, error) {
	var stocks []*models.Stock
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&models.Stock{}).
		Where("industry = ?", industry).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Where("industry = ?", industry).
		Offset(offset).Limit(limit).
		Order("symbol ASC").
		Find(&stocks).Error; err != nil {
		return nil, 0, err
	}

	return stocks, total, nil
}

// Search 搜索股票
func (r *stockRepository) Search(ctx context.Context, keyword string) ([]*models.Stock, error) {
	var stocks []*models.Stock
	
	query := r.db.WithContext(ctx).
		Where("symbol LIKE ? OR name LIKE ? OR full_name LIKE ?",
			"%"+keyword+"%",
			"%"+keyword+"%",
			"%"+keyword+"%").
		Order("symbol ASC")

	if err := query.Find(&stocks).Error; err != nil {
		return nil, err
	}

	return stocks, nil
}

// GetActiveStocks 获取活跃股票
func (r *stockRepository) GetActiveStocks(ctx context.Context) ([]*models.Stock, error) {
	var stocks []*models.Stock
	if err := r.db.WithContext(ctx).
		Where("status = ?", "active").
		Order("symbol ASC").
		Find(&stocks).Error; err != nil {
		return nil, err
	}
	return stocks, nil
}

// SymbolExists 检查股票代码是否存在
func (r *stockRepository) SymbolExists(ctx context.Context, symbol, exchange string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Stock{}).
		Where("symbol = ? AND exchange = ?", symbol, exchange).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
