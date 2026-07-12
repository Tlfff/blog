package comment

import (
	"blog/internal/model"
	"blog/pkg/iputil"
)

// CommentUserInfo 统一的评论相关用户信息
type CommentUserInfo struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	// IP       string `json:"ip"`
}

// ---- 一、 前台主评论列表返回 ----

type RootCommentItem struct {
	ID          uint64           `json:"id"`
	ArticleID   uint64           `json:"article_id"`
	User        *CommentUserInfo `json:"user"`
	Content     string           `json:"content"`
	ReplyCount  int64            `json:"reply_count"`
	IP          string           `json:"ip"`
	CreatedTime int64            `json:"created_time"`
	Status      int8             `json:"status"`
	LikeCount   uint64           `json:"like_count"`
}

type RootCommentListResponse struct {
	List  []*RootCommentItem `json:"list"`
	Total int64              `json:"total"` // 仅在走传统 Offset 分页时返回总数，走游标时返回 0
	// HasMore bool               `json:"has_more"` // 前端用来判断是否需要继续支持滑动加载
	LastID   uint64 `json:"last_id"`   // 游标锚点
	Page     uint64 `json:"page"`      // 页码
	PageSize uint64 `json:"page_size"` // 页面大小
}

// 构造主评论列表响应
func NewRootCommentListResponse(models []*model.Comment, userMap map[uint64]*CommentUserInfo, total int64, lastID, page, page_size uint64, likeMap map[uint64]uint64) *RootCommentListResponse {
	resp := &RootCommentListResponse{
		List:  make([]*RootCommentItem, 0),
		Total: total,
		// HasMore: hasMore,
		LastID:   lastID,
		Page:     page,
		PageSize: page_size,
	}

	for _, m := range models {
		// 从映射字典中安全捞取用户信息，没有则给个兜底空对象，防空指针
		userInfo, exists := userMap[m.UserID]
		if !exists {
			userInfo = &CommentUserInfo{UserID: m.UserID, Username: "未知用户", Avatar: ""}
		}
		// 从redis中获取评论对应点赞数
		likeCount, ok := likeMap[m.ID]
		if !ok {
			likeCount = uint64(m.LikeCount)
		}
		resp.List = append(resp.List, &RootCommentItem{
			ID:          m.ID,
			ArticleID:   m.ArticleID,
			User:        userInfo,
			Content:     m.Content,
			CreatedTime: m.CreatedTime.Unix(),
			Status:      m.Status,
			IP:          iputil.ConvertIPToRegion(m.IP),
			ReplyCount:  m.CommentCount,
			LikeCount:   likeCount,
		})
	}
	return resp
}

// ---- 二、 前台子评论（楼中楼）列表返回 ----

type ReplyCommentItem struct {
	ID          uint64           `json:"id"`
	ArticleID   uint64           `json:"article_id"`
	RootID      uint64           `json:"root_id"`
	User        *CommentUserInfo `json:"user"`
	ReplyToUser *CommentUserInfo `json:"reply_to_user"` // 贴吧精髓：彻底舍弃 parent_id，只渲染“被回复者”[cite: 1]
	Content     string           `json:"content"`
	CreatedTime int64            `json:"created_time"`
	Status      int8             `json:"status"`
	IP          string           `json:"ip"`
	LikeCount   uint64           `json:"like_count"`
}

type ReplyListResponse struct {
	List  []*ReplyCommentItem `json:"list"`
	Total int64               `json:"total"`
	// HasMore bool                `json:"has_more"`
	LastID   uint64 `json:"last_id"`
	Page     uint64 `json:"page"`      // 页码
	PageSize uint64 `json:"page_size"` // 页面大小
}

// NewReplyListResponse 构造楼中楼列表响应
func NewReplyListResponse(models []*model.Comment, userMap map[uint64]*CommentUserInfo, total int64, lastID, page, page_size uint64, likeMap map[uint64]uint64) *ReplyListResponse {
	resp := &ReplyListResponse{
		List:  make([]*ReplyCommentItem, 0),
		Total: total,
		// HasMore: hasMore,
		LastID:   lastID,
		Page:     page,
		PageSize: page_size,
	}

	for _, m := range models {
		userInfo, exists := userMap[m.UserID]
		if !exists {
			userInfo = &CommentUserInfo{UserID: m.UserID, Username: "未知用户", Avatar: ""}
		}

		// 处理被回复者（如果 reply_to_user_id > 0 说明是在回复别人）
		var replyToUserInfo *CommentUserInfo
		if m.ReplyToUserID > 0 {
			if targetUser, ok := userMap[m.ReplyToUserID]; ok {
				replyToUserInfo = targetUser
			} else {
				replyToUserInfo = &CommentUserInfo{UserID: m.ReplyToUserID, Username: "未知用户", Avatar: ""}
			}
		}
		// 从redis中获取评论对应点赞数
		likeCount, ok := likeMap[m.ID]
		if !ok {
			likeCount = uint64(m.LikeCount)
		}
		resp.List = append(resp.List, &ReplyCommentItem{
			ID:          m.ID,
			ArticleID:   m.ArticleID,
			RootID:      m.RootID,
			User:        userInfo,
			ReplyToUser: replyToUserInfo,
			Content:     m.Content,
			CreatedTime: m.CreatedTime.Unix(),
			Status:      m.Status,
			IP:          iputil.ConvertIPToRegion(m.IP),
			LikeCount:   likeCount,
		})
	}
	return resp
}

// ---- 三、 创建成功通用返回 ----

type CreateCommentResponse struct {
	ID          uint64 `json:"id"`
	CreatedTime int64  `json:"created_time"`
}
