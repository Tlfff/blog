package repository

import (
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
)

// 用于接收包含发评人及被回复人资料的复合结构体
type CommentWithUser struct {
	model.Comment
	// 发评人公共信息
	Nickname string `gorm:"column:nickname"`
	Avatar   string `gorm:"column:avatar"`
	IP       string `gorm:"column:last_login_ip"`
	// 被回复人公共信息 (如果是主评论则为空)
	ReplyNickname string `gorm:"column:reply_nickname"`
	ReplyAvatar   string `gorm:"column:reply_avatar"`
	ReplyCount    int64  `gorm:"column:reply_count"`
}

type CommentRepository interface {
	Insert(ctx context.Context, comment *model.Comment) error

	UpdateStatus(ctx context.Context, id uint64, status uint8) error

	FindByID(ctx context.Context, id uint64) (*model.Comment, error)

	// 游标分页：拉取文章主评论（支持正序/逆序、只看楼主）
	FindRootCommentsWithCursor(ctx context.Context, articleID uint64, lastID uint64, pageSize int, isDesc bool, authorID uint64) ([]*CommentWithUser, error)
	// Offset分页：拉取文章主评论（作为跳页/上一页兜底）
	FindRootCommentsWithOffset(ctx context.Context, articleID uint64, page int, pageSize int, isDesc bool, authorID uint64) ([]*CommentWithUser, error)
	//  游标分页：展开拉取子评论列表（固定升序）
	FindRepliesWithCursor(ctx context.Context, rootID uint64, lastID uint64, pageSize int) ([]*CommentWithUser, error)
	// Offset分页：展开拉取子评论列表（作为跳页/上一页兜底）
	FindRepliesWithOffset(ctx context.Context, rootID uint64, offset int, pageSize int) ([]*CommentWithUser, error)
	// 计算满足条件的主评论总数 (用于主列表分页的 total)
	CountRootComments(ctx context.Context, articleID uint64, authorID uint64) (int64, error)
	// 计算某个主评论下的子评论总数 (用于楼层内回复数展示)
	CountReplies(ctx context.Context, rootID uint64) (int64, error)
}

type commentRepository struct {
	db *gorm.DB
}

// 计算子评论数量
// select count(*) from comments where root_id=? and status=1
func (c *commentRepository) CountReplies(ctx context.Context, rootID uint64) (int64, error) {
	var count int64
	err := c.db.WithContext(ctx).Model(&model.Comment{}).Where("root_id=? AND status=1", rootID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// 计算文章中总评论数量
// select count(*) from comments where article=? and root_id=? and status=1
func (c *commentRepository) CountRootComments(ctx context.Context, articleID uint64, authorID uint64) (int64, error) {
	var count int64
	tx := c.db.WithContext(ctx).Model(&model.Comment{}).Where("article_id=? AND root_id=0 AND status=1", articleID)
	if authorID > 0 {
		tx = tx.Where("user_id=?", authorID)
	}
	err := tx.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// 软删除评论
// update comments set status = 0 where id=?
func (c *commentRepository) DeleteCommment(ctx context.Context, id uint64) error {
	return c.db.WithContext(ctx).Delete(&model.Comment{}).Error
}

// 通过id查找评论
// select id, article_id, user_id, reply_to_user_id, content, root_id, created_time, updated_time, status
// from comments where id=?
func (c *commentRepository) FindByID(ctx context.Context, id uint64) (*model.Comment, error) {
	var comment model.Comment
	err := c.db.WithContext(ctx).
		Select("id", "article_id", "user_id", "reply_to_user_id", "content", "root_id", "created_time", "updated_time", "status").
		Where("id=?", id).
		First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// 子评论：游标方式查找列表,固定升序
// SELECT
//
//	c.id, c.article_id, c.user_id, c.reply_to_user_id, c.content, c.root_id, c.created_time, c.updated_time, c.status,
//	u1.nickname AS nickname, u1.avatar AS avatar,u1.last_login_ip AS last_login_ip,
//	u2.nickname AS reply_nickname, u2.avatar AS reply_avatar
//
// FROM comments c
// LEFT JOIN users u1 ON c.user_id = u1.id
// LEFT JOIN users u2 ON c.reply_to_user_id = u2.id
// WHERE c.root_id = ？ AND c.status = 1 AND c.id > last_id
// ORDER BY c.id ASC
// LIMIT pageSize;
func (c *commentRepository) FindRepliesWithCursor(ctx context.Context, rootID uint64, lastID uint64, pageSize int) ([]*CommentWithUser, error) {
	var list []*CommentWithUser
	err := c.db.WithContext(ctx).Table("comments c").
		Select(`c.id, c.article_id, c.user_id, c.reply_to_user_id, c.content, c.root_id, c.created_time, c.updated_time, c.status,
				u1.nickname AS nickname, u1.avatar AS avatar, u1.last_login_ip AS last_login_ip, 
				u2.nickname AS reply_nickname, u2.avatar AS reply_avatar`).
		Joins("LEFT JOIN users u1 ON c.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON c.reply_to_user_id = u2.id").
		Where("c.root_id = ? AND c.status = 1 AND c.id > ?", rootID, lastID).
		Order("c.id ASC").Limit(pageSize).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 子评论：传统分页方式查找列表
// SELECT
//
//	c.id,c.article_id,c.user_id,c.reply_to_user_id,c.content,c.root_id,c.created_time,c.updated_time,c.status,
//	u1.nickname AS nickname,u1.avatar AS avatar,u1.last_login_ip AS last_login_ip,
//	u2.nickname AS reply_nickname,u2.avatar AS reply_avatar
//
// FROM comments c
// LEFT JOIN users u1 ON c.user_id = u1.id
// LEFT JOIN users u2 ON c.reply_to_user_id = u2.id
// WHERE c.root_id = ? AND c.status = 1
// ORDER BY c.id ASC
// LIMIT ? OFFSET ?;
func (c *commentRepository) FindRepliesWithOffset(ctx context.Context, rootID uint64, page int, pageSize int) ([]*CommentWithUser, error) {
	var list []*CommentWithUser
	err := c.db.WithContext(ctx).Table("comments c").
		Select(`c.id,c.article_id,c.user_id,c.reply_to_user_id,c.content,c.root_id,c.created_time,c.updated_time,c.status,
				u1.nickname AS nickname,u1.avatar AS avatar,u1.last_login_ip AS last_login_ip, 
				u2.nickname AS reply_nickname,u2.avatar AS reply_avatar`).
		Joins("LEFT JOIN users u1 ON c.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON c.reply_to_user_id = u2.id").
		Where("c.root_id = ? AND c.status = 1", rootID).
		Order("c.id ASC").Limit(pageSize).Offset((page - 1) * pageSize).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 主评论：游标方式查找列表
// SELECT
//
//	c.id,c.article_id,c.user_id,c.reply_to_user_id,c.content,c.root_id,c.created_time,c.updated_time,c.status,
//	u1.nickname AS nickname, u1.avatar AS avatar, u1.last_login_ip AS last_login_ip,
//	u2.nickname AS reply_nickname, u2.avatar AS reply_avatar,
//	IFNULL(rc.reply_count, 0) AS reply_count
//
// FROM comments c
// LEFT JOIN users u1 ON c.user_id = u1.id
// LEFT JOIN users u2 ON c.reply_to_user_id = u2.id
// LEFT JOIN (SELECT root_id, COUNT(*) AS reply_count FROM comments WHERE root_id > 0 AND status = 1 GROUP BY root_id) rc ON c.id = rc.root_id
// WHERE c.article_id = ? AND c.root_id = 0 AND c.status = 1
// -- 可选条件：authorID>0 拼接 AND c.user_id = ?
// -- 游标条件：isDesc=true 拼接 AND c.id < ? ORDER BY c.id DESC
// -- 游标条件：isDesc=false 拼接 AND c.id > ? ORDER BY c.id ASC
// LIMIT ?;
func (c *commentRepository) FindRootCommentsWithCursor(ctx context.Context, articleID uint64, lastID uint64, pageSize int, isDesc bool, authorID uint64) ([]*CommentWithUser, error) {
	var list []*CommentWithUser

	tx := c.db.WithContext(ctx).Table("comments").
		Select(`c.id,c.article_id,c.user_id,c.reply_to_user_id,c.content,c.root_id,c.created_time,c.updated_time,c.status,
			u1.nickname AS nickname, u1.avatar AS avatar,u1.last_login_ip AS last_login_ip, 
			u2.nickname AS reply_nickname, u2.avatar AS reply_avatar,
			IFNULL(rc.reply_count, 0) AS reply_count`).
		Joins("LEFT JOIN users u1 ON c.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON c.reply_to_user_id = u2.id").
		Joins(`LEFT JOIN (
			SELECT root_id, COUNT(*) AS reply_count 
			FROM comments 
			WHERE root_id > 0 AND status = 1 
			GROUP BY root_id
		) rc ON c.id = rc.root_id`).
		Where("c.article_id = ? AND c.root_id = 0 AND c.status = 1", articleID)

	if authorID > 0 {
		tx = tx.Where("c.user_id = ?", authorID)
	}
	if isDesc {
		tx = tx.Where("c.id < ?", lastID).Order("c.id DESC")
	} else {
		tx = tx.Where("c.id > ?", lastID).Order("c.id ASC")
	}
	err := tx.Limit(pageSize).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 主评论：传统分页方式查找列表
// SELECT
//
//	c.id,c.article_id,c.user_id,c.reply_to_user_id,c.content,c.root_id,c.created_time,c.updated_time,c.status,
//	u1.nickname AS nickname, u1.avatar AS avatar, u1.last_login_ip AS last_login_ip,
//	u2.nickname AS reply_nickname, u2.avatar AS reply_avatar,
//	IFNULL(rc.reply_count, 0) AS reply_count
//
// FROM comments c
// LEFT JOIN users u1 ON c.user_id = u1.id
// LEFT JOIN users u2 ON c.reply_to_user_id = u2.id
// LEFT JOIN (SELECT root_id, COUNT(*) AS reply_count FROM comments WHERE root_id > 0 AND status = 1 GROUP BY root_id) rc ON c.id = rc.root_id
// WHERE c.article_id = ? AND c.root_id = 0 AND c.status = 1
// -- 可选条件：authorID>0 自动拼接 AND c.user_id = ?
// -- 排序：isDesc=true ORDER BY c.id DESC，否则 ORDER BY c.id ASC
// LIMIT ? OFFSET ?;
func (c *commentRepository) FindRootCommentsWithOffset(ctx context.Context, articleID uint64, page int, pageSize int, isDesc bool, authorID uint64) ([]*CommentWithUser, error) {
	var list []*CommentWithUser
	tx := c.db.WithContext(ctx).Table("comments c").
		Select(`c.id,c.article_id,c.user_id,c.reply_to_user_id,c.content,c.root_id,c.created_time,c.updated_time,c.status,
			u1.nickname AS nickname, u1.avatar AS avatar,u1.last_login_ip AS last_login_ip, 
			u2.nickname AS reply_nickname, u2.avatar AS reply_avatar,
			IFNULL(rc.reply_count, 0) AS reply_count`).
		Joins("LEFT JOIN users u1 ON c.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON c.reply_to_user_id = u2.id").
		Joins(`LEFT JOIN (
			SELECT root_id, COUNT(*) AS reply_count 
			FROM comments 
			WHERE root_id > 0 AND status = 1 
			GROUP BY root_id
		) rc ON c.id = rc.root_id`).
		Where("c.article_id = ? AND c.root_id = 0 AND c.status = 1", articleID)

	if authorID > 0 {
		tx = tx.Where("c.user_id = ?", authorID)
	}
	if isDesc {
		tx = tx.Order("c.id DESC")
	} else {
		tx = tx.Order("c.id ASC")
	}

	err := tx.Limit(pageSize).Offset((page - 1) * pageSize).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 插入评论
// insert into comment
// article_id, user_id, reply_to_user_id, content, root_id, parent_id
// values(?,?,?,?,?,?)
func (c *commentRepository) Insert(ctx context.Context, comment *model.Comment) error {
	return c.db.WithContext(ctx).Create(&comment).Error
}

// 更新评论状态
// update comment set status=?
// where id =?
func (c *commentRepository) UpdateStatus(ctx context.Context, id uint64, status uint8) error {
	return c.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id=?", id).
		Update("status", status).Error
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{
		db: db,
	}
}
