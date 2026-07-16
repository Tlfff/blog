package ip

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/service"
)

var (
	ip2Region *service.Ip2Region
	once      sync.Once
	ipCache   sync.Map
)

const searcherPoolSize = 20

// 初始化（程序启动时调用一次）
func InitIPSearcher(dbPath string) error {
	var err error

	once.Do(func() {
		cfg, e := service.NewV4Config(
			service.BufferCache, // 全内存缓存，查询速度最快
			dbPath,
			20, // Searcher 池大小，可根据并发调整
		)
		if e != nil {
			err = e
			return
		}
		ip2Region, err = service.NewIp2Region(cfg, nil)
	})

	return err
}

// 关闭资源（程序退出时调用）
func Close() {
	if ip2Region != nil {
		ip2Region.Close()
	}
}

// IP 转属地
func ConvertIPToRegion(ip string) string {
	if ip2Region == nil {
		return "未知"
	}

	if v, ok := ipCache.Load(ip); ok {
		return v.(string)
	}

	parsedIP := net.ParseIP(ip)

	var region string

	switch {
	case ip == "", ip == "localhost":
		region = "内网"

	case parsedIP == nil:
		region = "未知"

	case parsedIP.IsLoopback():
		region = "内网"

	case parsedIP.IsPrivate():
		region = "内网"

	default:
		r, err := ip2Region.Search(ip)
		fmt.Println(r)
		if err != nil {
			region = "未知"
		} else {
			region = parseRegion(r)
		}
	}

	ipCache.Store(ip, region)

	return region
}

// 解析 ip2region 返回结果
func parseRegion(region string) string {
	chunks := strings.Split(region, "|")
	if len(chunks) < 5 {
		return "未知"
	}

	country := strings.TrimSpace(chunks[0])
	province := strings.TrimSpace(chunks[1]) // ← 注意这里改成 1

	// 没有数据
	if country == "" || country == "0" {
		return "未知"
	}

	// 国外 -> 国家
	if country != "中国" {
		if zh, ok := countryMap[country]; ok {
			return zh
		}
		return country
	}

	// 国内 -> 省份
	if province == "" || province == "0" {
		return "中国"
	}

	province = strings.TrimSuffix(province, "省")
	province = strings.TrimSuffix(province, "市")

	return province
}

var countryMap = map[string]string{
	"United States":  "美国",
	"Japan":          "日本",
	"Singapore":      "新加坡",
	"Australia":      "澳大利亚",
	"Germany":        "德国",
	"France":         "法国",
	"United Kingdom": "英国",
	"Russia":         "俄罗斯",
	"Canada":         "加拿大",
	"Korea":          "韩国",
	"South Korea":    "韩国",
	"North Korea":    "朝鲜",
	"India":          "印度",
	"Vietnam":        "越南",
	"Thailand":       "泰国",
	"Malaysia":       "马来西亚",
	"Indonesia":      "印度尼西亚",
	"Philippines":    "菲律宾",
	"Hong Kong":      "中国香港",
	"Taiwan":         "中国台湾",
	"Macao":          "中国澳门",
}
