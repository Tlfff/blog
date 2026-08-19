package domain

import "testing"

func TestUserRoleAndStatusRules(t *testing.T) {
	admin := &User{Role: RoleAdmin, Status: StatusNormal}
	user := &User{Role: RoleUser, Status: StatusNormal}

	if !admin.IsAdmin() || user.IsAdmin() {
		t.Fatal("管理员角色规则错误")
	}
	if !admin.IsNormal() {
		t.Fatal("正常用户状态规则错误")
	}
	admin.Status = StatusDeleted
	if admin.IsNormal() {
		t.Fatal("禁用用户不应为正常状态")
	}
}

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("生成密码哈希失败: %v", err)
	}
	ok, err := VerifyPassword("secret", hash)
	if err != nil || !ok {
		t.Fatalf("正确密码应通过校验: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword("wrong", hash)
	if err != nil || ok {
		t.Fatalf("错误密码不应通过校验: ok=%v err=%v", ok, err)
	}
}
