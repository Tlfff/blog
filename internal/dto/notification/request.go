package notification

type NotifyListRequest struct {
	Page     uint64 `form:"page" binding:"omitempty,min=0"`
	PageSize uint64 `form:"page_size" binding:"min=10,max=200"`
}
