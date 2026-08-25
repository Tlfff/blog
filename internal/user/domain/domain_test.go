package domain

import (
	"errors"
	"testing"
	"time"
)

// TestUserState 验证用户角色和状态领域行为。
func TestUserState(t *testing.T) {
	user := &User{Role: RoleAdmin, Status: StatusNormal}
	if !user.IsAdmin() {
		t.Fatal("管理员角色判断失败")
	}
	if !user.IsNormal() {
		t.Fatal("正常状态判断失败")
	}
}

// TestMinimumPasswordLengthKeepsContract 验证最小密码长度契约不变。
func TestMinimumPasswordLengthKeepsContract(t *testing.T) {
	if MinimumPasswordLength != 6 {
		t.Fatalf("最小密码长度发生变化: %d", MinimumPasswordLength)
	}
}

// TestPasswordValueObjects 验证明文密码规则和历史哈希重建。
func TestPasswordValueObjects(t *testing.T) {
	if _, err := NewPlainPassword("12345"); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("短密码错误不正确: %v", err)
	}
	plain, err := NewPlainPassword("123456")
	if err != nil || plain.String() != "123456" {
		t.Fatalf("创建明文密码失败: %v", err)
	}
	hash := RestorePasswordHash("legacy-hash")
	if hash.String() != "legacy-hash" {
		t.Fatalf("历史哈希重建失败: %s", hash.String())
	}
}

// TestUserDomainBehaviors 验证用户聚合封装状态变化。
func TestUserDomainBehaviors(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	user := NewUser("用户", "13800000000", RestorePasswordHash("hash:old"), "127.0.0.1", now)
	if user.Role != RoleUser || user.Status != StatusNormal || user.Avatar != DefaultAvatar {
		t.Fatalf("注册默认值错误: %+v", user)
	}

	later := now.Add(time.Hour)
	user.RecordLogin("10.0.0.1", later)
	user.UpdateProfile("新昵称", "avatar", later)
	user.ChangePhone("13900000000", later)
	user.ChangePassword(RestorePasswordHash("hash:new"), later)
	user.ChangeAvatar("avatar/1/new.png", later)
	if user.LastLoginIP != "10.0.0.1" || user.Nickname != "新昵称" || user.Phone != "13900000000" {
		t.Fatalf("用户领域行为未生效: %+v", user)
	}
	if user.Password.String() != "hash:new" || user.Avatar != "avatar/1/new.png" || !user.UpdatedTime.Equal(later) {
		t.Fatalf("用户密码或头像行为未生效: %+v", user)
	}
}
