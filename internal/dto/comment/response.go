package comment

import (
	"blog/internal/model"
	"blog/pkg/util/ip"
)

// CommentUserInfo 是评论相关的用户公开信息 DTO。
// CommentUserInfo 统一的评论相关用户信息
type CommentUserInfo struct {
	UserID   uint64 `json:"user_id"`  // 用户ID
	Username string `json:"username"` // 用户昵称
	Avatar   string `json:"avatar"`   // 用户头像URL
	// IP       string `json:"ip"`
}

// ---- 一、 前台主评论列表返回 ----

// RootCommentItem 是前台主评论列表项 DTO。
type RootCommentItem struct {
	ID          uint64           `json:"id"`           // 评论ID
	ArticleID   uint64           `json:"article_id"`   // 所属文章ID
	User        *CommentUserInfo `json:"user"`         // 评论发布者信息
	Content     string           `json:"content"`      // 评论正文内容
	ReplyCount  uint32           `json:"reply_count"`  // 该主楼下的回复数
	IP          string           `json:"ip"`           // 评论者IP归属地（已由IP转换为地区文案）
	CreatedTime int64            `json:"created_time"` // 创建时间（Unix 秒）
	Status      int8             `json:"status"`       // 评论状态：0-已删除 1-正常
	LikeCount   uint64           `json:"like_count"`   // 点赞数
}

// RootCommentListResponse 是前台主评论列表响应 DTO。
type RootCommentListResponse struct {
	List  []*RootCommentItem `json:"list"`  // 主评论列表
	Total int64              `json:"total"` // 总数，仅在走传统 Offset 分页时返回，走游标时返回 0
	// HasMore bool               `json:"has_more"` // 前端用来判断是否需要继续支持滑动加载
	LastID   uint64 `json:"last_id"`   // 游标锚点
	Page     uint64 `json:"page"`      // 页码
	PageSize uint64 `json:"page_size"` // 页面大小
}

// 构造主评论列表响应
func NewRootCommentListResponse(models []*model.Comment, userMap map[uint64]*CommentUserInfo, total int64, lastID, page, page_size uint64, likeMap map[uint64]uint64) *RootCommentListResponse {
	// 1. 初始化响应容器，回填分页信息
	resp := &RootCommentListResponse{
		List:  make([]*RootCommentItem, 0),
		Total: total,
		// HasMore: hasMore,
		LastID:   lastID,
		Page:     page,
		PageSize: page_size,
	}

	// 2. 逐条转换评论模型为响应列表项
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
		// 2.3 组装列表项，时间转为 Unix 秒，IP 转换为地区文案
		resp.List = append(resp.List, &RootCommentItem{
			ID:          m.ID,
			ArticleID:   m.ArticleID,
			User:        userInfo,
			Content:     m.Content,
			CreatedTime: m.CreatedTime.Unix(),
			Status:      m.Status,
			IP:          ip.ConvertIPToRegion(m.IP),
			ReplyCount:  m.CommentCount,
			LikeCount:   likeCount,
		})
	}
	// 3. 返回装配好的列表响应
	return resp
}

// ---- 二、 前台子评论（楼中楼）列表返回 ----

// ReplyCommentItem 是前台子评论（楼中楼）列表项 DTO。
type ReplyCommentItem struct {
	ID          uint64           `json:"id"`            // 评论ID
	ArticleID   uint64           `json:"article_id"`    // 所属文章ID
	RootID      uint64           `json:"root_id"`       // 所属主楼评论ID
	User        *CommentUserInfo `json:"user"`          // 回复发布者信息
	ReplyToUser *CommentUserInfo `json:"reply_to_user"` // 被回复者信息，不使用 parent_id，只渲染被回复者
	Content     string           `json:"content"`       // 评论正文内容
	CreatedTime int64            `json:"created_time"`  // 创建时间（Unix 秒）
	Status      int8             `json:"status"`        // 评论状态：0-已删除 1-正常
	IP          string           `json:"ip"`            // 评论者IP归属地（已由IP转换为地区文案）
	LikeCount   uint64           `json:"like_count"`    // 点赞数
}

// ReplyListResponse 是前台子评论（楼中楼）列表响应 DTO。
type ReplyListResponse struct {
	List  []*ReplyCommentItem `json:"list"`  // 楼中楼回复列表
	Total int64               `json:"total"` // 回复总数
	// HasMore bool                `json:"has_more"`
	LastID   uint64 `json:"last_id"`   // 游标锚点，取本页最后一条评论ID
	Page     uint64 `json:"page"`      // 页码
	PageSize uint64 `json:"page_size"` // 页面大小
}

// NewReplyListResponse 构造楼中楼列表响应
func NewReplyListResponse(models []*model.Comment, userMap map[uint64]*CommentUserInfo, total int64, lastID, page, page_size uint64, likeMap map[uint64]uint64) *ReplyListResponse {
	// 1. 初始化响应容器，回填分页信息
	resp := &ReplyListResponse{
		List:  make([]*ReplyCommentItem, 0),
		Total: total,
		// HasMore: hasMore,
		LastID:   lastID,
		Page:     page,
		PageSize: page_size,
	}

	// 2. 逐条转换回复模型为响应列表项
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
		// 2.4 组装列表项，时间转为 Unix 秒，IP 转换为地区文案
		resp.List = append(resp.List, &ReplyCommentItem{
			ID:          m.ID,
			ArticleID:   m.ArticleID,
			RootID:      m.RootID,
			User:        userInfo,
			ReplyToUser: replyToUserInfo,
			Content:     m.Content,
			CreatedTime: m.CreatedTime.Unix(),
			Status:      m.Status,
			IP:          ip.ConvertIPToRegion(m.IP),
			LikeCount:   likeCount,
		})
	}
	// 3. 返回装配好的列表响应
	return resp
}

// ---- 三、 创建成功通用返回 ----

// CreateCommentResponse 是创建评论成功后的通用响应 DTO。
type CreateCommentResponse struct {
	ID          uint64 `json:"id"`           // 新建评论的ID
	CreatedTime int64  `json:"created_time"` // 创建时间（Unix 秒）
}

// ----------------------------------- 二方服务 -----------------------------------
// CommentStatsResponse 是评论统计信息响应 DTO，供二方服务调用。
type CommentStatsResponse struct {
	ID        uint64 `json:"id"`         // 评论ID
	HotCount  uint64 `json:"hot_count"`  // 热度值
	LikeCount uint64 `json:"like_count"` // 点赞数
}

// 构造评论统计信息响应
func NewCommentStatsResponse(commentID, hotCount, likeCount uint64) *CommentStatsResponse {
	return &CommentStatsResponse{
		ID:        commentID,
		HotCount:  hotCount,
		LikeCount: likeCount,
	}
}
