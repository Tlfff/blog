package content

import "time"

// 文章状态取值与查询过滤值
const (
	StatusAll              int8 = -2 // 查询过滤：全部状态（含已删除）
	StatusAllExceptDeleted int8 = -1 // 查询过滤：除已删除外的全部状态
	StatusDeleted          int8 = 1  // 文章状态：已删除（软删除）
	StatusDraft            int8 = 2  // 文章状态：草稿
	StatusPublished        int8 = 3  // 文章状态：已发表
)

// Article 是 Content 领域的聚合根。
type Article struct {
	ID           uint64    // 文章唯一标识
	AuthorID     uint64    // 作者用户ID
	Title        string    // 文章标题
	Content      string    // 文章正文内容（支持Markdown）
	Tags         string    // 文章标签，多个标签以英文逗号分隔
	Status       int8      // 文章状态：1-已删除 2-草稿 3-已发表
	ViewCount    uint32    // 浏览量
	LikeCount    uint32    // 点赞数
	CommentCount uint32    // 评论数
	CreatedTime  time.Time // 创建时间
	UpdatedTime  time.Time // 最后更新时间
}

// 判断文章是否已被软删除
func (a *Article) IsDeleted() bool {
	return a.Status == StatusDeleted
}

// 判断文章是否已发表
func (a *Article) IsPublished() bool {
	return a.Status == StatusPublished
}

// 判断文章是否为草稿
func (a *Article) IsDraft() bool {
	return a.Status == StatusDraft
}

// 判断文章是否对外公开可见，只有已发表的文章允许游客访问
func (a *Article) IsPubliclyVisible() bool {
	return a.IsPublished()
}

// 判断指定用户是否有权编辑该文章，仅作者本人可编辑
func (a *Article) CanEdit(userID uint64) bool {
	return a.AuthorID == userID
}

// 判断指定用户是否有权删除该文章，仅作者本人可删除
func (a *Article) CanDelete(userID uint64) bool {
	return a.AuthorID == userID
}

// 判断指定用户是否有权发表该文章，仅作者本人可发表
func (a *Article) CanPublish(userID uint64) bool {
	return a.AuthorID == userID
}

// 将文章标记为已删除（软删除，数据仍保留在库中）
func (a *Article) SoftDelete() {
	a.Status = StatusDeleted
}

// 将文章标记为已发表
func (a *Article) Publish() {
	a.Status = StatusPublished
}

// 将已删除的文章恢复为草稿状态
func (a *Article) Recover() {
	a.Status = StatusDraft
}

// ArticleWithAuthor 是文章详情查询模型，在聚合根基础上附带作者公开信息。
type ArticleWithAuthor struct {
	Article            // 内嵌文章聚合根字段
	Nickname    string // 作者昵称
	Avatar      string // 作者头像URL
	LastLoginIP string // 作者最后登录IP，用于展示归属地
}
