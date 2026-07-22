package ip

import "testing"

func TestConvertIPToRegion(t *testing.T) {
	dbPath := "../resource/ip2region.xdb"

	if err := InitIPSearcher(dbPath); err != nil {
		t.Fatalf("初始化 IP 查询器失败: %v", err)
	}
	defer Close()

	tests := []struct {
		name     string
		ip       string
		expected string
		exact    bool // true：要求完全一致；false：只要求不是"未知"
	}{
		// ===== 国内 =====
		{
			name:     "湖北武汉",
			ip:       "113.57.65.51",
			expected: "湖北",
			exact:    true,
		},
		{
			name:     "阿里DNS",
			ip:       "223.5.5.5",
			expected: "浙江",
			exact:    true,
		},
		{
			name:     "腾讯DNS",
			ip:       "119.29.29.29",
			expected: "北京",
			exact:    true,
		},
		{
			name:     "百度DNS",
			ip:       "180.76.76.76",
			expected: "北京",
			exact:    true,
		},

		// ===== 国外 =====
		// 国外归属可能随数据库更新，因此这里只要求不是"未知"
		{
			name:     "Google DNS",
			ip:       "8.8.8.8",
			expected: "",
			exact:    false,
		},
		{
			name:     "Cloudflare DNS",
			ip:       "1.1.1.1",
			expected: "",
			exact:    false,
		},
		{
			name:     "OpenDNS",
			ip:       "208.67.222.222",
			expected: "",
			exact:    false,
		},

		// ===== 特殊情况 =====
		{
			name:     "本地回环",
			ip:       "127.0.0.1",
			expected: "内网",
			exact:    true,
		},
		{
			name:     "局域网192",
			ip:       "192.168.1.100",
			expected: "内网",
			exact:    true,
		},
		{
			name:     "局域网10",
			ip:       "10.10.10.10",
			expected: "内网",
			exact:    true,
		},
		{
			name:     "空字符串",
			ip:       "",
			expected: "内网",
			exact:    true,
		},
		{
			name:     "非法IP",
			ip:       "abc.def.ghi.jkl",
			expected: "未知",
			exact:    true,
		},
	}

	t.Log("========================================")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertIPToRegion(tt.ip)

			if tt.exact {
				if got != tt.expected {
					t.Errorf("IP=%s 期望=%s 实际=%s", tt.ip, tt.expected, got)
				}
			} else {
				if got == "未知" {
					t.Errorf("IP=%s 应该解析成功，但返回了'未知'", tt.ip)
				}
			}

			t.Logf("%-15s %-18s => %s", tt.name, tt.ip, got)
		})
	}

	t.Log("========================================")
}
