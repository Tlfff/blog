package infrastructure

import (
	articledomain "blog/internal/article/domain"
	"context"
)

// UserFacade 是 Article 查询作者信息所需的 User Application Facade。
type UserFacade interface {
	// GetUserSnapshot 返回最小用户公开快照。
	GetUserSnapshot(ctx context.Context, userID uint64) (id uint64, nickname, avatar, lastLoginIP string, err error)
}

type userQuery struct {
	facade UserFacade // User Application Facade
}

// NewUserQuery 创建 Article 到 User 的本地查询适配器。
func NewUserQuery(facade UserFacade) articledomain.UserQuery {
	return &userQuery{facade: facade}
}

// FindUserByID 查询文章详情所需的作者信息。
func (q *userQuery) FindUserByID(ctx context.Context, id uint64) (*articledomain.UserInfo, error) {
	userID, nickname, avatar, lastLoginIP, err := q.facade.GetUserSnapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	return &articledomain.UserInfo{ID: userID, Nickname: nickname, Avatar: avatar, LastLoginIP: lastLoginIP}, nil
}
