package app

import (
	"blog/internal/like/domain"
	"context"
	"log"
	"time"
)

// Service 编排文章和评论点赞用例。
type Service struct {
	articles    domain.ArticleLikeRepository // 文章点赞关系 Repository
	comments    domain.CommentLikeRepository // 评论点赞关系 Repository
	cache       domain.LikeCache             // 点赞状态缓存
	events      domain.EventPublisher        // 当前文章点赞通知发布器
	projections domain.ProjectionUpdater     // 目标点赞数写入 Port
	tx          TransactionManager           // 本地事务协调 Port
}

// NewService 创建 Like 上下文应用服务。
func NewService(
	articles domain.ArticleLikeRepository,
	comments domain.CommentLikeRepository,
	cache domain.LikeCache,
	events domain.EventPublisher,
	projections domain.ProjectionUpdater,
	tx TransactionManager,
) *Service {
	return &Service{
		articles: articles, comments: comments, cache: cache,
		events: events, projections: projections, tx: tx,
	}
}

// ArticleLike 点赞文章，重复点赞按成功处理。
func (s *Service) ArticleLike(ctx context.Context, userID, articleID uint64) error {
	// 1. 查询当前点赞状态，重复点赞直接成功
	liked, err := s.isLiked(ctx, domain.LikeTargetArticle, articleID, userID)
	if err != nil || liked {
		return err
	}

	// 2. 在事务中更新点赞关系和文章点赞数
	changed, err := s.setLiked(ctx, domain.LikeTargetArticle, articleID, userID, true)
	if err != nil || !changed {
		return err
	}
	// 3. 数据库成功后更新点赞缓存
	if s.cache != nil {
		if err := s.cache.Add(ctx, domain.LikeTargetArticle, articleID, userID); err != nil {
			log.Printf("更新文章点赞缓存失败,article_id:%d,user_id:%d,err:%v", articleID, userID, err)
		}
	}
	// 4. 缓存更新后发布文章点赞通知
	if s.events != nil {
		if err := s.events.PublishLikeCreated(ctx, domain.LikeCreated{
			UserID: userID, Target: domain.LikeTargetArticle, TargetID: articleID, OccurredAt: time.Now(),
		}); err != nil {
			log.Printf("发布文章点赞通知失败,article_id:%d,user_id:%d,err:%v", articleID, userID, err)
		}
	}
	return nil
}

// ArticleCancelLike 取消文章点赞，重复取消按成功处理。
func (s *Service) ArticleCancelLike(ctx context.Context, userID, articleID uint64) error {
	// 1. 查询当前点赞状态，重复取消直接成功
	liked, err := s.isLiked(ctx, domain.LikeTargetArticle, articleID, userID)
	if err != nil || !liked {
		return err
	}
	// 2. 在事务中更新点赞关系和文章点赞数
	changed, err := s.setLiked(ctx, domain.LikeTargetArticle, articleID, userID, false)
	if err != nil || !changed {
		return err
	}
	// 3. 数据库成功后移除点赞缓存
	if s.cache != nil {
		if err := s.cache.Remove(ctx, domain.LikeTargetArticle, articleID, userID); err != nil {
			log.Printf("更新取消文章点赞缓存失败,article_id:%d,user_id:%d,err:%v", articleID, userID, err)
		}
	}
	return nil
}

// CommentLike 点赞评论，重复点赞按成功处理。
func (s *Service) CommentLike(ctx context.Context, userID, commentID uint64) error {
	// 1. 查询当前点赞状态，重复点赞直接成功
	liked, err := s.isLiked(ctx, domain.LikeTargetComment, commentID, userID)
	if err != nil || liked {
		return err
	}
	// 2. 在事务中更新点赞关系和评论点赞数
	changed, err := s.setLiked(ctx, domain.LikeTargetComment, commentID, userID, true)
	if err != nil || !changed {
		return err
	}
	// 3. 数据库成功后更新点赞缓存
	if s.cache != nil {
		if err := s.cache.Add(ctx, domain.LikeTargetComment, commentID, userID); err != nil {
			log.Printf("更新评论点赞缓存失败,comment_id:%d,user_id:%d,err:%v", commentID, userID, err)
		}
	}
	return nil
}

// CommentCancelLike 取消评论点赞，重复取消按成功处理。
func (s *Service) CommentCancelLike(ctx context.Context, userID, commentID uint64) error {
	// 1. 查询当前点赞状态，重复取消直接成功
	liked, err := s.isLiked(ctx, domain.LikeTargetComment, commentID, userID)
	if err != nil || !liked {
		return err
	}
	// 2. 在事务中更新点赞关系和评论点赞数
	changed, err := s.setLiked(ctx, domain.LikeTargetComment, commentID, userID, false)
	if err != nil || !changed {
		return err
	}
	// 3. 数据库成功后移除点赞缓存
	if s.cache != nil {
		if err := s.cache.Remove(ctx, domain.LikeTargetComment, commentID, userID); err != nil {
			log.Printf("更新取消评论点赞缓存失败,comment_id:%d,user_id:%d,err:%v", commentID, userID, err)
		}
	}
	return nil
}

// IsUserLikedArticle 查询用户是否点赞文章。
func (s *Service) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	return s.isLiked(ctx, domain.LikeTargetArticle, articleID, userID)
}

// isLiked 从缓存或 Repository 查询点赞状态。
func (s *Service) isLiked(ctx context.Context, target domain.LikeTarget, targetID, userID uint64) (bool, error) {
	if s.cache != nil {
		return s.cache.IsLiked(ctx, target, targetID, userID)
	}
	if target == domain.LikeTargetArticle {
		return s.articles.IsLiked(ctx, userID, targetID)
	}
	return s.comments.IsLiked(ctx, userID, targetID)
}

// setLiked 在同一事务内更新点赞关系和目标点赞数。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - target：点赞目标类型。
//   - targetID：点赞目标唯一标识。
//   - userID：操作用户唯一标识。
//   - liked：目标点赞状态，true 表示点赞，false 表示取消。
func (s *Service) setLiked(ctx context.Context, target domain.LikeTarget, targetID, userID uint64, liked bool) (bool, error) {
	// 1. 构造需要在事务内执行的点赞写入逻辑
	changed := false
	operation := func(txCtx context.Context) error {
		var err error
		if target == domain.LikeTargetArticle {
			changed, err = s.articles.SetLiked(txCtx, userID, targetID, liked)
		} else {
			changed, err = s.comments.SetLiked(txCtx, userID, targetID, liked)
		}
		if err != nil || !changed || s.projections == nil {
			return err
		}
		delta := int64(1)
		if !liked {
			delta = -1
		}
		return s.projections.ApplyLikeDelta(txCtx, target, targetID, delta)
	}
	// 2. 优先通过事务协调器执行写入
	if s.tx != nil {
		return changed, s.tx.WithinTransaction(ctx, operation)
	}
	return changed, operation(ctx)
}
