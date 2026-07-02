package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "blog",                   // 命令名称
	Short: "林风的博客系统",                // 简短介绍（在父命令的帮助列表里显示）
	Long:  `一个基于gin+cobra架构开发的博客系统`, // 详细介绍
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help() // 如果直接输入 blog，什么都不带，就显示帮助信息
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func init() {

}
