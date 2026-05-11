package server

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/rgb-24bit/taskdeck/internal/model"
)

//go:embed templates/*
var templateFS embed.FS

var tmpl *template.Template

func init() {
	funcMap := template.FuncMap{
		"fmtTime":       fmtTime,
		"fmtRemain":     fmtRemain,
		"sourceIcon":    sourceIcon,
		"progressValue": progressValue,
	}
	tmpl = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html", "templates/partials/*.html"))
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func fmtRemain(t *time.Time, timeout int64) string {
	if t == nil || timeout == 0 {
		return ""
	}
	expireAt := t.Add(time.Duration(timeout) * time.Second)
	remain := time.Until(expireAt)
	if remain <= 0 {
		return "已过期"
	}
	if remain < time.Minute {
		return "即将激活"
	}
	if remain < time.Hour {
		return fmt.Sprintf("%d分钟后", int(remain.Minutes()))
	}
	if remain < 24*time.Hour {
		return fmt.Sprintf("%d小时后", int(remain.Hours()))
	}
	return fmt.Sprintf("%d天后", int(remain.Hours()/24))
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

func progressValue(enteredAt *time.Time, timeout int64) int {
	if enteredAt == nil || timeout == 0 {
		return 0
	}
	elapsed := time.Since(*enteredAt)
	total := time.Duration(timeout) * time.Second
	pct := int(elapsed * 100 / total)
	if pct > 100 {
		pct = 100
	}
	return pct
}

func renderIndex(w http.ResponseWriter, s *Server) {
	active, _ := s.store.List(model.ListParams{Status: model.StatusActive})
	waiting, _ := s.store.List(model.ListParams{Status: model.StatusWaiting})
	if active == nil {
		active = []*model.Task{}
	}
	if waiting == nil {
		waiting = []*model.Task{}
	}

	// Group waiting by condition type
	now := time.Now()
	var timeoutGroup, manualGroup []*model.Task
	for _, t := range waiting {
		if t.ConditionType == model.ConditionTimeout && t.ConditionTimeout > 0 && t.EnteredWaitAt != nil {
			expireAt := t.EnteredWaitAt.Add(time.Duration(t.ConditionTimeout) * time.Second)
			if now.After(expireAt) {
				t.ConditionType = "expired"
			}
			timeoutGroup = append(timeoutGroup, t)
		} else {
			manualGroup = append(manualGroup, t)
		}
	}

	data := map[string]interface{}{
		"Active":       active,
		"TimeoutGroup": timeoutGroup,
		"ManualGroup":  manualGroup,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "index.html", data)
}

func renderHistory(w http.ResponseWriter, s *Server) {
	from := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	tasks, _ := s.store.List(model.ListParams{Status: model.StatusDone, From: from})
	if tasks == nil {
		tasks = []*model.Task{}
	}
	data := map[string]interface{}{
		"Tasks": tasks,
		"From":  time.Now().AddDate(0, 0, -7).Format("2006-01-02"),
		"To":    time.Now().Format("2006-01-02"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "history.html", data)
}
