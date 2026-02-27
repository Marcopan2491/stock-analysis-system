package repository

import (
	"context"

	"gorm.io/gorm"
	"stock-analysis-system/backend/pkg/models"
)

// StrategyRepository 策略数据仓库接口
type StrategyRepository interface {
	Create(ctx context.Context, strategy *models.Strategy) error
	Update(ctx context.Context, strategy *models.Strategy) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*models.Strategy, error)
	GetByUserID(ctx context.Context, userID uint, strategyType string, page, pageSize int) ([]*models.Strategy, int64, error)
	
	// 交易信号相关
	GetSignalsByStrategyID(ctx context.Context, strategyID uint, page, pageSize int) ([]*models.TradeSignal, int64, error)
	GetSignalsByUserID(ctx context.Context, userID uint, symbol, signalType string, page, pageSize int) ([]*models.TradeSignal, int64, error)
	CreateSignal(ctx context.Context, signal *models.TradeSignal) error
}

// strategyRepository 策略数据仓库实现
type strategyRepository struct {
	db *gorm.DB
}

// NewStrategyRepository 创建策略数据仓库
func NewStrategyRepository(db *gorm.DB) StrategyRepository {
	return &strategyRepository{db: db}
}

// Create 创建策略
func (r *strategyRepository) Create(ctx context.Context, strategy *models.Strategy) error {
	return r.db.WithContext(ctx).Create(strategy).Error
}

// Update 更新策略
func (r *strategyRepository) Update(ctx context.Context, strategy *models.Strategy) error {
	return r.db.WithContext(ctx).Save(strategy).Error
}

// Delete 删除策略
func (r *strategyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Strategy{}, id).Error
}

// GetByID 根据ID获取策略
func (r *strategyRepository) GetByID(ctx context.Context, id uint) (*models.Strategy, error) {
	var strategy models.Strategy
	if err := r.db.WithContext(ctx).First(&strategy, id).Error; err != nil {
		return nil, err
	}
	return &strategy, nil
}

// GetByUserID 获取用户的策略列表
func (r *strategyRepository) GetByUserID(ctx context.Context, userID uint, strategyType string, page, pageSize int) ([]*models.Strategy, int64, error) {
	var strategies []*models.Strategy
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Strategy{}).Where("user_id = ? OR is_public = true", userID)
	
	if strategyType != "" {
		query = query.Where("type = ?", strategyType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&strategies).Error; err != nil {
		return nil, 0, err
	}

	return strategies, total, nil
}

// GetSignalsByStrategyID 获取策略的交易信号
func (r *strategyRepository) GetSignalsByStrategyID(ctx context.Context, strategyID uint, page, pageSize int) ([]*models.TradeSignal, int64, error) {
	var signals []*models.TradeSignal
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.TradeSignal{}).Where("strategy_id = ?", strategyID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Where("strategy_id = ?", strategyID).Offset((page - 1) * pageSize).Limit(pageSize).Find(&signals).Error; err != nil {
		return nil, 0, err
	}

	return signals, total, nil
}

// GetSignalsByUserID 获取用户的交易信号
func (r *strategyRepository) GetSignalsByUserID(ctx context.Context, userID uint, symbol, signalType string, page, pageSize int) ([]*models.TradeSignal, int64, error) {
	var signals []*models.TradeSignal
	var total int64

	// 先获取用户的所有策略ID
	var strategyIDs []uint
	if err := r.db.WithContext(ctx).Model(&models.Strategy{}).Where("user_id = ?", userID).Pluck("id", &strategyIDs).Error; err != nil {
		return nil, 0, err
	}

	query := r.db.WithContext(ctx).Model(&models.TradeSignal{}).Where("strategy_id IN ?", strategyIDs)
	
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}
	if signalType != "" {
		query = query.Where("signal_type = ?", signalType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&signals).Error; err != nil {
		return nil, 0, err
	}

	return signals, total, nil
}

// CreateSignal 创建交易信号
func (r *strategyRepository) CreateSignal(ctx context.Context, signal *models.TradeSignal) error {
	return r.db.WithContext(ctx).Create(signal).Error
}
