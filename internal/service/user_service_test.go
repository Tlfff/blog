package service

import (
	"blog/internal/model"
	"blog/internal/repository"
	"testing"
)

// -------------------- GetProfile --------------------

func TestUserService_GetProfile(t *testing.T) {
	repo := repository.NewUserRepository()
	service := NewUserService(repo)

	// 先行往内存仓储注入一个测试用户
	_ = repo.CreateUser(&model.User{
		ID:       10,
		Phone:    "13800000000",
		Nickname: "原昵称",
	})

	// 测试标准获取
	user, err := service.GetProfile(10)
	if err != nil {
		t.Fatalf("获取用户资料失败: %v", err)
	}
	if user.Nickname != "原昵称" {
		t.Errorf("预期昵称为 '原昵称'，实际得到: %s", user.Nickname)
	}
}

// -------------------- UpdateProfile --------------------

func TestUserService_UpdateProfile(t *testing.T) {
	repo := repository.NewUserRepository()
	service := NewUserService(repo)

	_ = repo.CreateUser(&model.User{
		ID:       20,
		Phone:    "13800000001",
		Nickname: "老名字",
		Avatar:   "old.png",
	})

	// 执行基本资料更新
	err := service.UpdateProfile(20, "新名字", "new.png")
	if err != nil {
		t.Fatalf("更新用户资料失败: %v", err)
	}

	// 从底层验证落盘结果
	user, _ := repo.FindUserByID(20)
	if user.Nickname != "新名字" || user.Avatar != "new.png" {
		t.Errorf("资料未正确更新，当前: 昵称=%s, 头像=%s", user.Nickname, user.Avatar)
	}
	if user.UpdateTime.IsZero() {
		t.Errorf("更新时间没有被正确设置")
	}
}

// -------------------- UpdateAccount --------------------

// func TestUserService_UpdateAccount_成功与密码错误分支(t *testing.T) {
// 	repo := repository.NewUserRepository()
// 	service := NewUserService(repo)

// 	// 💡 核心：先生成一个合法的 PBKDF2 加密密文存入 mock 用户
// 	correctRawPassword := "123456"
// 	hashedPassword, err := auth.HashPassword(correctRawPassword)
// 	if err != nil {
// 		t.Fatalf("测试前置准备：密码哈希失败: %v", err)
// 	}

// 	_ = repo.CreateUser(&model.User{
// 		ID:       30,
// 		Phone:    "13800000002",
// 		Password: hashedPassword, // 注入合法哈希串
// 	})

// 	// 分支 1：输入错误的旧密码，断言拦截
// 	err = service.UpdateAccount(30, "13911112222", "wrong_pwd", "new_pwd_666")
// 	if err != common.ErrPasswordFailed {
// 		t.Errorf("输入错误密码时，预期返回 ErrPasswordFailed，实际得到: %v", err)
// 	}

// 	// 分支 2：输入正确的旧密码，断言更新成功
// 	err = service.UpdateAccount(30, "13911112222", correctRawPassword, "new_pwd_666")
// 	if err != nil {
// 		t.Fatalf("账户信息更新失败: %v", err)
// 	}

// 	// 从仓储取出来进行最终验证
// 	updatedUser, _ := repo.FindUserByID(30)
// 	if updatedUser.Phone != "13911112222" {
// 		t.Errorf("手机号未能成功修改")
// 	}

// 	// 验证新密码是否已经成功被重置并能通过校验
// 	ok, _ := auth.VerifyPassword("new_pwd_666", updatedUser.Password)
// 	if !ok {
// 		t.Errorf("新密码的哈希不正确，无法通过 VerifyPassword 验证")
// 	}
// }
