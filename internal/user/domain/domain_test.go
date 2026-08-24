package domain

import "testing"

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
