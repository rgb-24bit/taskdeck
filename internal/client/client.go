package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rgb-24bit/taskdeck/internal/model"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(port int) *Client {
	return &Client{
		BaseURL: fmt.Sprintf("http://localhost:%d", port),
		HTTP:    &http.Client{},
	}
}

func (c *Client) Add(tc model.TaskCreate) (*model.Task, error) {
	data, _ := json.Marshal(tc)
	resp, err := c.HTTP.Post(c.BaseURL+"/api/tasks", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return nil, apiError(resp)
	}
	var task model.Task
	json.NewDecoder(resp.Body).Decode(&task)
	return &task, nil
}

func (c *Client) List(params model.ListParams) ([]*model.Task, error) {
	query := ""
	if params.Status != "" {
		query += "status=" + params.Status
	}
	if params.From != "" {
		if query != "" {
			query += "&"
		}
		query += "from=" + params.From
	}
	if params.To != "" {
		if query != "" {
			query += "&"
		}
		query += "to=" + params.To
	}
	url := c.BaseURL + "/api/tasks"
	if query != "" {
		url += "?" + query
	}
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, apiError(resp)
	}
	var tasks []*model.Task
	json.NewDecoder(resp.Body).Decode(&tasks)
	return tasks, nil
}

func (c *Client) Get(id int64) (*model.Task, error) {
	resp, err := c.HTTP.Get(fmt.Sprintf("%s/api/tasks/%d", c.BaseURL, id))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, apiError(resp)
	}
	var task model.Task
	json.NewDecoder(resp.Body).Decode(&task)
	return &task, nil
}

func (c *Client) Update(id int64, tu model.TaskUpdate) (*model.Task, error) {
	data, _ := json.Marshal(tu)
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/tasks/%d", c.BaseURL, id), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, apiError(resp)
	}
	var task model.Task
	json.NewDecoder(resp.Body).Decode(&task)
	return &task, nil
}

func (c *Client) Done(id int64) error {
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/api/tasks/%d/done", c.BaseURL, id), "", nil)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return apiError(resp)
	}
	return nil
}

func (c *Client) Delete(id int64) error {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/tasks/%d", c.BaseURL, id), nil)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return apiError(resp)
	}
	return nil
}

func (c *Client) Activate(id int64) (*model.Task, error) {
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/api/tasks/%d/activate", c.BaseURL, id), "", nil)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, apiError(resp)
	}
	var task model.Task
	json.NewDecoder(resp.Body).Decode(&task)
	return &task, nil
}

func (c *Client) Wait(id int64, conditionType string, timeout int64) (*model.Task, error) {
	body := map[string]interface{}{
		"condition_type":    conditionType,
		"condition_timeout": timeout,
	}
	data, _ := json.Marshal(body)
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/api/tasks/%d/wait", c.BaseURL, id), "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, apiError(resp)
	}
	var task model.Task
	json.NewDecoder(resp.Body).Decode(&task)
	return &task, nil
}

func (c *Client) Reorder(id int64, req model.ReorderRequest) error {
	data, _ := json.Marshal(req)
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/api/tasks/%d/reorder", c.BaseURL, id), "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return apiError(resp)
	}
	return nil
}

func (c *Client) Cleanup(olderThan string) (int64, error) {
	data, _ := json.Marshal(map[string]string{"older_than": olderThan})
	resp, err := c.HTTP.Post(c.BaseURL+"/api/tasks/cleanup", "application/json", bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, apiError(resp)
	}
	var result map[string]int64
	json.NewDecoder(resp.Body).Decode(&result)
	return result["deleted"], nil
}

func apiError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
