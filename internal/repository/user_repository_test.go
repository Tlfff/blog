package repository

import (
	"testing"

	"blog/internal/model"
)

func newTestUser(id int64, phone string, status int8) *model.User {
	return &model.User{
		ID:       id,
		Phone:    phone,
		Nickname: "test",
		Password: "pwd",
		Status:   status,
	}
}

func TestUserRepository_CreateUser(t *testing.T) {
	repo := NewUserRepository()

	user := newTestUser(1, "13800138000", 1)

	err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	got, err := repo.GetUserByAccount("13800138000")
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	if got.Phone != "13800138000" {
		t.Fatalf("手机号不匹配")
	}
}

func TestUserRepository_GetUserByPhone_NotFound(t *testing.T) {
	repo := NewUserRepository()

	_, err := repo.GetUserByAccount("not-exist")
	if err == nil {
		t.Fatalf("期望用户不存在错误，但没有返回")
	}
}

func TestUserRepository_GetUserByPhone_Disabled(t *testing.T) {
	repo := NewUserRepository()

	repo.CreateUser(newTestUser(1, "13800138000", 0)) // status=0 禁用

	_, err := repo.GetUserByAccount("13800138000")
	if err == nil {
		t.Fatalf("期望禁用用户错误，但没有返回")
	}
}

func TestUserRepository_FindUserByID(t *testing.T) {
	repo := NewUserRepository()

	repo.CreateUser(newTestUser(100, "13800138000", 1))

	user, err := repo.FindUserByID(100)
	if err != nil {
		t.Fatalf("查找失败: %v", err)
	}

	if user.ID != 100 {
		t.Fatalf("用户ID不匹配")
	}
}

func TestUserRepository_FindUserByID_NotFound(t *testing.T) {
	repo := NewUserRepository()

	_, err := repo.FindUserByID(999)
	if err == nil {
		t.Fatalf("期望找不到用户错误")
	}
}

func TestUserRepository_UpdateUser(t *testing.T) {
	repo := NewUserRepository()

	repo.CreateUser(newTestUser(1, "13800138000", 1))

	updated := newTestUser(1, "13800138000", 1)
	updated.Nickname = "updated-name"

	err := repo.UpdateUser(updated)
	if err != nil {
		t.Fatalf("更新失败: %v", err)
	}

	user, _ := repo.GetUserByAccount("13800138000")

	if user.Nickname != "updated-name" {
		t.Fatalf("更新未生效")
	}
}
