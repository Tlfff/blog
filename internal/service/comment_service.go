package service

import (
	"blog/internal/common"
	"blog/internal/consts"
	commentDto "blog/internal/dto/comment"
	"strconv"

	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/database"
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 核心业务层：主评论列表查询条件
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
	articleRepo *repository.ArticleRepository
	rdb         *redis.Client
}

func NewCommentService(commentRepo repository.CommentRepository, articleRepo *repository.ArticleRepository, rdb *redis.Client) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		articleRepo: articleRepo,
		rdb:         rdb,
	}
}

// 发表评论
func (s *CommentService) CreateComment(ctx context.Context, articleID uint64, rootID uint64, userID uint64, replyToUserID uint64, content string, ip string) (*commentDto.CreateCommentResponse, error) {

	// 1. 构造底层实体模型
	commentModel := &model.Comment{
		ArticleID:     articleID,
		RootID:        rootID,
		UserID:        userID,
		ReplyToUserID: replyToUserID,
		Content:       content,
		IP:            ip,
		Status:        1, // 1: 正常展示
	}
	// 2. 开启事务，插入同时更新文章中的评论数字段
	db := s.commentRepo.GetDB().WithContext(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		// 3. 如果是子评论 (rootID > 0)，校验主楼状态
		if rootID > 0 {
			// 用当前读的方式上锁，避免并发问题
			rootComment, err := s.commentRepo.FindByIDForUpdate(ctx, tx, rootID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return common.ErrCommentNotFound
				}
				return err
			}
			// 状态为 0 说明主楼被删了
			if rootComment.Status == 0 {
				return common.ErrCommentRootDeleted
			}
		}
		// 插入评论
		if err := s.commentRepo.Insert(ctx, tx, commentModel); err != nil {
			return err
		}
		// 如果是发表子评论，需要同步让对应的主楼 comment_count 原子 +1
		if rootID > 0 {
			if err := s.commentRepo.UpdateCommentCountDelta(ctx, tx, rootID, 1); err != nil {
				return err
			}
		}
		// 文章计数更新，tx透传
		return s.articleRepo.UpdateCommentCountDelta(ctx, tx, articleID, 1)
	})

	if err != nil {
		return nil, err
	}
	return &commentDto.CreateCommentResponse{ID: commentModel.ID, CreatedTime: commentModel.CreatedTime.Unix()}, nil
}

// 获取主评论列表
func (s *CommentService) GetRootCommentList(ctx context.Context, cond *CommentQueryCondition) (*commentDto.RootCommentListResponse, error) {
	page := cond.Page
	if page <= 0 {
		page = 1
	}

	var joinsData []*repository.CommentWithUser
	var err error

	// 1. 获取评论数，pagesize+1是为了用于计算是否还有下一页
	if cond.LastID > 0 {
		joinsData, err = s.commentRepo.FindRootCommentsWithCursor(ctx, cond.ArticleID, cond.LastID, cond.PageSize, cond.IsDesc, cond.AuthorID)
	} else {
		offset := (page - 1) * cond.PageSize
		joinsData, err = s.commentRepo.FindRootCommentsWithOffset(ctx, cond.ArticleID, offset, cond.PageSize, cond.IsDesc, cond.AuthorID)
	}

	// 获取满足条件的主评论总数
	totalCount, _ := s.commentRepo.CountRootComments(ctx, cond.ArticleID, cond.AuthorID)

	if err != nil {
		return nil, err
	}

	// 2. 核心转换：从连表数据中提取出原生的 models 和满足 DTO 所需的 userMap
	models := make([]*model.Comment, len(joinsData))
	userMap := make(map[uint64]*commentDto.CommentUserInfo)

	for i, jd := range joinsData {
		// 复制出纯粹的单表底层模型引用
		models[i] = &jd.Comment
		models[i].IP = jd.IP
		// 将发评人数据直接抓进 Map
		userMap[jd.UserID] = &commentDto.CommentUserInfo{
			UserID:   jd.UserID,
			Username: jd.Nickname,
			Avatar:   jd.Avatar,
		}
		// 主评论一般没有被回复人，但为了防防御，只要有也顺便抓进去
		if jd.ReplyToUserID > 0 {
			userMap[jd.ReplyToUserID] = &commentDto.CommentUserInfo{
				UserID:   jd.ReplyToUserID,
				Username: jd.ReplyNickname,
				Avatar:   jd.ReplyAvatar,
			}
		}
	}
	// 3. 计算额外的返回信息
	var nextLastID uint64 = 0
	if len(models) > 0 {
		nextLastID = models[len(models)-1].ID
	}

	// 4. 获取点赞数
	likeMap := s.getCommentsLikeCountMap(ctx, models)
	// 组装返回信息
	return commentDto.NewRootCommentListResponse(models, userMap, totalCount, nextLastID, uint64(page), uint64(cond.PageSize), likeMap), nil
}

// 获取子评论列表
func (s *CommentService) GetReplyList(ctx context.Context, cond *ReplyQueryCondition) (*commentDto.ReplyListResponse, error) {
	// 1. 查看主评论是否存在或者被删除
	rootComment, err := s.commentRepo.FindByID(ctx, cond.RootID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrCommentNotFound
		}
		return nil, err
	}

	if rootComment.Status == 0 {
		return &commentDto.ReplyListResponse{
			List: make([]*commentDto.ReplyCommentItem, 0),
			// HasMore: false,
			LastID: 0,
		}, nil
	}

	page := cond.Page
	if page <= 0 {
		page = 1
	}

	var joinsData []*repository.CommentWithUser

	// 2. 走连表游标或分页
	if cond.LastID > 0 {
		joinsData, err = s.commentRepo.FindRepliesWithCursor(ctx, cond.RootID, cond.LastID, cond.PageSize+1)
	} else {
		offset := (page - 1) * cond.PageSize
		joinsData, err = s.commentRepo.FindRepliesWithOffset(ctx, cond.RootID, offset, cond.PageSize+1)
	}

	if err != nil {
		return nil, err
	}

	// 3. 从连表数据中提取出原生的子评论 models 以及对应的发评人/回复人画像
	models := make([]*model.Comment, len(joinsData))
	userMap := make(map[uint64]*commentDto.CommentUserInfo)

	for i, jd := range joinsData {
		models[i] = &jd.Comment
		models[i].IP = jd.IP
		// 填充发评人
		userMap[jd.UserID] = &commentDto.CommentUserInfo{
			UserID:   jd.UserID,
			Username: jd.Nickname,
			Avatar:   jd.Avatar,
			// IP:       jd.IP,
		}
		// 填充被回复人
		if jd.ReplyToUserID > 0 {
			userMap[jd.ReplyToUserID] = &commentDto.CommentUserInfo{
				UserID:   jd.ReplyToUserID,
				Username: jd.ReplyNickname,
				Avatar:   jd.ReplyAvatar,
			}
		}
	}

	// 4. 计算子评论游标及数量边界

	var nextLastID uint64 = 0
	if len(models) > 0 {
		nextLastID = models[len(models)-1].ID
	}

	// 计算当前主楼下子评论总数
	count, _ := s.commentRepo.CountReplies(ctx, cond.RootID)

	// 5. 获取点赞数
	likeMap := s.getCommentsLikeCountMap(ctx, models)

	return commentDto.NewReplyListResponse(models, userMap, count, nextLastID, uint64(page), uint64(cond.PageSize), likeMap), nil
}

// 删除主评论 todo: 可加分布式锁，进来之后先查评论是否还存在，不存在则直接返回，不进入事务中
func (s *CommentService) DeleteRootComment(ctx context.Context, comment *model.Comment) error {

	return database.RunTx(ctx, s.commentRepo.GetDB(), func(tx *gorm.DB) error {
		// 1. 批量软删除该主评论下所有子评论，同时拿到有效子评论数量
		replyCount, err := s.commentRepo.BatchUpdateChildCommentStatus(ctx, tx, comment.ID)
		if err != nil {
			return err
		}

		// 2. 软删除当前主评论
		affected, err := s.commentRepo.UpdateStatus(ctx, tx, comment.ID, 0)
		if err != nil {
			return err
		}

		// 并发兜底：当前评论已被其他请求删除，无数据变更，直接提交
		if affected == 0 {
			return nil
		}

		// 3. 文章总评论数一次性扣减：自身1条 + 所有子评论条数
		totalDecrease := 1 + replyCount
		return s.articleRepo.UpdateCommentCountDelta(ctx, tx, comment.ArticleID, -totalDecrease)
	})

}

// 删除子评论
func (s *CommentService) DeleteSonComment(ctx context.Context, comment *model.Comment) error {

	return database.RunTx(ctx, s.commentRepo.GetDB(), func(tx *gorm.DB) error {
		// 1. 软删除当前子评论
		affected, err := s.commentRepo.UpdateStatus(ctx, tx, comment.ID, 0)
		if err != nil {
			return err
		}

		if affected == 0 {
			return nil
		}

		// 2. 对应顶层主评论回复计数 -1
		if err := s.commentRepo.UpdateCommentCountDelta(ctx, tx, comment.RootID, -1); err != nil {
			return err
		}

		// 3. 文章总评论数 -1
		return s.articleRepo.UpdateCommentCountDelta(ctx, tx, comment.ArticleID, -1)
	})
}

// 删除评论入口，在这里就行校验，同时分流主评论和子评论
func (s *CommentService) DeleteComment(ctx context.Context, commentID uint64, userID uint64, isAdmin bool) error {
	oldComment, err := s.checkCommentAuth(ctx, commentID, userID, isAdmin)
	if err != nil {
		return err
	}
	if oldComment.RootID == 0 {
		// 如果是主评论
		return s.DeleteRootComment(ctx, oldComment)
	} else {
		// 如果是子评论
		return s.DeleteSonComment(ctx, oldComment)
	}
}

// 校验删除评论是否有权限
func (s *CommentService) checkCommentAuth(ctx context.Context, commentID uint64, userID uint64, isAdmin bool) (*model.Comment, error) {
	// 1. 检查评论是否存在
	oldComment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrCommentNotFound
		}
		return nil, err
	}
	// 已删除的话直接返回
	if oldComment.Status == 0 {
		return nil, common.ErrCommentDeleted
	}
	// 2. 验证身份
	if !isAdmin && oldComment.UserID != userID {
		return nil, common.ErrCommentPermission
	}
	return oldComment, nil
}

// 💡 辅助方法：通过 MGet 批量高效获取多条评论在 Redis 中的点赞数
func (s *CommentService) getCommentsLikeCountMap(ctx context.Context, comments []*model.Comment) map[uint64]uint64 {
	likeMap := make(map[uint64]uint64)
	if len(comments) == 0 {
		return likeMap
	}

	// 1. 组装所有评论对应的 Redis Count Key
	keys := make([]string, len(comments))
	for i, c := range comments {
		keys[i] = consts.KeyCommentLikeCountPrefix + strconv.FormatUint(c.ID, 10)
	}

	// 2. 批量一次性调 Redis，防止循环内多次单请求
	vals, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return likeMap // 出错则直接走兜底逻辑
	}

	// 3. 解析结果，存入映射表中
	for i, val := range vals {
		if val != nil {
			if countStr, ok := val.(string); ok {
				if count, err := strconv.ParseUint(countStr, 10, 64); err == nil {
					likeMap[comments[i].ID] = count
					continue
				}
			}
		}
		// 如果 Redis 中还未生成此 Key，则用 MySQL 中原本的 LikeCount 字段兜底
		likeMap[comments[i].ID] = uint64(comments[i].LikeCount)
	}

	return likeMap
}
