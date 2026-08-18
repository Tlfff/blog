package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
)

type articleQueryAdapter struct {
	repo *repository.ArticleRepository
}

// NewArticleQuery 将 GORM 文章 Repository 适配为 Community 只读查询 Port。
func NewArticleQuery(repo *repository.ArticleRepository) domaincommunity.ArticleQuery {
	return &articleQueryAdapter{repo: repo}
}

func (a *articleQueryAdapter) FindByID(ctx context.Context, id uint64) (*domaincommunity.ArticleInfo, error) {
	article, err := a.repo.FindArticleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toArticleInfo(article), nil
}

func (a *articleQueryAdapter) GetHotListByIDs(ctx context.Context, ids []uint64) ([]*domaincommunity.ArticleInfo, error) {
	articles, err := a.repo.GetHotListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return toArticleInfos(articles), nil
}

func (a *articleQueryAdapter) GetTopHotArticles(ctx context.Context, limit int) ([]*domaincommunity.ArticleInfo, error) {
	articles, err := a.repo.GetTopHotArticles(ctx, limit)
	if err != nil {
		return nil, err
	}
	return toArticleInfos(articles), nil
}

type userInfoQueryAdapter struct {
	repo *repository.UserRepository
}

// NewUserInfoQuery 将 GORM 用户 Repository 适配为 Community 只读查询 Port。
func NewUserInfoQuery(repo *repository.UserRepository) domaincommunity.UserInfoQuery {
	return &userInfoQueryAdapter{repo: repo}
}

func (a *userInfoQueryAdapter) FindUserByID(ctx context.Context, id uint64) (*domaincommunity.UserInfo, error) {
	user, err := a.repo.FindUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domaincommunity.UserInfo{
		ID:       user.ID,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}, nil
}

func toArticleInfo(m *model.Article) *domaincommunity.ArticleInfo {
	return &domaincommunity.ArticleInfo{
		ID:           m.ID,
		AuthorID:     m.AuthorID,
		Title:        m.Title,
		ViewCount:    m.ViewCount,
		LikeCount:    m.LikeCount,
		CommentCount: m.CommentCount,
	}
}

func toArticleInfos(articles []*model.Article) []*domaincommunity.ArticleInfo {
	list := make([]*domaincommunity.ArticleInfo, 0, len(articles))
	for _, article := range articles {
		list = append(list, toArticleInfo(article))
	}
	return list
}
