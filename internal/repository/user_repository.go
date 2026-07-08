package repository

import (
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// 创建新用户
// insert into users (phone, password, nickname, avatar, role, status) values (?, ?, ?, ?, ?, ?)
func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// 根据账户获取用户信息
// select id,phone,password,nickname,avatar,role from users where phone = ? and status=1
// 或 select id,phone,password,nickname,avatar,role from users where nickname = ? and status=1
func (r *UserRepository) GetUserByAccount(ctx context.Context, phone, nickname string) (*model.User, error) {
	var user model.User
	tx := r.db.WithContext(ctx).Model(&model.User{}).
		Select("id,phone,password,nickname,avatar,role").
		Where("status = 1")

	if phone != "" {
		tx = tx.Where("phone = ?", phone)
	}
	if nickname != "" {
		tx = tx.Where("nickname = ?", nickname)
	}

	err := tx.Take(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 根据用户ID获取用户信息
// select id,phone,password,nickname,avatar,role,last_login_ip,last_login_time from users where id =? and status=1
func (r *UserRepository) FindUserByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ? AND status = ?", id, 1).Take(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 批量根据用户ID获取用户信息 (组装用户字典专用)
// select id,phone,password,nickname,avatar,role,last_login_ip,last_login_time from users where id in (?, ?, ...) and status=1
func (r *UserRepository) FindUsersByIDs(ctx context.Context, ids []uint64) ([]*model.User, error) {
	var users []*model.User

	// 兜底防御：如果传进来的 ids 切片是空的，直接返回空切片
	if len(ids) == 0 {
		return users, nil
	}

	// 使用 Where("id IN ?", ids) 配合 status=1 进行高性能批量检索
	err := r.db.WithContext(ctx).
		Where("id IN ? AND status = ?", ids, 1).
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

// 更新用户信息
// update users set nickname=?,avatar=?,phone=?,password=? where id=? and status=1
func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND status = ?", user.ID, 1).
		Updates(user)

	if result.Error != nil {
		return result.Error
	}

	return nil
}
