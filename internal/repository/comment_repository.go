package repository

import (
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
)

type CommentRepository interface {
	Insert(ctx context.Context, comment *model.Comment) error
	UpdateStatus(ctx context.Context, id uint64, status uint8) error
	FindByID(ctx context.Context, id uint64) (*model.Comment, error)

	// 游标分页：拉取文章主评论（支持正序/逆序、只看楼主）
	FindRootCommentsWithCursor(ctx context.Context, articleID uint64, lastID uint64, pageSize int, isDesc bool, authorID uint64) ([]*model.Comment, error)

	// Offset分页：拉取文章主评论（作为跳页/上一页兜底）
	FindRootCommentsWithOffset(ctx context.Context, articleID uint64, page int, pageSize int, isDesc bool, authorID uint64) ([]*model.Comment, error)

	//  游标分页：展开拉取子评论列表（固定升序）
	FindRepliesWithCursor(ctx context.Context, rootID uint64, lastID uint64, pageSize int) ([]*model.Comment, error)

	// Offset分页：展开拉取子评论列表（作为跳页/上一页兜底）
	FindRepliesWithOffset(ctx context.Context, rootID uint64, offset int, pageSize int) ([]*model.Comment, error)

	// Offset分页：展开拉取子评论列表（作为跳页/上一页兜底）
	DeleteCommment(ctx context.Context, id uint64) error
}

type commentRepository struct {
	db *gorm.DB
}

// 软删除评论
// update comments set status = 0 where id=?
func (c *commentRepository) DeleteCommment(ctx context.Context, id uint64) error {
	return c.db.WithContext(ctx).Delete(&model.Comment{}).Error
}

// 通过id查找评论
// select id, article_id, user_id, reply_to_user_id, content, root_id, parent_id, created_time, updated_time, status
// from comments where id=?
func (c *commentRepository) FindByID(ctx context.Context, id uint64) (*model.Comment, error) {
	var comment model.Comment
	err := c.db.WithContext(ctx).
		Select("id", "article_id", "user_id", "reply_to_user_id", "content", "root_id", "parent_id", "created_time", "updated_time", "status").
		Where("id=?", id).
		First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// 子评论：游标方式查找列表,固定升序
// select id, article_id, user_id, reply_to_user_id, content, root_id, parent_id, created_time, updated_time, status
// from comments where root_id = ? and id > lastID and status = 1
// limit pageSize
func (c *commentRepository) FindRepliesWithCursor(ctx context.Context, rootID uint64, lastID uint64, pageSize int) ([]*model.Comment, error) {
	var list []*model.Comment
	err := c.db.WithContext(ctx).
		Select("id", "article_id", "user_id", "reply_to_user_id", "content", "root_id", "parent_id", "created_time", "updated_time", "status").
		Where("root_id=?  AND status=1 AND id>?", rootID, lastID).
		Order("id ASC").Limit(pageSize).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 子评论：传统分页方式查找列表
// select id, article_id, user_id, reply_to_user_id, content, root_id, parent_id, created_time, updated_time, status
// from comments
// where root_id = rootID and status=1
// limit pageSize offset page
func (c *commentRepository) FindRepliesWithOffset(ctx context.Context, rootID uint64, page int, pageSize int) ([]*model.Comment, error) {
	var list []*model.Comment
	err := c.db.WithContext(ctx).
		Select("id", "article_id", "user_id", "reply_to_user_id", "content", "root_id", "parent_id", "created_time", "updated_time", "status").
		Where("root_id=?  AND status=1", rootID).
		Order("id ASC").Limit(pageSize).Offset(page).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 主评论：游标方式查找列表
// select id, article_id, user_id, reply_to_user_id, content, root_id, parent_id, created_time, updated_time, status
// from comments
// where article_id = articleID and root_id=0 and status=1 and id > lastID
// limit pageSize
func (c *commentRepository) FindRootCommentsWithCursor(ctx context.Context, articleID uint64, lastID uint64, pageSize int, isDesc bool, authorID uint64) ([]*model.Comment, error) {
	var list []*model.Comment
	tx := c.db.WithContext(ctx).
		Select("id", "article_id", "user_id", "reply_to_user_id", "content", "root_id", "parent_id", "created_time", "updated_time", "status").
		Where("article_id=? AND root_id=0 AND status=1", articleID)
	// 只看楼主
	if authorID > 0 {
		tx = tx.Where("user_id=?", authorID)
	}
	// 判断排序形式
	if isDesc {
		tx = tx.Where("id<?", lastID).Order("id DESC") // 降序就要找比上一个id小的
	} else {
		tx = tx.Where("id>?", lastID).Order("id ASC")
	}
	err := tx.Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 主评论：传统分页方式查找列表
// select id, article_id, user_id, reply_to_user_id, content, root_id, parent_id, created_time, updated_time, status
// from comments
// where article_id = articleID and root_id = 0 and status = 1
// limit pageSize offset page
func (c *commentRepository) FindRootCommentsWithOffset(ctx context.Context, articleID uint64, page int, pageSize int, isDesc bool, authorID uint64) ([]*model.Comment, error) {
	var list []*model.Comment
	tx := c.db.WithContext(ctx).
		Select("id", "article_id", "user_id", "reply_to_user_id", "content", "root_id", "parent_id", "created_time", "updated_time", "status").
		Where("article_id=?  AND root_id=0 AND status=1", articleID)
	// 只看楼主
	if authorID > 0 {
		tx = tx.Where("user_id=?", authorID)
	}
	if isDesc {
		tx = tx.Order("id DESC")
	} else {
		tx = tx.Order("id ASC")
	}
	err := tx.Limit(pageSize).Offset(page).Find(&list).Error
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
