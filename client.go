package vkplayliveapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	baseUrl string
	client  HTTPClient
}

func NewClient(baseUrl string, client HTTPClient) *Client {
	return &Client{
		baseUrl: baseUrl,
		client:  client,
	}
}

func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader, response any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+url, body)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get API response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("response code is %v", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return nil
}
