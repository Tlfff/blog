package article

import (
	"blog/internal/common"
)

// 创建文章
type CreateArticleRequest struct {
	Title   string   `json:"title" binding:"required"` //不能为空
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status"`
}

// 修改文章
type UpdateArticleRequest struct {
	ID      int64    `json:"id" binding:"required,min=1"` // id是否大于0
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status" binding:"oneof=0 1 2"` // 状态只能是0,1,2
}

// func (r *UpdateArticleRequest) Validate() error {
// 	if r.ID <= 0 {
// 		return common.ErrArticleIDInvalid
// 	}
// 	if r.Title == "" || r.Content == "" {
// 		return common.ErrArticleContentEmpty
// 	}

// 	if err := model.FindStatusById(int(r.Status)); err != nil {
// 		return common.ErrArticleStatusError
// 	}
// 	return nil
// }

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
	ID int64 `form:"id" binding:"required,min=1"`
}

// func (r *PublishArticleRequest) Validate() error {
// 	if r.ID <= 0 {
// 		return common.ErrArticleIDInvalid
// 	}
// 	return nil
// }

// 获取文章详情
type GetDetailRequest struct {
	ID int64 `form:"id" binding:"required,min=1"` // form:"id" 告诉 Gin 去 URL 参数中找 ?id=xxx
}

// func (r *GetDetailRequest) Validate() error {
// 	if r.ID <= 0 {
// 		return common.ErrArticleIDInvalid
// 	}
// 	return nil
// }

// 获取用户文章列表
type GetPublishListRequest struct {
	AuthorID int64 `form:"author_id" binding:"required,min=1"`
}

// func (r *GetPublishListRequest) Validate() error {
// 	if r.AuthorID <= 0 {
// 		return common.ErrUserExists // 建议未来将该错误定义改为更精准的 ErrUserIDInvalid
// 	}
// 	return nil
// }
