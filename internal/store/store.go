package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rgb-24bit/taskdeck/internal/model"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			title           TEXT NOT NULL,
			context         TEXT DEFAULT '',
			status          TEXT NOT NULL DEFAULT 'active',
			source_type     TEXT NOT NULL DEFAULT 'manual',
			source_label    TEXT DEFAULT '',
			condition_type  TEXT DEFAULT 'manual',
			condition_timeout INTEGER DEFAULT 0,
			sort_order      INTEGER NOT NULL DEFAULT 0,
			entered_wait_at TEXT,
			created_at      TEXT NOT NULL,
			done_at         TEXT,
			deleted_at      TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_sort ON tasks(sort_order);
	`)
	return err
}

func (s *Store) Create(tc model.TaskCreate) (*model.Task, error) {
	now := time.Now().Format(time.RFC3339)
	var maxOrder int64
	s.db.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM tasks WHERE status = 'active' AND deleted_at IS NULL").Scan(&maxOrder)

	task := &model.Task{
		Title:            tc.Title,
		Context:          tc.Context,
		Status:           tc.Status,
		SourceType:       tc.SourceType,
		SourceLabel:      tc.SourceLabel,
		ConditionType:    tc.ConditionType,
		ConditionTimeout: tc.ConditionTimeout,
		SortOrder:        maxOrder + 1,
		CreatedAt:        time.Now(),
	}

	if task.Status == "" {
		task.Status = model.StatusActive
	}
	if task.SourceType == "" {
		task.SourceType = model.SourceManual
	}
	if task.ConditionType == "" {
		task.ConditionType = model.ConditionManual
	}

	nowStr := now
	var enteredWaitAt *string
	if task.Status == model.StatusWaiting {
		enteredWaitAt = &nowStr
	}

	result, err := s.db.Exec(
		`INSERT INTO tasks (title, context, status, source_type, source_label, condition_type, condition_timeout, sort_order, entered_wait_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.Title, task.Context, task.Status, task.SourceType, task.SourceLabel,
		task.ConditionType, task.ConditionTimeout, task.SortOrder, enteredWaitAt, nowStr,
	)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	id, _ := result.LastInsertId()
	task.ID = id
	return task, nil
}

func (s *Store) Get(id int64) (*model.Task, error) {
	var t taskRow
	err := s.db.QueryRow(
		`SELECT id, title, context, status, source_type, source_label, condition_type, condition_timeout, sort_order, entered_wait_at, created_at, done_at, deleted_at
		 FROM tasks WHERE id = ? AND deleted_at IS NULL`, id,
	).Scan(&t.ID, &t.Title, &t.Context, &t.Status, &t.SourceType, &t.SourceLabel,
		&t.ConditionType, &t.ConditionTimeout, &t.SortOrder, &t.EnteredWaitAt, &t.CreatedAt, &t.DoneAt, &t.DeletedAt)
	if err != nil {
		return nil, err
	}
	return t.toTask(), nil
}

func (s *Store) List(params model.ListParams) ([]*model.Task, error) {
	var query string
	var args []interface{}

	switch params.Status {
	case model.StatusDone:
		query = `SELECT id, title, context, status, source_type, source_label, condition_type, condition_timeout, sort_order, entered_wait_at, created_at, done_at, deleted_at
			 FROM tasks WHERE status = 'done' AND deleted_at IS NULL`
		if params.From != "" {
			query += " AND done_at >= ?"
			args = append(args, params.From)
		}
		if params.To != "" {
			query += " AND done_at <= ?"
			args = append(args, params.To)
		}
		query += " ORDER BY done_at DESC"
	default:
		query = `SELECT id, title, context, status, source_type, source_label, condition_type, condition_timeout, sort_order, entered_wait_at, created_at, done_at, deleted_at
			 FROM tasks WHERE status IN ('active', 'waiting') AND deleted_at IS NULL`
		if params.Status == model.StatusActive || params.Status == model.StatusWaiting {
			query += " AND status = ?"
			args = append(args, params.Status)
		}
		query += " ORDER BY sort_order ASC"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		var t taskRow
		err := rows.Scan(&t.ID, &t.Title, &t.Context, &t.Status, &t.SourceType, &t.SourceLabel,
			&t.ConditionType, &t.ConditionTimeout, &t.SortOrder, &t.EnteredWaitAt, &t.CreatedAt, &t.DoneAt, &t.DeletedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t.toTask())
	}
	return tasks, nil
}

func (s *Store) Update(id int64, tu model.TaskUpdate) (*model.Task, error) {
	task, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	if tu.Title != nil {
		task.Title = *tu.Title
	}
	if tu.ConditionType != nil {
		task.ConditionType = *tu.ConditionType
	}
	if tu.ConditionTimeout != nil {
		task.ConditionTimeout = *tu.ConditionTimeout
	}
	if tu.SortOrder != nil {
		task.SortOrder = *tu.SortOrder
	}
	if tu.ContextAppend != nil {
		task.Context += *tu.ContextAppend
	} else if tu.Context != nil {
		task.Context = *tu.Context
	}

	_, err = s.db.Exec(
		`UPDATE tasks SET title=?, context=?, condition_type=?, condition_timeout=?, sort_order=? WHERE id=?`,
		task.Title, task.Context, task.ConditionType, task.ConditionTimeout, task.SortOrder, task.ID,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Store) Done(id int64) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE tasks SET status='done', done_at=? WHERE id=? AND deleted_at IS NULL`, now, id)
	return err
}

func (s *Store) Delete(id int64) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE tasks SET deleted_at=? WHERE id=?`, now, id)
	return err
}

func (s *Store) Activate(id int64) (*model.Task, error) {
	task, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if task.Status != model.StatusWaiting {
		return nil, fmt.Errorf("task is not in waiting pool")
	}

	var maxOrder int64
	s.db.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM tasks WHERE status = 'active' AND deleted_at IS NULL").Scan(&maxOrder)

	_, err = s.db.Exec(
		`UPDATE tasks SET status='active', entered_wait_at=NULL, sort_order=? WHERE id=?`,
		maxOrder+1, id,
	)
	if err != nil {
		return nil, err
	}
	task.Status = model.StatusActive
	task.SortOrder = maxOrder + 1
	task.EnteredWaitAt = nil
	return task, nil
}

func (s *Store) Wait(id int64, conditionType string, timeout int64) (*model.Task, error) {
	task, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().Format(time.RFC3339)

	if conditionType == "" {
		conditionType = task.ConditionType
	}
	if conditionType == "" {
		conditionType = model.ConditionManual
	}

	_, err = s.db.Exec(
		`UPDATE tasks SET status='waiting', entered_wait_at=?, condition_type=?, condition_timeout=? WHERE id=?`,
		now, conditionType, timeout, id,
	)
	if err != nil {
		return nil, err
	}
	task.Status = model.StatusWaiting
	nowTime := time.Now()
	task.EnteredWaitAt = &nowTime
	task.ConditionType = conditionType
	task.ConditionTimeout = timeout
	return task, nil
}

func (s *Store) Reorder(id int64, req model.ReorderRequest) error {
	task, err := s.Get(id)
	if err != nil {
		return err
	}
	if task.Status != model.StatusActive {
		return fmt.Errorf("can only reorder active tasks")
	}

	switch req.Position {
	case "top":
		var minOrder int64
		s.db.QueryRow("SELECT COALESCE(MIN(sort_order), 0) FROM tasks WHERE status = 'active' AND deleted_at IS NULL").Scan(&minOrder)
		_, err = s.db.Exec(`UPDATE tasks SET sort_order = sort_order + 1 WHERE status = 'active' AND deleted_at IS NULL`)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(`UPDATE tasks SET sort_order = ? WHERE id = ?`, minOrder-1, id)
		return err
	case "bottom":
		var maxOrder int64
		s.db.QueryRow("SELECT COALESCE(MAX(sort_order), 0) FROM tasks WHERE status = 'active' AND deleted_at IS NULL").Scan(&maxOrder)
		_, err = s.db.Exec(`UPDATE tasks SET sort_order = ? WHERE id = ?`, maxOrder+1, id)
		return err
	default:
		afterTask, err := s.Get(req.AfterID)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(
			`UPDATE tasks SET sort_order = sort_order + 1 WHERE status = 'active' AND deleted_at IS NULL AND sort_order > ?`,
			afterTask.SortOrder,
		)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(`UPDATE tasks SET sort_order = ? WHERE id = ?`, afterTask.SortOrder+1, id)
		return err
	}
}

func (s *Store) Cleanup(olderThan time.Time) (int64, error) {
	result, err := s.db.Exec(
		`UPDATE tasks SET deleted_at = ? WHERE status = 'done' AND done_at <= ? AND deleted_at IS NULL`,
		time.Now().Format(time.RFC3339), olderThan.Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func (s *Store) GetExpiredWaiting() ([]*model.Task, error) {
	now := time.Now()
	rows, err := s.db.Query(
		`SELECT id, title, context, status, source_type, source_label, condition_type, condition_timeout, sort_order, entered_wait_at, created_at, done_at, deleted_at
		 FROM tasks WHERE status = 'waiting' AND condition_type = 'timeout' AND condition_timeout > 0 AND entered_wait_at IS NOT NULL AND deleted_at IS NULL`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		var t taskRow
		err := rows.Scan(&t.ID, &t.Title, &t.Context, &t.Status, &t.SourceType, &t.SourceLabel,
			&t.ConditionType, &t.ConditionTimeout, &t.SortOrder, &t.EnteredWaitAt, &t.CreatedAt, &t.DoneAt, &t.DeletedAt)
		if err != nil {
			return nil, err
		}
		task := t.toTask()
		if task.EnteredWaitAt != nil {
			expireAt := task.EnteredWaitAt.Add(time.Duration(task.ConditionTimeout) * time.Second)
			if now.After(expireAt) || now.Equal(expireAt) {
				tasks = append(tasks, task)
			}
		}
	}
	return tasks, nil
}

// taskRow is used for scanning from SQLite (all TEXT time fields scan to *string)
type taskRow struct {
	ID               int64
	Title            string
	Context          string
	Status           string
	SourceType       string
	SourceLabel      string
	ConditionType    string
	ConditionTimeout int64
	SortOrder        int64
	EnteredWaitAt    *string
	CreatedAt        string
	DoneAt           *string
	DeletedAt        *string
}

func (r *taskRow) toTask() *model.Task {
	t := &model.Task{
		ID:               r.ID,
		Title:            r.Title,
		Context:          r.Context,
		Status:           r.Status,
		SourceType:       r.SourceType,
		SourceLabel:      r.SourceLabel,
		ConditionType:    r.ConditionType,
		ConditionTimeout: r.ConditionTimeout,
		SortOrder:        r.SortOrder,
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, r.CreatedAt)
	t.EnteredWaitAt = parseTimeStr(r.EnteredWaitAt)
	t.DoneAt = parseTimeStr(r.DoneAt)
	t.DeletedAt = parseTimeStr(r.DeletedAt)
	return t
}

func parseTimeStr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}
