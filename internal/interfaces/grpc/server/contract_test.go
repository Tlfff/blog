package server

import (
	grpchandler "blog/internal/interfaces/grpc/handler"
	"testing"
)

func TestOpenGRPCServiceContract(t *testing.T) {
	app := &AppHandler{
		Article: grpchandler.NewArticleHandler(nil),
		User:    grpchandler.NewUserHandler(nil),
		Comment: grpchandler.NewCommentHandler(nil),
	}

	s := NewGRPCServer(app, nil, nil)
	info := s.GetServiceInfo()

	services := map[string][]string{
		"blogopen.v1.UserService":    {"GetUserBasicInfo", "GetPublicUserInfo"},
		"blogopen.v1.ArticleService": {"GetAvailableList"},
		"blogopen.v1.CommentService": {"GetCommentStats"},
	}

	for service, methods := range services {
		serviceInfo, ok := info[service]
		if !ok {
			t.Errorf("缺少 gRPC 服务: %s", service)
			continue
		}
		gotMethods := make(map[string]struct{})
		for _, method := range serviceInfo.Methods {
			gotMethods[method.Name] = struct{}{}
		}
		for _, method := range methods {
			if _, ok := gotMethods[method]; !ok {
				t.Errorf("服务 %s 缺少方法 %s", service, method)
			}
		}
	}
}
