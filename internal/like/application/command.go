package application

// ChangeLikeCommand 表示点赞或取消点赞用例输入。
type ChangeLikeCommand struct {
	UserID   uint64 // 操作用户唯一标识
	TargetID uint64 // 点赞目标唯一标识
}
