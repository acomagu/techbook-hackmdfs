package hackmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	sessionID  string
	httpClient *http.Client
}

// NewClient creates a Client instance. If the httpClient is nil, the
// http.DefaultClient is used.
func NewClient(sessionID string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		sessionID:  sessionID,
		httpClient: httpClient,
	}
}

func (c *Client) GetNoteBody(ctx context.Context, id string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://hackmd.io/%s/download", id), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, err
}

type HistoryItem struct {
	ID   string   `json:"id"`
	Text string   `json:"text"`
	Time int      `json:"time"`
	Tags []string `json:"tags"`
}

type HistoryResponse struct {
	History []*HistoryItem `json:"history"`
}

func (c *Client) GetHistory(ctx context.Context) ([]*HistoryItem, error) {
	req, err := http.NewRequest(http.MethodGet, "https://hackmd.io/history", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.AddCookie(&http.Cookie{
		Name:  "connect.sid",
		Value: c.sessionID,
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	tresp := new(HistoryResponse)
	if err := json.NewDecoder(resp.Body).Decode(tresp); err != nil {
		return nil, err
	}

	return tresp.History, nil
}

func (c *Client) Write(ctx context.Context, id string, offset uint64, data []byte) error {
	panic("this function is not implemented yet")
}
