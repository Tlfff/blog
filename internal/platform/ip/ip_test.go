package ip

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestConvertIPToRegion 验证离线 IP 归属地解析和特殊地址处理。
func TestConvertIPToRegion(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法获取测试文件路径")
	}
	dbPath := filepath.Join(filepath.Dir(currentFile), "resource", "ip2region.xdb")
	if err := InitIPSearcher(dbPath); err != nil {
		t.Fatalf("初始化 IP 查询器失败: %v", err)
	}
	defer Close()

	tests := []struct {
		name string // 测试场景名称
		ip   string // 待解析 IP
		want string // 期望归属地
	}{
		{name: "本地回环", ip: "127.0.0.1", want: "内网"},
		{name: "局域网", ip: "192.168.1.100", want: "内网"},
		{name: "空地址", ip: "", want: "内网"},
		{name: "非法地址", ip: "abc.def", want: "未知"},
		{name: "国内公网", ip: "223.5.5.5", want: "浙江"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ConvertIPToRegion(test.ip); got != test.want {
				t.Fatalf("归属地不一致: got=%s want=%s", got, test.want)
			}
		})
	}
}
