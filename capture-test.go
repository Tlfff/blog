package main

import "net/http"

func main() {
	// 测试接口，带参数返回文本
	http.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("本地抓包测试成功！"))
	})
	// 监听本地回环 127.0.0.1:8080
	http.ListenAndServe("127.0.0.1:8080", nil)
}
