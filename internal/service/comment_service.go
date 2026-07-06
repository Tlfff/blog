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
}

func NewCommentService(commentRepo repository.CommentRepository) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
	}
}

// 发表评论
func (s *CommentService) CreateComment(ctx context.Context, articleID uint64, rootID uint64, userID uint64, replyToUserID uint64, content string, ip string) (*commentDto.CreateCommentResponse, error) {
	// 1. 如果是子评论 (rootID > 0)，校验主楼状态
	if rootID > 0 {
		rootComment, err := s.commentRepo.FindByID(ctx, rootID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, common.ErrCommentNotFound
			}
			return nil, err
		}
		// 状态为 0 说明主楼被删了
		if rootComment.Status == 0 {
			return nil, common.ErrCommentRootDeleted
		}
	}

	// 2. 构造底层实体模型 (剔除 ParentID)
	commentModel := &model.Comment{
		ArticleID:     articleID,
		RootID:        rootID,
		UserID:        userID,
		ReplyToUserID: replyToUserID,
		Content:       content,
		IP:            ip,
		Status:        1, // 1: 正常展示
	}

	// 3. 压入数据库
	if err := s.commentRepo.Insert(ctx, commentModel); err != nil {
		return nil, err
	}

	return &commentDto.CreateCommentResponse{
		ID:          commentModel.ID,
		CreatedTime: commentModel.CreatedTime.Unix(),
	}, nil
}

// 获取主评论列表
func (s *CommentService) GetRootCommentList(ctx context.Context, cond *CommentQueryCondition) (*commentDto.RootCommentListResponse, error) {
	page := cond.Page
	if page <= 0 {
		page = 1
	}

	var joinsData []*repository.CommentWithUser
	var err error

	// 1. 同样的分流策略，只是换成了连表查询
	if cond.LastID > 0 {
		joinsData, err = s.commentRepo.FindRootCommentsWithCursor(ctx, cond.ArticleID, cond.LastID, cond.PageSize, cond.IsDesc, cond.AuthorID)
	} else {
		offset := (page - 1) * cond.PageSize
		joinsData, err = s.commentRepo.FindRootCommentsWithOffset(ctx, cond.ArticleID, offset, cond.PageSize, cond.IsDesc, cond.AuthorID)
	}

	// 获取满足条件的主评论总数
	totalCount, _ := s.commentRepo.CountRootComments(ctx, cond.ArticleID, cond.AuthorID)

	if err != nil {
		return nil, err
	}

	// 2. 核心转换：从连表数据中提取出原生的 models 和满足 DTO 所需的 userMap
	models := make([]*model.Comment, len(joinsData))
	userMap := make(map[uint64]*commentDto.CommentUserInfo)

	for i, jd := range joinsData {
		// 复制出纯粹的单表底层模型引用[cite: 9]
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

	// 3. 完美继承你原本的游标逻辑[cite: 9]
	hasMore := len(models) == cond.PageSize
	var nextLastID uint64 = 0
	if len(models) > 0 {
		nextLastID = models[len(models)-1].ID
	}

	// 4. 调用你原本的工厂函数，确保原有业务行为 100% 一致[cite: 9]
	resp := commentDto.NewRootCommentListResponse(models, userMap, totalCount, hasMore, nextLastID)

	// 🌟 5. 补回你最关心的：遍历主评论列表，将子评论数量绑定到对应项目上！[cite: 9]
	for i, item := range resp.List {
		item.ReplyCount = joinsData[i].ReplyCount
	}

	return resp, nil
}

// 获取子评论列表
func (s *CommentService) GetReplyList(ctx context.Context, cond *ReplyQueryCondition) (*commentDto.ReplyListResponse, error) {
	// 1. 查看主评论是否存在或者被删除[cite: 9]
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
		joinsData, err = s.commentRepo.FindRepliesWithCursor(ctx, cond.RootID, cond.LastID, cond.PageSize)
	} else {
		offset := (page - 1) * cond.PageSize
		joinsData, err = s.commentRepo.FindRepliesWithOffset(ctx, cond.RootID, offset, cond.PageSize)
	}

	if err != nil {
		return nil, err
	}

	// 3. 从连表数据中提取出原生的子评论 models 以及对应的发评人/回复人画像
	models := make([]*model.Comment, len(joinsData))
	userMap := make(map[uint64]*commentDto.CommentUserInfo)

	for i, jd := range joinsData {
		models[i] = &jd.Comment
		models[i].IP = jd.IP
		// 填充发评人[cite: 9]
		userMap[jd.UserID] = &commentDto.CommentUserInfo{
			UserID:   jd.UserID,
			Username: jd.Nickname,
			Avatar:   jd.Avatar,
			// IP:       jd.IP,
		}
		// 填充被回复人[cite: 9]
		if jd.ReplyToUserID > 0 {
			userMap[jd.ReplyToUserID] = &commentDto.CommentUserInfo{
				UserID:   jd.ReplyToUserID,
				Username: jd.ReplyNickname,
				Avatar:   jd.ReplyAvatar,
			}
		}
	}

	// 4. 计算子评论游标及数量边界[cite: 9]
	hasMore := len(models) == cond.PageSize
	var nextLastID uint64 = 0
	if len(models) > 0 {
		nextLastID = models[len(models)-1].ID
	}

	// 计算当前主楼下子评论总数[cite: 9]
	count, _ := s.commentRepo.CountReplies(ctx, cond.RootID)

	// 5. 完好无损地交给你原本的 DTO 渲染机制返回[cite: 9]
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

	return s.commentRepo.UpdateStatus(ctx, commentID, 0)
}
