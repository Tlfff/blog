package application

// CreateArticleCommand 表示创建文章用例输入。
type CreateArticleCommand struct {
	AuthorID uint64   // 作者用户唯一标识
	Title    string   // 文章标题
	Content  string   // Markdown 正文
	Tags     []string // 文章标签
	Status   int8     // 文章状态：2-草稿；3-已发表
}

// UpdateArticleCommand 表示修改文章用例输入。
type UpdateArticleCommand struct {
	ArticleID uint64   // 文章唯一标识
	AuthorID  uint64   // 当前用户唯一标识
	Title     string   // 文章标题
	Content   string   // Markdown 正文
	Tags      []string // 文章标签
	Status    int8     // 文章状态：2-草稿；3-已发表
}
