package interceptor

import (
	"strconv"
	"testing"
	"time"
)

func TestCheckTimestamp(t *testing.T) {
	now := time.Now()

	// 1. 当前时间应通过
	if err := checkTimestamp(strconv.FormatInt(now.Unix(), 10)); err != nil {
		t.Errorf("当前时间应通过: %v", err)
	}

	// 2. 窗口内（30秒前）应通过
	if err := checkTimestamp(strconv.FormatInt(now.Add(-30*time.Second).Unix(), 10)); err != nil {
		t.Errorf("窗口内时间应通过: %v", err)
	}

	// 3. 窗口外（2分钟前）应拒绝
	if err := checkTimestamp(strconv.FormatInt(now.Add(-2*time.Minute).Unix(), 10)); err == nil {
		t.Error("过期时间应被拒绝")
	}

	// 4. 未来时间（2分钟后）应拒绝
	if err := checkTimestamp(strconv.FormatInt(now.Add(2*time.Minute).Unix(), 10)); err == nil {
		t.Error("未来时间应被拒绝")
	}

	// 5. 非法格式应拒绝
	if err := checkTimestamp("not-a-number"); err == nil {
		t.Error("非法时间戳应被拒绝")
	}
}
