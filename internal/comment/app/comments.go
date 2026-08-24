package app

import (
	commentdto "blog/internal/comment/app/dto"
	domaincommunity "blog/internal/comment/domain"
	iputil "blog/internal/platform/ip"
	apperrors "blog/internal/shared/apperrors"
	"context"
	"errors"
)

// CreateComment 创建评论，rootID 大于 0 时创建楼中楼回复。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - articleID：所属文章唯一标识。
//   - rootID：根评论唯一标识，主评论为 0。
//   - userID：评论用户唯一标识。
//   - replyToUserID：被回复用户唯一标识，可以为 0。
//   - content：评论正文。
//   - ip：评论来源 IP。
func (s *Service) CreateComment(ctx context.Context, articleID, rootID, userID, replyToUserID uint64, content, ip string) (*commentdto.CreateCommentResponse, error) {
	// 1. 根据 rootID 通过领域构造函数创建主评论或回复
	var comment *domaincommunity.Comment
	if rootID > 0 {
		comment = domaincommunity.NewReply(articleID, rootID, userID, replyToUserID, content, ip)
	} else {
		comment = domaincommunity.NewRootComment(articleID, userID, content, ip)
	}
	// 2. 写入评论并同步维护主楼回复数与文章评论数
	var err error
	if s.tx != nil {
		err = s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
			return s.comments.CreateWithCounts(txCtx, comment, rootID > 0)
		})
	} else {
		err = s.comments.CreateWithCounts(ctx, comment, rootID > 0)
	}
	if err != nil {
		return nil, mapCommentError(err)
	}
	// 3. 返回新评论ID与创建时间
	return &commentdto.CreateCommentResponse{ID: comment.ID, CreatedTime: comment.CreatedTime.Unix()}, nil
}

// ListRootComments 分页查询文章的主评论列表，支持游标与只看楼主。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - articleID：文章唯一标识。
//   - lastID：游标评论唯一标识，为 0 时使用 Offset 分页。
//   - page：当前页码。
//   - pageSize：每页数量。
//   - isDesc：是否按评论唯一标识倒序排列。
//   - authorID：只看楼主时的作者用户唯一标识，为 0 时不过滤。
func (s *Service) ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) (*commentdto.RootCommentListResponse, error) {
	// 1. 兜底页码
	if page <= 0 {
		page = 1
	}
	// 2. 查询主楼评论
	rows, err := s.comments.ListRootComments(ctx, articleID, lastID, page, pageSize, isDesc, authorID)
	if err != nil {
		return nil, err
	}
	// 3. 查询总数用于分页展示
	total, _ := s.comments.CountRootComments(ctx, articleID, authorID)
	userMap := buildCommentUserMap(rows)
	// 4. 汇总评论涉及的用户信息
	nextLastID := uint64(0)
	// 5. 计算下一页游标
	if len(rows) > 0 {
		nextLastID = rows[len(rows)-1].ID
	}
	likeMap := s.getCommentLikeCounts(ctx, rows)
	// 6. 批量读取点赞数并组装响应
	return buildRootCommentListResponse(rows, userMap, total, nextLastID, uint64(page), uint64(pageSize), likeMap), nil
}

// ListReplies 分页查询指定主评论下的回复列表。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - rootID：根评论唯一标识。
//   - lastID：游标回复唯一标识，为 0 时使用 Offset 分页。
//   - page：当前页码。
//   - pageSize：每页数量。
func (s *Service) ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) (*commentdto.ReplyListResponse, error) {
	// 1. 先校验主楼评论是否存在
	rootComment, err := s.comments.FindByID(ctx, rootID)
	if err != nil {
		return nil, mapCommentError(err)
	}
	// 2. 由评论聚合判断主楼是否允许继续查询回复
	if err := rootComment.EnsureReplyable(); errors.Is(err, domaincommunity.ErrCommentRootDeleted) {
		return &commentdto.ReplyListResponse{
			List:   make([]*commentdto.ReplyCommentItem, 0),
			LastID: 0,
		}, nil
	}
	// 3. 兜底页码
	if page <= 0 {
		page = 1
	}
	// 4. 查询回复列表
	rows, err := s.comments.ListReplies(ctx, rootID, lastID, page, pageSize)
	if err != nil {
		return nil, err
	}
	// 5. 查询总数并计算下一页游标
	total, _ := s.comments.CountReplies(ctx, rootID)
	nextLastID := uint64(0)
	if len(rows) > 0 {
		nextLastID = rows[len(rows)-1].ID
	}
	// 6. 汇总用户信息与点赞数并组装响应
	userMap := buildCommentUserMap(rows)
	likeMap := s.getCommentLikeCounts(ctx, rows)
	return buildReplyListResponse(rows, userMap, total, nextLastID, uint64(page), uint64(pageSize), likeMap), nil
}

// DeleteComment 删除评论，管理员可删除任意评论，普通用户只能删除自己的评论。
func (s *Service) DeleteComment(ctx context.Context, commentID, userID uint64, isAdmin bool) error {
	// 1. 查询评论是否存在
	comment, err := s.comments.FindByID(ctx, commentID)
	if err != nil {
		return mapCommentError(err)
	}
	// 2. 由评论聚合统一校验删除状态和操作者权限
	if err := comment.DeleteBy(userID, isAdmin); err != nil {
		return mapCommentError(err)
	}

	// 3. 在本地事务中软删除评论并同步维护相关计数
	if s.tx != nil {
		return s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
			return s.comments.DeleteWithCounts(txCtx, comment)
		})
	}
	return s.comments.DeleteWithCounts(ctx, comment)
}

// GetCommentStats 查询单条评论的热度与点赞数。
func (s *Service) GetCommentStats(ctx context.Context, commentID uint64) (*commentdto.CommentStatsResponse, error) {
	// 1. 查询评论
	comment, err := s.comments.FindByID(ctx, commentID)
	if err != nil {
		return nil, mapCommentError(err)
	}
	// 2. 由评论聚合计算热度值
	likeCount := uint64(comment.LikeCount)
	return commentdto.NewCommentStatsResponse(commentID, comment.HotValue(), likeCount), nil
}

// getCommentLikeCounts 批量读取评论点赞数，缓存缺失时回退到评论表计数。
func (s *Service) getCommentLikeCounts(ctx context.Context, rows []*domaincommunity.CommentWithUser) map[uint64]uint64 {
	// 1. 无评论或未接入缓存时返回空字典
	likeMap := make(map[uint64]uint64)
	if len(rows) == 0 || s.likeCounts == nil {
		return likeMap
	}
	// 2. 收集评论ID并批量读取缓存点赞数
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	cached, err := s.likeCounts.GetCommentLikeCounts(ctx, ids)
	if err != nil {
		return likeMap
	}
	// 3. 缓存命中用缓存值，未命中回退到评论表计数
	for _, row := range rows {
		if count, ok := cached[row.ID]; ok {
			likeMap[row.ID] = count
		} else {
			likeMap[row.ID] = uint64(row.LikeCount)
		}
	}
	return likeMap
}

// buildCommentUserMap 把评论行里的作者与被回复者信息汇总为用户字典。
func buildCommentUserMap(rows []*domaincommunity.CommentWithUser) map[uint64]*commentdto.CommentUserInfo {
	// 1. 逐行登记评论作者信息
	userMap := make(map[uint64]*commentdto.CommentUserInfo)
	for _, row := range rows {
		userMap[row.UserID] = &commentdto.CommentUserInfo{
			UserID:   row.UserID,
			Username: row.Nickname,
			Avatar:   row.Avatar,
		}
		// 2. 存在被回复者时一并登记
		if row.ReplyToUserID > 0 {
			userMap[row.ReplyToUserID] = &commentdto.CommentUserInfo{
				UserID:   row.ReplyToUserID,
				Username: row.ReplyNickname,
				Avatar:   row.ReplyAvatar,
			}
		}
	}
	return userMap
}

// buildRootCommentListResponse 组装主评论列表响应。
//
// 参数说明：
//   - rows：带用户展示信息的评论查询结果。
//   - userMap：用户唯一标识到展示信息的映射。
//   - total：符合条件的主评论总数。
//   - lastID：下一页查询使用的游标评论 ID。
//   - page：当前页码。
//   - pageSize：每页数量。
//   - likeMap：评论唯一标识到点赞数的映射。
func buildRootCommentListResponse(rows []*domaincommunity.CommentWithUser, userMap map[uint64]*commentdto.CommentUserInfo, total int64, lastID, page, pageSize uint64, likeMap map[uint64]uint64) *commentdto.RootCommentListResponse {
	resp := &commentdto.RootCommentListResponse{
		List:     make([]*commentdto.RootCommentItem, 0, len(rows)),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: pageSize,
	}
	// 1. 逐行组装响应项，用户信息缺失时兜底为未知用户
	for _, row := range rows {
		userInfo := userMap[row.UserID]
		if userInfo == nil {
			userInfo = &commentdto.CommentUserInfo{UserID: row.UserID, Username: "未知用户"}
		}
		likeCount := likeMap[row.ID]
		resp.List = append(resp.List, &commentdto.RootCommentItem{
			ID:          row.ID,
			ArticleID:   row.ArticleID,
			User:        userInfo,
			Content:     row.Content,
			CreatedTime: row.CreatedTime.Unix(),
			Status:      row.Status,
			IP:          iputil.ConvertIPToRegion(row.IP),
			ReplyCount:  row.CommentCount,
			LikeCount:   likeCount,
		})
	}
	return resp
}

// buildReplyListResponse 组装楼中楼回复列表响应。
//
// 参数说明：
//   - rows：带用户展示信息的回复查询结果。
//   - userMap：用户唯一标识到展示信息的映射。
//   - total：符合条件的回复总数。
//   - lastID：下一页查询使用的游标回复 ID。
//   - page：当前页码。
//   - pageSize：每页数量。
//   - likeMap：评论唯一标识到点赞数的映射。
func buildReplyListResponse(rows []*domaincommunity.CommentWithUser, userMap map[uint64]*commentdto.CommentUserInfo, total int64, lastID, page, pageSize uint64, likeMap map[uint64]uint64) *commentdto.ReplyListResponse {
	resp := &commentdto.ReplyListResponse{
		List:     make([]*commentdto.ReplyCommentItem, 0, len(rows)),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: pageSize,
	}
	// 1. 逐行组装响应项，用户信息缺失时兜底为未知用户
	for _, row := range rows {
		userInfo := userMap[row.UserID]
		if userInfo == nil {
			userInfo = &commentdto.CommentUserInfo{UserID: row.UserID, Username: "未知用户"}
		}
		// 2. 存在被回复者时填充被回复者信息
		var replyToUserInfo *commentdto.CommentUserInfo
		if row.ReplyToUserID > 0 {
			replyToUserInfo = userMap[row.ReplyToUserID]
			if replyToUserInfo == nil {
				replyToUserInfo = &commentdto.CommentUserInfo{UserID: row.ReplyToUserID, Username: "未知用户"}
			}
		}
		likeCount := likeMap[row.ID]
		resp.List = append(resp.List, &commentdto.ReplyCommentItem{
			ID:          row.ID,
			ArticleID:   row.ArticleID,
			RootID:      row.RootID,
			User:        userInfo,
			ReplyToUser: replyToUserInfo,
			Content:     row.Content,
			CreatedTime: row.CreatedTime.Unix(),
			Status:      row.Status,
			IP:          iputil.ConvertIPToRegion(row.IP),
			LikeCount:   likeCount,
		})
	}
	return resp
}

// mapCommentError 把领域评论错误映射为对外统一业务错误。
func mapCommentError(err error) error {
	switch {
	case errors.Is(err, domaincommunity.ErrCommentNotFound):
		return apperrors.ErrCommentNotFound
	case errors.Is(err, domaincommunity.ErrCommentDeleted):
		return apperrors.ErrCommentDeleted
	case errors.Is(err, domaincommunity.ErrCommentRootDeleted):
		return apperrors.ErrCommentRootDeleted
	case errors.Is(err, domaincommunity.ErrCommentPermission):
		return apperrors.ErrCommentPermission
	default:
		return err
	}
}

// GetInteractionTarget 返回 Like、Notification 等上下文所需的最小评论信息。
func (s *Service) GetInteractionTarget(ctx context.Context, commentID uint64) (*InteractionTarget, error) {
	comment, err := s.comments.FindByID(ctx, commentID)
	if err != nil {
		return nil, mapCommentError(err)
	}
	return &InteractionTarget{
		ID:        comment.ID,
		ArticleID: comment.ArticleID,
		UserID:    comment.UserID,
		Active:    comment.IsNormal(),
	}, nil
}
