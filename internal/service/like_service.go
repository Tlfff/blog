package service

import (
	"blog/internal/consts"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/util/cache"
	"context"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type LikeService struct {
	articleLikeRepo repository.ArticleLikeRepository
	commentLikeRepo repository.CommentLikeRepository
	commentRepo     repository.CommentRepository
	rdb             *redis.Client
}

func NewLikeService(
	articleLikeRepo repository.ArticleLikeRepository,
	commentLikeRepo repository.CommentLikeRepository,
	commentRepo repository.CommentRepository,
	rdb *redis.Client,
) *LikeService {
	return &LikeService{
		articleLikeRepo: articleLikeRepo,
		commentLikeRepo: commentLikeRepo,
		commentRepo:     commentRepo,
		rdb:             rdb,
	}
}

// ====================== Redis Key 辅助函数 ======================
func (s *LikeService) getArticleStatusKey(articleID uint64) string {
	return consts.KeyArticleLikeStatePrefix + strconv.FormatUint(articleID, 10)
}

func (s *LikeService) getArticleInfo(articleID uint64) string {
	return consts.KeyArticleInfoHashPrefix + strconv.FormatUint(articleID, 10)
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

// ---------------------- 底层缓存加载辅助函数 ----------------------

// 读取文章点赞数据库数据
func (s *LikeService) loadArticleLikeDBData(ctx context.Context, articleID uint64) (map[string]string, error) {
	// 1. 初始化Map,用于存储用户点赞状态
	fields := make(map[string]string)
	// 2. 从数据库中捞出该文章下的所有用户点赞记录（包含点赞和取消点赞的历史记录）
	records, err := s.articleLikeRepo.GetLikesByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		// 3. 将 UserID 转为 string 作为 Hash 的 Field，Status 转为 string 作为 Value
		fields[strconv.FormatUint(r.UserID, 10)] = strconv.Itoa(int(r.Status))
	}
	// // 4.统计文章获得的点赞总数
	// totalCount, err := s.articleLikeRepo.CountValidByID(ctx, articleID)
	return fields, err
}

// 读取评论点赞数据库数据
func (s *LikeService) loadCommentLikeDBData(ctx context.Context, commentID uint64) (map[string]string, error) {
	// 1. 初始化Map,用于存储用户点赞状态
	fields := make(map[string]string)
	// 2. 从数据库中捞出该评论下的所有用户点赞记录（包含点赞和取消点赞的历史记录）
	records, err := s.commentLikeRepo.GetLikesByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		// 3. 将 UserID 转为 string 作为 Hash 的 Field，Status 转为 string 作为 Value
		fields[strconv.FormatUint(r.UserID, 10)] = strconv.Itoa(int(r.Status))
	}
	// // 4.统计评论获得的点赞总数
	// totalCount, err := s.commentLikeRepo.CountValidByCommentID(ctx, commentID)
	return fields, err
}

// 确定点赞状态缓存存在模版
// loadDataFunc: 加载DB原始状态数据的回调函数
func (s *LikeService) doEnsureLikeStatusCacheExists(ctx context.Context, statusKey string, lockKey string, id uint64, loadDataFunc func(ctx context.Context, id uint64) (map[string]string, error)) error {
	// 将锁唯一标识转为10进制
	lockValue := strconv.FormatUint(id, 10)

	// 1. 检查状态缓存是否存在函数
	checkExists := func(ctx context.Context) (bool, error) {
		// 是否存在该key
		exists, err := s.rdb.Exists(ctx, statusKey).Result()
		return exists > 0, err
	}

	// 2. 初始化回调：加载数据并写入状态Hash
	initCache := func(ctx context.Context) error {
		// 3.调用传入的底层函数获取数据
		fields, err := loadDataFunc(ctx, id)
		if err != nil {
			return err
		}
		// 开启 Redis 的管道机制，可以把所有命令打包，一次性发送给 Redis 执行
		pipe := s.rdb.Pipeline()
		// 4.如果返回数据为0，则缓存个占位符，表示查询过了，下次再请求则不会查询数据库。避免缓存穿透。有数据则缓存到对应key
		if len(fields) == 0 {
			pipe.HSet(ctx, statusKey, "placeholder", "0")
		} else {
			pipe.HSet(ctx, statusKey, fields)
		}
		// 5. 设置过期时间
		pipe.Expire(ctx, statusKey, consts.ExpirePeriod)
		// 6. 把上面打包好的 HSet 和 Expire 命令一起发给 Redis 真正执行，并返回最终结果
		_, err = pipe.Exec(ctx)
		return err
	}

	// 调用全局双重检查初始化工具
	return cache.DoubleCheckInitCache(
		ctx,
		s.rdb,
		statusKey,
		lockKey,
		lockValue,
		consts.LockExpire,
		consts.RetryCount,
		consts.RetryDelay,
		checkExists,
		initCache,
	)
}

// 初始化点赞文章的hash缓存
func (s *LikeService) doEnsureArticleCacheExists(ctx context.Context, articleID uint64) error {
	// 1. 初始化key，记录用户操作hash和锁string
	statusKey := s.getArticleStatusKey(articleID)
	lockKey := s.getArticleLockKey(articleID)

	// 2. 加载文章对应数据的底层函数
	loadArticleData := func(ctx context.Context, id uint64) (map[string]string, error) {
		fields, err := s.loadArticleLikeDBData(ctx, id)
		return fields, err
	}

	// 3. 调用初始化模版
	return s.doEnsureLikeStatusCacheExists(ctx, statusKey, lockKey, articleID, loadArticleData)
}

// 初始化点赞评论的hash缓存
func (s *LikeService) doEnsureCommentCacheExists(ctx context.Context, commentID uint64) error {
	// 1. 初始化key，记录用户操作hash和锁string
	statusKey := s.getCommentStatusKey(commentID)
	lockKey := s.getCommentLockKey(commentID)

	// 2. 评论加载回调函数
	loadCommentData := func(ctx context.Context, id uint64) (map[string]string, error) {
		fields, err := s.loadCommentLikeDBData(ctx, id)
		return fields, err
	}

	return s.doEnsureLikeStatusCacheExists(ctx, statusKey, lockKey, commentID, loadCommentData)
}

// 文章点赞，更新redis中 info hash的点赞数，以及用户点赞hash的状态
func (s *LikeService) doArticleLike(ctx context.Context, statusKey, articleHashKey string, userID uint64) error {
	// 1.查询用户原来的点赞状态
	field := strconv.FormatUint(userID, 10)
	val, err := s.rdb.HGet(ctx, statusKey, field).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	// 2. 如果已经点赞过了，直接返回
	if val == "1" {
		return nil
	}
	// 3. 开启管道，将多个命令打包成一次请求io发送
	pipe := s.rdb.Pipeline()
	// 3.1 将用户操作设为1-点赞,同时给key续期
	pipe.HSet(ctx, statusKey, field, "1")
	pipe.Expire(ctx, statusKey, 10*24*time.Hour)
	// 3.2 增加对应的文章点赞计数，同时给key续期
	pipe.HIncrBy(ctx, articleHashKey, consts.FeildArticleLikeCount, 1)
	pipe.Expire(ctx, articleHashKey, 10*24*time.Hour)
	// 4. 将命令打包，redis串行执行
	_, err = pipe.Exec(ctx)
	return err
}

// 取消文章点赞，更新redis中 info hash的点赞数，以及用户点赞hash的状态
func (s *LikeService) doArticleCancelLike(ctx context.Context, statusKey, articleHashKey string, userID uint64) error {
	// 1.查询用户原来的点赞状态
	field := strconv.FormatUint(userID, 10)
	val, err := s.rdb.HGet(ctx, statusKey, field).Result()
	// 如果是网络、系统问题
	if err != nil && err != redis.Nil {
		return err
	}
	// 2. 如果已经点赞过了，直接返回。redis.nil表示缓存未命中。能取消点赞说明一定有缓存在redis中，没有说明这次操作非法。
	if err == redis.Nil || val == "2" {
		return nil
	}
	// 3. 开启管道，将多个命令打包成一次请求io发送
	pipe := s.rdb.Pipeline()
	// 3.1 将用户操作设为2-取消点赞,同时给key续期
	pipe.HSet(ctx, statusKey, field, "2")
	pipe.Expire(ctx, statusKey, 10*24*time.Hour)
	// 3.2 减少对应的文章点赞计数，同时给key续期
	pipe.HIncrBy(ctx, articleHashKey, consts.FeildArticleLikeCount, -1)
	pipe.Expire(ctx, articleHashKey, 10*24*time.Hour)
	// 4. 将命令打包，redis串行执行
	_, err = pipe.Exec(ctx)
	return err
}

// 评论点赞，更新redis中 info hash 的点赞数，以及用户点赞hash的状态
func (s *LikeService) doCommentLike(ctx context.Context, statusKey, articleHashKey string, userID uint64) error {
	// 1.查询用户原来的点赞状态
	field := strconv.FormatUint(userID, 10)
	val, err := s.rdb.HGet(ctx, statusKey, field).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	// 2. 如果已经点赞过了，直接返回
	if val == "1" {
		return nil
	}
	// 3. 开启管道，将多个命令打包成一次请求io发送
	pipe := s.rdb.Pipeline()
	// 3.1 将用户操作设为1-点赞
	pipe.HSet(ctx, statusKey, field, "1")
	pipe.Expire(ctx, statusKey, 10*24*time.Hour)
	// 3.2 增加对应的文章点赞计数，同时给key续期
	pipe.HIncrBy(ctx, articleHashKey, consts.FeildArticleCommentCount, 1)
	pipe.Expire(ctx, articleHashKey, 10*24*time.Hour)
	// 4. 执行管道命令
	_, err = pipe.Exec(ctx)
	return err
}

// 取消评论点赞，更新redis中 info hash的点赞数，以及用户点赞hash的状态
func (s *LikeService) doCommentCancelLike(ctx context.Context, statusKey, articleHashKey string, userID uint64) error {
	// 1.查询用户原来的点赞状态
	field := strconv.FormatUint(userID, 10)
	val, err := s.rdb.HGet(ctx, statusKey, field).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	// 2. 如果已经取消点赞/未点赞，直接返回
	if err == redis.Nil || val == "2" {
		return nil
	}
	// 3. 开启管道，将多个命令打包成一次请求io发送
	pipe := s.rdb.Pipeline()
	// 3.1 将用户操作设为2-取消点赞
	pipe.HSet(ctx, statusKey, field, "2")
	pipe.Expire(ctx, statusKey, 10*24*time.Hour)
	// 3.2 减少评论独立计数
	pipe.HIncrBy(ctx, articleHashKey, consts.FeildArticleCommentCount, -1)
	pipe.Expire(ctx, articleHashKey, 10*24*time.Hour)
	// 4. 执行管道命令
	_, err = pipe.Exec(ctx)
	return err
}

// ----------------------- 文章点赞业务 -----------------------

// 点赞文章
func (s *LikeService) ArticleLike(ctx context.Context, userID, articleID uint64) error {
	// 1. 确保缓存已初始化
	err := s.doEnsureArticleCacheExists(ctx, articleID)
	if err != nil {
		return err
	}
	// 2. 执行点赞缓存更新
	err = s.doArticleLike(ctx, s.getArticleStatusKey(articleID), s.getArticleInfo(articleID), userID)
	if err != nil {
		return err
	}
	// 3. 同步更新全局榜单热度
	s.updateRankZSet(ctx, articleID)
	return nil
}

// 取消文章点赞
func (s *LikeService) ArticleCancelLike(ctx context.Context, userID, articleID uint64) error {
	// 1. 确保缓存已初始化
	err := s.doEnsureArticleCacheExists(ctx, articleID)
	if err != nil {
		return err
	}
	// 2. 执行取消点赞缓存更新
	err = s.doArticleCancelLike(ctx, s.getArticleStatusKey(articleID), s.getArticleInfo(articleID), userID)
	if err != nil {
		return err
	}
	// 3. 同步更新全局榜单热度
	s.updateRankZSet(ctx, articleID)
	return nil
}

// 计算文章热度
func calcHotScore(viewCount, likeCount, commentCount uint32) float64 {
	return float64(1*viewCount + 2*likeCount + 1*commentCount)
}

// 同步更新全局文章热度排行榜（ZSet）
func (s *LikeService) updateRankZSet(ctx context.Context, articleID uint64) {
	// 1.获取文章info的 Hash Key
	hashKey := s.getArticleInfo(articleID)

	// 读取该文章在 Redis Hash 中当前的三个计数器
	res, err := s.rdb.HMGet(ctx, hashKey, "view_count", "like_count", "comment_count").Result()
	if err != nil {
		log.Printf("[RankSync] HMGet读取计数异常 articleID=%d err=%v", articleID, err)
		return
	}

	// 2. 规避 go-redis 底层 interface{} 转 string 转换问题
	var viewCount, likeCount, commentCount uint64
	if res[0] != nil {
		viewCount, _ = strconv.ParseUint(res[0].(string), 10, 32)
	}
	if res[1] != nil {
		likeCount, _ = strconv.ParseUint(res[1].(string), 10, 32)
	}
	if res[2] != nil {
		commentCount, _ = strconv.ParseUint(res[2].(string), 10, 32)
	}

	// 3. 计算最新热度
	hotScore := calcHotScore(uint32(viewCount), uint32(likeCount), uint32(commentCount))
	member := strconv.FormatUint(articleID, 10)

	// 4. 使用 Pipeline 保证更新与裁剪一次性执行
	pipe := s.rdb.Pipeline()

	// 4.1 更新 ZSet 榜单分数
	pipe.ZAdd(ctx, consts.KeyArticleHotRankZSet, redis.Z{
		Score:  hotScore,
		Member: member,
	})

	// 4.2 顺手裁剪掉 100 名开外的冷门文章（防止 ZSet 无限膨胀，规避 BigKey 隐患）
	// 0：分数最低，-101：分数排101的（倒序）
	pipe.ZRemRangeByRank(ctx, consts.KeyArticleHotRankZSet, 0, -101)

	// 5. 提交执行
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Printf("[RankSync] ZAdd/ZRemRange 榜单更新异常 articleID=%d err=%v", articleID, err)
	}
}

// 获取文章点赞总数,从redis的文章info hash中获取,对应field:like_count
func (s *LikeService) GetArticleLikeCount(ctx context.Context, articleID uint64) (uint64, error) {
	// 1. 确保缓存已初始化
	err := s.doEnsureArticleCacheExists(ctx, articleID)
	if err != nil {
		return 0, err
	}
	// 2. 从hash中读取计数
	countKey := s.getArticleInfo(articleID)
	val, err := s.rdb.HGet(ctx, countKey, consts.FeildArticleLikeCount).Result()
	// 如果没有缓存，则返回0，但正常情况不会发生
	if err == redis.Nil {
		return 0, nil
	}
	// 连接、redis错误
	if err != nil {
		return 0, err
	}
	// 3. 将结果转换成int64
	count, _ := strconv.ParseUint(val, 10, 64)
	return count, nil
}

// 查询用户是否点赞文章
func (s *LikeService) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	// 1. 确保缓存已初始化
	err := s.doEnsureArticleCacheExists(ctx, articleID)
	if err != nil {
		return false, err
	}
	// 2. 查询用户点赞状态
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

// ---------------------------- 评论点赞业务 ----------------------------

// 评论点赞
func (s *LikeService) CommentLike(ctx context.Context, userID, commentID uint64) error {
	// 1. 确保缓存已初始化
	err := s.doEnsureCommentCacheExists(ctx, commentID)
	if err != nil {
		return err
	}
	// 获取评论对应的文章id
	articleID, err := s.commentRepo.GetArticleId(ctx, commentID)
	if err != nil {
		return err
	}
	// 2. 执行点赞缓存更新
	err = s.doCommentLike(ctx, s.getCommentStatusKey(commentID), s.getArticleInfo(articleID), userID)
	if err != nil {
		return err
	}
	// 3. 同步更新全局榜单热度
	s.updateRankZSet(ctx, articleID)
	return nil
}

// 取消评论点赞
func (s *LikeService) CommentCancelLike(ctx context.Context, userID, commentID uint64) error {
	// 1. 确保缓存已初始化
	err := s.doEnsureCommentCacheExists(ctx, commentID)
	if err != nil {
		return err
	}
	// 获取评论对应的文章id
	articleID, err := s.commentRepo.GetArticleId(ctx, commentID)
	if err != nil {
		return err
	}
	// 2. 执行取消点赞缓存更新
	err = s.doCommentCancelLike(ctx, s.getCommentStatusKey(commentID), s.getArticleInfo(articleID), userID)
	if err != nil {
		return err
	}
	// 3. 同步更新全局榜单热度
	s.updateRankZSet(ctx, articleID)
	return nil
}

// ---------------------------- 定时刷盘同步 ----------------------------
// 从 Redis Key 中提取数字 ID
func parseID(key string, prefix string) uint64 {
	idStr := key[len(prefix):]
	id, _ := strconv.ParseUint(idStr, 10, 64)
	return id
}

// 专门负责扫描所有用户的点赞文章状态 Hash，批量同步明细到 article_likes 表
// 外层循环处理文章 Key，内层循环处理用户 Field。
// like:article:status{1}: {userid}:{1}
func (s *LikeService) syncArticleUserLikes(ctx context.Context) error {
	var cursor uint64
	pattern := consts.KeyArticleLikeStatePrefix + "*" // 扫描 like:article:status:*

	for {
		// 1. 外层循环：每次批量扫描出 100 个文章的明细大 Key
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		var likes []*model.ArticleLike // 声明本批次的批量写入切片

		for _, key := range keys {
			// 解析大 Key 上的文章 ID
			articleID := parseID(key, consts.KeyArticleInfoHashPrefix)

			// 获取该文章下所有的用户点赞状态
			userStates, err := s.rdb.HGetAll(ctx, key).Result()
			if err != nil || len(userStates) == 0 {
				continue
			}

			// 2. 内层循环：遍历当前文章下的所有用户
			for userIDStr, statusStr := range userStates {
				if userIDStr == "placeholder" {
					continue // 过滤防穿透的占位符
				}
				userID, _ := strconv.ParseUint(userIDStr, 10, 64)
				status, _ := strconv.Atoi(statusStr)

				// 记录用户操作
				likes = append(likes, &model.ArticleLike{
					UserID:    userID,
					ArticleID: articleID,
					Status:    int8(status),
				})
			}
		}

		// 3. 批量更新落库
		if len(likes) > 0 {
			if err := s.articleLikeRepo.BatchInsertOrUpdateStatus(ctx, likes); err != nil {
				log.Printf("[Cron][Detail] 用户明细批量落库失败, err=%v", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// 将文章点赞数同步到article表中
func (s *LikeService) syncArticleTotalCounts(ctx context.Context) error {
	var cursor uint64
	pattern := consts.KeyArticleInfoHashPrefix + "*" // 扫描 article:info:*

	for {
		// 1. 批量扫描出 100 个文章的信息大 Key
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		var articles []*model.Article // 声明本批次待更新的文章总数切片

		for _, key := range keys {
			// 解析大 Key 上的文章 ID
			articleID := parseID(key, consts.KeyArticleInfoHashPrefix)

			// 获取文章对应的点赞数
			val, err := s.rdb.HGet(ctx, key, consts.FeildArticleLikeCount).Result()
			if err != nil {
				continue // 如果该文章 info 结构里没有 like_count 字段，直接跳过
			}

			count, _ := strconv.ParseUint(val, 10, 32)

			articles = append(articles, &model.Article{
				ID:        articleID,
				LikeCount: uint32(count),
			})
		}

		// 2. 批量落库
		if len(articles) > 0 {
			if err := s.articleLikeRepo.BatchUpdateArticleLikeCount(ctx, articles); err != nil {
				log.Printf("[Cron][Count] 文章点赞总数批量更新失败, err=%v", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// 批量同步用户点赞评论批明细到 comment_likes 表
func (s *LikeService) SyncCommentUserLikes(ctx context.Context) error {
	var cursor uint64
	pattern := consts.KeyCommentLikeStatePrefix + "*" // 扫描 like:comment:status:*

	for {
		// 1. 外层循环：每次批量扫描出 100 个评论的明细大 Key
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		var likes []*model.CommentLike // 本批次批量写入切片

		for _, key := range keys {
			// 解析评论ID
			commentID := parseID(key, consts.KeyCommentLikeStatePrefix)
			if commentID == 0 {
				continue
			}

			// 获取该评论下所有用户点赞状态
			userStates, err := s.rdb.HGetAll(ctx, key).Result()
			if err != nil || len(userStates) == 0 {
				continue
			}

			// 遍历每条用户点赞记录
			for userIDStr, statusStr := range userStates {
				if userIDStr == "placeholder" {
					continue // 过滤占位符
				}
				userID, _ := strconv.ParseUint(userIDStr, 10, 64)
				status, _ := strconv.Atoi(statusStr)

				likes = append(likes, &model.CommentLike{
					UserID:    userID,
					CommentID: commentID,
					Status:    int8(status),
				})
			}
		}

		// 批量 upsert 落库
		if len(likes) > 0 {
			if err := s.commentLikeRepo.BatchInsertOrUpdateStatus(ctx, likes); err != nil {
				log.Printf("[Cron][Detail] 评论用户明细批量落库失败, err=%v", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// 将评论点赞总数同步到 comment 主表
func (s *LikeService) SyncCommentTotalCounts(ctx context.Context) error {
	var cursor uint64
	pattern := consts.KeyCommentLikeCountPrefix + "*" // 扫描 like:comment:count:*

	for {
		// 1. 批量扫描评论计数key
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		var comments []*model.Comment // 待更新评论数组

		for _, key := range keys {
			// 解析评论ID
			commentID := parseID(key, consts.KeyCommentLikeCountPrefix)
			if commentID == 0 {
				continue
			}

			// 读取评论独立计数
			val, err := s.rdb.Get(ctx, key).Result()
			if err != nil {
				continue
			}
			count, _ := strconv.ParseUint(val, 10, 32)

			comments = append(comments, &model.Comment{
				ID:        commentID,
				LikeCount: uint32(count),
			})
		}

		// 批量更新评论主表like_count
		if len(comments) > 0 {
			if err := s.commentLikeRepo.BatchUpdateCommentLikeCount(ctx, comments); err != nil {
				log.Printf("[Cron][Count] 评论点赞总数批量更新失败, err=%v", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// 统一点赞刷盘入口
func (s *LikeService) SyncAllLikes(ctx context.Context) error {
	log.Println("[LikeSync] 开始同步文章点赞明细")
	if err := s.syncArticleUserLikes(ctx); err != nil {
		log.Printf("[LikeSync] 文章点赞明细同步错误: %v", err)
		return err
	}

	log.Println("[LikeSync] 开始同步文章点赞总数")
	if err := s.syncArticleTotalCounts(ctx); err != nil {
		log.Printf("[LikeSync] 文章点赞总数同步错误: %v", err)
		return err
	}

	log.Println("[LikeSync] 开始同步评论点赞明细")
	if err := s.SyncCommentUserLikes(ctx); err != nil {
		log.Printf("[LikeSync] 评论点赞明细同步错误: %v", err)
		return err
	}

	log.Println("[LikeSync] 开始同步评论点赞总数")
	if err := s.SyncCommentTotalCounts(ctx); err != nil {
		log.Printf("[LikeSync] 评论点赞总数同步错误: %v", err)
		return err
	}

	log.Println("[LikeSync] 全部点赞同步完成")
	return nil
}
