package domain

import "time"

// 评论状态取值
const (
	CommentStatusDeleted int8 = 0 // 评论状态：已删除
	CommentStatusNormal  int8 = 1 // 评论状态：正常
)

// Comment 是 Comment 领域的评论聚合。
type Comment struct {
	ID            uint64    // 评论唯一标识
	ArticleID     uint64    // 所属文章ID
	UserID        uint64    // 评论发布者用户ID
	ReplyToUserID uint64    // 被回复者用户ID，主楼评论为 0
	Content       string    // 评论正文内容
	RootID        uint64    // 主楼评论ID，为 0 表示自身即主楼
	IP            string    // 发布评论时的来源IP，用于展示归属地
	LikeCount     uint32    // 点赞数
	CommentCount  uint32    // 回复数，仅主楼评论累计
	CreatedTime   time.Time // 创建时间
	UpdatedTime   time.Time // 最后更新时间
	Status        int8      // 评论状态：0-已删除 1-正常
}

// 判断评论是否处于正常状态（未被删除）
func (c *Comment) IsNormal() bool {
	return c.Status == CommentStatusNormal
}

// 判断评论是否为主楼评论（RootID 为 0 即主楼）
func (c *Comment) IsRoot() bool {
	return c.RootID == 0
}

// 判断评论是否属于指定用户，用于删除等权限校验
func (c *Comment) BelongsTo(userID uint64) bool {
	return c.UserID == userID
}

// CommentWithUser 是评论列表查询模型，在评论聚合基础上附带作者与被回复者公开信息。
type CommentWithUser struct {
	Comment              // 内嵌评论聚合字段
	Nickname      string // 评论发布者昵称
	Avatar        string // 评论发布者头像URL
	ReplyNickname string // 被回复者昵称，主楼评论为空
	ReplyAvatar   string // 被回复者头像URL，主楼评论为空
}
