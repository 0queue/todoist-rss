package todoist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	client *http.Client
	token  string
}

func New(token string) *Client {
	return &Client{
		client: &http.Client{Timeout: 10 * time.Second},
		token:  token,
	}
}

type Task struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	AddedAt     time.Time `json:"added_at"`
}

type result struct {
	Results    []Task `json:"results"`
	NextCursor string `json:"next_cursor"`
}

func (c *Client) GetTasks(ctx context.Context, label string) ([]Task, error) {
	u, err := url.Parse("https://api.todoist.com/api/v1/tasks")
	if err != nil {
		return nil, fmt.Errorf("url parse: %w", err)
	}

	q := u.Query()
	q.Set("label", label)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	response, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer response.Body.Close()

	dec := json.NewDecoder(response.Body)

	var res result
	err = dec.Decode(&res)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return res.Results, nil
}

func (c *Client) CloseTask(ctx context.Context, taskID string) error {
	s, err := url.JoinPath("https://todoist.com/api/v1/tasks", taskID, "close")
	if err != nil {
		return fmt.Errorf("join path: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	response, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer response.Body.Close()

	return nil
}
