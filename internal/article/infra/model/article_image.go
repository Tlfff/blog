package model

import "time"

// ArticleImage 是文章图片数据模型（GORM 映射 article_images 表）。
type ArticleImage struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`          // 图片唯一标识
	ArticleID   *uint64   `gorm:"column:article_id;index:idx_article_id"`      // 所属文章唯一标识，为空表示尚未绑定文章
	ObjectKey   string    `gorm:"column:object_key;uniqueIndex:uk_object_key"` // 对象存储 Key
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime"`          // 图片记录创建时间
}
