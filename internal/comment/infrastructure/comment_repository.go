package infrastructure

import (
	domaincommunity "blog/internal/comment/domain"
	"blog/internal/comment/infrastructure/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"
	"errors"

	"gorm.io/gorm"
)

// commentRepository 是基于 GORM 的评论 Repository 适配器。
type commentRepository struct {
	db         *gorm.DB                          // GORM 数据库连接
	statistics domaincommunity.ArticleStatistics // Article 统计 Port
}

// 创建 Comment 上下文的 GORM 评论 Repository
func NewCommentRepository(db *gorm.DB, statistics ...domaincommunity.ArticleStatistics) domaincommunity.CommentRepository {
	var stats domaincommunity.ArticleStatistics
	if len(statistics) > 0 {
		stats = statistics[0]
	}
	return &commentRepository{db: db, statistics: stats}
}

// 创建评论，并在同一事务内维护主楼回复数与文章评论数
func (r *commentRepository) CreateWithCounts(ctx context.Context, comment *domaincommunity.Comment, incrementReply bool) error {
	// 1. 从事务上下文取得连接，无事务时回退默认连接
	db := platformtransaction.DB(ctx, r.db)
	commentModel := toModelComment(comment)

	// 2. 回复场景下先加锁校验主楼状态
	if incrementReply {
		rootComment, err := r.findByIDForUpdate(ctx, db, comment.RootID)
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

	// 3. 写入评论并维护根评论回复数
	if err := db.WithContext(ctx).Create(commentModel).Error; err != nil {
		return err
	}
	if incrementReply {
		if err := r.updateCommentCountDelta(ctx, db, comment.RootID, 1); err != nil {
			return err
		}
	}

	// 4. 通过 Article Port 在同一事务中更新文章评论数
	if r.statistics != nil {
		if err := r.statistics.IncrementCommentCount(ctx, comment.ArticleID, 1); err != nil {
			return err
		}
	}

	// 5. 回填自增 ID 和时间字段
	comment.ID = commentModel.ID
	comment.CreatedTime = commentModel.CreatedTime
	comment.UpdatedTime = commentModel.UpdatedTime
	return nil
}

// 按ID查询单条评论，未命中时返回领域错误
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

// 分页查询文章的主楼评论列表，同时联表补齐作者与被回复者信息
func (r *commentRepository) ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) ([]*domaincommunity.CommentWithUser, error) {
	// 1. 联表查询评论及作者、被回复者的公开信息
	tx := r.db.WithContext(ctx).Table("comments c").
		Select(`c.id, c.article_id, c.user_id, c.reply_to_user_id, c.content, c.root_id, c.like_count, c.comment_count, c.ip, c.created_time, c.updated_time, c.status,
			u1.nickname AS nickname, u1.avatar AS avatar, u1.last_login_ip AS last_login_ip,
			u2.nickname AS reply_nickname, u2.avatar AS reply_avatar`).
		Joins("LEFT JOIN users u1 ON c.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON c.reply_to_user_id = u2.id").
		Where("c.article_id = ? AND c.root_id = 0 AND c.status = ?", articleID, domaincommunity.CommentStatusNormal)
	// 2. 只看楼主时追加作者过滤
	if authorID > 0 {
		tx = tx.Where("c.user_id = ?", authorID)
	}
	// 3. 有游标走游标分页，否则走传统 Offset 分页
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
	// 4. 扫描结果并转换为领域对象
	var rows []*commentWithUserRow
	if err := tx.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return toDomainCommentWithUsers(rows), nil
}

// 统计文章的主楼评论总数，支持只看楼主
func (r *commentRepository) CountRootComments(ctx context.Context, articleID, authorID uint64) (int64, error) {
	var count int64
	// 1. 统计条件：指定文章的正常状态主楼评论
	tx := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("article_id=? AND root_id=0 AND status=?", articleID, domaincommunity.CommentStatusNormal)
	// 2. 只看楼主时追加作者过滤
	if authorID > 0 {
		tx = tx.Where("user_id=?", authorID)
	}
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// 分页查询指定主楼下的回复列表（楼中楼）
func (r *commentRepository) ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) ([]*domaincommunity.CommentWithUser, error) {
	// 1. 联表查询回复及作者、被回复者的公开信息
	tx := r.db.WithContext(ctx).Table("comments c").
		Select(`c.id, c.article_id, c.user_id, c.reply_to_user_id, c.content, c.root_id, c.like_count, c.comment_count, c.ip, c.created_time, c.updated_time, c.status,
			u1.nickname AS nickname, u1.avatar AS avatar, u1.last_login_ip AS last_login_ip,
			u2.nickname AS reply_nickname, u2.avatar AS reply_avatar`).
		Joins("LEFT JOIN users u1 ON c.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON c.reply_to_user_id = u2.id").
		Where("c.root_id = ? AND c.status = ?", rootID, domaincommunity.CommentStatusNormal)
	// 2. 有游标走游标分页，否则走传统 Offset 分页
	if lastID > 0 {
		tx = tx.Where("c.id > ?", lastID).Order("c.id ASC").Limit(pageSize)
	} else {
		tx = tx.Order("c.id ASC").Limit(pageSize).Offset((page - 1) * pageSize)
	}
	// 3. 扫描结果并转换为领域对象
	var rows []*commentWithUserRow
	if err := tx.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return toDomainCommentWithUsers(rows), nil
}

// 统计指定主楼下的回复总数
func (r *commentRepository) CountReplies(ctx context.Context, rootID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("root_id=? AND status=?", rootID, domaincommunity.CommentStatusNormal).
		Count(&count).Error
	return count, err
}

// 删除评论，并在同一事务内维护主楼回复数与文章评论数
func (r *commentRepository) DeleteWithCounts(ctx context.Context, comment *domaincommunity.Comment) error {
	// 1. 从事务上下文取得连接，无事务时回退默认连接
	db := platformtransaction.DB(ctx, r.db)
	commentModel := toModelComment(comment)

	// 2. 主楼评论连带软删除全部回复，并一次扣减文章评论数
	if commentModel.RootID == 0 {
		replyCount, err := r.batchUpdateChildCommentStatus(ctx, db, commentModel.ID)
		if err != nil {
			return err
		}
		affected, err := r.updateStatus(ctx, db, commentModel.ID, uint8(domaincommunity.CommentStatusDeleted))
		if err != nil || affected == 0 {
			return err
		}
		if r.statistics != nil {
			return r.statistics.IncrementCommentCount(ctx, commentModel.ArticleID, -(replyCount + 1))
		}
		return nil
	}

	// 3. 回复评论软删除自身并扣减根评论回复数和文章评论数
	affected, err := r.updateStatus(ctx, db, commentModel.ID, uint8(domaincommunity.CommentStatusDeleted))
	if err != nil || affected == 0 {
		return err
	}
	if err := r.updateCommentCountDelta(ctx, db, commentModel.RootID, -1); err != nil {
		return err
	}
	if r.statistics != nil {
		return r.statistics.IncrementCommentCount(ctx, commentModel.ArticleID, -1)
	}
	return nil
}

// 在事务内加行锁查询评论，避免并发回复与删除产生计数错乱
func (r *commentRepository) findByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*model.Comment, error) {
	// 1. 使用 FOR UPDATE 加行锁读取评论
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

// 按增量更新主楼评论的回复数
func (r *commentRepository) updateCommentCountDelta(ctx context.Context, tx *gorm.DB, id uint64, delta int64) error {
	return tx.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

// 按增量更新文章的评论数

// 批量把指定主楼下的正常回复标记为已删除，返回受影响行数
func (r *commentRepository) batchUpdateChildCommentStatus(ctx context.Context, tx *gorm.DB, rootID uint64) (int64, error) {
	// 1. 把该主楼下所有正常回复批量标记为已删除
	res := tx.WithContext(ctx).
		Model(&model.Comment{}).
		Where("root_id = ? AND status = ?", rootID, domaincommunity.CommentStatusNormal).
		Update("status", domaincommunity.CommentStatusDeleted)
	return res.RowsAffected, res.Error
}

// 把指定评论从正常状态更新为目标状态，返回受影响行数
func (r *commentRepository) updateStatus(ctx context.Context, tx *gorm.DB, id uint64, status uint8) (int64, error) {
	// 1. 仅当评论当前为正常状态时才更新，避免重复扣减计数
	res := tx.WithContext(ctx).Model(&model.Comment{}).
		Where("id=? AND status = ?", id, domaincommunity.CommentStatusNormal).
		Update("status", status)
	return res.RowsAffected, res.Error
}

// commentWithUserRow 是评论联表查询的行映射结构。
type commentWithUserRow struct {
	model.Comment        // 内嵌评论表字段
	Nickname      string `gorm:"column:nickname"`       // 评论发布者昵称
	Avatar        string `gorm:"column:avatar"`         // 评论发布者头像URL
	IP            string `gorm:"column:last_login_ip"`  // 评论发布者最后登录IP，用于展示归属地
	ReplyNickname string `gorm:"column:reply_nickname"` // 被回复者昵称
	ReplyAvatar   string `gorm:"column:reply_avatar"`   // 被回复者头像URL
}

// 把评论领域对象转换为数据库模型
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

// 把数据库模型转换为评论领域对象
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

// 把联表查询结果批量转换为带用户信息的评论领域对象
func toDomainCommentWithUsers(rows []*commentWithUserRow) []*domaincommunity.CommentWithUser {
	// 1. 逐行转换评论主体并附加用户展示信息
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
