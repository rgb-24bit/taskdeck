package main

import (
	"fmt"

	"github.com/rgb-24bit/taskdeck/internal/config"
	"github.com/rgb-24bit/taskdeck/internal/daemon"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the TaskDeck daemon",
	Annotations: map[string]string{
		"skip-daemon": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRestart(cfg)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cfg *config.Config) error {
	if daemon.IsRunning(cfg.PidPath) {
		if err := runShutdown(cfg); err != nil {
			return err
		}
	}
	fmt.Println("starting daemon...")
	return daemon.Run(cfg)
}
