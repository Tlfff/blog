package community

import (
	domaincommunity "blog/internal/domain/community"
	"context"
	"time"
)

// 点赞文章：写入点赞记录、刷新缓存并异步发送点赞通知
func (s *Service) ArticleLike(ctx context.Context, userID, articleID uint64) error {
	// 1. 先查缓存判断是否已点赞，避免重复写库
	liked, err := s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetArticle, articleID, userID)
	if err != nil {
		return err
	}
	// 2. 已点赞则幂等返回
	if liked {
		return nil
	}
	// 3. 写入点赞记录并同步文章点赞数
	if err := s.articleLikes.SetLiked(ctx, userID, articleID, true); err != nil {
		return err
	}
	// 4. 更新点赞缓存，失败只记日志（缓存可由冷启动重建）
	logError("更新文章点赞缓存失败", s.likeCache.Add(ctx, domaincommunity.LikeTargetArticle, articleID, userID))
	// 5. 异步发送点赞通知给文章作者
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

// 取消点赞文章：更新点赞记录并同步清理缓存
func (s *Service) ArticleCancelLike(ctx context.Context, userID, articleID uint64) error {
	// 1. 先查缓存判断是否已点赞
	liked, err := s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetArticle, articleID, userID)
	if err != nil {
		return err
	}
	// 2. 未点赞则无需取消，幂等返回
	if !liked {
		return nil
	}
	// 3. 更新点赞记录并同步文章点赞数
	if err := s.articleLikes.SetLiked(ctx, userID, articleID, false); err != nil {
		return err
	}
	// 4. 从点赞缓存中移除该用户
	logError("更新取消文章点赞缓存失败", s.likeCache.Remove(ctx, domaincommunity.LikeTargetArticle, articleID, userID))
	return nil
}

// 点赞评论：写入点赞记录并刷新缓存
func (s *Service) CommentLike(ctx context.Context, userID, commentID uint64) error {
	// 1. 先查缓存判断是否已点赞
	liked, err := s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetComment, commentID, userID)
	if err != nil {
		return err
	}
	// 2. 已点赞则幂等返回
	if liked {
		return nil
	}
	// 3. 写入点赞记录并同步评论点赞数
	if err := s.commentLikes.SetLiked(ctx, userID, commentID, true); err != nil {
		return err
	}
	// 4. 更新点赞缓存，失败只记日志
	logError("更新评论点赞缓存失败", s.likeCache.Add(ctx, domaincommunity.LikeTargetComment, commentID, userID))
	return nil
}

// 取消点赞评论：更新点赞记录并同步清理缓存
func (s *Service) CommentCancelLike(ctx context.Context, userID, commentID uint64) error {
	// 1. 先查缓存判断是否已点赞
	liked, err := s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetComment, commentID, userID)
	if err != nil {
		return err
	}
	// 2. 未点赞则无需取消，幂等返回
	if !liked {
		return nil
	}
	// 3. 更新点赞记录并同步评论点赞数
	if err := s.commentLikes.SetLiked(ctx, userID, commentID, false); err != nil {
		return err
	}
	// 4. 从点赞缓存中移除该用户
	logError("更新取消评论点赞缓存失败", s.likeCache.Remove(ctx, domaincommunity.LikeTargetComment, commentID, userID))
	return nil
}

// 查询当前用户是否已点赞指定文章，游客直接返回未点赞
func (s *Service) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	// 1. 游客无点赞状态，直接返回 false
	if userID == 0 {
		return false, nil
	}
	return s.likeCache.IsLiked(ctx, domaincommunity.LikeTargetArticle, articleID, userID)
}
