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

// CommentQueryCondition 核心业务层：主评论列表查询条件
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
	userRepo    *repository.UserRepository // 注入用户 Repository 供批量导出使用[cite: 9]
}

func NewCommentService(commentRepo repository.CommentRepository, userRepo *repository.UserRepository) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		userRepo:    userRepo,
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

	// 3. 压入数据库[cite: 10]
	if err := s.commentRepo.Insert(ctx, commentModel); err != nil {
		return nil, err
	}

	return &commentDto.CreateCommentResponse{
		ID:          commentModel.ID,
		CreatedTime: commentModel.CreatedTime.Unix(),
	}, nil
}

// GetRootCommentList 获取主评论列表
func (s *CommentService) GetRootCommentList(ctx context.Context, cond *CommentQueryCondition) (*commentDto.RootCommentListResponse, error) {
	page := cond.Page
	if page <= 0 {
		page = 1
	}

	var models []*model.Comment
	var err error
	var totalCount int64 = 0

	// 核心分流策略：传入了 last_id，走游标分页
	if cond.LastID > 0 {
		models, err = s.commentRepo.FindRootCommentsWithCursor(ctx, cond.ArticleID, cond.LastID, cond.PageSize, cond.IsDesc, cond.AuthorID)
	} else {
		// 否则走传统跳页 Offset 路由
		offset := (page - 1) * cond.PageSize
		models, err = s.commentRepo.FindRootCommentsWithOffset(ctx, cond.ArticleID, offset, cond.PageSize, cond.IsDesc, cond.AuthorID)
	}
	totalCount, _ = s.commentRepo.CountRootComments(ctx, cond.ArticleID, cond.AuthorID)

	if err != nil {
		return nil, err
	}

	// 批量拉取发评人与被回复人的真实资料
	userMap, err := s.GetUserMapByComments(ctx, models)
	if err != nil {
		return nil, err
	}

	hasMore := len(models) == cond.PageSize
	var nextLastID uint64 = 0
	if len(models) > 0 {
		nextLastID = models[len(models)-1].ID
	}

	// 获取主评论数据
	resp := commentDto.NewRootCommentListResponse(models, userMap, totalCount, hasMore, nextLastID)
	// 将子评论数量绑定在对应主评论上
	for _, item := range resp.List {
		count, _ := s.commentRepo.CountReplies(ctx, item.ID)
		item.ReplyCount = count
	}
	return resp, nil
}

// GetReplyList 获取子评论列表
func (s *CommentService) GetReplyList(ctx context.Context, cond *ReplyQueryCondition) (*commentDto.ReplyListResponse, error) {
	// 1. 查看主评论是否存在或者被删除
	rootComment, err := s.commentRepo.FindByID(ctx, cond.RootID)
	if err != nil {
		// 如果主楼干脆不存在，直接抛出未找到错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrCommentNotFound
		}
		return nil, err
	}

	// 如果主楼的 status 是 0（已被删除），直接断流，返回空列表
	if rootComment.Status == 0 {
		return &commentDto.ReplyListResponse{
			List:    make([]*commentDto.ReplyCommentItem, 0),
			HasMore: false,
			LastID:  0,
		}, nil
	}
	// 2. 获取子评论列表
	page := cond.Page
	if page <= 0 {
		page = 1
	}

	var models []*model.Comment

	if cond.LastID > 0 {
		models, err = s.commentRepo.FindRepliesWithCursor(ctx, cond.RootID, cond.LastID, cond.PageSize)
	} else {
		offset := (page - 1) * cond.PageSize
		models, err = s.commentRepo.FindRepliesWithOffset(ctx, cond.RootID, offset, cond.PageSize)
	}

	if err != nil {
		return nil, err
	}

	// 解决缺口一：批量拉取相关用户头像和昵称[cite: 9]
	userMap, err := s.GetUserMapByComments(ctx, models)
	if err != nil {
		return nil, err
	}

	hasMore := len(models) == cond.PageSize
	var nextLastID uint64 = 0
	if len(models) > 0 {
		nextLastID = models[len(models)-1].ID
	}

	count, _ := s.commentRepo.CountReplies(ctx, cond.RootID)
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

// 批量汇聚并清洗导出用户对应信息方法
func (s *CommentService) GetUserMapByComments(ctx context.Context, models []*model.Comment) (map[uint64]*commentDto.CommentUserInfo, error) {
	userMap := make(map[uint64]*commentDto.CommentUserInfo)
	if len(models) == 0 {
		return userMap, nil
	}

	userIdsMap := make(map[uint64]struct{})
	for _, m := range models {
		userIdsMap[m.UserID] = struct{}{}
		if m.ReplyToUserID > 0 {
			userIdsMap[m.ReplyToUserID] = struct{}{}
		}
	}

	ids := make([]uint64, 0, len(userIdsMap))
	for id := range userIdsMap {
		ids = append(ids, id)
	}

	users, err := s.userRepo.FindUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		userMap[u.ID] = &commentDto.CommentUserInfo{
			UserID:   u.ID,
			Username: u.Nickname,
			Avatar:   u.Avatar,
		}
	}

	return userMap, nil
}
