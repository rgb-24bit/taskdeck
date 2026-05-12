package main

import (
	"fmt"

	"github.com/rgb-24bit/taskdeck/internal/model"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [-w] [-d]",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		wait, _ := cmd.Flags().GetBool("wait")
		done, _ := cmd.Flags().GetBool("done")

		status := model.StatusActive
		if wait {
			status = model.StatusWaiting
		} else if done {
			status = model.StatusDone
		}

		tasks, err := apiClient.List(model.ListParams{Status: status})
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("no tasks")
			return nil
		}

		for _, t := range tasks {
			icon := statusIcon(t)
			idStr := fmt.Sprintf("\033[33m#%d\033[0m", t.ID)
			statusTag := statusTag(t)
			keyStr := ""
			if t.Key != nil {
				keyStr = fmt.Sprintf(" \033[90m[%s]\033[0m", *t.Key)
			}
			fmt.Printf("  %s %s%s %s %s\n", icon, idStr, keyStr, statusTag, t.Title)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolP("wait", "w", false, "list waiting tasks")
	listCmd.Flags().BoolP("done", "d", false, "list done tasks")
	rootCmd.AddCommand(listCmd)
}
