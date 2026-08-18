package community

import (
	"blog/internal/common"
	domaincommunity "blog/internal/domain/community"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type fakeCommentRepo struct {
	comments map[uint64]*domaincommunity.Comment
	nextID   uint64
}

func newFakeCommentRepo() *fakeCommentRepo {
	return &fakeCommentRepo{comments: make(map[uint64]*domaincommunity.Comment), nextID: 1}
}

func (f *fakeCommentRepo) CreateWithCounts(_ context.Context, comment *domaincommunity.Comment, incrementReply bool) error {
	if incrementReply {
		root, ok := f.comments[comment.RootID]
		if !ok {
			return domaincommunity.ErrCommentNotFound
		}
		if root.Status == domaincommunity.CommentStatusDeleted {
			return domaincommunity.ErrCommentRootDeleted
		}
		root.CommentCount++
	}
	comment.ID = f.nextID
	f.nextID++
	now := time.Now()
	comment.CreatedTime = now
	comment.UpdatedTime = now
	f.comments[comment.ID] = cloneComment(comment)
	return nil
}

func (f *fakeCommentRepo) FindByID(_ context.Context, id uint64) (*domaincommunity.Comment, error) {
	comment, ok := f.comments[id]
	if !ok {
		return nil, domaincommunity.ErrCommentNotFound
	}
	return cloneComment(comment), nil
}

func (f *fakeCommentRepo) ListRootComments(_ context.Context, _, _ uint64, _, _ int, _ bool, _ uint64) ([]*domaincommunity.CommentWithUser, error) {
	return nil, nil
}

func (f *fakeCommentRepo) CountRootComments(_ context.Context, _, _ uint64) (int64, error) {
	return 0, nil
}

func (f *fakeCommentRepo) ListReplies(_ context.Context, _ uint64, _ uint64, _, _ int) ([]*domaincommunity.CommentWithUser, error) {
	return nil, nil
}

func (f *fakeCommentRepo) CountReplies(_ context.Context, _ uint64) (int64, error) {
	return 0, nil
}

func (f *fakeCommentRepo) DeleteWithCounts(_ context.Context, comment *domaincommunity.Comment) error {
	current, ok := f.comments[comment.ID]
	if !ok {
		return domaincommunity.ErrCommentNotFound
	}
	current.Status = domaincommunity.CommentStatusDeleted
	return nil
}

type fakeArticleLikeRepo struct {
	states map[string]bool
	writes int
}

func newFakeArticleLikeRepo() *fakeArticleLikeRepo {
	return &fakeArticleLikeRepo{states: make(map[string]bool)}
}

func (f *fakeArticleLikeRepo) SetLiked(_ context.Context, userID, articleID uint64, liked bool) error {
	f.states[fmt.Sprintf("%d:%d", userID, articleID)] = liked
	f.writes++
	return nil
}

func (f *fakeArticleLikeRepo) IsLiked(_ context.Context, userID, articleID uint64) (bool, error) {
	return f.states[fmt.Sprintf("%d:%d", userID, articleID)], nil
}

func (f *fakeArticleLikeRepo) GetLikedUserIDs(_ context.Context, _ uint64) ([]uint64, error) {
	return nil, nil
}

type fakeCommentLikeRepo struct {
	states map[string]bool
	writes int
}

func newFakeCommentLikeRepo() *fakeCommentLikeRepo {
	return &fakeCommentLikeRepo{states: make(map[string]bool)}
}

func (f *fakeCommentLikeRepo) SetLiked(_ context.Context, userID, commentID uint64, liked bool) error {
	f.states[fmt.Sprintf("%d:%d", userID, commentID)] = liked
	f.writes++
	return nil
}

func (f *fakeCommentLikeRepo) IsLiked(_ context.Context, userID, commentID uint64) (bool, error) {
	return f.states[fmt.Sprintf("%d:%d", userID, commentID)], nil
}

func (f *fakeCommentLikeRepo) GetLikedUserIDs(_ context.Context, _ uint64) ([]uint64, error) {
	return nil, nil
}

type fakeViewRepo struct {
	histories int
	increments int
}

func (f *fakeViewRepo) Create(_ context.Context, _ *domaincommunity.ViewHistory) error {
	f.histories++
	return nil
}

func (f *fakeViewRepo) IncrementViewCount(_ context.Context, _ uint64) error {
	f.increments++
	return nil
}

type fakeNotificationRepo struct {
	notifications []*domaincommunity.Notification
	unread        int64
}

func (f *fakeNotificationRepo) Insert(_ context.Context, notification *domaincommunity.Notification) error {
	f.notifications = append(f.notifications, notification)
	f.unread++
	return nil
}

func (f *fakeNotificationRepo) GetList(_ context.Context, _ uint64, _, _ int64) ([]*domaincommunity.Notification, error) {
	return f.notifications, nil
}

func (f *fakeNotificationRepo) MarkAllAsRead(_ context.Context, _ uint64) error {
	f.unread = 0
	for _, n := range f.notifications {
		n.IsRead = true
	}
	return nil
}

func (f *fakeNotificationRepo) GetUnreadCount(_ context.Context, _ uint64) (int64, error) {
	return f.unread, nil
}

type fakeArticleQuery struct {
	articles map[uint64]*domaincommunity.ArticleInfo
}

func (f *fakeArticleQuery) FindByID(_ context.Context, id uint64) (*domaincommunity.ArticleInfo, error) {
	article, ok := f.articles[id]
	if !ok {
		return nil, errors.New("article not found")
	}
	return article, nil
}

func (f *fakeArticleQuery) GetHotListByIDs(_ context.Context, ids []uint64) ([]*domaincommunity.ArticleInfo, error) {
	result := make([]*domaincommunity.ArticleInfo, 0, len(ids))
	for _, id := range ids {
		if article, ok := f.articles[id]; ok {
			result = append(result, article)
		}
	}
	return result, nil
}

func (f *fakeArticleQuery) GetTopHotArticles(_ context.Context, limit int) ([]*domaincommunity.ArticleInfo, error) {
	result := make([]*domaincommunity.ArticleInfo, 0, limit)
	for _, article := range f.articles {
		result = append(result, article)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

type fakeUserQuery struct {
	users map[uint64]*domaincommunity.UserInfo
}

func (f *fakeUserQuery) FindUserByID(_ context.Context, id uint64) (*domaincommunity.UserInfo, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

type fakeLikeCache struct {
	states map[string]map[uint64]bool
}

func newFakeLikeCache() *fakeLikeCache {
	return &fakeLikeCache{states: make(map[string]map[uint64]bool)}
}

func (f *fakeLikeCache) IsLiked(_ context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) (bool, error) {
	key := string(target) + ":" + fmt.Sprint(targetID)
	return f.states[key][userID], nil
}

func (f *fakeLikeCache) Add(_ context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) error {
	key := string(target) + ":" + fmt.Sprint(targetID)
	if f.states[key] == nil {
		f.states[key] = make(map[uint64]bool)
	}
	f.states[key][userID] = true
	return nil
}

func (f *fakeLikeCache) Remove(_ context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) error {
	key := string(target) + ":" + fmt.Sprint(targetID)
	delete(f.states[key], userID)
	return nil
}

type fakeLikeCountStore struct{}

func (fakeLikeCountStore) GetCommentLikeCounts(_ context.Context, _ []uint64) (map[uint64]uint64, error) {
	return map[uint64]uint64{}, nil
}

type fakeHotRankStore struct {
	entries []domaincommunity.HotRankItem
}

func (f *fakeHotRankStore) GetTop(_ context.Context, limit int) ([]domaincommunity.HotRankItem, error) {
	if len(f.entries) > limit {
		return f.entries[:limit], nil
	}
	return f.entries, nil
}

func (f *fakeHotRankStore) Rebuild(_ context.Context, entries []domaincommunity.HotRankItem) error {
	f.entries = entries
	return nil
}

type fakeEvents struct {
	notifications []domaincommunity.NotificationEvent
	views         []domaincommunity.ViewHistoryEvent
}

func (f *fakeEvents) SendLikeNotification(_ context.Context, event domaincommunity.NotificationEvent) error {
	f.notifications = append(f.notifications, event)
	return nil
}

func (f *fakeEvents) SendViewHistory(_ context.Context, event domaincommunity.ViewHistoryEvent) error {
	f.views = append(f.views, event)
	return nil
}

func newCommunityTestService() (*Service, *fakeCommentRepo, *fakeArticleLikeRepo, *fakeCommentLikeRepo, *fakeViewRepo, *fakeNotificationRepo, *fakeEvents) {
	commentRepo := newFakeCommentRepo()
	articleLikes := newFakeArticleLikeRepo()
	commentLikes := newFakeCommentLikeRepo()
	views := &fakeViewRepo{}
	notifications := &fakeNotificationRepo{}
	articles := &fakeArticleQuery{articles: map[uint64]*domaincommunity.ArticleInfo{
		1: {ID: 1, AuthorID: 2, Title: "文章A", ViewCount: 10, LikeCount: 5, CommentCount: 3},
	}}
	users := &fakeUserQuery{users: map[uint64]*domaincommunity.UserInfo{
		1: {ID: 1, Nickname: "用户1", Avatar: "a"},
		2: {ID: 2, Nickname: "作者", Avatar: "b"},
	}}
	events := &fakeEvents{}
	service := NewService(
		commentRepo,
		articleLikes,
		commentLikes,
		views,
		notifications,
		articles,
		users,
		newFakeLikeCache(),
		fakeLikeCountStore{},
		&fakeHotRankStore{},
		events,
	)
	return service, commentRepo, articleLikes, commentLikes, views, notifications, events
}

func TestCommunityService_CommentsAndPermissions(t *testing.T) {
	service, repo, _, _, _, _, _ := newCommunityTestService()
	ctx := context.Background()

	root, err := service.CreateComment(ctx, 1, 0, 1, 0, "主评论", "127.0.0.1")
	if err != nil {
		t.Fatalf("创建主评论失败: %v", err)
	}
	if root.ID == 0 {
		t.Fatal("主评论返回 ID 为空")
	}
	if _, err := service.CreateComment(ctx, 1, 999, 2, 0, "回复不存在主楼", "127.0.0.1"); !errors.Is(err, common.ErrCommentNotFound) {
		t.Fatalf("回复不存在主楼应报错, got %v", err)
	}
	if _, err := service.CreateComment(ctx, 1, root.ID, 2, 1, "回复", "127.0.0.1"); err != nil {
		t.Fatalf("创建回复失败: %v", err)
	}
	if err := service.DeleteComment(ctx, root.ID, 999, false); !errors.Is(err, common.ErrCommentPermission) {
		t.Fatalf("非作者删除应被拒绝, got %v", err)
	}
	if err := service.DeleteComment(ctx, root.ID, 1, false); err != nil {
		t.Fatalf("作者删除失败: %v", err)
	}
	if repo.comments[root.ID].Status != domaincommunity.CommentStatusDeleted {
		t.Fatal("删除后评论状态应为已删除")
	}
}

func TestCommunityService_LikeIdempotency(t *testing.T) {
	service, _, articleLikes, commentLikes, _, _, _ := newCommunityTestService()
	ctx := context.Background()

	if err := service.ArticleLike(ctx, 1, 1); err != nil {
		t.Fatalf("点赞失败: %v", err)
	}
	if err := service.ArticleLike(ctx, 1, 1); err != nil {
		t.Fatalf("重复点赞失败: %v", err)
	}
	if articleLikes.writes != 1 {
		t.Fatalf("重复点赞不应重复写库, writes=%d", articleLikes.writes)
	}
	liked, err := service.IsUserLikedArticle(ctx, 1, 1)
	if err != nil || !liked {
		t.Fatalf("点赞状态查询错误: %v %v", liked, err)
	}
	if err := service.ArticleCancelLike(ctx, 1, 1); err != nil {
		t.Fatalf("取消点赞失败: %v", err)
	}
	if err := service.ArticleCancelLike(ctx, 1, 1); err != nil {
		t.Fatalf("重复取消点赞失败: %v", err)
	}
	if articleLikes.writes != 2 {
		t.Fatalf("取消点赞应只写一次, writes=%d", articleLikes.writes)
	}

	if err := service.CommentLike(ctx, 1, 100); err != nil {
		t.Fatalf("评论点赞失败: %v", err)
	}
	if err := service.CommentLike(ctx, 1, 100); err != nil {
		t.Fatalf("重复评论点赞失败: %v", err)
	}
	if commentLikes.writes != 1 {
		t.Fatalf("重复评论点赞不应重复写库, writes=%d", commentLikes.writes)
	}
}

func TestCommunityService_NotificationAndViewRules(t *testing.T) {
	service, _, _, _, views, notifications, events := newCommunityTestService()
	ctx := context.Background()

	if err := service.CreateLikeNotification(ctx, domaincommunity.NotificationEvent{
		NotifyType:  domaincommunity.NotifyTypeLikeArticle,
		SenderID:    2,
		TargetID:    1,
		CreatedTime: time.Now(),
	}); err != nil {
		t.Fatalf("自赞通知处理失败: %v", err)
	}
	if len(notifications.notifications) != 0 {
		t.Fatal("自己点赞自己不应产生通知")
	}
	if err := service.CreateLikeNotification(ctx, domaincommunity.NotificationEvent{
		NotifyType:  domaincommunity.NotifyTypeLikeArticle,
		SenderID:    1,
		TargetID:    1,
		CreatedTime: time.Now(),
	}); err != nil {
		t.Fatalf("点赞通知处理失败: %v", err)
	}
	if len(notifications.notifications) != 1 {
		t.Fatalf("应产生一条通知, got %d", len(notifications.notifications))
	}
	if notifications.notifications[0].ReceiverID != 2 {
		t.Fatalf("通知接收者应为文章作者, got %d", notifications.notifications[0].ReceiverID)
	}

	service.SendViewHistory(ctx, 1, 1)
	if len(events.views) != 1 {
		t.Fatalf("应发布浏览事件, got %d", len(events.views))
	}
	service.CreateViewHistory(ctx, 1, 1, time.Now())
	service.CreateViewHistory(ctx, 0, 2, time.Now())
	if views.histories != 1 || views.increments != 2 {
		t.Fatalf("浏览统计不变量错误: histories=%d increments=%d", views.histories, views.increments)
	}
}

func TestCommunityService_HotRank(t *testing.T) {
	service, _, _, _, _, _, _ := newCommunityTestService()
	ctx := context.Background()

	if err := service.RebuildHotRank(ctx); err != nil {
		t.Fatalf("重建热榜失败: %v", err)
	}
	rank, err := service.GetHotRank(ctx)
	if err != nil {
		t.Fatalf("获取热榜失败: %v", err)
	}
	if len(*rank.List) != 1 || (*rank.List)[0].Title != "文章A" || (*rank.List)[0].Hot != domaincommunity.CalcHotScore(10, 5, 3) {
		t.Fatalf("热榜数据错误: %+v", rank.List)
	}
	if strings.TrimSpace((*rank.List)[0].Title) == "" {
		t.Fatal("热榜标题为空")
	}
}

func cloneComment(c *domaincommunity.Comment) *domaincommunity.Comment {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}
