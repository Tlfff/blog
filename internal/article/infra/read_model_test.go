package infra

import (
	articlemodel "blog/internal/article/infra/model"
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// readModelUser 是文章详情 JOIN 测试使用的 users 表映射。
type readModelUser struct {
	ID          uint64 `gorm:"column:id;primaryKey"` // 用户唯一标识
	Nickname    string `gorm:"column:nickname"`      // 用户昵称
	Avatar      string `gorm:"column:avatar"`        // 用户头像
	LastLoginIP string `gorm:"column:last_login_ip"` // 最后登录 IP
}

// TableName 指定测试用户模型映射 users 表。
func (readModelUser) TableName() string { return "users" }

// TestFindWithAuthorByIDUsesCallerOwnedReadModel 验证文章详情只读 JOIN 返回作者展示字段。
func TestFindWithAuthorByIDUsesCallerOwnedReadModel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&articlemodel.Article{}, &readModelUser{}); err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	if err := db.Create(&readModelUser{ID: 2, Nickname: "作者", Avatar: "avatar", LastLoginIP: "127.0.0.1"}).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	if err := db.Create(&articlemodel.Article{ID: 1, AuthorID: 2, Title: "文章", Status: 3}).Error; err != nil {
		t.Fatalf("创建测试文章失败: %v", err)
	}

	detail, err := NewArticleRepository(db).FindWithAuthorByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("查询文章详情失败: %v", err)
	}
	if detail.Nickname != "作者" || detail.Avatar != "avatar" || detail.LastLoginIP != "127.0.0.1" {
		t.Fatalf("作者展示字段不正确: %+v", detail)
	}
}
