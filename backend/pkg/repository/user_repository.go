package repository

import (
	"context"

	"gorm.io/gorm"
	"stock-analysis-system/backend/pkg/models"
)

// UserRepository 用户数据仓库接口
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	
	// 自选股相关
	GetWatchlists(ctx context.Context, userID uint) ([]*models.Watchlist, error)
	GetWatchlistByID(ctx context.Context, id uint) (*models.Watchlist, error)
	CreateWatchlist(ctx context.Context, watchlist *models.Watchlist) error
	AddToWatchlist(ctx context.Context, item *models.WatchlistItem) error
	RemoveFromWatchlist(ctx context.Context, watchlistID uint, symbol, exchange string) error
}

// userRepository 用户数据仓库实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据仓库
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 更新用户
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetWatchlists 获取用户的自选股分组
func (r *userRepository) GetWatchlists(ctx context.Context, userID uint) ([]*models.Watchlist, error) {
	var watchlists []*models.Watchlist
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("user_id = ?", userID).
		Find(&watchlists).Error; err != nil {
		return nil, err
	}
	return watchlists, nil
}

// GetWatchlistByID 根据ID获取自选股分组
func (r *userRepository) GetWatchlistByID(ctx context.Context, id uint) (*models.Watchlist, error) {
	var watchlist models.Watchlist
	if err := r.db.WithContext(ctx).First(&watchlist, id).Error; err != nil {
		return nil, err
	}
	return &watchlist, nil
}

// CreateWatchlist 创建自选股分组
func (r *userRepository) CreateWatchlist(ctx context.Context, watchlist *models.Watchlist) error {
	return r.db.WithContext(ctx).Create(watchlist).Error
}

// AddToWatchlist 添加自选股
func (r *userRepository) AddToWatchlist(ctx context.Context, item *models.WatchlistItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// RemoveFromWatchlist 移除自选股
func (r *userRepository) RemoveFromWatchlist(ctx context.Context, watchlistID uint, symbol, exchange string) error {
	return r.db.WithContext(ctx).
		Where("watchlist_id = ? AND symbol = ? AND exchange = ?", watchlistID, symbol, exchange).
		Delete(&models.WatchlistItem{}).Error
}
