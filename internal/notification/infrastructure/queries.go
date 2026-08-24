package infrastructure

import (
	notificationdomain "blog/internal/notification/domain"
	"context"
)

// ArticleFacade 是 Notification 查询文章快照所需的本地应用门面。
type ArticleFacade interface {
	// GetArticleSnapshot 查询通知所需的最小文章快照。
	GetArticleSnapshot(ctx context.Context, articleID uint64) (id, authorID uint64, title string, err error)
}

// UserFacade 是 Notification 查询用户快照所需的本地应用门面。
type UserFacade interface {
	// GetUserSnapshot 查询通知所需的最小用户快照。
	GetUserSnapshot(ctx context.Context, userID uint64) (id uint64, nickname, avatar, lastLoginIP string, err error)
}

type articleQuery struct {
	facade ArticleFacade // Article Application 本地门面
}

// NewArticleQuery 创建 Notification 到 Article 的本地查询适配器。
func NewArticleQuery(facade ArticleFacade) notificationdomain.ArticleQuery {
	return &articleQuery{facade: facade}
}

// FindByID 查询通知所需的文章快照。
func (q *articleQuery) FindByID(ctx context.Context, id uint64) (*notificationdomain.ArticleInfo, error) {
	articleID, authorID, title, err := q.facade.GetArticleSnapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	return &notificationdomain.ArticleInfo{ID: articleID, AuthorID: authorID, Title: title}, nil
}

type userInfoQuery struct {
	facade UserFacade // User Application 本地门面
}

// NewUserInfoQuery 创建 Notification 到 User 的本地查询适配器。
func NewUserInfoQuery(facade UserFacade) notificationdomain.UserInfoQuery {
	return &userInfoQuery{facade: facade}
}

// FindUserByID 查询通知所需的用户快照。
func (q *userInfoQuery) FindUserByID(ctx context.Context, id uint64) (*notificationdomain.UserInfo, error) {
	userID, nickname, avatar, _, err := q.facade.GetUserSnapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	return &notificationdomain.UserInfo{ID: userID, Nickname: nickname, Avatar: avatar}, nil
}
