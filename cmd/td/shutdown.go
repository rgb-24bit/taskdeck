package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/rgb-24bit/taskdeck/internal/config"
	"github.com/rgb-24bit/taskdeck/internal/daemon"
	"github.com/spf13/cobra"
)

var shutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "Stop the TaskDeck daemon",
	Annotations: map[string]string{
		"skip-daemon": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShutdown(cfg)
	},
}

func init() {
	rootCmd.AddCommand(shutdownCmd)
}

func runShutdown(cfg *config.Config) error {
	pid, err := daemon.ReadPID(cfg.PidPath)
	if err != nil {
		fmt.Println("daemon is not running")
		return nil
	}
	if !daemon.IsRunning(cfg.PidPath) {
		fmt.Println("daemon is not running")
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	fmt.Println("shutting down daemon...")
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send signal: %w", err)
	}

	for i := 0; i < 150; i++ {
		time.Sleep(100 * time.Millisecond)
		if !daemon.IsRunning(cfg.PidPath) {
			fmt.Println("daemon stopped")
			return nil
		}
	}
	return fmt.Errorf("daemon did not stop in time")
}
