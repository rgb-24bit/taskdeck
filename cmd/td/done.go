package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done <id|key>",
	Short: "Mark task as done",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		if err := apiClient.Done(args[0]); err != nil {
			return fmt.Errorf("done: %w", err)
		}
		fmt.Printf("task %s marked done\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
}
