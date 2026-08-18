package notification

// NotifyListRequest 是分页获取通知列表的请求 DTO。
type NotifyListRequest struct {
	Page     uint64 `form:"page" binding:"omitempty,min=0"`     // 页码，从1开始，不传时由服务端取默认值
	PageSize uint64 `form:"page_size" binding:"min=10,max=200"` // 每页条数，取值范围 10~200
}
