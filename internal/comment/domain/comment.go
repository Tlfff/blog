package domain

import "time"

// 评论状态取值。
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

// NewRootComment 创建正常状态的主评论。
func NewRootComment(articleID, userID uint64, content, ip string) *Comment {
	return &Comment{
		ArticleID: articleID,
		UserID:    userID,
		Content:   content,
		IP:        ip,
		Status:    CommentStatusNormal,
	}
}

// NewReply 创建正常状态的回复评论。
//
// 参数说明：
//   - articleID：所属文章唯一标识。
//   - rootID：根评论唯一标识。
//   - userID：回复用户唯一标识。
//   - replyToUserID：被回复用户唯一标识，可以为 0。
//   - content：回复正文。
//   - ip：回复来源 IP。
func NewReply(articleID, rootID, userID, replyToUserID uint64, content, ip string) *Comment {
	return &Comment{
		ArticleID:     articleID,
		RootID:        rootID,
		UserID:        userID,
		ReplyToUserID: replyToUserID,
		Content:       content,
		IP:            ip,
		Status:        CommentStatusNormal,
	}
}

// IsNormal 判断评论是否处于正常状态。
func (c *Comment) IsNormal() bool {
	return c.Status == CommentStatusNormal
}

// IsRoot 判断评论是否为主楼评论。
func (c *Comment) IsRoot() bool {
	return c.RootID == 0
}

// BelongsTo 判断评论是否属于指定用户。
func (c *Comment) BelongsTo(userID uint64) bool {
	return c.UserID == userID
}

// EnsureReplyable 校验评论是否允许被回复。
func (c *Comment) EnsureReplyable() error {
	if !c.IsNormal() {
		return ErrCommentRootDeleted
	}
	return nil
}

// DeleteBy 校验删除权限并将评论标记为已删除。
func (c *Comment) DeleteBy(operatorID uint64, isAdmin bool) error {
	// 1. 保持现有顺序，先检查删除状态
	if !c.IsNormal() {
		return ErrCommentDeleted
	}

	// 2. 管理员可以删除任意评论，普通用户只能删除自己的评论
	if !isAdmin && !c.BelongsTo(operatorID) {
		return ErrCommentPermission
	}
	c.Status = CommentStatusDeleted
	return nil
}

// HotValue 返回现有评论热度值：点赞数加回复数。
func (c *Comment) HotValue() uint64 {
	return uint64(c.LikeCount) + uint64(c.CommentCount)
}

// CommentWithUser 是评论列表查询模型，在评论聚合基础上附带作者与被回复者公开信息。
type CommentWithUser struct {
	Comment              // 内嵌评论聚合字段
	Nickname      string // 评论发布者昵称
	Avatar        string // 评论发布者头像URL
	ReplyNickname string // 被回复者昵称，主楼评论为空
	ReplyAvatar   string // 被回复者头像URL，主楼评论为空
}
