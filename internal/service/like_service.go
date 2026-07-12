package service

import (
	"blog/internal/consts"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type LikeService struct {
	articleLikeRepo repository.ArticleLikeRepository
	commentLikeRepo repository.CommentLikeRepository
	rdb             *redis.Client
}

func NewLikeService(articleLikeRepo repository.ArticleLikeRepository, commentLikeRepo repository.CommentLikeRepository, rdb *redis.Client) *LikeService {
	return &LikeService{
		articleLikeRepo: articleLikeRepo,
		commentLikeRepo: commentLikeRepo,
		rdb:             rdb,
	}
}

// ====================== Redis Key 拼接辅助函数 ======================

func (s *LikeService) getArticleStatusKey(articleID uint64) string {
	return consts.KeyArticleLikeStatePrefix + strconv.FormatUint(articleID, 10)
}

func (s *LikeService) getArticleCountKey(articleID uint64) string {
	return consts.KeyArticleLikeCountPrefix + strconv.FormatUint(articleID, 10)
}

func (s *LikeService) getArticleLockKey(articleID uint64) string {
	return consts.KeyArticleLikeLockPrefix + strconv.FormatUint(articleID, 10)
}

func (s *LikeService) getCommentStatusKey(commentID uint64) string {
	return consts.KeyCommentLikeStatePrefix + strconv.FormatUint(commentID, 10)
}

func (s *LikeService) getCommentCountKey(commentID uint64) string {
	return consts.KeyCommentLikeCountPrefix + strconv.FormatUint(commentID, 10)
}

func (s *LikeService) getCommentLockKey(commentID uint64) string {
	return consts.KeyCommentLikeLockPrefix + strconv.FormatUint(commentID, 10)
}

// ====================== 核心抽离：仿照 CommentService 分流模式的通用底层函数 ======================

func (s *LikeService) doEnsureCacheExists(ctx context.Context, statusKey, countKey, lockKey string, selfFunc func() error, loadDB func() (map[string]string, int64, error)) error {
	exists, err := s.rdb.Exists(ctx, statusKey).Result()
	if err != nil || exists > 0 {
		return err
	}

	success, err := s.rdb.SetNX(ctx, lockKey, "1", 3*time.Second).Result()
	if err != nil {
		return err
	}
	if !success {
		time.Sleep(50 * time.Millisecond)
		return selfFunc()
	}
	defer s.rdb.Del(ctx, lockKey)

	exists, err = s.rdb.Exists(ctx, statusKey).Result()
	if err != nil || exists > 0 {
		return err
	}

	fields, totalCount, err := loadDB()
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()
	if len(fields) == 0 {
		pipe.HSet(ctx, statusKey, "placeholder", "0")
		pipe.Set(ctx, countKey, 0, 0)
	} else {
		pipe.HSet(ctx, statusKey, fields)
		pipe.Set(ctx, countKey, totalCount, 0)
	}

	pipe.Expire(ctx, statusKey, 10*24*time.Hour)
	pipe.Expire(ctx, countKey, 10*24*time.Hour)

	_, err = pipe.Exec(ctx)
	return err
}

func (s *LikeService) doLike(ctx context.Context, ensureCache func() error, statusKey, countKey string, userID uint64) error {
	if err := ensureCache(); err != nil {
		return err
	}

	field := strconv.FormatUint(userID, 10)
	val, err := s.rdb.HGet(ctx, statusKey, field).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if val == "1" {
		return nil
	}

	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, statusKey, field, "1")
	pipe.Incr(ctx, countKey)
	pipe.Expire(ctx, statusKey, 10*24*time.Hour)
	pipe.Expire(ctx, countKey, 10*24*time.Hour)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *LikeService) doCancelLike(ctx context.Context, ensureCache func() error, statusKey, countKey string, userID uint64) error {
	if err := ensureCache(); err != nil {
		return err
	}

	field := strconv.FormatUint(userID, 10)
	val, err := s.rdb.HGet(ctx, statusKey, field).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == redis.Nil || val == "2" {
		return nil
	}

	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, statusKey, field, "2")
	pipe.Decr(ctx, countKey)
	pipe.Expire(ctx, statusKey, 10*24*time.Hour)
	pipe.Expire(ctx, countKey, 10*24*time.Hour)
	_, err = pipe.Exec(ctx)
	return err
}

// ====================== 对外暴露的文章点赞业务入口 ======================

func (s *LikeService) ensureArticleCacheExists(ctx context.Context, articleID uint64) error {
	statusKey := s.getArticleStatusKey(articleID)
	countKey := s.getArticleCountKey(articleID)
	lockKey := s.getArticleLockKey(articleID)

	return s.doEnsureCacheExists(ctx, statusKey, countKey, lockKey, func() error {
		return s.ensureArticleCacheExists(ctx, articleID)
	}, func() (map[string]string, int64, error) {
		records, err := s.articleLikeRepo.GetDB().WithContext(ctx).Where("article_id = ?", articleID).Find(&model.ArticleLike{}).Rows()
		if err != nil {
			return nil, 0, err
		}
		defer records.Close()

		fields := make(map[string]string)
		for records.Next() {
			var r model.ArticleLike
			if err := s.articleLikeRepo.GetDB().ScanRows(records, &r); err == nil {
				fields[strconv.FormatUint(r.UserID, 10)] = strconv.Itoa(int(r.Status))
			}
		}

		var totalCount int64
		err = s.articleLikeRepo.GetDB().WithContext(ctx).Model(&model.ArticleLike{}).Where("article_id = ? AND status = ?", articleID, 1).Count(&totalCount).Error

		return fields, totalCount, err
	})
}

func (s *LikeService) ArticleLike(ctx context.Context, userID, articleID uint64) error {
	return s.doLike(ctx, func() error {
		return s.ensureArticleCacheExists(ctx, articleID)
	}, s.getArticleStatusKey(articleID), s.getArticleCountKey(articleID), userID)
}

func (s *LikeService) ArticleCancelLike(ctx context.Context, userID, articleID uint64) error {
	return s.doCancelLike(ctx, func() error {
		return s.ensureArticleCacheExists(ctx, articleID)
	}, s.getArticleStatusKey(articleID), s.getArticleCountKey(articleID), userID)
}

func (s *LikeService) GetArticleLikeCount(ctx context.Context, articleID uint64) (uint64, error) {
	if err := s.ensureArticleCacheExists(ctx, articleID); err != nil {
		return 0, err
	}
	countKey := s.getArticleCountKey(articleID)
	val, err := s.rdb.Get(ctx, countKey).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	count, _ := strconv.ParseUint(val, 10, 64)
	return count, nil
}

// 💡 替换目标：供详情页调用，高效判断当前登录用户是否点赞过该文章
func (s *LikeService) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	if err := s.ensureArticleCacheExists(ctx, articleID); err != nil {
		return false, err
	}
	statusKey := s.getArticleStatusKey(articleID)
	field := strconv.FormatUint(userID, 10)
	val, err := s.rdb.HGet(ctx, statusKey, field).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

// ====================== 对外暴露的评论点赞业务入口 ======================

func (s *LikeService) ensureCommentCacheExists(ctx context.Context, commentID uint64) error {
	statusKey := s.getCommentStatusKey(commentID)
	countKey := s.getCommentCountKey(commentID)
	lockKey := s.getCommentLockKey(commentID)

	return s.doEnsureCacheExists(ctx, statusKey, countKey, lockKey, func() error {
		return s.ensureCommentCacheExists(ctx, commentID)
	}, func() (map[string]string, int64, error) {
		records, err := s.commentLikeRepo.GetDB().WithContext(ctx).Where("comment_id = ?", commentID).Find(&model.CommentLike{}).Rows()
		if err != nil {
			return nil, 0, err
		}
		defer records.Close()

		fields := make(map[string]string)
		for records.Next() {
			var r model.CommentLike
			if err := s.commentLikeRepo.GetDB().ScanRows(records, &r); err == nil {
				fields[strconv.FormatUint(r.UserID, 10)] = strconv.Itoa(int(r.Status))
			}
		}

		var totalCount int64
		err = s.commentLikeRepo.GetDB().WithContext(ctx).Model(&model.CommentLike{}).Where("comment_id = ? AND status = ?", commentID, 1).Count(&totalCount).Error

		return fields, totalCount, err
	})
}

func (s *LikeService) CommentLike(ctx context.Context, userID, commentID uint64) error {
	return s.doLike(ctx, func() error {
		return s.ensureCommentCacheExists(ctx, commentID)
	}, s.getCommentStatusKey(commentID), s.getCommentCountKey(commentID), userID)
}

func (s *LikeService) CommentCancelLike(ctx context.Context, userID, commentID uint64) error {
	return s.doCancelLike(ctx, func() error {
		return s.ensureCommentCacheExists(ctx, commentID)
	}, s.getCommentStatusKey(commentID), s.getCommentCountKey(commentID), userID)
}

// 💡 镜像对齐：未来如果评论详情或列表需要展现用户点赞状态，可用此方法判定
func (s *LikeService) IsUserLikedComment(ctx context.Context, userID, commentID uint64) (bool, error) {
	if err := s.ensureCommentCacheExists(ctx, commentID); err != nil {
		return false, err
	}
	statusKey := s.getCommentStatusKey(commentID)
	field := strconv.FormatUint(userID, 10)
	val, err := s.rdb.HGet(ctx, statusKey, field).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

// ====================== 定时离线刷盘异步写回 MySQL 逻辑 ======================

func (s *LikeService) SyncArticleLikesToDB(ctx context.Context) error {
	cursor := uint64(0)
	pattern := consts.KeyArticleLikeStatePrefix + "*"

	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		for _, key := range keys {
			articleIDStr := key[len(consts.KeyArticleLikeStatePrefix):]
			articleID, _ := strconv.ParseUint(articleIDStr, 10, 64)

			userStates, err := s.rdb.HGetAll(ctx, key).Result()
			if err != nil {
				continue
			}

			var hasActivity bool
			for userIDStr, statusStr := range userStates {
				if userIDStr == "placeholder" {
					continue
				}
				userID, _ := strconv.ParseUint(userIDStr, 10, 64)
				redisStatus, _ := strconv.Atoi(statusStr)

				dbStatus, err := s.articleLikeRepo.GetLikeStatus(ctx, userID, articleID)
				if err != nil {
					if redisStatus == 0 || redisStatus == 2 {
						continue
					}
					likeRecord := &model.ArticleLike{
						UserID:    userID,
						ArticleID: articleID,
						Status:    int8(redisStatus),
					}
					_ = s.articleLikeRepo.Insert(ctx, likeRecord)
					hasActivity = true
				} else {
					if dbStatus != int8(redisStatus) {
						_ = s.articleLikeRepo.Update(ctx, userID, articleID, int8(redisStatus))
						hasActivity = true
					}
				}
			}

			countKey := s.getArticleCountKey(articleID)
			if hasActivity || len(userStates) > 3 {
				val, err := s.rdb.Get(ctx, countKey).Result()
				if err == nil {
					count, _ := strconv.ParseInt(val, 10, 64)
					_ = s.articleLikeRepo.GetDB().WithContext(ctx).
						Table("articles").
						Where("id = ?", articleID).
						Update("like_count", count).Error
				}
			} else {
				s.rdb.Del(ctx, key, countKey)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (s *LikeService) SyncCommentLikesToDB(ctx context.Context) error {
	cursor := uint64(0)
	pattern := consts.KeyCommentLikeStatePrefix + "*"

	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		for _, key := range keys {
			commentIDStr := key[len(consts.KeyCommentLikeStatePrefix):]
			commentID, _ := strconv.ParseUint(commentIDStr, 10, 64)

			userStates, err := s.rdb.HGetAll(ctx, key).Result()
			if err != nil {
				continue
			}

			var hasActivity bool
			for userIDStr, statusStr := range userStates {
				if userIDStr == "placeholder" {
					continue
				}
				userID, _ := strconv.ParseUint(userIDStr, 10, 64)
				redisStatus, _ := strconv.Atoi(statusStr)

				dbStatus, err := s.commentLikeRepo.GetLikeStatus(ctx, userID, commentID)
				if err != nil {
					if redisStatus == 0 || redisStatus == 2 {
						continue
					}
					_ = s.commentLikeRepo.Insert(ctx, &model.CommentLike{
						UserID:    userID,
						CommentID: commentID,
						Status:    int8(redisStatus),
					})
					hasActivity = true
				} else {
					if dbStatus != int8(redisStatus) {
						_ = s.commentLikeRepo.Update(ctx, userID, commentID, int8(redisStatus))
						hasActivity = true
					}
				}
			}

			countKey := s.getCommentCountKey(commentID)
			if hasActivity || len(userStates) > 3 {
				val, err := s.rdb.Get(ctx, countKey).Result()
				if err == nil {
					count, _ := strconv.ParseInt(val, 10, 64)
					_ = s.commentLikeRepo.GetDB().WithContext(ctx).
						Table("comments").
						Where("id = ?", commentID).
						Update("like_count", count).Error
				}
			} else {
				s.rdb.Del(ctx, key, countKey)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
