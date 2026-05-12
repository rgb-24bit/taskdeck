package main

import (
	"github.com/rgb-24bit/taskdeck/internal/daemon"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the TaskDeck daemon",
	Annotations: map[string]string{
		"skip-daemon": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return daemon.Run(cfg)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
