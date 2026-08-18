// Package ip 提供 IP 归属地解析与进程内缓存能力。
package ip

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/service"
)

var (
	ip2Region *service.Ip2Region // ip2region 查询器，进程内单例
	once      sync.Once          // 保证查询器只初始化一次
	ipCache   sync.Map           // IP 到归属地的进程内缓存，避免重复查询
)

const searcherPoolSize = 20 // Searcher 池大小，可根据并发量调整

// 初始化 IP 归属地查询器，程序启动时调用一次
func InitIPSearcher(dbPath string) error {
	var err error

	once.Do(func() {
		// 1. 以全内存缓存模式构建 ip2region 配置
		cfg, e := service.NewV4Config(
			service.BufferCache, // 全内存缓存，查询速度最快
			dbPath,
			20, // Searcher 池大小，可根据并发调整
		)
		if e != nil {
			err = e
			return
		}
		// 2. 基于配置创建查询器单例
		ip2Region, err = service.NewIp2Region(cfg, nil)
	})

	return err
}

// 释放查询器占用的资源，程序退出时调用
func Close() {
	if ip2Region != nil {
		ip2Region.Close()
	}
}

// 将 IP 转换为归属地文案，优先命中进程内缓存
func ConvertIPToRegion(ip string) string {
	// 1. 查询器未初始化时统一返回未知
	if ip2Region == nil {
		return "未知"
	}

	// 2. 命中缓存直接返回，避免重复查询
	if v, ok := ipCache.Load(ip); ok {
		return v.(string)
	}

	// 3. 解析 IP 并按地址类型判定归属地
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
		// 3.1 公网 IP 交给 ip2region 查询并解析结果
		r, err := ip2Region.Search(ip)
		fmt.Println(r)
		if err != nil {
			region = "未知"
		} else {
			region = parseRegion(r)
		}
	}

	// 4. 结果写入缓存后返回
	ipCache.Store(ip, region)

	return region
}

// 解析 ip2region 返回的结果串，国内取省份、国外取国家
func parseRegion(region string) string {
	// 1. 结果串以竖线分隔，字段不足视为未知
	chunks := strings.Split(region, "|")
	if len(chunks) < 5 {
		return "未知"
	}

	// 2. 取出国家与省份字段
	country := strings.TrimSpace(chunks[0])
	province := strings.TrimSpace(chunks[1]) // 省份字段（国外为州/省）

	// 3. 国家字段为空或为 0 表示库中无数据
	if country == "" || country == "0" {
		return "未知"
	}

	// 4. 非中国 IP 返回国家中文名，未收录时返回原始值
	if country != "中国" {
		if zh, ok := countryMap[country]; ok {
			return zh
		}
		return country
	}

	// 5. 中国 IP 返回省份，并去掉省/市后缀
	if province == "" || province == "0" {
		return "中国"
	}

	province = strings.TrimSuffix(province, "省")
	province = strings.TrimSuffix(province, "市")

	return province
}

// countryMap 是 ip2region 英文国家名到中文名的映射表。
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
