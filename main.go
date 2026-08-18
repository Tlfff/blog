package main

import (
	"blog/cmd"
)

// 程序入口，交由 cobra 命令行执行具体子命令
func main() {

	cmd.Execute()
}
