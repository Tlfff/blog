package repository

import (
	"blog/internal/model"

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
func (r *UserRepository) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

// 根据账户获取用户信息
// select id,phone,password,nickname,avatar,role from users where phone =? and status=1
func (r *UserRepository) GetUserByAccount(account string) (*model.User, error) {
	var user model.User
	err := r.db.Where("phone = ? AND status = ?", account, 1).Take(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 根据用户ID获取用户信息
// select id,phone,password,nickname,avatar,role from users where id =? and status=1
func (r *UserRepository) FindUserByID(id uint64) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ? AND status = ?", id, 1).Take(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 更新用户信息
// update users set nickname=?,avatar=?,phone=?,password=? where id=? and status=1
func (r *UserRepository) UpdateUser(user *model.User) error {
	result := r.db.Model(&model.User{}).
		Where("id = ? AND status = ?", user.ID, 1).
		Updates(user)

	if result.Error != nil {
		return result.Error
	}

	return nil
}
