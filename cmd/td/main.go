package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rgb-24bit/taskdeck/internal/client"
	"github.com/rgb-24bit/taskdeck/internal/config"
	"github.com/rgb-24bit/taskdeck/internal/daemon"
	"github.com/rgb-24bit/taskdeck/internal/model"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "serve":
		cmdServe()
	case "add":
		cmdAdd(args)
	case "list":
		cmdList(args)
	case "show":
		cmdShow(args)
	case "edit":
		cmdEdit(args)
	case "done":
		cmdDone(args)
	case "delete":
		cmdDelete(args)
	case "move":
		cmdMove(args)
	case "activate":
		cmdActivate(args)
	case "context":
		cmdContext(args)
	case "cleanup":
		cmdCleanup(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func mustConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func mustClient() *client.Client {
	cfg := mustConfig()
	return client.New(cfg.Host, cfg.Port)
}

func ensureDaemon() {
	cfg := mustConfig()
	if !isLocalHost(cfg.Host) {
		return
	}
	if daemon.IsRunning(cfg.PidPath) {
		return
	}
	fmt.Println("starting taskdeck daemon...")
	if err := daemon.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "start daemon: %v\n", err)
		os.Exit(1)
	}
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if daemon.IsRunning(cfg.PidPath) {
			return
		}
	}
	fmt.Fprintln(os.Stderr, "daemon failed to start")
	os.Exit(1)
}

func isLocalHost(host string) bool {
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func cmdServe() {
	cfg := mustConfig()
	if err := daemon.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

func cmdAdd(args []string) {
	ensureDaemon()
	cl := mustClient()

	status := model.StatusActive
	conditionType := model.ConditionManual
	sourceType := model.SourceManual
	sourceLabel := ""
	var timeout int64
	var key *string

	// Parse flags
	var titleParts []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-w", "--wait":
			status = model.StatusWaiting
		case "-t", "--timeout":
			i++
			if i < len(args) {
				conditionType = model.ConditionTimeout
				timeout = parseTimeout(args[i])
			}
		case "-s", "--source":
			i++
			if i < len(args) {
				sourceType, sourceLabel = parseSource(args[i])
			}
		case "-k", "--key":
			i++
			if i < len(args) {
				key = &args[i]
			}
		default:
			titleParts = append(titleParts, args[i])
		}
	}

	title := strings.Join(titleParts, " ")
	if title == "" {
		fmt.Fprintln(os.Stderr, "usage: td add [-w] [-t duration] [-s source[:label]] [-k key] <title>")
		os.Exit(1)
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

	task, isUpdate, err := cl.Add(tc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "add: %v\n", err)
		os.Exit(1)
	}
	if isUpdate {
		fmt.Printf("updated task #%d: %s\n", task.ID, task.Title)
	} else {
		fmt.Printf("created task #%d: %s\n", task.ID, task.Title)
	}
}

func cmdList(args []string) {
	ensureDaemon()
	cl := mustClient()

	status := ""
	waitMode := false
	for _, a := range args {
		if a == "-w" || a == "--wait" {
			waitMode = true
		} else if a == "-d" || a == "--done" {
			status = model.StatusDone
		}
	}

	if waitMode {
		status = model.StatusWaiting
	} else if status == "" {
		status = model.StatusActive
	}

	tasks, err := cl.List(model.ListParams{Status: status})
	if err != nil {
		fmt.Fprintf(os.Stderr, "list: %v\n", err)
		os.Exit(1)
	}

	if len(tasks) == 0 {
		fmt.Println("no tasks")
		return
	}

	// Rich output
	for _, t := range tasks {
		icon := statusIcon(t)
		idStr := fmt.Sprintf("\033[33m#%d\033[0m", t.ID) // yellow
		statusTag := statusTag(t)
		keyStr := ""
		if t.Key != nil {
			keyStr = fmt.Sprintf(" \033[90m[%s]\033[0m", *t.Key)
		}
		fmt.Printf("  %s %s%s %s %s\n", icon, idStr, keyStr, statusTag, t.Title)
	}
}

func cmdShow(args []string) {
	ensureDaemon()
	cl := mustClient()

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: td show <id|key>")
		os.Exit(1)
	}
	task, err := cl.Get(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "show: %v\n", err)
		os.Exit(1)
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
}

func cmdEdit(args []string) {
	ensureDaemon()
	cl := mustClient()

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: td edit <id|key> title <new title>")
		fmt.Fprintln(os.Stderr, "       td edit <id|key> timeout <duration>")
		os.Exit(1)
	}

	var tu model.TaskUpdate
	switch args[1] {
	case "title":
		title := strings.Join(args[2:], " ")
		tu.Title = &title
	case "timeout":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: td edit <id|key> timeout <duration>")
			os.Exit(1)
		}
		t := parseTimeout(args[2])
		conditionType := model.ConditionTimeout
		tu.ConditionType = &conditionType
		tu.ConditionTimeout = &t
	default:
		fmt.Fprintf(os.Stderr, "unknown field: %s\n", args[1])
		os.Exit(1)
	}

	task, err := cl.Update(args[0], tu)
	if err != nil {
		fmt.Fprintf(os.Stderr, "edit: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("updated task #%d: %s\n", task.ID, task.Title)
}

func cmdDone(args []string) {
	ensureDaemon()
	cl := mustClient()

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: td done <id|key>")
		os.Exit(1)
	}
	if err := cl.Done(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "done: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("task %s marked done\n", args[0])
}

func cmdDelete(args []string) {
	ensureDaemon()
	cl := mustClient()

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: td delete <id|key>")
		os.Exit(1)
	}
	if err := cl.Delete(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "delete: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("task %s deleted\n", args[0])
}

func cmdMove(args []string) {
	ensureDaemon()
	cl := mustClient()

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: td move <id|key> --after <id|key> | --top | --bottom | --wait")
		os.Exit(1)
	}

	switch args[1] {
	case "--wait", "-w":
		conditionType := model.ConditionManual
		var timeout int64
		if len(args) > 2 && args[2] == "--timeout" {
			conditionType = model.ConditionTimeout
			timeout = parseTimeout(args[3])
		}
		task, err := cl.Wait(args[0], conditionType, timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "move --wait: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("task #%d moved to waiting pool\n", task.ID)
	case "--after":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: td move <id|key> --after <target-id|key>")
			os.Exit(1)
		}
		if err := cl.Reorder(args[0], model.ReorderRequest{After: args[2]}); err != nil {
			fmt.Fprintf(os.Stderr, "move: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("task %s moved after %s\n", args[0], args[2])
	case "--top":
		if err := cl.Reorder(args[0], model.ReorderRequest{Position: "top"}); err != nil {
			fmt.Fprintf(os.Stderr, "move: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("task %s moved to top\n", args[0])
	case "--bottom":
		if err := cl.Reorder(args[0], model.ReorderRequest{Position: "bottom"}); err != nil {
			fmt.Fprintf(os.Stderr, "move: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("task %s moved to bottom\n", args[0])
	default:
		fmt.Fprintf(os.Stderr, "unknown move target: %s\n", args[1])
		os.Exit(1)
	}
}

func cmdActivate(args []string) {
	ensureDaemon()
	cl := mustClient()

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: td activate <id|key>")
		os.Exit(1)
	}
	task, err := cl.Activate(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "activate: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("task #%d activated: %s\n", task.ID, task.Title)
}

func cmdContext(args []string) {
	ensureDaemon()
	cl := mustClient()

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: td context <id|key> [--append <text>]")
		os.Exit(1)
	}

	if len(args) >= 3 && args[1] == "--append" {
		appendText := strings.Join(args[2:], " ")
		var tu model.TaskUpdate
		tu.ContextAppend = &appendText
		task, err := cl.Update(args[0], tu)
		if err != nil {
			fmt.Fprintf(os.Stderr, "context: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("context appended to task #%d\n", task.ID)
	} else if len(args) >= 3 && args[1] == "--set" {
		newContext := strings.Join(args[2:], " ")
		var tu model.TaskUpdate
		tu.Context = &newContext
		task, err := cl.Update(args[0], tu)
		if err != nil {
			fmt.Fprintf(os.Stderr, "context: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("context updated for task #%d\n", task.ID)
	} else {
		// Display context
		task, err := cl.Get(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "context: %v\n", err)
			os.Exit(1)
		}
		if task.Context == "" {
			fmt.Println("(empty)")
		} else {
			fmt.Println(task.Context)
		}
	}
}

func cmdCleanup(args []string) {
	ensureDaemon()
	cl := mustClient()

	olderThan := "30d"
	if len(args) > 0 {
		olderThan = args[0]
	}

	n, err := cl.Cleanup(olderThan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cleanup: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("cleaned up %d done tasks older than %s\n", n, olderThan)
}

func printUsage() {
	fmt.Println(`taskdeck - context task management

commands:
  serve                   start the daemon
  add [-w] [-t dur] [-s type[:label]] [-k key] <title>  add a task
  list [-w] [-d]          list active/waiting/done tasks
  show <id|key>           show task details
  edit <id|key> title <text>  edit task title
  edit <id|key> timeout <dur> set timeout condition
  done <id|key>           mark task done
  delete <id|key>         hard delete task
  move <id|key> --after <id|key>  reorder in queue
  move <id|key> --top|--bottom move to top/bottom
  move <id|key> --wait [-t dur] move to waiting pool
  activate <id|key>       activate from waiting pool
  context <id|key>        show task context
  context <id|key> --append <text>  append to context
  context <id|key> --set <text>     set context
  cleanup [duration]      delete done tasks older than duration (default 30d)`)
}

func parseSource(s string) (string, string) {
	parts := strings.SplitN(s, ":", 2)
	typ := parts[0]
	label := ""
	if len(parts) == 2 {
		label = parts[1]
	}
	switch typ {
	case model.SourceAgent, model.SourceExternal, model.SourceManual:
		return typ, label
	default:
		fmt.Fprintf(os.Stderr, "unknown source type '%s', using 'manual'\n", typ)
		return model.SourceManual, ""
	}
}

func parseTimeout(s string) int64 {
	s = strings.TrimSpace(s)
	// Check for hour marker
	if strings.HasSuffix(s, "h") {
		numStr := strings.TrimSuffix(s, "h")
		num, err := strconv.ParseFloat(numStr, 64)
		if err == nil {
			return int64(num * 3600)
		}
	}
	if strings.HasSuffix(s, "m") {
		numStr := strings.TrimSuffix(s, "m")
		num, err := strconv.ParseFloat(numStr, 64)
		if err == nil {
			return int64(num * 60)
		}
	}
	if strings.HasSuffix(s, "s") {
		numStr := strings.TrimSuffix(s, "s")
		num, err := strconv.ParseFloat(numStr, 64)
		if err == nil {
			return int64(num)
		}
	}
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return num
}

func statusIcon(t *model.Task) string {
	switch t.Status {
	case model.StatusActive:
		return "\033[32m●\033[0m" // green
	case model.StatusWaiting:
		if t.ConditionType == model.ConditionTimeout && t.EnteredWaitAt != nil {
			return "\033[33m◐\033[0m" // yellow half
		}
		return "\033[36m○\033[0m" // cyan
	case model.StatusDone:
		return "\033[90m✓\033[0m" // grey check
	default:
		return " "
	}
}

func statusTag(t *model.Task) string {
	switch t.Status {
	case model.StatusActive:
		return "\033[32m[active]\033[0m"
	case model.StatusWaiting:
		if t.ConditionType == model.ConditionTimeout {
			remain := ""
			if t.EnteredWaitAt != nil && t.ConditionTimeout > 0 {
				remain = " " + fmtRemainCLI(t)
			}
			return "\033[33m[wait:timeout" + remain + "]\033[0m"
		}
		return "\033[36m[wait:manual]\033[0m"
	case model.StatusDone:
		return "\033[90m[done]\033[0m"
	default:
		return ""
	}
}

func sourceIcon(st string) string {
	switch st {
	case model.SourceAgent:
		return "🤖"
	case model.SourceExternal:
		return "🔗"
	default:
		return "👤"
	}
}

func fmtRemainCLI(t *model.Task) string {
	if t.EnteredWaitAt == nil || t.ConditionTimeout == 0 {
		return ""
	}
	expireAt := t.EnteredWaitAt.Add(time.Duration(t.ConditionTimeout) * time.Second)
	remain := time.Until(expireAt)
	if remain <= 0 {
		return "expired"
	}
	if remain < time.Minute {
		return "soon"
	}
	if remain < time.Hour {
		return fmt.Sprintf("%dm", int(remain.Minutes()))
	}
	if remain < 24*time.Hour {
		return fmt.Sprintf("%dh", int(remain.Hours()))
	}
	return fmt.Sprintf("%dd", int(remain.Hours()/24))
}

func fmtDuration(seconds int64) string {
	if seconds >= 3600 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	if seconds >= 60 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}

