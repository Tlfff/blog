package application

import notificationdomain "blog/internal/notification/domain"

// NotificationRepository 是 Notification Application 使用的持久化 Port。
type NotificationRepository = notificationdomain.NotificationRepository

// ArticleQuery 是 Notification Application 使用的文章最小查询 Port。
type ArticleQuery = notificationdomain.ArticleQuery

// UserInfoQuery 是 Notification Application 使用的用户最小查询 Port。
type UserInfoQuery = notificationdomain.UserInfoQuery
