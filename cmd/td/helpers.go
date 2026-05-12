package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rgb-24bit/taskdeck/internal/model"
)

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
		return "\033[32m●\033[0m"
	case model.StatusWaiting:
		if t.ConditionType == model.ConditionTimeout && t.EnteredWaitAt != nil {
			return "\033[33m◐\033[0m"
		}
		return "\033[36m○\033[0m"
	case model.StatusDone:
		return "\033[90m✓\033[0m"
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
