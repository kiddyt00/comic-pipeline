package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/kiddyt00/comic-pipeline/pkg/server"
	"github.com/spf13/cobra"
)

var (
	n8nURL string
	port   int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 Web 面板",
	Long:  "启动 comic-pipeline Web 管理面板",
	RunE: func(c *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		return server.Serve(ctx, "comic-pipeline.db", n8nURL, "output", port)
	},
}

var rootCmd = &cobra.Command{
	Use:   "comic-pipeline",
	Short: "comic-pipeline CLI",
	Long:  "comic-pipeline - 漫画处理流水线",
}

func init() {
	serveCmd.Flags().StringVar(&n8nURL, "n8n-url", "http://localhost:5678", "n8n base URL")
	serveCmd.Flags().IntVar(&port, "port", 8080, "web 面板端口")
	rootCmd.AddCommand(serveCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
