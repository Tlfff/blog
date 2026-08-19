package client

import (
	articledto "blog/internal/article/application/dto"
	commentdto "blog/internal/comment/application/dto"
	notificationdto "blog/internal/notification/application/dto"
	"blog/internal/shared/common"
	internalv1 "blog/shared/contracts/gen/internalv1"
	platformerrors "blog/shared/platform/errors"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CommunityClient 实现统一入口需要的 Community 用例接口。
type CommunityClient struct {
	client internalv1.CommunityServiceClient
}

func NewCommunityClient(client internalv1.CommunityServiceClient) *CommunityClient {
	return &CommunityClient{client: client}
}

func (c *CommunityClient) CreateComment(ctx context.Context, articleID, rootID, userID, replyToUserID uint64, content, ip string) (*commentdto.CreateCommentResponse, error) {
	resp, err := c.client.CreateComment(ctx, &internalv1.CreateCommentRequest{
		ArticleId:     articleID,
		RootId:        rootID,
		UserId:        userID,
		ReplyToUserId: replyToUserID,
		Content:       content,
		Ip:            ip,
	})
	if err != nil {
		return nil, toCommunityError(err)
	}
	return &commentdto.CreateCommentResponse{ID: resp.Id, CreatedTime: resp.CreatedTime}, nil
}

func (c *CommunityClient) ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) (*commentdto.RootCommentListResponse, error) {
	resp, err := c.client.ListRootComments(ctx, &internalv1.ListRootCommentsRequest{
		ArticleId: articleID,
		LastId:    lastID,
		Page:      uint64(page),
		PageSize:  uint64(pageSize),
		IsDesc:    isDesc,
		AuthorId:  authorID,
	})
	if err != nil {
		return nil, toCommunityError(err)
	}
	list := &commentdto.RootCommentListResponse{
		List:     make([]*commentdto.RootCommentItem, 0, len(resp.Items)),
		Total:    resp.Total,
		LastID:   resp.LastId,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}
	for _, item := range resp.Items {
		user := &commentdto.CommentUserInfo{UserID: item.User.UserId, Username: item.User.Username, Avatar: item.User.Avatar}
		list.List = append(list.List, &commentdto.RootCommentItem{
			ID:          item.Id,
			ArticleID:   item.ArticleId,
			User:        user,
			Content:     item.Content,
			ReplyCount:  item.ReplyCount,
			IP:          item.Ip,
			CreatedTime: item.CreatedTime,
			Status:      int8(item.Status),
			LikeCount:   item.LikeCount,
		})
	}
	return list, nil
}

func (c *CommunityClient) ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) (*commentdto.ReplyListResponse, error) {
	resp, err := c.client.ListReplies(ctx, &internalv1.ListRepliesRequest{
		RootId:   rootID,
		LastId:   lastID,
		Page:     uint64(page),
		PageSize: uint64(pageSize),
	})
	if err != nil {
		return nil, toCommunityError(err)
	}
	list := &commentdto.ReplyListResponse{
		List:     make([]*commentdto.ReplyCommentItem, 0, len(resp.Items)),
		Total:    resp.Total,
		LastID:   resp.LastId,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}
	for _, item := range resp.Items {
		user := &commentdto.CommentUserInfo{UserID: item.User.UserId, Username: item.User.Username, Avatar: item.User.Avatar}
		var replyTo *commentdto.CommentUserInfo
		if item.ReplyToUser != nil {
			replyTo = &commentdto.CommentUserInfo{UserID: item.ReplyToUser.UserId, Username: item.ReplyToUser.Username, Avatar: item.ReplyToUser.Avatar}
		}
		list.List = append(list.List, &commentdto.ReplyCommentItem{
			ID:          item.Id,
			ArticleID:   item.ArticleId,
			RootID:      item.RootId,
			User:        user,
			ReplyToUser: replyTo,
			Content:     item.Content,
			CreatedTime: item.CreatedTime,
			Status:      int8(item.Status),
			IP:          item.Ip,
			LikeCount:   item.LikeCount,
		})
	}
	return list, nil
}

func (c *CommunityClient) DeleteComment(ctx context.Context, commentID, userID uint64, isAdmin bool) error {
	_, err := c.client.DeleteComment(ctx, &internalv1.DeleteCommentRequest{
		CommentId: commentID,
		UserId:    userID,
		IsAdmin:   isAdmin,
	})
	return toCommunityError(err)
}

func (c *CommunityClient) GetCommentStats(ctx context.Context, commentID uint64) (*commentdto.CommentStatsResponse, error) {
	resp, err := c.client.GetCommentStats(ctx, &internalv1.CommentIDRequest{CommentId: commentID})
	if err != nil {
		return nil, toCommunityError(err)
	}
	return commentdto.NewCommentStatsResponse(resp.CommentId, resp.HotValue, resp.LikeCount), nil
}

func (c *CommunityClient) ArticleLike(ctx context.Context, userID, articleID uint64) error {
	_, err := c.client.ArticleLike(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	return toCommunityError(err)
}

func (c *CommunityClient) ArticleCancelLike(ctx context.Context, userID, articleID uint64) error {
	_, err := c.client.ArticleCancelLike(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	return toCommunityError(err)
}

func (c *CommunityClient) CommentLike(ctx context.Context, userID, commentID uint64) error {
	_, err := c.client.CommentLike(ctx, &internalv1.CommentUserRequest{CommentId: commentID, UserId: userID})
	return toCommunityError(err)
}

func (c *CommunityClient) CommentCancelLike(ctx context.Context, userID, commentID uint64) error {
	_, err := c.client.CommentCancelLike(ctx, &internalv1.CommentUserRequest{CommentId: commentID, UserId: userID})
	return toCommunityError(err)
}

func (c *CommunityClient) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	resp, err := c.client.IsUserLikedArticle(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	if err != nil {
		return false, toCommunityError(err)
	}
	return resp.Liked, nil
}

func (c *CommunityClient) GetHotRank(ctx context.Context) (*articledto.HotRankResponse, error) {
	resp, err := c.client.GetHotRank(ctx, &internalv1.HotRankRequest{Limit: 10})
	if err != nil {
		return nil, toCommunityError(err)
	}
	items := make([]articledto.HotRankItem, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, articledto.HotRankItem{
			ArticleID:    item.ArticleId,
			Title:        item.Title,
			Hot:          item.Hot,
			ViewCount:    item.ViewCount,
			CommentCount: item.CommentCount,
			LikeCount:    item.LikeCount,
		})
	}
	return articledto.NewHotRankResponse(items), nil
}

func (c *CommunityClient) RebuildHotRank(ctx context.Context) error {
	_, err := c.client.RebuildHotRank(ctx, &internalv1.Empty{})
	return toCommunityError(err)
}

func (c *CommunityClient) SendViewHistory(ctx context.Context, userID, articleID uint64) error {
	_, err := c.client.SendViewHistory(ctx, &internalv1.ViewHistoryRequest{ArticleId: articleID, UserId: userID})
	return toCommunityError(err)
}

func (c *CommunityClient) GetMyNotifications(ctx context.Context, userID uint64, page, pageSize int64) (*notificationdto.NotificationListResponse, error) {
	resp, err := c.client.GetNotifications(ctx, &internalv1.NotificationListRequest{
		UserId:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, toCommunityError(err)
	}
	list := &notificationdto.NotificationListResponse{
		List:     make([]*notificationdto.NotifyListItem, 0, len(resp.Items)),
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}
	for _, item := range resp.Items {
		list.List = append(list.List, &notificationdto.NotifyListItem{
			ID:             item.Id,
			Type:           int8(item.Type),
			IsRead:         item.IsRead,
			CreatedTime:    item.CreatedTime,
			SenderID:       item.SenderId,
			SenderNickname: item.SenderNickname,
			SenderAvatar:   item.SenderAvatar,
			ActionText:     item.ActionText,
			ArticleID:      item.ArticleId,
			Title:          item.Title,
		})
	}
	return list, nil
}

func (c *CommunityClient) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	resp, err := c.client.GetUnreadCount(ctx, &internalv1.UserIDRequest{UserId: userID})
	if err != nil {
		return 0, toCommunityError(err)
	}
	return resp.Count, nil
}

func (c *CommunityClient) ClearUnread(ctx context.Context, userID uint64) error {
	_, err := c.client.ClearUnread(ctx, &internalv1.UserIDRequest{UserId: userID})
	return toCommunityError(err)
}

func toCommunityError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
		return common.ErrCommentNotFound
	}
	switch platformerrors.FromGRPC(err) {
	case common.CodeForbidden:
		return common.ErrForbidden
	case common.CodeUnauthorized:
		return common.ErrTokenInvalid
	case common.CodeInvalidParameter:
		return common.ErrParameter
	default:
		return common.ErrSystem
	}
}
