package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var activateCmd = &cobra.Command{
	Use:   "activate <id|key>",
	Short: "Activate a task from the waiting pool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		task, err := apiClient.Activate(args[0])
		if err != nil {
			return fmt.Errorf("activate: %w", err)
		}
		fmt.Printf("task #%d activated: %s\n", task.ID, task.Title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(activateCmd)
}
