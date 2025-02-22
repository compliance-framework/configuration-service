package sdk

import (
	"net/http"
)

type Config struct {
	BaseURL string
}

type Client struct {
	httpClient *http.Client

	config *Config

	Results *resultClient
}

func NewClient(client *http.Client, config *Config) *Client {
	results := &resultClient{
		httpClient: client,
		config:     config,
	}
	return &Client{
		Results: results,
	}
}
