package sdk

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type Config struct {
	BaseURL string
}

type Client struct {
	httpClient *http.Client

	config *Config

	Observations *observationsClient
	Findings     *findingsClient
}

func NewClient(client *http.Client, config *Config) *Client {
	return &Client{
		httpClient: client,
		config:     config,
		Observations: &observationsClient{
			httpClient: client,
			config:     config,
		},
		Findings: &findingsClient{
			httpClient: client,
			config:     config,
		},
	}
}

func (c *Client) NewRequest(ctx context.Context, method string, path string, reader io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s/%s", c.config.BaseURL, path), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}
