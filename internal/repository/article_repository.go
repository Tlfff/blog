package repository

import (
	"blog/internal/common"
	"blog/internal/model"
	"errors"
)

type ArticleRepository struct {
	articles map[int64]*model.Article
}

func NewArticleRepository() *ArticleRepository {
	return &ArticleRepository{
		articles: make(map[int64]*model.Article),
	}
}

// 创建文章
func (a *ArticleRepository) CreateArticle(acticle *model.Article) error {
	a.articles[acticle.ID] = acticle
	return nil
}

// 更新文章（包括状态）
func (a *ArticleRepository) UpdateArticle(acticle *model.Article) error {
	if a.articles[acticle.ID].AuthorID == acticle.AuthorID {
		a.articles[acticle.ID] = acticle
	}
	return nil
}

// 删除文章
func (a *ArticleRepository) DeleteArticle(acticleId int64, userId int64) error {
	if a.articles[acticleId].AuthorID == userId {
		a.articles[acticleId].Status = 0
	}
	return nil
}

// 硬删除文章
func (a *ArticleRepository) ClearArticle(acticleId int64, userId int64) error {
	if a.articles[acticleId].AuthorID == userId {
		delete(a.articles, acticleId)
	}
	return nil
}

// 根据id查找文章
func (a *ArticleRepository) FindArticleByID(id int64) (*model.Article, error) {
	for _, article := range a.articles {
		if article.ID == id {
			return article, nil
		}
	}
	return nil, errors.New(common.ErrArticleNotFound.Error())
}

// 列表查询
func (a *ArticleRepository) GetListByStatus(AuthorID int64, status int8) ([]*model.Article, error) {
	var list []*model.Article
	// 传入的userId为0时，认为要获取全部
	if AuthorID == 0 {
		for _, article := range a.articles {
			if article.Status == status {
				list = append(list, article)
			}
		}
	} else {
		for _, article := range a.articles {
			if article.Status == status && article.AuthorID == AuthorID {
				list = append(list, article)
			}
		}
	}

	return list, nil
}
