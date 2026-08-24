package application

// UserProfileQuery 表示用户资料查询条件。
type UserProfileQuery struct {
	UserID uint64 // 用户唯一标识
}

// BatchUserProfileQuery 表示批量公开资料查询条件。
type BatchUserProfileQuery struct {
	UserIDs []uint64 // 用户唯一标识列表
}
