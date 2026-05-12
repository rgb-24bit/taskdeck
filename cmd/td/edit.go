package main

import (
	"fmt"
	"strings"

	"github.com/rgb-24bit/taskdeck/internal/model"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a task",
}

var editTitleCmd = &cobra.Command{
	Use:   "title <id|key> <new title>",
	Short: "Edit task title",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		title := strings.Join(args[1:], " ")
		tu := model.TaskUpdate{Title: &title}
		task, err := apiClient.Update(args[0], tu)
		if err != nil {
			return fmt.Errorf("edit: %w", err)
		}
		fmt.Printf("updated task #%d: %s\n", task.ID, task.Title)
		return nil
	},
}

var editTimeoutCmd = &cobra.Command{
	Use:   "timeout <id|key> <duration>",
	Short: "Set timeout condition on task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		t := parseTimeout(args[1])
		conditionType := model.ConditionTimeout
		tu := model.TaskUpdate{
			ConditionType:    &conditionType,
			ConditionTimeout: &t,
		}
		task, err := apiClient.Update(args[0], tu)
		if err != nil {
			return fmt.Errorf("edit: %w", err)
		}
		fmt.Printf("updated task #%d: %s\n", task.ID, task.Title)
		return nil
	},
}

func init() {
	editCmd.AddCommand(editTitleCmd)
	editCmd.AddCommand(editTimeoutCmd)
	rootCmd.AddCommand(editCmd)
}
