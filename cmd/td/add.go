package main

import (
	"fmt"
	"strings"

	"github.com/rgb-24bit/taskdeck/internal/model"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [flags] <title>",
	Short: "Add a task",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		title := strings.Join(args, " ")

		status := model.StatusActive
		if wait, _ := cmd.Flags().GetBool("wait"); wait {
			status = model.StatusWaiting
		}

		conditionType := model.ConditionManual
		var timeout int64
		if timeoutStr, _ := cmd.Flags().GetString("timeout"); timeoutStr != "" {
			conditionType = model.ConditionTimeout
			timeout = parseTimeout(timeoutStr)
		}

		sourceType := model.SourceManual
		sourceLabel := ""
		if sourceStr, _ := cmd.Flags().GetString("source"); sourceStr != "" {
			sourceType, sourceLabel = parseSource(sourceStr)
		}

		var key *string
		if keyStr, _ := cmd.Flags().GetString("key"); keyStr != "" {
			key = &keyStr
		}

		tc := model.TaskCreate{
			Key:              key,
			Title:            title,
			Status:           status,
			ConditionType:    conditionType,
			ConditionTimeout: timeout,
			SourceType:       sourceType,
			SourceLabel:      sourceLabel,
		}

		task, isUpdate, err := apiClient.Add(tc)
		if err != nil {
			return fmt.Errorf("add: %w", err)
		}
		if isUpdate {
			fmt.Printf("updated task #%d: %s\n", task.ID, task.Title)
		} else {
			fmt.Printf("created task #%d: %s\n", task.ID, task.Title)
		}
		return nil
	},
}

func init() {
	addCmd.Flags().BoolP("wait", "w", false, "create task in waiting status")
	addCmd.Flags().StringP("timeout", "t", "", "timeout duration (e.g. 30m, 2h)")
	addCmd.Flags().StringP("source", "s", "", "source type[:label] (manual, agent, external)")
	addCmd.Flags().StringP("key", "k", "", "unique string key for idempotent upsert")
	rootCmd.AddCommand(addCmd)
}
