package infra

import (
	domainidentity "blog/internal/user/domain"
	"blog/internal/user/infra/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

// userRepository 是 Identity 用户仓储的 GORM 实现。
type userRepository struct {
	db *gorm.DB // GORM 数据库连接
}

// NewUserRepository 返回直接持有 GORM 的 Identity 用户 Repository 实现。
func NewUserRepository(db *gorm.DB) domainidentity.UserRepository {
	// 1. 保存数据库连接并返回 Repository 实现
	return &userRepository{db: db}
}

// CreateUser 创建用户，并把数据库生成的主键与时间回填到领域对象。
func (r *userRepository) CreateUser(ctx context.Context, user *domainidentity.User) error {
	// 1. 领域对象转换为数据库模型
	m := toModelUser(user)
	// 2. 写入数据库
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	// 3. 回填自增ID与时间字段
	user.ID = m.ID
	user.CreatedTime = m.CreatedTime
	user.UpdatedTime = m.UpdatedTime
	return nil
}

// GetUserByAccount 按手机号或昵称查询正常状态的用户，用于登录校验。
func (r *userRepository) GetUserByAccount(ctx context.Context, phone, nickname string) (*domainidentity.User, error) {
	var m model.User
	// 1. 只查询登录所需字段，并限定用户状态正常
	tx := r.db.WithContext(ctx).Model(&model.User{}).
		Select("id,phone,password,nickname,avatar,role").
		Where("status = ?", model.UserNormal)
	// 2. 手机号与昵称任一非空即作为查询条件
	if phone != "" {
		tx = tx.Where("phone = ?", phone)
	}
	if nickname != "" {
		tx = tx.Where("nickname = ?", nickname)
	}
	// 3. 查询无结果时返回领域错误，便于上层映射为业务错误码
	if err := tx.Take(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainidentity.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(&m), nil
}

// FindUserByID 按用户 ID 查询正常状态的用户。
func (r *userRepository) FindUserByID(ctx context.Context, id uint64) (*domainidentity.User, error) {
	// 1. 查询正常状态的指定用户
	var m model.User
	err := r.db.WithContext(ctx).Where("id = ? AND status = ?", id, model.UserNormal).Take(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainidentity.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(&m), nil
}

// FindUsersByIDs 批量按用户 ID 查询正常状态的用户。
func (r *userRepository) FindUsersByIDs(ctx context.Context, ids []uint64) ([]*domainidentity.User, error) {
	// 1. ID 列表为空时直接返回空切片，避免无意义查询
	if len(ids) == 0 {
		return []*domainidentity.User{}, nil
	}
	// 2. 批量查询
	var models []*model.User
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND status = ?", ids, model.UserNormal).
		Find(&models).Error; err != nil {
		return nil, err
	}
	// 3. 逐条转换为领域对象
	users := make([]*domainidentity.User, 0, len(models))
	for _, m := range models {
		users = append(users, toDomainUser(m))
	}
	return users, nil
}

// UpdateUser 更新用户信息，仅允许更新状态正常的用户。
func (r *userRepository) UpdateUser(ctx context.Context, user *domainidentity.User) error {
	// 1. 仅更新状态正常的指定用户
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND status = ?", user.ID, model.UserNormal).
		Updates(toModelUser(user)).Error
}

// GetUserList 分页查询用户列表，按 ID 正序或倒序排列。
func (r *userRepository) GetUserList(ctx context.Context, page, pageSize int, isDesc bool) ([]*domainidentity.User, error) {
	// 1. 只查询列表所需字段，并限定用户状态正常
	tx := r.db.WithContext(ctx).
		Select("ID,Nickname,Avatar").
		Where("status = ?", model.UserNormal)
	// 2. 按 isDesc 决定排序方向
	if isDesc {
		tx = tx.Order("id desc")
	} else {
		tx = tx.Order("id asc")
	}
	// 3. 按页码换算 offset 后查询
	var models []*model.User
	if err := tx.Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return nil, err
	}
	// 4. 逐条转换为领域对象
	users := make([]*domainidentity.User, 0, len(models))
	for _, m := range models {
		users = append(users, toDomainUser(m))
	}
	return users, nil
}

// CountUsers 统计正常状态的用户总数。
func (r *userRepository) CountUsers(ctx context.Context) (int64, error) {
	// 1. 统计正常状态的用户总数
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", model.UserNormal).Count(&count).Error
	return count, err
}

// toDomainUser 把数据库模型转换为 User 领域用户对象。
func toDomainUser(m *model.User) *domainidentity.User {
	// 1. 映射数据库字段到领域对象
	return &domainidentity.User{
		ID:            m.ID,
		Nickname:      m.Nickname,
		Phone:         m.Phone,
		Password:      domainidentity.RestorePasswordHash(m.Password),
		Avatar:        m.Avatar,
		Role:          m.Role,
		Status:        m.Status,
		LastLoginIP:   m.LastLoginIp,
		LastLoginTime: m.LastLoginTime,
		CreatedTime:   m.CreatedTime,
		UpdatedTime:   m.UpdatedTime,
	}
}

// toModelUser 把 User 领域用户对象转换为数据库模型。
func toModelUser(u *domainidentity.User) *model.User {
	// 1. 映射领域字段到数据库模型
	return &model.User{
		ID:            u.ID,
		Nickname:      u.Nickname,
		Phone:         u.Phone,
		Password:      u.Password.String(),
		Avatar:        u.Avatar,
		Role:          u.Role,
		Status:        u.Status,
		LastLoginIp:   u.LastLoginIP,
		LastLoginTime: u.LastLoginTime,
		CreatedTime:   u.CreatedTime,
		UpdatedTime:   u.UpdatedTime,
	}
}
