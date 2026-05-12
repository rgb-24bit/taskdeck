package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup [duration]",
	Short: "Delete done tasks older than duration (default 30d)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		olderThan := "30d"
		if len(args) > 0 {
			olderThan = args[0]
		}

		n, err := apiClient.Cleanup(olderThan)
		if err != nil {
			return fmt.Errorf("cleanup: %w", err)
		}
		fmt.Printf("cleaned up %d done tasks older than %s\n", n, olderThan)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}
