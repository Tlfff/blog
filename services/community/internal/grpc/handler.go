// Package grpc 实现 Community Service 的内部 gRPC 接口。
package grpc

import (
	communityapp "blog/internal/application/community"
	articledto "blog/internal/dto/article"
	commentdto "blog/internal/dto/comment"
	notificationdto "blog/internal/dto/notification"
	internalv1 "blog/shared/contracts/gen/internalv1"
	platformerrors "blog/shared/platform/errors"
	"context"
)

// CommunityServer 把 Community Application 用例暴露为内部 gRPC 服务。
type CommunityServer struct {
	internalv1.UnimplementedCommunityServiceServer
	app *communityapp.Service
}

func NewCommunityServer(app *communityapp.Service) *CommunityServer {
	return &CommunityServer{app: app}
}

func (s *CommunityServer) CreateComment(ctx context.Context, req *internalv1.CreateCommentRequest) (*internalv1.CreateCommentResponse, error) {
	resp, err := s.app.CreateComment(ctx, req.ArticleId, req.RootId, req.UserId, req.ReplyToUserId, req.Content, req.Ip)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.CreateCommentResponse{Id: resp.ID, CreatedTime: resp.CreatedTime}, nil
}

func (s *CommunityServer) ListRootComments(ctx context.Context, req *internalv1.ListRootCommentsRequest) (*internalv1.RootCommentList, error) {
	resp, err := s.app.ListRootComments(ctx, req.ArticleId, req.LastId, int(req.Page), int(req.PageSize), req.IsDesc, req.AuthorId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	items := make([]*internalv1.RootCommentItem, 0, len(resp.List))
	for _, item := range resp.List {
		user := &internalv1.CommentUserInfo{UserId: item.User.UserID, Username: item.User.Username, Avatar: item.User.Avatar}
		items = append(items, &internalv1.RootCommentItem{
			Id:          item.ID,
			ArticleId:   item.ArticleID,
			User:        user,
			Content:     item.Content,
			ReplyCount:  item.ReplyCount,
			Ip:          item.IP,
			CreatedTime: item.CreatedTime,
			Status:      int32(item.Status),
			LikeCount:   item.LikeCount,
		})
	}
	return &internalv1.RootCommentList{
		Items:    items,
		Total:    resp.Total,
		LastId:   resp.LastID,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}

func (s *CommunityServer) ListReplies(ctx context.Context, req *internalv1.ListRepliesRequest) (*internalv1.ReplyList, error) {
	resp, err := s.app.ListReplies(ctx, req.RootId, req.LastId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	items := make([]*internalv1.ReplyCommentItem, 0, len(resp.List))
	for _, item := range resp.List {
		user := &internalv1.CommentUserInfo{UserId: item.User.UserID, Username: item.User.Username, Avatar: item.User.Avatar}
		var replyTo *internalv1.CommentUserInfo
		if item.ReplyToUser != nil {
			replyTo = &internalv1.CommentUserInfo{UserId: item.ReplyToUser.UserID, Username: item.ReplyToUser.Username, Avatar: item.ReplyToUser.Avatar}
		}
		items = append(items, &internalv1.ReplyCommentItem{
			Id:          item.ID,
			ArticleId:   item.ArticleID,
			RootId:      item.RootID,
			User:        user,
			ReplyToUser: replyTo,
			Content:     item.Content,
			CreatedTime: item.CreatedTime,
			Status:      int32(item.Status),
			Ip:          item.IP,
			LikeCount:   item.LikeCount,
		})
	}
	return &internalv1.ReplyList{
		Items:    items,
		Total:    resp.Total,
		LastId:   resp.LastID,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}

func (s *CommunityServer) DeleteComment(ctx context.Context, req *internalv1.DeleteCommentRequest) (*internalv1.Empty, error) {
	if err := s.app.DeleteComment(ctx, req.CommentId, req.UserId, req.IsAdmin); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *CommunityServer) GetCommentStats(ctx context.Context, req *internalv1.CommentIDRequest) (*internalv1.CommentStats, error) {
	stats, err := s.app.GetCommentStats(ctx, req.CommentId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.CommentStats{CommentId: stats.ID, HotValue: stats.HotCount, LikeCount: stats.LikeCount}, nil
}

func (s *CommunityServer) ArticleLike(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.Empty, error) {
	if err := s.app.ArticleLike(ctx, req.UserId, req.ArticleId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *CommunityServer) ArticleCancelLike(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.Empty, error) {
	if err := s.app.ArticleCancelLike(ctx, req.UserId, req.ArticleId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *CommunityServer) CommentLike(ctx context.Context, req *internalv1.CommentUserRequest) (*internalv1.Empty, error) {
	if err := s.app.CommentLike(ctx, req.UserId, req.CommentId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *CommunityServer) CommentCancelLike(ctx context.Context, req *internalv1.CommentUserRequest) (*internalv1.Empty, error) {
	if err := s.app.CommentCancelLike(ctx, req.UserId, req.CommentId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *CommunityServer) IsUserLikedArticle(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.LikeState, error) {
	liked, err := s.app.IsUserLikedArticle(ctx, req.UserId, req.ArticleId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.LikeState{Liked: liked}, nil
}

func (s *CommunityServer) GetHotRank(ctx context.Context, req *internalv1.HotRankRequest) (*internalv1.HotRank, error) {
	rank, err := s.app.GetHotRank(ctx)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	items := make([]*internalv1.HotRankItem, 0, len(*rank.List))
	for _, item := range *rank.List {
		items = append(items, &internalv1.HotRankItem{
			ArticleId:    item.ArticleID,
			Title:        item.Title,
			Hot:          item.Hot,
			ViewCount:    item.ViewCount,
			CommentCount: item.CommentCount,
			LikeCount:    item.LikeCount,
		})
	}
	return &internalv1.HotRank{Items: items}, nil
}

func (s *CommunityServer) RebuildHotRank(ctx context.Context, _ *internalv1.Empty) (*internalv1.Empty, error) {
	if err := s.app.RebuildHotRank(ctx); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *CommunityServer) SendViewHistory(ctx context.Context, req *internalv1.ViewHistoryRequest) (*internalv1.Empty, error) {
	if err := s.app.SendViewHistory(ctx, req.UserId, req.ArticleId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *CommunityServer) GetNotifications(ctx context.Context, req *internalv1.NotificationListRequest) (*internalv1.NotificationList, error) {
	resp, err := s.app.GetMyNotifications(ctx, req.UserId, req.Page, req.PageSize)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	items := make([]*internalv1.NotificationItem, 0, len(resp.List))
	for _, item := range resp.List {
		items = append(items, &internalv1.NotificationItem{
			Id:             item.ID,
			Type:           int32(item.Type),
			IsRead:         item.IsRead,
			CreatedTime:    item.CreatedTime,
			SenderId:       item.SenderID,
			SenderNickname: item.SenderNickname,
			SenderAvatar:   item.SenderAvatar,
			ActionText:     item.ActionText,
			ArticleId:      item.ArticleID,
			Title:          item.Title,
		})
	}
	return &internalv1.NotificationList{Items: items, Page: resp.Page, PageSize: resp.PageSize}, nil
}

func (s *CommunityServer) GetUnreadCount(ctx context.Context, req *internalv1.UserIDRequest) (*internalv1.UnreadCount, error) {
	count, err := s.app.GetUnreadCount(ctx, req.UserId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.UnreadCount{Count: count}, nil
}

func (s *CommunityServer) ClearUnread(ctx context.Context, req *internalv1.UserIDRequest) (*internalv1.Empty, error) {
	if err := s.app.ClearUnread(ctx, req.UserId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

var (
	_ articledto.HotRankItem
	_ commentdto.CreateCommentResponse
	_ notificationdto.NotificationListResponse
)
