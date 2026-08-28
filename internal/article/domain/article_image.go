package domain

import "time"

// ArticleImage 表示由文章正文引用的图片资源。
type ArticleImage struct {
	ID          uint64    // 图片唯一标识
	ArticleID   uint64    // 所属文章唯一标识，0 表示尚未绑定文章
	ObjectKey   string    // 对象存储 Key
	CreatedTime time.Time // 图片记录创建时间
}
