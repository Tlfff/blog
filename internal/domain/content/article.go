package content

import "time"

const (
	StatusAll             int8 = -2
	StatusAllExceptDeleted int8 = -1
	StatusDeleted          int8 = 1
	StatusDraft            int8 = 2
	StatusPublished        int8 = 3
)

// Article 是 Content 领域的聚合根。
type Article struct {
	ID           uint64
	AuthorID     uint64
	Title        string
	Content      string
	Tags         string
	Status       int8
	ViewCount    uint32
	LikeCount    uint32
	CommentCount uint32
	CreatedTime  time.Time
	UpdatedTime  time.Time
}

func (a *Article) IsDeleted() bool {
	return a.Status == StatusDeleted
}

func (a *Article) IsPublished() bool {
	return a.Status == StatusPublished
}

func (a *Article) IsDraft() bool {
	return a.Status == StatusDraft
}

func (a *Article) IsPubliclyVisible() bool {
	return a.IsPublished()
}

func (a *Article) CanEdit(userID uint64) bool {
	return a.AuthorID == userID
}

func (a *Article) CanDelete(userID uint64) bool {
	return a.AuthorID == userID
}

func (a *Article) CanPublish(userID uint64) bool {
	return a.AuthorID == userID
}

func (a *Article) SoftDelete() {
	a.Status = StatusDeleted
}

func (a *Article) Publish() {
	a.Status = StatusPublished
}

func (a *Article) Recover() {
	a.Status = StatusDraft
}

// ArticleWithAuthor 是文章详情所需的作者公开信息。
type ArticleWithAuthor struct {
	Article
	Nickname    string
	Avatar      string
	LastLoginIP string
}
