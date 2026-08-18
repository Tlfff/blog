package community

import (
	"blog/internal/common"
	domaincommunity "blog/internal/domain/community"
	commentdto "blog/internal/dto/comment"
	iputil "blog/pkg/util/ip"
	"context"
	"errors"
	"log"
)

func (s *Service) CreateComment(ctx context.Context, articleID, rootID, userID, replyToUserID uint64, content, ip string) (*commentdto.CreateCommentResponse, error) {
	comment := &domaincommunity.Comment{
		ArticleID:     articleID,
		RootID:        rootID,
		UserID:        userID,
		ReplyToUserID: replyToUserID,
		Content:       content,
		IP:            ip,
		Status:        domaincommunity.CommentStatusNormal,
	}
	err := s.comments.CreateWithCounts(ctx, comment, rootID > 0)
	if err != nil {
		return nil, mapCommentError(err)
	}
	return &commentdto.CreateCommentResponse{ID: comment.ID, CreatedTime: comment.CreatedTime.Unix()}, nil
}

func (s *Service) ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) (*commentdto.RootCommentListResponse, error) {
	if page <= 0 {
		page = 1
	}
	rows, err := s.comments.ListRootComments(ctx, articleID, lastID, page, pageSize, isDesc, authorID)
	if err != nil {
		return nil, err
	}
	total, _ := s.comments.CountRootComments(ctx, articleID, authorID)
	userMap := buildCommentUserMap(rows)
	nextLastID := uint64(0)
	if len(rows) > 0 {
		nextLastID = rows[len(rows)-1].ID
	}
	likeMap := s.getCommentLikeCounts(ctx, rows)
	return buildRootCommentListResponse(rows, userMap, total, nextLastID, uint64(page), uint64(pageSize), likeMap), nil
}

func (s *Service) ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) (*commentdto.ReplyListResponse, error) {
	rootComment, err := s.comments.FindByID(ctx, rootID)
	if err != nil {
		return nil, mapCommentError(err)
	}
	if rootComment.Status == domaincommunity.CommentStatusDeleted {
		return &commentdto.ReplyListResponse{
			List:   make([]*commentdto.ReplyCommentItem, 0),
			LastID: 0,
		}, nil
	}
	if page <= 0 {
		page = 1
	}
	rows, err := s.comments.ListReplies(ctx, rootID, lastID, page, pageSize)
	if err != nil {
		return nil, err
	}
	total, _ := s.comments.CountReplies(ctx, rootID)
	nextLastID := uint64(0)
	if len(rows) > 0 {
		nextLastID = rows[len(rows)-1].ID
	}
	userMap := buildCommentUserMap(rows)
	likeMap := s.getCommentLikeCounts(ctx, rows)
	return buildReplyListResponse(rows, userMap, total, nextLastID, uint64(page), uint64(pageSize), likeMap), nil
}

func (s *Service) DeleteComment(ctx context.Context, commentID, userID uint64, isAdmin bool) error {
	comment, err := s.comments.FindByID(ctx, commentID)
	if err != nil {
		return mapCommentError(err)
	}
	if comment.Status == domaincommunity.CommentStatusDeleted {
		return common.ErrCommentDeleted
	}
	if !isAdmin && !comment.BelongsTo(userID) {
		return common.ErrCommentPermission
	}
	return s.comments.DeleteWithCounts(ctx, comment)
}

func (s *Service) GetCommentStats(ctx context.Context, commentID uint64) (*commentdto.CommentStatsResponse, error) {
	comment, err := s.comments.FindByID(ctx, commentID)
	if err != nil {
		return nil, mapCommentError(err)
	}
	likeCount := uint64(comment.LikeCount)
	commentCount := uint64(comment.CommentCount)
	return commentdto.NewCommentStatsResponse(commentID, likeCount+commentCount, likeCount), nil
}

func (s *Service) getCommentLikeCounts(ctx context.Context, rows []*domaincommunity.CommentWithUser) map[uint64]uint64 {
	likeMap := make(map[uint64]uint64)
	if len(rows) == 0 || s.likeCounts == nil {
		return likeMap
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	cached, err := s.likeCounts.GetCommentLikeCounts(ctx, ids)
	if err != nil {
		return likeMap
	}
	for _, row := range rows {
		if count, ok := cached[row.ID]; ok {
			likeMap[row.ID] = count
		} else {
			likeMap[row.ID] = uint64(row.LikeCount)
		}
	}
	return likeMap
}

func buildCommentUserMap(rows []*domaincommunity.CommentWithUser) map[uint64]*commentdto.CommentUserInfo {
	userMap := make(map[uint64]*commentdto.CommentUserInfo)
	for _, row := range rows {
		userMap[row.UserID] = &commentdto.CommentUserInfo{
			UserID:   row.UserID,
			Username: row.Nickname,
			Avatar:   row.Avatar,
		}
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

func buildRootCommentListResponse(rows []*domaincommunity.CommentWithUser, userMap map[uint64]*commentdto.CommentUserInfo, total int64, lastID, page, pageSize uint64, likeMap map[uint64]uint64) *commentdto.RootCommentListResponse {
	resp := &commentdto.RootCommentListResponse{
		List:     make([]*commentdto.RootCommentItem, 0, len(rows)),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: pageSize,
	}
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

func buildReplyListResponse(rows []*domaincommunity.CommentWithUser, userMap map[uint64]*commentdto.CommentUserInfo, total int64, lastID, page, pageSize uint64, likeMap map[uint64]uint64) *commentdto.ReplyListResponse {
	resp := &commentdto.ReplyListResponse{
		List:     make([]*commentdto.ReplyCommentItem, 0, len(rows)),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: pageSize,
	}
	for _, row := range rows {
		userInfo := userMap[row.UserID]
		if userInfo == nil {
			userInfo = &commentdto.CommentUserInfo{UserID: row.UserID, Username: "未知用户"}
		}
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

func mapCommentError(err error) error {
	switch {
	case errors.Is(err, domaincommunity.ErrCommentNotFound):
		return common.ErrCommentNotFound
	case errors.Is(err, domaincommunity.ErrCommentDeleted):
		return common.ErrCommentDeleted
	case errors.Is(err, domaincommunity.ErrCommentRootDeleted):
		return common.ErrCommentRootDeleted
	case errors.Is(err, domaincommunity.ErrCommentPermission):
		return common.ErrCommentPermission
	default:
		return err
	}
}

func logError(prefix string, err error) {
	if err != nil {
		log.Printf("%s: %v", prefix, err)
	}
}
