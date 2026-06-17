package repository

import (
	"errors"
	"log"

	"blog/internal/common"
	"blog/internal/model"
)

type UserRepository struct {
	users map[string]*model.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: make(map[string]*model.User),
	}
}

// 创建新用户
func (m *UserRepository) CreateUser(user *model.User) error {
	m.users[user.Phone] = user
	return nil
}

// 根据手机号获取用户信息
func (m *UserRepository) GetUserByPhone(phone string) (*model.User, error) {
	user, ok := m.users[phone]
	if !ok || user.Status != 1 {
		log.Printf("用户%s不存在或已被禁用", phone)
		return nil, errors.New(common.ErrUserNotFound.Error())
	}
	return user, nil
}

// 根据用户ID获取用户信息(因为现在map用的phone作key)
func (m *UserRepository) FindUserByID(id int64) (*model.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New(common.ErrUserNotFound.Error())
}

// 更新用户信息
func (m *UserRepository) UpdateUser(user *model.User) error {
	m.users[user.Phone] = user
	return nil
}
