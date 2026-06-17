package article

import (
	"blog/internal/common"
	"blog/internal/model"
)

// 创建文章
type CreateArticleRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status"`
}

func (r *CreateArticleRequest) Validate() error {
	if r.Title == "" || r.Content == "" {
		return common.ErrArticleContentEmpty
	}
	return nil
}

// 修改文章
type UpdateArticleRequest struct {
	ID      int64    `json:"id"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status"`
}

func (r *UpdateArticleRequest) Validate() error {
	if r.ID <= 0 {
		return common.ErrArticleIDInvalid
	}
	if r.Title == "" || r.Content == "" {
		return common.ErrArticleContentEmpty
	}

	if err := model.FindStatusById(int(r.Status)); err != nil {
		return common.ErrArticleStatusError
	}
	return nil
}

// 删除文章
type DeleteArticleRequest struct {
	ID int64 `json:"id"`
}

func (r *DeleteArticleRequest) Validate() error {
	if r.ID <= 0 {
		return common.ErrArticleIDInvalid
	}
	return nil
}

// 发布文章
type PublishArticleRequest struct {
	ID int64 `json:"id"`
}

func (r *PublishArticleRequest) Validate() error {
	if r.ID <= 0 {
		return common.ErrArticleIDInvalid
	}
	return nil
}

// 获取文章详情
type GetDetailRequest struct {
	ID int64 `json:"id"`
}

func (r *GetDetailRequest) Validate() error {
	if r.ID <= 0 {
		return common.ErrArticleIDInvalid
	}
	return nil
}

// 获取用户文章列表
type GetPublishListRequest struct {
	AuthorID int64 `json:"author_id"`
}

func (r *GetPublishListRequest) Validate() error {
	if r.AuthorID <= 0 {
		return common.ErrUserExists // 建议未来将该错误定义改为更精准的 ErrUserIDInvalid
	}
	return nil
}
