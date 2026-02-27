package repository

import (
	"context"

	"gorm.io/gorm"
	"stock-analysis-system/backend/pkg/models"
)

// BacktestRepository 回测数据仓库接口
type BacktestRepository interface {
	Create(ctx context.Context, record *models.BacktestRecord) error
	Update(ctx context.Context, record *models.BacktestRecord) error
	GetByID(ctx context.Context, id uint) (*models.BacktestRecord, error)
	GetByStrategyID(ctx context.Context, strategyID uint, page, pageSize int) ([]*models.BacktestRecord, int64, error)
	GetByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*models.BacktestRecord, int64, error)
}

// backtestRepository 回测数据仓库实现
type backtestRepository struct {
	db *gorm.DB
}

// NewBacktestRepository 创建回测数据仓库
func NewBacktestRepository(db *gorm.DB) BacktestRepository {
	return &backtestRepository{db: db}
}

// Create 创建回测记录
func (r *backtestRepository) Create(ctx context.Context, record *models.BacktestRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// Update 更新回测记录
func (r *backtestRepository) Update(ctx context.Context, record *models.BacktestRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// GetByID 根据ID获取回测记录
func (r *backtestRepository) GetByID(ctx context.Context, id uint) (*models.BacktestRecord, error) {
	var record models.BacktestRecord
	if err := r.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByStrategyID 获取策略的回测记录
func (r *backtestRepository) GetByStrategyID(ctx context.Context, strategyID uint, page, pageSize int) ([]*models.BacktestRecord, int64, error) {
	var records []*models.BacktestRecord
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.BacktestRecord{}).Where("strategy_id = ?", strategyID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Where("strategy_id = ?", strategyID).Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetByUserID 获取用户的所有回测记录
func (r *backtestRepository) GetByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*models.BacktestRecord, int64, error) {
	var records []*models.BacktestRecord
	var total int64

	// 通过策略ID关联查询
	subQuery := r.db.Model(&models.Strategy{}).Where("user_id = ?", userID).Select("id")

	if err := r.db.WithContext(ctx).Model(&models.BacktestRecord{}).Where("strategy_id IN (?)", subQuery).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Where("strategy_id IN (?)", subQuery).Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}
