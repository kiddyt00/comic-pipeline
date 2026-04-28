package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 Web 面板",
	Long:  "启动 comic-pipeline Web 管理面板",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🎬 comic-pipeline Web 面板启动中...")
		fmt.Println("   地址: http://localhost:8080")
	},
}

var rootCmd = &cobra.Command{
	Use:   "comic-pipeline",
	Short: "comic-pipeline CLI",
	Long:  "comic-pipeline - 漫画处理流水线",
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
