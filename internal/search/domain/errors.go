package domain

import "errors"

var (
	ErrKeywordEmpty      = errors.New("搜索关键词不能为空")            // 搜索关键词缺失或去除空格后为空
	ErrPageInvalid       = errors.New("搜索页码必须从 1 开始")         // 搜索页码小于 1
	ErrPageSizeInvalid   = errors.New("搜索每页数量必须在 10 到 20 之间") // 搜索每页数量超出范围
	ErrSearchUnavailable = errors.New("文章搜索服务暂不可用")           // Elasticsearch 查询或写入不可用
	ErrChangeTypeInvalid = errors.New("文章搜索变更类型无效")           // Canal 事件类型无法映射
)
