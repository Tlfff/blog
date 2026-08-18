package identity

import (
	domainidentity "blog/internal/domain/identity"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
	"errors"

	"gorm.io/gorm"
)

type userRepositoryAdapter struct {
	repo *repository.UserRepository
}

// NewUserRepository 将现有 GORM UserRepository 适配为 Identity 领域 Port。
func NewUserRepository(repo *repository.UserRepository) domainidentity.UserRepository {
	return &userRepositoryAdapter{repo: repo}
}

func (a *userRepositoryAdapter) CreateUser(ctx context.Context, user *domainidentity.User) error {
	m := toModelUser(user)
	if err := a.repo.CreateUser(ctx, m); err != nil {
		return err
	}
	user.ID = m.ID
	user.CreatedTime = m.CreatedTime
	user.UpdatedTime = m.UpdatedTime
	return nil
}

func (a *userRepositoryAdapter) GetUserByAccount(ctx context.Context, phone, nickname string) (*domainidentity.User, error) {
	m, err := a.repo.GetUserByAccount(ctx, phone, nickname)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainidentity.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(m), nil
}

func (a *userRepositoryAdapter) FindUserByID(ctx context.Context, id uint64) (*domainidentity.User, error) {
	m, err := a.repo.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainidentity.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(m), nil
}

func (a *userRepositoryAdapter) FindUsersByIDs(ctx context.Context, ids []uint64) ([]*domainidentity.User, error) {
	models, err := a.repo.FindUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	users := make([]*domainidentity.User, 0, len(models))
	for _, m := range models {
		users = append(users, toDomainUser(m))
	}
	return users, nil
}

func (a *userRepositoryAdapter) UpdateUser(ctx context.Context, user *domainidentity.User) error {
	return a.repo.UpdateUser(ctx, toModelUser(user))
}

func (a *userRepositoryAdapter) GetUserList(ctx context.Context, page, pageSize int, isDesc bool) ([]*domainidentity.User, error) {
	models, err := a.repo.GetUserList(ctx, page, pageSize, isDesc)
	if err != nil {
		return nil, err
	}
	users := make([]*domainidentity.User, 0, len(models))
	for _, m := range models {
		users = append(users, toDomainUser(m))
	}
	return users, nil
}

func (a *userRepositoryAdapter) CountUsers(ctx context.Context) (int64, error) {
	return a.repo.CountUsers(ctx)
}

func toDomainUser(m *model.User) *domainidentity.User {
	return &domainidentity.User{
		ID:            m.ID,
		Nickname:      m.Nickname,
		Phone:         m.Phone,
		Password:      m.Password,
		Avatar:        m.Avatar,
		Role:          m.Role,
		Status:        m.Status,
		LastLoginIP:   m.LastLoginIp,
		LastLoginTime: m.LastLoginTime,
		CreatedTime:   m.CreatedTime,
		UpdatedTime:   m.UpdatedTime,
	}
}

func toModelUser(u *domainidentity.User) *model.User {
	return &model.User{
		ID:            u.ID,
		Nickname:      u.Nickname,
		Phone:         u.Phone,
		Password:      u.Password,
		Avatar:        u.Avatar,
		Role:          u.Role,
		Status:        u.Status,
		LastLoginIp:   u.LastLoginIP,
		LastLoginTime: u.LastLoginTime,
		CreatedTime:   u.CreatedTime,
		UpdatedTime:   u.UpdatedTime,
	}
}
