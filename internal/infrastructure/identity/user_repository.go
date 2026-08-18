package identity

import (
	domainidentity "blog/internal/domain/identity"
	"blog/internal/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 返回直接持有 GORM 的 Identity 用户 Repository 实现。
func NewUserRepository(db *gorm.DB) domainidentity.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *domainidentity.User) error {
	m := toModelUser(user)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	user.ID = m.ID
	user.CreatedTime = m.CreatedTime
	user.UpdatedTime = m.UpdatedTime
	return nil
}

func (r *userRepository) GetUserByAccount(ctx context.Context, phone, nickname string) (*domainidentity.User, error) {
	var m model.User
	tx := r.db.WithContext(ctx).Model(&model.User{}).
		Select("id,phone,password,nickname,avatar,role").
		Where("status = ?", model.UserNormal)
	if phone != "" {
		tx = tx.Where("phone = ?", phone)
	}
	if nickname != "" {
		tx = tx.Where("nickname = ?", nickname)
	}
	if err := tx.Take(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainidentity.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(&m), nil
}

func (r *userRepository) FindUserByID(ctx context.Context, id uint64) (*domainidentity.User, error) {
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

func (r *userRepository) FindUsersByIDs(ctx context.Context, ids []uint64) ([]*domainidentity.User, error) {
	if len(ids) == 0 {
		return []*domainidentity.User{}, nil
	}
	var models []*model.User
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND status = ?", ids, model.UserNormal).
		Find(&models).Error; err != nil {
		return nil, err
	}
	users := make([]*domainidentity.User, 0, len(models))
	for _, m := range models {
		users = append(users, toDomainUser(m))
	}
	return users, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *domainidentity.User) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND status = ?", user.ID, model.UserNormal).
		Updates(toModelUser(user)).Error
}

func (r *userRepository) GetUserList(ctx context.Context, page, pageSize int, isDesc bool) ([]*domainidentity.User, error) {
	tx := r.db.WithContext(ctx).
		Select("ID,Nickname,Avatar").
		Where("status = ?", model.UserNormal)
	if isDesc {
		tx = tx.Order("id desc")
	} else {
		tx = tx.Order("id asc")
	}
	var models []*model.User
	if err := tx.Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return nil, err
	}
	users := make([]*domainidentity.User, 0, len(models))
	for _, m := range models {
		users = append(users, toDomainUser(m))
	}
	return users, nil
}

func (r *userRepository) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", model.UserNormal).Count(&count).Error
	return count, err
}

func toDomainUser(m *model.User) *domainidentity.User {
	return &domainidentity.User{
		ID:            m.ID,
		Nickname:      m.Nickname,
		Phone:         m.Phone,
		Password:      m.Password,
		Avatar:        m.Avatar,
		Role:          m.Role,
		Status:        m.Status,
		LastLoginIP:   m.LastLoginIp,
		LastLoginTime: m.LastLoginTime,
		CreatedTime:   m.CreatedTime,
		UpdatedTime:   m.UpdatedTime,
	}
}

func toModelUser(u *domainidentity.User) *model.User {
	return &model.User{
		ID:            u.ID,
		Nickname:      u.Nickname,
		Phone:         u.Phone,
		Password:      u.Password,
		Avatar:        u.Avatar,
		Role:          u.Role,
		Status:        u.Status,
		LastLoginIp:   u.LastLoginIP,
		LastLoginTime: u.LastLoginTime,
		CreatedTime:   u.CreatedTime,
		UpdatedTime:   u.UpdatedTime,
	}
}
