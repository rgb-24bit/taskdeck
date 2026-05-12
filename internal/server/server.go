package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rgb-24bit/taskdeck/internal/model"
	"github.com/rgb-24bit/taskdeck/internal/store"
)

type Server struct {
	store  *store.Store
	mux    *http.ServeMux
}

func New(st *store.Store) *Server {
	s := &Server{store: st, mux: http.NewServeMux()}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/tasks", s.handleTasksCollection)
	s.mux.HandleFunc("/api/tasks/", s.handleTask)
	s.mux.HandleFunc("/api/tasks/cleanup", s.handleCleanup)
	s.mux.HandleFunc("/", s.handleWeb)
}

func (s *Server) handleTasksCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listTasks(w, r)
	case http.MethodPost:
		s.createTask(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	// /api/tasks/{id-or-key}, /api/tasks/{id-or-key}/done, etc.
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing task identifier", http.StatusBadRequest)
		return
	}

	id, err := s.resolveID(parts[0])
	if err != nil {
		http.Error(w, "task not found: "+parts[0], http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.getTask(w, r, id)
		case http.MethodPatch:
			s.updateTask(w, r, id)
		case http.MethodDelete:
			s.deleteTask(w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	action := parts[1]
	switch action {
	case "done":
		if r.Method == http.MethodPost {
			s.doneTask(w, r, id)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "activate":
		if r.Method == http.MethodPost {
			s.activateTask(w, r, id)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "wait":
		if r.Method == http.MethodPost {
			s.waitTask(w, r, id)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "reorder":
		if r.Method == http.MethodPost {
			s.reorderTask(w, r, id)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, r)
	}
}

// resolveID resolves a task identifier (numeric ID or string key) to an int64 ID.
func (s *Server) resolveID(identifier string) (int64, error) {
	if id, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		return id, nil
	}
	task, err := s.store.GetByKey(identifier)
	if err != nil {
		return 0, err
	}
	return task.ID, nil
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	params := model.ListParams{
		Status: r.URL.Query().Get("status"),
		From:   r.URL.Query().Get("from"),
		To:     r.URL.Query().Get("to"),
	}
	tasks, err := s.store.List(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []*model.Task{}
	}
	writeJSON(w, 200, tasks)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var tc model.TaskCreate
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		r.ParseForm()
		tc.Title = r.FormValue("title")
		tc.Status = r.FormValue("status")
		tc.SourceType = r.FormValue("source_type")
		tc.SourceLabel = r.FormValue("source_label")
		tc.ConditionType = r.FormValue("condition_type")
		if timeoutStr := r.FormValue("condition_timeout"); timeoutStr != "" {
			if d, err := parseDuration(timeoutStr); err == nil {
				tc.ConditionTimeout = int64(d.Seconds())
				tc.ConditionType = model.ConditionTimeout
			}
		}
		if key := r.FormValue("key"); key != "" {
			tc.Key = &key
		}
		if tc.Status == "" {
			tc.Status = model.StatusActive
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&tc); err != nil {
			http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	task, isUpdate, err := s.store.Create(tc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	status := 201
	if isUpdate {
		status = 200
	}
	writeJSON(w, status, task)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request, id int64) {
	task, err := s.store.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.ExecuteTemplate(w, "drawer.html", task)
		return
	}
	writeJSON(w, 200, task)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request, id int64) {
	var tu model.TaskUpdate
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		r.ParseForm()
		if v := r.FormValue("context"); v != "" {
			tu.Context = &v
		}
		if v := r.FormValue("title"); v != "" {
			tu.Title = &v
		}
		// An empty context sent via form means clear it
		if _, ok := r.Form["context"]; ok && r.FormValue("context") == "" {
			empty := ""
			tu.Context = &empty
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&tu); err != nil {
			http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	task, err := s.store.Update(id, tu)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.ExecuteTemplate(w, "drawer.html", task)
		return
	}
	writeJSON(w, 200, task)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.store.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) doneTask(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.store.Done(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) activateTask(w http.ResponseWriter, r *http.Request, id int64) {
	task, err := s.store.Activate(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, task)
}

func (s *Server) waitTask(w http.ResponseWriter, r *http.Request, id int64) {
	var body struct {
		ConditionType    string `json:"condition_type"`
		ConditionTimeout int64  `json:"condition_timeout"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	task, err := s.store.Wait(id, body.ConditionType, body.ConditionTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, task)
}

func (s *Server) reorderTask(w http.ResponseWriter, r *http.Request, id int64) {
	var req model.ReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.Reorder(id, req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.CleanupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	d, err := parseDuration(req.OlderThan)
	if err != nil {
		http.Error(w, "invalid duration: "+err.Error(), http.StatusBadRequest)
		return
	}
	n, err := s.store.Cleanup(time.Now().Add(-d))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, map[string]int64{"deleted": n})
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		renderIndex(w, s)
	} else if r.URL.Path == "/history" {
		renderHistory(w, s)
	} else {
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func parseDuration(s string) (time.Duration, error) {
	// Simple parsing: 7d -> 7 days
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	// Try standard Go duration first
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}
	// Support d (days)
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		return time.Duration(num) * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("invalid duration: %s", s)
}
