package domain

import "time"

// ArticleStatus 表示文章生命周期状态值对象。
type ArticleStatus int8

// 文章状态取值与查询过滤值。
const (
	StatusAll              int8          = -2 // 查询过滤：全部状态（含已删除）
	StatusAllExceptDeleted int8          = -1 // 查询过滤：除已删除外的全部状态
	StatusUnspecified      ArticleStatus = 0  // 兼容现有未传状态的输入
	StatusDeleted          ArticleStatus = 1  // 文章状态：已删除（软删除）
	StatusDraft            ArticleStatus = 2  // 文章状态：草稿
	StatusPublished        ArticleStatus = 3  // 文章状态：已发表
)

// NewArticleStatus 根据现有接口取值创建文章状态。
func NewArticleStatus(value int8) (ArticleStatus, error) {
	// 1. 将协议状态值转换为领域状态
	status := ArticleStatus(value)
	// 2. 仅接受现有兼容状态值
	switch status {
	case StatusUnspecified, StatusDeleted, StatusDraft, StatusPublished:
		return status, nil
	default:
		return StatusUnspecified, ErrArticleStatusInvalid
	}
}

// Int8 返回持久化和协议兼容所需的状态值。
func (s ArticleStatus) Int8() int8 {
	// 1. 转换为持久化和协议使用的基础类型
	return int8(s)
}

// IsDeleted 判断状态是否为已删除。
func (s ArticleStatus) IsDeleted() bool {
	// 1. 判断是否为已删除状态
	return s == StatusDeleted
}

// IsPublished 判断状态是否为已发表。
func (s ArticleStatus) IsPublished() bool {
	// 1. 判断是否为已发表状态
	return s == StatusPublished
}

// IsDraft 判断状态是否为草稿。
func (s ArticleStatus) IsDraft() bool {
	// 1. 判断是否为草稿状态
	return s == StatusDraft
}

// Article 是 Article 领域的聚合根。
type Article struct {
	ID           uint64        // 文章唯一标识
	AuthorID     uint64        // 作者用户ID
	Title        string        // 文章标题
	Content      string        // 文章正文内容（支持Markdown）
	Tags         string        // 文章标签，多个标签以英文逗号分隔
	Status       ArticleStatus // 文章状态：0-兼容未指定；1-已删除；2-草稿；3-已发表
	ViewCount    uint32        // 浏览量
	LikeCount    uint32        // 点赞数
	CommentCount uint32        // 评论数
	CreatedTime  time.Time     // 创建时间
	UpdatedTime  time.Time     // 最后更新时间
}

// NewArticle 创建符合当前业务规则的文章聚合。
//
// 参数说明：
//   - authorID：唯一管理员兼作者的用户标识。
//   - title：文章标题，不能为空。
//   - content：文章正文，不能为空。
//   - tags：英文逗号分隔的标签字符串。
//   - statusValue：兼容现有接口的文章状态值。
func NewArticle(authorID uint64, title, content, tags string, statusValue int8) (*Article, error) {
	// 1. 校验作者、标题和正文
	if authorID == 0 {
		return nil, ErrArticlePermissionDenied
	}
	if title == "" {
		return nil, ErrArticleTitleEmpty
	}
	if content == "" {
		return nil, ErrArticleContentEmpty
	}
	// 2. 校验文章初始状态
	status, err := NewArticleStatus(statusValue)
	if err != nil {
		return nil, err
	}
	// 3. 创建文章聚合
	return &Article{
		AuthorID: authorID,
		Title:    title,
		Content:  content,
		Tags:     tags,
		Status:   status,
	}, nil
}

// NewDraftArticle 初始化等待填写内容的文章草稿。
func NewDraftArticle(authorID uint64) (*Article, error) {
	// 1. 校验草稿作者身份
	if authorID == 0 {
		return nil, ErrArticlePermissionDenied
	}

	// 2. 创建不要求标题和正文的初始化草稿
	return &Article{
		AuthorID: authorID,
		Status:   StatusDraft,
	}, nil
}

// IsDeleted 判断文章是否已被软删除。
func (a *Article) IsDeleted() bool {
	// 1. 委托状态值对象判断删除状态
	return a.Status.IsDeleted()
}

// IsPublished 判断文章是否已发表。
func (a *Article) IsPublished() bool {
	// 1. 委托状态值对象判断发表状态
	return a.Status.IsPublished()
}

// IsDraft 判断文章是否为草稿。
func (a *Article) IsDraft() bool {
	// 1. 委托状态值对象判断草稿状态
	return a.Status.IsDraft()
}

// IsPubliclyVisible 判断文章是否对外公开可见。
func (a *Article) IsPubliclyVisible() bool {
	// 1. 仅已发表文章允许公开访问
	return a.IsPublished()
}

// EnsureCanUploadImageBy 校验操作者能否为文章上传图片。
func (a *Article) EnsureCanUploadImageBy(operatorID uint64) error {
	// 1. 已删除文章不允许继续写入图片
	if a.IsDeleted() {
		return ErrArticleDeleted
	}

	// 2. 仅文章作者可以上传图片
	if a.AuthorID != operatorID {
		return ErrArticlePermissionDenied
	}
	return nil
}

// EditBy 校验作者和文章状态后修改可编辑字段。
//
// 参数说明：
//   - operatorID：当前操作者用户标识。
//   - title：新文章标题。
//   - content：新文章正文。
//   - tags：英文逗号分隔的新标签。
//   - statusValue：兼容现有接口的新文章状态值。
func (a *Article) EditBy(operatorID uint64, title, content, tags string, statusValue int8) error {
	// 1. 保持现有顺序，先判断删除状态
	if a.IsDeleted() {
		return ErrArticleDeleted
	}

	// 2. 校验唯一作者身份
	if a.AuthorID != operatorID {
		return ErrArticlePermissionDenied
	}
	if title == "" {
		return ErrArticleTitleEmpty
	}
	if content == "" {
		return ErrArticleContentEmpty
	}
	status, err := NewArticleStatus(statusValue)
	if err != nil {
		return err
	}

	// 3. 更新聚合字段
	a.Title = title
	a.Content = content
	a.Tags = tags
	a.Status = status
	return nil
}

// MoveToTrashBy 将作者文章移入垃圾箱。
func (a *Article) MoveToTrashBy(operatorID uint64) error {
	// 1. 校验操作者是否为文章作者
	if a.AuthorID != operatorID {
		return ErrArticlePermissionDenied
	}
	// 2. 将文章状态改为已删除
	a.Status = StatusDeleted
	return nil
}

// PublishBy 将作者文章状态改为已发表。
func (a *Article) PublishBy(operatorID uint64) error {
	// 1. 保持现有顺序，先校验作者身份
	if a.AuthorID != operatorID {
		return ErrArticlePermissionDenied
	}

	// 2. 已删除文章不能发布
	if a.IsDeleted() {
		return ErrArticleDeleted
	}
	a.Status = StatusPublished
	return nil
}

// RecoverBy 将作者的垃圾箱文章恢复为草稿。
func (a *Article) RecoverBy(operatorID uint64) error {
	// 1. 校验操作者是否为文章作者
	if a.AuthorID != operatorID {
		return ErrArticlePermissionDenied
	}
	// 2. 仅垃圾箱文章允许恢复
	if !a.IsDeleted() {
		return ErrArticleStatusError
	}
	// 3. 将文章状态恢复为草稿
	a.Status = StatusDraft
	return nil
}

// EnsureCanPermanentlyDeleteBy 校验文章能否被作者彻底删除。
func (a *Article) EnsureCanPermanentlyDeleteBy(operatorID uint64) error {
	// 1. 校验操作者是否为文章作者
	if a.AuthorID != operatorID {
		return ErrArticlePermissionDenied
	}
	// 2. 仅垃圾箱文章允许彻底删除
	if !a.IsDeleted() {
		return ErrArticleStatusError
	}
	return nil
}

// ArticleWithAuthor 是文章详情查询模型，在聚合根基础上附带作者公开信息。
type ArticleWithAuthor struct {
	Article            // 内嵌文章聚合根字段
	Nickname    string // 作者昵称
	Avatar      string // 作者头像URL
	LastLoginIP string // 作者最后登录IP，用于展示归属地
}
