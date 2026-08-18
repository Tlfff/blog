package community

import "time"

const (
	CommentStatusDeleted int8 = 0
	CommentStatusNormal  int8 = 1
)

// Comment 是 Community 领域的评论聚合。
type Comment struct {
	ID            uint64
	ArticleID     uint64
	UserID        uint64
	ReplyToUserID uint64
	Content       string
	RootID        uint64
	IP            string
	LikeCount     uint32
	CommentCount  uint32
	CreatedTime   time.Time
	UpdatedTime   time.Time
	Status        int8
}

func (c *Comment) IsNormal() bool {
	return c.Status == CommentStatusNormal
}

func (c *Comment) IsRoot() bool {
	return c.RootID == 0
}

func (c *Comment) BelongsTo(userID uint64) bool {
	return c.UserID == userID
}

// CommentWithUser 是评论列表所需的作者与被回复者公开信息。
type CommentWithUser struct {
	Comment
	Nickname      string
	Avatar        string
	ReplyNickname string
	ReplyAvatar   string
}
