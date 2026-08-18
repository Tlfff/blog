package identity

import (
	domainidentity "blog/internal/domain/identity"
	"blog/internal/model"
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepositoryCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法启动内存测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}

	repo := NewUserRepository(db)
	ctx := context.Background()
	now := time.Now()
	user := &domainidentity.User{
		Nickname:      "测试用户",
		Phone:         "13800138000",
		Password:      "hash",
		Avatar:        "avatar",
		Role:          domainidentity.RoleUser,
		Status:        domainidentity.StatusNormal,
		LastLoginIP:   "127.0.0.1",
		LastLoginTime: now,
		CreatedTime:   now,
		UpdatedTime:   now,
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("创建用户后 ID 未回填")
	}

	got, err := repo.FindUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if got.Nickname != "测试用户" {
		t.Fatalf("用户昵称不一致: %s", got.Nickname)
	}

	got.Nickname = "新昵称"
	if err := repo.UpdateUser(ctx, got); err != nil {
		t.Fatalf("更新用户失败: %v", err)
	}
	after, err := repo.FindUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("更新后查询失败: %v", err)
	}
	if after.Nickname != "新昵称" {
		t.Fatalf("更新未生效: %s", after.Nickname)
	}
}
