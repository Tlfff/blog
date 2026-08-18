package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"blog/pkg/database"
	"context"
	"errors"

	"gorm.io/gorm"
)

type commentRepository struct {
	db *gorm.DB
}

// NewCommentRepository 返回直接持有 GORM 的 Community 评论 Repository 实现。
func NewCommentRepository(db *gorm.DB) domaincommunity.CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) CreateWithCounts(ctx context.Context, comment *domaincommunity.Comment, incrementReply bool) error {
	commentModel := toModelComment(comment)
	err := database.RunTx(ctx, r.db, func(tx *gorm.DB) error {
		if incrementReply {
			rootComment, err := r.findByIDForUpdate(ctx, tx, comment.RootID)
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
		if err := tx.WithContext(ctx).Create(commentModel).Error; err != nil {
			return err
		}
		if incrementReply {
			if err := r.updateCommentCountDelta(ctx, tx, comment.RootID, 1); err != nil {
				return err
			}
		}
		return r.updateArticleCommentCountDelta(ctx, tx, comment.ArticleID, 1)
	})
	if err != nil {
		return err
	}
	comment.ID = commentModel.ID
	comment.CreatedTime = commentModel.CreatedTime
	comment.UpdatedTime = commentModel.UpdatedTime
	return nil
}

func (r *commentRepository) FindByID(ctx context.Context, id uint64) (*domaincommunity.Comment, error) {
	var m model.Comment
	err := r.db.WithContext(ctx).
		Select("id", "article_id", "user_id", "reply_to_user_id", "content", "root_id", "like_count", "comment_count", "ip", "created_time", "updated_time", "status").
		Where("id=?", id).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincommunity.ErrCommentNotFound
		}
		return nil, err
	}
	return toDomainComment(&m), nil
}

func (r *commentRepository) ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) ([]*domaincommunity.CommentWithUser, error) {
	tx := r.db.WithContext(ctx).Table("comments c").
		Select(`c.id, c.article_id, c.user_id, c.reply_to_user_id, c.content, c.root_id, c.like_count, c.comment_count, c.ip, c.created_time, c.updated_time, c.status,
			u1.nickname AS nickname, u1.avatar AS avatar, u1.last_login_ip AS last_login_ip,
			u2.nickname AS reply_nickname, u2.avatar AS reply_avatar`).
		Joins("LEFT JOIN users u1 ON c.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON c.reply_to_user_id = u2.id").
		Where("c.article_id = ? AND c.root_id = 0 AND c.status = ?", articleID, model.CommentLiked)
	if authorID > 0 {
		tx = tx.Where("c.user_id = ?", authorID)
	}
	if lastID > 0 {
		if isDesc {
			tx = tx.Where("c.id < ?", lastID).Order("c.id DESC")
		} else {
			tx = tx.Where("c.id > ?", lastID).Order("c.id ASC")
		}
		tx = tx.Limit(pageSize)
	} else {
		if isDesc {
			tx = tx.Order("c.id DESC")
		} else {
			tx = tx.Order("c.id ASC")
		}
		tx = tx.Limit(pageSize).Offset((page - 1) * pageSize)
	}
	var rows []*commentWithUserRow
	if err := tx.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return toDomainCommentWithUsers(rows), nil
}

func (r *commentRepository) CountRootComments(ctx context.Context, articleID, authorID uint64) (int64, error) {
	var count int64
	tx := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("article_id=? AND root_id=0 AND status=?", articleID, model.Published)
	if authorID > 0 {
		tx = tx.Where("user_id=?", authorID)
	}
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *commentRepository) ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) ([]*domaincommunity.CommentWithUser, error) {
	tx := r.db.WithContext(ctx).Table("comments c").
		Select(`c.id, c.article_id, c.user_id, c.reply_to_user_id, c.content, c.root_id, c.like_count, c.comment_count, c.ip, c.created_time, c.updated_time, c.status,
			u1.nickname AS nickname, u1.avatar AS avatar, u1.last_login_ip AS last_login_ip,
			u2.nickname AS reply_nickname, u2.avatar AS reply_avatar`).
		Joins("LEFT JOIN users u1 ON c.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON c.reply_to_user_id = u2.id").
		Where("c.root_id = ? AND c.status = ?", rootID, model.CommentLiked)
	if lastID > 0 {
		tx = tx.Where("c.id > ?", lastID).Order("c.id ASC").Limit(pageSize)
	} else {
		tx = tx.Order("c.id ASC").Limit(pageSize).Offset((page - 1) * pageSize)
	}
	var rows []*commentWithUserRow
	if err := tx.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return toDomainCommentWithUsers(rows), nil
}

func (r *commentRepository) CountReplies(ctx context.Context, rootID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("root_id=? AND status=?", rootID, model.CommentLiked).
		Count(&count).Error
	return count, err
}

func (r *commentRepository) DeleteWithCounts(ctx context.Context, comment *domaincommunity.Comment) error {
	commentModel := toModelComment(comment)
	if commentModel.RootID == 0 {
		return database.RunTx(ctx, r.db, func(tx *gorm.DB) error {
			replyCount, err := r.batchUpdateChildCommentStatus(ctx, tx, commentModel.ID)
			if err != nil {
				return err
			}
			affected, err := r.updateStatus(ctx, tx, commentModel.ID, uint8(domaincommunity.CommentStatusDeleted))
			if err != nil {
				return err
			}
			if affected == 0 {
				return nil
			}
			return r.updateArticleCommentCountDelta(ctx, tx, commentModel.ArticleID, -(1 + replyCount))
		})
	}
	return database.RunTx(ctx, r.db, func(tx *gorm.DB) error {
		affected, err := r.updateStatus(ctx, tx, commentModel.ID, uint8(domaincommunity.CommentStatusDeleted))
		if err != nil {
			return err
		}
		if affected == 0 {
			return nil
		}
		if err := r.updateCommentCountDelta(ctx, tx, commentModel.RootID, -1); err != nil {
			return err
		}
		return r.updateArticleCommentCountDelta(ctx, tx, commentModel.ArticleID, -1)
	})
}

func (r *commentRepository) findByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*model.Comment, error) {
	var comment model.Comment
	err := tx.WithContext(ctx).
		Select("id", "article_id", "user_id", "reply_to_user_id", "content", "root_id", "like_count", "comment_count", "ip", "created_time", "updated_time", "status").
		Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", id).
		First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *commentRepository) updateCommentCountDelta(ctx context.Context, tx *gorm.DB, id uint64, delta int64) error {
	return tx.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

func (r *commentRepository) updateArticleCommentCountDelta(ctx context.Context, tx *gorm.DB, articleID uint64, delta int64) error {
	return tx.WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", articleID).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

func (r *commentRepository) batchUpdateChildCommentStatus(ctx context.Context, tx *gorm.DB, rootID uint64) (int64, error) {
	res := tx.WithContext(ctx).
		Model(&model.Comment{}).
		Where("root_id = ? AND status = ?", rootID, model.CommentLiked).
		Update("status", model.CommentCancelLiked)
	return res.RowsAffected, res.Error
}

func (r *commentRepository) updateStatus(ctx context.Context, tx *gorm.DB, id uint64, status uint8) (int64, error) {
	res := tx.WithContext(ctx).Model(&model.Comment{}).
		Where("id=? AND status = ?", id, model.CommentLiked).
		Update("status", status)
	return res.RowsAffected, res.Error
}

type commentWithUserRow struct {
	model.Comment
	Nickname      string `gorm:"column:nickname"`
	Avatar        string `gorm:"column:avatar"`
	IP            string `gorm:"column:last_login_ip"`
	ReplyNickname string `gorm:"column:reply_nickname"`
	ReplyAvatar   string `gorm:"column:reply_avatar"`
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

func toDomainCommentWithUsers(rows []*commentWithUserRow) []*domaincommunity.CommentWithUser {
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
