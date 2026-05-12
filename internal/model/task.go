package model

import "time"

const (
	StatusActive  = "active"
	StatusWaiting = "waiting"
	StatusDone    = "done"
)

const (
	SourceManual   = "manual"
	SourceAgent    = "agent"
	SourceExternal = "external"
)

const (
	ConditionManual  = "manual"
	ConditionTimeout = "timeout"
)

type Task struct {
	ID               int64      `json:"id"`
	Key              *string    `json:"key,omitempty"`
	Title            string     `json:"title"`
	Context          string     `json:"context"`
	Status           string     `json:"status"`
	SourceType       string     `json:"source_type"`
	SourceLabel      string     `json:"source_label"`
	ConditionType    string     `json:"condition_type"`
	ConditionTimeout int64      `json:"condition_timeout"`
	SortOrder        int64      `json:"sort_order"`
	EnteredWaitAt    *time.Time `json:"entered_wait_at"`
	CreatedAt        time.Time  `json:"created_at"`
	DoneAt           *time.Time `json:"done_at"`
	DeletedAt        *time.Time `json:"deleted_at"`
}

type TaskCreate struct {
	Key             *string `json:"key,omitempty"`
	Title           string  `json:"title"`
	Context         string `json:"context"`
	Status          string `json:"status"`
	SourceType      string `json:"source_type"`
	SourceLabel     string `json:"source_label"`
	ConditionType   string `json:"condition_type"`
	ConditionTimeout int64 `json:"condition_timeout"`
}

type TaskUpdate struct {
	Title            *string `json:"title,omitempty"`
	Context          *string `json:"context,omitempty"`
	ContextAppend    *string `json:"context_append,omitempty"`
	ConditionType    *string `json:"condition_type,omitempty"`
	ConditionTimeout *int64  `json:"condition_timeout,omitempty"`
	SortOrder        *int64  `json:"sort_order,omitempty"`
}

type ReorderRequest struct {
	After    string `json:"after"`
	Position string `json:"position"` // "top" | "bottom"
}

type CleanupRequest struct {
	OlderThan string `json:"older_than"`
}

type ListParams struct {
	Status string
	From   string
	To     string
}
