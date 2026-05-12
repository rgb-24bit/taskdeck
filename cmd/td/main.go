package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rgb-24bit/taskdeck/internal/client"
	"github.com/rgb-24bit/taskdeck/internal/config"
	"github.com/rgb-24bit/taskdeck/internal/daemon"
	"github.com/spf13/cobra"
)

var (
	cfg       *config.Config
	apiClient *client.Client
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "td",
	Short: "TaskDeck - context task management",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		return nil
	},
}

func ensureDaemonAndClient() error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}
	if isLocalHost(cfg.Host) && !daemon.IsRunning(cfg.PidPath) {
		fmt.Println("starting taskdeck daemon...")
		if err := daemon.Run(cfg); err != nil {
			return fmt.Errorf("start daemon: %w", err)
		}
		for i := 0; i < 50; i++ {
			time.Sleep(100 * time.Millisecond)
			if daemon.IsRunning(cfg.PidPath) {
				break
			}
		}
		if !daemon.IsRunning(cfg.PidPath) {
			return fmt.Errorf("daemon failed to start")
		}
	}
	apiClient = client.New(cfg.Host, cfg.Port)
	return nil
}

func isLocalHost(host string) bool {
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1"
}
