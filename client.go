package vkplayliveapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPClient interface for standard http-client to perform requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type client struct {
	baseURL string
	client  HTTPClient
}

func newClient(baseURL string, httpClient HTTPClient) *client {
	return &client{
		baseURL: baseURL,
		client:  httpClient,
	}
}

func (c *client) doRequest(ctx context.Context, method, url string, body io.Reader, response any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+url, body)
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
