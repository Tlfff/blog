// Package notification 组装 Notification 上下文的应用和基础设施依赖。
package notification

import (
	notificationapp "blog/internal/notification/application"
	notificationinfra "blog/internal/notification/infrastructure"
	notificationhttp "blog/internal/notification/interfaces/http"
	notificationkafka "blog/internal/notification/interfaces/kafka"
	platformkafka "blog/internal/platform/kafka"

	"go.mongodb.org/mongo-driver/mongo"
)

// ArticleFacade 表示 Notification 依赖的 Article Application Facade。
type ArticleFacade = notificationinfra.ArticleFacade

// UserFacade 表示 Notification 依赖的 User Application Facade。
type UserFacade = notificationinfra.UserFacade

// Module 表示 Notification 上下文对组合根公开的能力。
type Module struct {
	Application *notificationapp.Service              // Notification Application Facade
	HTTP        *notificationhttp.NotificationHandler // Notification HTTP Adapter
	Kafka       platformkafka.MessageHandler          // Notification Kafka Adapter
}

// NewModule 创建 Notification 上下文模块。
func NewModule(db *mongo.Database, articles ArticleFacade, users UserFacade) *Module {
	repo := notificationinfra.NewNotificationRepository(db)
	application := notificationapp.NewService(
		repo,
		notificationinfra.NewArticleQuery(articles),
		notificationinfra.NewUserInfoQuery(users),
	)
	return &Module{
		Application: application,
		HTTP:        notificationhttp.NewNotificationHandler(application),
		Kafka:       notificationkafka.NewHandler(application),
	}
}
