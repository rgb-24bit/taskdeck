package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/rgb-24bit/taskdeck/internal/model"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(host string, port int) *Client {
	if host == "" {
		host = "localhost"
	}
	return &Client{
		BaseURL: "http://" + net.JoinHostPort(host, fmt.Sprintf("%d", port)),
		HTTP:    &http.Client{},
	}
}

func (c *Client) Add(tc model.TaskCreate) (*model.Task, bool, error) {
	data, _ := json.Marshal(tc)
	resp, err := c.HTTP.Post(c.BaseURL+"/api/tasks", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, false, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return nil, false, apiError(resp)
	}
	var task model.Task
	json.NewDecoder(resp.Body).Decode(&task)
	return &task, resp.StatusCode == 200, nil
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

func (c *Client) Get(identifier string) (*model.Task, error) {
	resp, err := c.HTTP.Get(fmt.Sprintf("%s/api/tasks/%s", c.BaseURL, identifier))
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

func (c *Client) Update(identifier string, tu model.TaskUpdate) (*model.Task, error) {
	data, _ := json.Marshal(tu)
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/tasks/%s", c.BaseURL, identifier), bytes.NewReader(data))
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

func (c *Client) Done(identifier string) error {
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/api/tasks/%s/done", c.BaseURL, identifier), "", nil)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return apiError(resp)
	}
	return nil
}

func (c *Client) Delete(identifier string) error {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/tasks/%s", c.BaseURL, identifier), nil)
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

func (c *Client) Activate(identifier string) (*model.Task, error) {
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/api/tasks/%s/activate", c.BaseURL, identifier), "", nil)
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

func (c *Client) Wait(identifier string, conditionType string, timeout int64) (*model.Task, error) {
	body := map[string]interface{}{
		"condition_type":    conditionType,
		"condition_timeout": timeout,
	}
	data, _ := json.Marshal(body)
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/api/tasks/%s/wait", c.BaseURL, identifier), "application/json", bytes.NewReader(data))
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

func (c *Client) Reorder(identifier string, req model.ReorderRequest) error {
	data, _ := json.Marshal(req)
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/api/tasks/%s/reorder", c.BaseURL, identifier), "application/json", bytes.NewReader(data))
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
