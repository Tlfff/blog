package service

import (
	"blog/internal/common"
	commentDto "blog/internal/dto/comment"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
	"errors"

	"gorm.io/gorm"
)

// 核心业务层：主评论列表查询条件
type CommentQueryCondition struct {
	ArticleID uint64
	Page      int
	PageSize  int
	LastID    uint64
	IsDesc    bool
	AuthorID  uint64
}

// 核心业务层：子评论列表查询条件
type ReplyQueryCondition struct {
	RootID   uint64
	Page     int
	PageSize int
	LastID   uint64
}

type CommentService struct {
	commentRepo repository.CommentRepository
	articleRepo *repository.ArticleRepository
}

func NewCommentService(commentRepo repository.CommentRepository, articleRepo *repository.ArticleRepository) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		articleRepo: articleRepo,
	}
}

// 发表评论
func (s *CommentService) CreateComment(ctx context.Context, articleID uint64, rootID uint64, userID uint64, replyToUserID uint64, content string, ip string) (*commentDto.CreateCommentResponse, error) {

	// 1. 构造底层实体模型
	commentModel := &model.Comment{
		ArticleID:     articleID,
		RootID:        rootID,
		UserID:        userID,
		ReplyToUserID: replyToUserID,
		Content:       content,
		IP:            ip,
		Status:        1, // 1: 正常展示
	}
	// 2. 开启事务，插入同时更新文章中的评论数字段
	db := s.commentRepo.GetDB().WithContext(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		// 3. 如果是子评论 (rootID > 0)，校验主楼状态
		if rootID > 0 {
			// 用当前读的方式上锁，避免并发问题
			rootComment, err := s.commentRepo.FindByIDForUpdate(ctx, tx, rootID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return common.ErrCommentNotFound
				}
				return err
			}
			// 状态为 0 说明主楼被删了
			if rootComment.Status == 0 {
				return common.ErrCommentRootDeleted
			}
		}
		// 插入评论
		if err := s.commentRepo.Insert(ctx, tx, commentModel); err != nil {
			return err
		}
		// 如果是发表子评论，需要同步让对应的主楼 comment_count 原子 +1
		if rootID > 0 {
			if err := s.commentRepo.UpdateCommentCountDelta(ctx, tx, rootID, 1); err != nil {
				return err
			}
		}
		// 文章计数更新，tx透传
		return s.articleRepo.UpdateCommentCountDelta(ctx, tx, articleID, 1)
	})

	if err != nil {
		return nil, err
	}
	return &commentDto.CreateCommentResponse{ID: commentModel.ID, CreatedTime: commentModel.CreatedTime.Unix()}, nil
}

// 获取主评论列表
func (s *CommentService) GetRootCommentList(ctx context.Context, cond *CommentQueryCondition) (*commentDto.RootCommentListResponse, error) {
	page := cond.Page
	if page <= 0 {
		page = 1
	}

	var joinsData []*repository.CommentWithUser
	var err error

	// 1. 获取评论数，pagesize+1是为了用于计算是否还有下一页
	if cond.LastID > 0 {
		joinsData, err = s.commentRepo.FindRootCommentsWithCursor(ctx, cond.ArticleID, cond.LastID, cond.PageSize+1, cond.IsDesc, cond.AuthorID)
	} else {
		offset := (page - 1) * cond.PageSize
		joinsData, err = s.commentRepo.FindRootCommentsWithOffset(ctx, cond.ArticleID, offset, cond.PageSize+1, cond.IsDesc, cond.AuthorID)
	}

	// 获取满足条件的主评论总数
	totalCount, _ := s.commentRepo.CountRootComments(ctx, cond.ArticleID, cond.AuthorID)

	if err != nil {
		return nil, err
	}
	// 计算是否还有下一页
	hasMore := len(joinsData) > cond.PageSize
	if hasMore {
		// 截断，只保留原 pageSize 数量的数据，把第 pageSize + 1 条抛弃
		joinsData = joinsData[:cond.PageSize]
	}

	// 2. 核心转换：从连表数据中提取出原生的 models 和满足 DTO 所需的 userMap
	models := make([]*model.Comment, len(joinsData))
	userMap := make(map[uint64]*commentDto.CommentUserInfo)

	for i, jd := range joinsData {
		// 复制出纯粹的单表底层模型引用
		models[i] = &jd.Comment
		models[i].IP = jd.IP
		// 将发评人数据直接抓进 Map
		userMap[jd.UserID] = &commentDto.CommentUserInfo{
			UserID:   jd.UserID,
			Username: jd.Nickname,
			Avatar:   jd.Avatar,
			// IP:       jd.IP,
		}
		// 主评论一般没有被回复人，但为了防防御，只要有也顺便抓进去
		if jd.ReplyToUserID > 0 {
			userMap[jd.ReplyToUserID] = &commentDto.CommentUserInfo{
				UserID:   jd.ReplyToUserID,
				Username: jd.ReplyNickname,
				Avatar:   jd.ReplyAvatar,
			}
		}
	}
	var nextLastID uint64 = 0
	if len(models) > 0 {
		nextLastID = models[len(models)-1].ID
	}

	//
	return commentDto.NewRootCommentListResponse(models, userMap, totalCount, hasMore, nextLastID), nil
}

// 获取子评论列表
func (s *CommentService) GetReplyList(ctx context.Context, cond *ReplyQueryCondition) (*commentDto.ReplyListResponse, error) {
	// 1. 查看主评论是否存在或者被删除
	rootComment, err := s.commentRepo.FindByID(ctx, cond.RootID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrCommentNotFound
		}
		return nil, err
	}

	if rootComment.Status == 0 {
		return &commentDto.ReplyListResponse{
			List:    make([]*commentDto.ReplyCommentItem, 0),
			HasMore: false,
			LastID:  0,
		}, nil
	}

	page := cond.Page
	if page <= 0 {
		page = 1
	}

	var joinsData []*repository.CommentWithUser

	// 2. 走连表游标或分页
	if cond.LastID > 0 {
		joinsData, err = s.commentRepo.FindRepliesWithCursor(ctx, cond.RootID, cond.LastID, cond.PageSize+1)
	} else {
		offset := (page - 1) * cond.PageSize
		joinsData, err = s.commentRepo.FindRepliesWithOffset(ctx, cond.RootID, offset, cond.PageSize+1)
	}

	if err != nil {
		return nil, err
	}
	// 计算是否还有下一页
	hasMore := len(joinsData) > cond.PageSize
	if hasMore {
		// 截断，只保留原 pageSize 数量的数据，把第 pageSize + 1 条抛弃
		joinsData = joinsData[:cond.PageSize]
	}

	// 3. 从连表数据中提取出原生的子评论 models 以及对应的发评人/回复人画像
	models := make([]*model.Comment, len(joinsData))
	userMap := make(map[uint64]*commentDto.CommentUserInfo)

	for i, jd := range joinsData {
		models[i] = &jd.Comment
		models[i].IP = jd.IP
		// 填充发评人
		userMap[jd.UserID] = &commentDto.CommentUserInfo{
			UserID:   jd.UserID,
			Username: jd.Nickname,
			Avatar:   jd.Avatar,
			// IP:       jd.IP,
		}
		// 填充被回复人
		if jd.ReplyToUserID > 0 {
			userMap[jd.ReplyToUserID] = &commentDto.CommentUserInfo{
				UserID:   jd.ReplyToUserID,
				Username: jd.ReplyNickname,
				Avatar:   jd.ReplyAvatar,
			}
		}
	}

	// 4. 计算子评论游标及数量边界

	var nextLastID uint64 = 0
	if len(models) > 0 {
		nextLastID = models[len(models)-1].ID
	}

	// 计算当前主楼下子评论总数
	count, _ := s.commentRepo.CountReplies(ctx, cond.RootID)

	// 5. 完好无损地交给你原本的 DTO 渲染机制返回
	return commentDto.NewReplyListResponse(models, userMap, count, hasMore, nextLastID), nil
}

// 删除评论
func (s *CommentService) DeleteComment(ctx context.Context, commentID uint64, userID uint64, isAdmin bool) error {
	oldComment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrCommentNotFound
		}
		return err
	}

	if oldComment.Status == 0 {
		return common.ErrCommentDeleted
	}

	if !isAdmin && oldComment.UserID != userID {
		return common.ErrCommentPermission
	}

	// 2.开启事务
	db := s.commentRepo.GetDB().WithContext(ctx)

	// 抽取事务内部逻辑，返回error
	execTx := func(tx *gorm.DB) error {
		// 记录待扣减的子评论总数，默认为0
		var replyCount int64 = 0
		// 如果是删除主评论的话，在当前事务快照内查询有效子评论条数，保证数据一致性
		if oldComment.RootID == 0 {
			// 批量软删除该主评论下所有子评论
			replyCount, err = s.commentRepo.BatchUpdateChildCommentStatus(ctx, tx, commentID)
			if err != nil {
				return err
			}
		}

		// 软删除当前评论，接收本次更新实际影响行数
		affected, err := s.commentRepo.UpdateStatus(ctx, tx, commentID, 0)
		if err != nil {
			return err
		}

		// 影响行数=0：评论早已被删除，无需扣减
		if affected == 0 {
			return nil
		}
		// 如果是删除“子评论”，需要同步让它对应的主楼 comment_count 原子 -1
		if oldComment.RootID > 0 {
			if err := s.commentRepo.UpdateCommentCountDelta(ctx, tx, oldComment.RootID, -1); err != nil {
				return err
			}
		}

		// 正常删除，执行扣减
		totalDecrease := 1 + replyCount
		// 事务内同步更新文章评论总数，扣减对应数量，保证评论数与实际评论数据原子一致
		return s.articleRepo.UpdateCommentCountDelta(ctx, tx, oldComment.ArticleID, -totalDecrease)
	}

	return db.Transaction(execTx)
}
