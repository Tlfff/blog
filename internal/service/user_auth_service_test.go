package service

import (
	"blog/internal/model"
	"blog/internal/repository"
	"fmt"
	"log"
	"testing"
)

func TestUserAuthServiceRegister(t *testing.T) {
	repo := repository.NewUserRepository()
	userAuthService := NewUserAuthService(repo)
	log.Printf("测试注册功能: %+v\n", repo)

	// 1.测试注册新用户
	user := &model.User{
		ID:       0,
		Nickname: "testuser",
		Phone:    "1234567890",
	}
	err := userAuthService.Register(user, "123456")
	if err != nil {
		t.Fatalf("注册用户失败: %v", err)
	}
	// 2.测试注册重复用户
	err = userAuthService.Register(user, "123456")
	if err != nil {
		log.Printf("注册重复用户失败: %v", err)
	}
	// 3.校验手机号、用户名
	savedUser, err := repo.GetUserByPhone("1234567890")
	if err != nil {
		log.Printf("获取用户失败: %v", err)
	}
	fmt.Printf("注册成功的用户: %+v\n", savedUser)
	// 4.测试密码哈希
	if savedUser.Password == "123456" {
		log.Printf("密码没有被哈希处理")
	}
	fmt.Printf("注册用户的哈希密码: %s\n", savedUser.Password)

}

func TestUserAuthServiceLogin(t *testing.T) {
	repo := repository.NewUserRepository()
	userAuthService := NewUserAuthService(repo)
	log.Printf("测试登录功能: %+v\n", repo)
	// 1.注册一个用户
	user := &model.User{
		ID:       0,
		Nickname: "testuser",
		Phone:    "1234567890",
		Status:   1,
	}
	err := userAuthService.Register(user, "123456")
	if err != nil {
		t.Fatalf("注册用户失败: %v", err)
	}

	// 2.测试正确登录
	loggedInUser, err := userAuthService.Login("1234567890", "123456")
	if err != nil {
		log.Printf("登录失败: %v", err)
	}
	fmt.Printf("登录成功的用户: %+v\n", loggedInUser)

	// 3.测试错误密码
	_, err = userAuthService.Login("1234567890", "wrongpassword")
	if err != nil {
		log.Printf("使用错误密码登录失败: %v", err)
	}

	// 4.测试不存在的用户
	_, err = userAuthService.Login("0987654321", "123456")
	if err != nil {
		log.Printf("使用不存在的手机号登录失败: %v", err)
	}

	// 5.测试禁用用户
	user.Status = 0
	err = repo.UpdateUser(user)
	if err != nil {
		log.Printf("更新用户状态失败: %v", err)
	}
	_, err = userAuthService.Login("1234567890", "123456")
	if err != nil {
		log.Printf("禁用用户登录失败: %v", err)
	}

}
