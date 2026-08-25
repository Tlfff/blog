package app

// NotificationListQuery 表示通知列表查询条件。
type NotificationListQuery struct {
	UserID   uint64 // 当前用户唯一标识
	Page     int64  // 页码
	PageSize int64  // 每页数量
}
