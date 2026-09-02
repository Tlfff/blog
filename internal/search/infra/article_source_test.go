package infra

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// articleSourceTestRow 表示 ArticleSource 测试表结构。
type articleSourceTestRow struct {
	ID          uint64    `gorm:"column:id;primaryKey"` // 文章唯一标识
	Title       string    `gorm:"column:title"`         // 文章标题
	Content     string    `gorm:"column:content"`       // Markdown 正文
	Tags        string    `gorm:"column:tags"`          // 标签字符串
	Status      int8      `gorm:"column:status"`        // 文章状态：1-已删除；2-草稿；3-已发表
	UpdatedTime time.Time `gorm:"column:updated_time"`  // 最后更新时间
}

// TableName 返回 ArticleSource 测试表名。
func (articleSourceTestRow) TableName() string {
	// 1. 复用生产查询使用的 articles 表名
	return "articles"
}

// TestArticleSourceListPublishedAfter 验证只读来源按状态和 ID 游标分页。
func TestArticleSourceListPublishedAfter(t *testing.T) {
	// 1. 创建内存数据库并插入草稿和已发表文章
	db, err := gorm.Open(sqlite.Open("file:search_source?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&articleSourceTestRow{}); err != nil {
		t.Fatalf("创建文章测试表失败: %v", err)
	}
	rows := []articleSourceTestRow{
		{ID: 1, Title: "已发表一", Status: int8(searchdomain.ArticleStatusPublished)},
		{ID: 2, Title: "草稿", Status: int8(searchdomain.ArticleStatusDraft)},
		{ID: 3, Title: "已发表二", Status: int8(searchdomain.ArticleStatusPublished)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("插入文章测试数据失败: %v", err)
	}
	source := NewArticleSource(db)

	// 2. 从游标 1 开始只返回后续已发表文章
	articles, err := source.ListPublishedAfter(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("读取已发表文章失败: %v", err)
	}
	if len(articles) != 1 || articles[0].ID != 3 {
		t.Fatalf("文章游标分页结果不符合预期: %+v", articles)
	}
	count, err := source.CountPublished(context.Background())
	if err != nil || count != 2 {
		t.Fatalf("已发表文章数量不符合预期: count=%d err=%v", count, err)
	}
}

// TestArticleSourceRejectsInvalidLimit 验证非法批次大小不会访问数据库。
func TestArticleSourceRejectsInvalidLimit(t *testing.T) {
	// 1. 使用空数据库连接验证参数边界优先返回
	source := NewArticleSource(&gorm.DB{})
	if _, err := source.ListPublishedAfter(context.Background(), 0, 0); err == nil {
		t.Fatal("非法重建批次大小未返回错误")
	}
}
