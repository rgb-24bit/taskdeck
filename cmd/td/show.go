package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id|key>",
	Short: "Show task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		task, err := apiClient.Get(args[0])
		if err != nil {
			return fmt.Errorf("show: %w", err)
		}

		fmt.Printf("\033[1m#%d %s\033[0m\n", task.ID, task.Title)
		if task.Key != nil {
			fmt.Printf("  Key:         %s\n", *task.Key)
		}
		fmt.Printf("  Status:       %s\n", statusTag(task))
		fmt.Printf("  Source:       %s %s\n", sourceIcon(task.SourceType), task.SourceType)
		if task.SourceLabel != "" {
			fmt.Printf("                %s\n", task.SourceLabel)
		}
		fmt.Printf("  Condition:    %s", task.ConditionType)
		if task.ConditionTimeout > 0 {
			fmt.Printf(" (%s)", fmtDuration(task.ConditionTimeout))
		}
		fmt.Println()
		if task.EnteredWaitAt != nil {
			fmt.Printf("  Waiting since: %s\n", task.EnteredWaitAt.Format("2006-01-02 15:04"))
		}
		fmt.Printf("  Created:      %s\n", task.CreatedAt.Format("2006-01-02 15:04"))
		if task.DoneAt != nil {
			fmt.Printf("  Done:         %s\n", task.DoneAt.Format("2006-01-02 15:04"))
		}
		if task.Context != "" {
			fmt.Println()
			fmt.Println(task.Context)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
