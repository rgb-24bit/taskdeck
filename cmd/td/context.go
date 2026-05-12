package main

import (
	"fmt"
	"strings"

	"github.com/rgb-24bit/taskdeck/internal/model"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context [show] <id|key>",
	Short: "View or modify task context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		task, err := apiClient.Get(args[0])
		if err != nil {
			return fmt.Errorf("context: %w", err)
		}
		if task.Context == "" {
			fmt.Println("(empty)")
		} else {
			fmt.Println(task.Context)
		}
		return nil
	},
}

var contextShowCmd = &cobra.Command{
	Use:   "show <id|key>",
	Short: "Show task context",
	Args:  cobra.ExactArgs(1),
	RunE:  contextCmd.RunE,
}

var contextAppendCmd = &cobra.Command{
	Use:   "append <id|key> <text>",
	Short: "Append text to task context",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		appendText := strings.Join(args[1:], " ")
		tu := model.TaskUpdate{ContextAppend: &appendText}
		task, err := apiClient.Update(args[0], tu)
		if err != nil {
			return fmt.Errorf("context: %w", err)
		}
		fmt.Printf("context appended to task #%d\n", task.ID)
		return nil
	},
}

var contextSetCmd = &cobra.Command{
	Use:   "set <id|key> <text>",
	Short: "Set task context",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureDaemonAndClient(); err != nil {
			return err
		}

		newContext := strings.Join(args[1:], " ")
		tu := model.TaskUpdate{Context: &newContext}
		task, err := apiClient.Update(args[0], tu)
		if err != nil {
			return fmt.Errorf("context: %w", err)
		}
		fmt.Printf("context updated for task #%d\n", task.ID)
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextShowCmd)
	contextCmd.AddCommand(contextAppendCmd)
	contextCmd.AddCommand(contextSetCmd)
	rootCmd.AddCommand(contextCmd)
}
