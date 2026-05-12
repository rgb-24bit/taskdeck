package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id|key>",
	Short: "Hard delete a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		if err := apiClient.Delete(args[0]); err != nil {
			return fmt.Errorf("delete: %w", err)
		}
		fmt.Printf("task %s deleted\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
