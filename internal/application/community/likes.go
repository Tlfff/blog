package community

import (
	domaincommunity "blog/internal/domain/community"
	"context"
	"time"
)

func (s *Service) ArticleLike(ctx context.Context, userID, articleID uint64) error {
	liked, err := s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetArticle, articleID, userID)
	if err != nil {
		return err
	}
	if liked {
		return nil
	}
	if err := s.articleLikes.SetLiked(ctx, userID, articleID, true); err != nil {
		return err
	}
	logError("更新文章点赞缓存失败", s.likeCache.Add(ctx, domaincommunity.LikeTargetArticle, articleID, userID))
	if s.events != nil {
		logError("发送点赞通知失败", s.events.SendLikeNotification(ctx, domaincommunity.NotificationEvent{
			NotifyType:  domaincommunity.NotifyTypeLikeArticle,
			SenderID:    userID,
			TargetID:    articleID,
			CreatedTime: time.Now(),
		}))
	}
	return nil
}

func (s *Service) ArticleCancelLike(ctx context.Context, userID, articleID uint64) error {
	liked, err := s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetArticle, articleID, userID)
	if err != nil {
		return err
	}
	if !liked {
		return nil
	}
	if err := s.articleLikes.SetLiked(ctx, userID, articleID, false); err != nil {
		return err
	}
	logError("更新取消文章点赞缓存失败", s.likeCache.Remove(ctx, domaincommunity.LikeTargetArticle, articleID, userID))
	return nil
}

func (s *Service) CommentLike(ctx context.Context, userID, commentID uint64) error {
	liked, err := s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetComment, commentID, userID)
	if err != nil {
		return err
	}
	if liked {
		return nil
	}
	if err := s.commentLikes.SetLiked(ctx, userID, commentID, true); err != nil {
		return err
	}
	logError("更新评论点赞缓存失败", s.likeCache.Add(ctx, domaincommunity.LikeTargetComment, commentID, userID))
	return nil
}

func (s *Service) CommentCancelLike(ctx context.Context, userID, commentID uint64) error {
	liked, err := s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetComment, commentID, userID)
	if err != nil {
		return err
	}
	if !liked {
		return nil
	}
	if err := s.commentLikes.SetLiked(ctx, userID, commentID, false); err != nil {
		return err
	}
	logError("更新取消评论点赞缓存失败", s.likeCache.Remove(ctx, domaincommunity.LikeTargetComment, commentID, userID))
	return nil
}

func (s *Service) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	return s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetArticle, articleID, userID)
}
