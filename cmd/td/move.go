package main

import (
	"fmt"

	"github.com/rgb-24bit/taskdeck/internal/model"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <id|key> --after <ref> | --top | --bottom | --wait [--timeout <d>]",
	Short: "Reorder or move task to waiting pool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		id := args[0]

		if after, _ := cmd.Flags().GetString("after"); after != "" {
			if err := apiClient.Reorder(id, model.ReorderRequest{After: after}); err != nil {
				return fmt.Errorf("move: %w", err)
			}
			fmt.Printf("task %s moved after %s\n", id, after)
			return nil
		}

		if top, _ := cmd.Flags().GetBool("top"); top {
			if err := apiClient.Reorder(id, model.ReorderRequest{Position: "top"}); err != nil {
				return fmt.Errorf("move: %w", err)
			}
			fmt.Printf("task %s moved to top\n", id)
			return nil
		}

		if bottom, _ := cmd.Flags().GetBool("bottom"); bottom {
			if err := apiClient.Reorder(id, model.ReorderRequest{Position: "bottom"}); err != nil {
				return fmt.Errorf("move: %w", err)
			}
			fmt.Printf("task %s moved to bottom\n", id)
			return nil
		}

		if wait, _ := cmd.Flags().GetBool("wait"); wait {
			conditionType := model.ConditionManual
			var timeout int64
			if timeoutStr, _ := cmd.Flags().GetString("timeout"); timeoutStr != "" {
				conditionType = model.ConditionTimeout
				timeout = parseTimeout(timeoutStr)
			}
			task, err := apiClient.Wait(id, conditionType, timeout)
			if err != nil {
				return fmt.Errorf("move --wait: %w", err)
			}
			fmt.Printf("task #%d moved to waiting pool\n", task.ID)
			return nil
		}

		return fmt.Errorf("specify one of --after, --top, --bottom, or --wait")
	},
}

func init() {
	moveCmd.Flags().String("after", "", "move after specified task (by id or key)")
	moveCmd.Flags().Bool("top", false, "move to top of queue")
	moveCmd.Flags().Bool("bottom", false, "move to bottom of queue")
	moveCmd.Flags().Bool("wait", false, "move to waiting pool")
	moveCmd.Flags().String("timeout", "", "timeout duration (only with --wait)")
	moveCmd.MarkFlagsMutuallyExclusive("after", "top", "bottom", "wait")
	rootCmd.AddCommand(moveCmd)
}
