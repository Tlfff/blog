package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/database"
	"context"
	"errors"

	"gorm.io/gorm"
)

type commentRepositoryAdapter struct {
	commentRepo repository.CommentRepository
	articleRepo *repository.ArticleRepository
}

// NewCommentRepository 将 GORM 评论 Repository 适配为 Community 领域 Port。
func NewCommentRepository(commentRepo repository.CommentRepository, articleRepo *repository.ArticleRepository) domaincommunity.CommentRepository {
	return &commentRepositoryAdapter{commentRepo: commentRepo, articleRepo: articleRepo}
}

func (a *commentRepositoryAdapter) CreateWithCounts(ctx context.Context, comment *domaincommunity.Comment, incrementReply bool) error {
	commentModel := toModelComment(comment)
	err := database.RunTx(ctx, a.commentRepo.GetDB(), func(tx *gorm.DB) error {
		if incrementReply {
			rootComment, err := a.commentRepo.FindByIDForUpdate(ctx, tx, comment.RootID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domaincommunity.ErrCommentNotFound
				}
				return err
			}
			if rootComment.Status == domaincommunity.CommentStatusDeleted {
				return domaincommunity.ErrCommentRootDeleted
			}
		}
		if err := a.commentRepo.Insert(ctx, tx, commentModel); err != nil {
			return err
		}
		if incrementReply {
			if err := a.commentRepo.UpdateCommentCountDelta(ctx, tx, comment.RootID, 1); err != nil {
				return err
			}
		}
		return a.articleRepo.UpdateCommentCountDelta(ctx, tx, comment.ArticleID, 1)
	})
	if err != nil {
		return err
	}
	comment.ID = commentModel.ID
	comment.CreatedTime = commentModel.CreatedTime
	comment.UpdatedTime = commentModel.UpdatedTime
	return nil
}

func (a *commentRepositoryAdapter) FindByID(ctx context.Context, id uint64) (*domaincommunity.Comment, error) {
	comment, err := a.commentRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincommunity.ErrCommentNotFound
		}
		return nil, err
	}
	return toDomainComment(comment), nil
}

func (a *commentRepositoryAdapter) ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) ([]*domaincommunity.CommentWithUser, error) {
	var rows []*repository.CommentWithUser
	var err error
	if lastID > 0 {
		rows, err = a.commentRepo.FindRootCommentsWithCursor(ctx, articleID, lastID, pageSize, isDesc, authorID)
	} else {
		rows, err = a.commentRepo.FindRootCommentsWithOffset(ctx, articleID, (page-1)*pageSize, pageSize, isDesc, authorID)
	}
	if err != nil {
		return nil, err
	}
	return toDomainCommentWithUsers(rows), nil
}

func (a *commentRepositoryAdapter) CountRootComments(ctx context.Context, articleID, authorID uint64) (int64, error) {
	return a.commentRepo.CountRootComments(ctx, articleID, authorID)
}

func (a *commentRepositoryAdapter) ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) ([]*domaincommunity.CommentWithUser, error) {
	var rows []*repository.CommentWithUser
	var err error
	if lastID > 0 {
		rows, err = a.commentRepo.FindRepliesWithCursor(ctx, rootID, lastID, pageSize+1)
	} else {
		rows, err = a.commentRepo.FindRepliesWithOffset(ctx, rootID, (page-1)*pageSize, pageSize+1)
	}
	if err != nil {
		return nil, err
	}
	return toDomainCommentWithUsers(rows), nil
}

func (a *commentRepositoryAdapter) CountReplies(ctx context.Context, rootID uint64) (int64, error) {
	return a.commentRepo.CountReplies(ctx, rootID)
}

func (a *commentRepositoryAdapter) DeleteWithCounts(ctx context.Context, comment *domaincommunity.Comment) error {
	commentModel := toModelComment(comment)
	if commentModel.RootID == 0 {
		return database.RunTx(ctx, a.commentRepo.GetDB(), func(tx *gorm.DB) error {
			replyCount, err := a.commentRepo.BatchUpdateChildCommentStatus(ctx, tx, commentModel.ID)
			if err != nil {
				return err
			}
			affected, err := a.commentRepo.UpdateStatus(ctx, tx, commentModel.ID, uint8(domaincommunity.CommentStatusDeleted))
			if err != nil {
				return err
			}
			if affected == 0 {
				return nil
			}
			return a.articleRepo.UpdateCommentCountDelta(ctx, tx, commentModel.ArticleID, -(1 + replyCount))
		})
	}
	return database.RunTx(ctx, a.commentRepo.GetDB(), func(tx *gorm.DB) error {
		affected, err := a.commentRepo.UpdateStatus(ctx, tx, commentModel.ID, uint8(domaincommunity.CommentStatusDeleted))
		if err != nil {
			return err
		}
		if affected == 0 {
			return nil
		}
		if err := a.commentRepo.UpdateCommentCountDelta(ctx, tx, commentModel.RootID, -1); err != nil {
			return err
		}
		return a.articleRepo.UpdateCommentCountDelta(ctx, tx, commentModel.ArticleID, -1)
	})
}

func toModelComment(c *domaincommunity.Comment) *model.Comment {
	return &model.Comment{
		ID:            c.ID,
		ArticleID:     c.ArticleID,
		UserID:        c.UserID,
		ReplyToUserID: c.ReplyToUserID,
		Content:       c.Content,
		RootID:        c.RootID,
		IP:            c.IP,
		LikeCount:     c.LikeCount,
		CommentCount:  c.CommentCount,
		CreatedTime:   c.CreatedTime,
		UpdatedTime:   c.UpdatedTime,
		Status:        c.Status,
	}
}

func toDomainComment(m *model.Comment) *domaincommunity.Comment {
	return &domaincommunity.Comment{
		ID:            m.ID,
		ArticleID:     m.ArticleID,
		UserID:        m.UserID,
		ReplyToUserID: m.ReplyToUserID,
		Content:       m.Content,
		RootID:        m.RootID,
		IP:            m.IP,
		LikeCount:     m.LikeCount,
		CommentCount:  m.CommentCount,
		CreatedTime:   m.CreatedTime,
		UpdatedTime:   m.UpdatedTime,
		Status:        m.Status,
	}
}

func toDomainCommentWithUsers(rows []*repository.CommentWithUser) []*domaincommunity.CommentWithUser {
	result := make([]*domaincommunity.CommentWithUser, 0, len(rows))
	for _, row := range rows {
		comment := toDomainComment(&row.Comment)
		result = append(result, &domaincommunity.CommentWithUser{
			Comment:       *comment,
			Nickname:      row.Nickname,
			Avatar:        row.Avatar,
			ReplyNickname: row.ReplyNickname,
			ReplyAvatar:   row.ReplyAvatar,
		})
	}
	return result
}
