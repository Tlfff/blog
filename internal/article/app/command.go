package app

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

// ImageUploadFileCommand 表示单张文章图片上传凭证输入。
type ImageUploadFileCommand struct {
	ClientID string // 前端生成的图片标识，用于关联上传结果
	FileExt  string // 图片文件扩展名，如 png、jpg
}

// GetImageUploadURLsCommand 表示批量获取文章图片上传凭证的输入。
type GetImageUploadURLsCommand struct {
	ArticleID uint64                   // 图片所属文章唯一标识
	AuthorID  uint64                   // 当前作者用户唯一标识
	Files     []ImageUploadFileCommand // 待上传图片列表
}
