package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

const pollInterval = time.Second

type Client struct {
	baseURL      *url.URL
	httpClient   *http.Client
	pollInterval time.Duration
	timeout      time.Duration
}

type response struct {
	Status   string `json:"status"`
	JobID    string `json:"job_id,omitempty"`
	Title    string `json:"title,omitempty"`
	Filename string `json:"filename,omitempty"`
	Error    string `json:"error,omitempty"`
}

func NewClient(rawURL string, timeout time.Duration) (*Client, error) {
	if rawURL == "" {
		return nil, nil
	}
	if timeout <= 0 {
		return nil, errors.New("downloader timeout must be positive")
	}

	baseURL, err := url.Parse(rawURL)
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" {
		return nil, fmt.Errorf("invalid downloader URL %q", rawURL)
	}

	return &Client{
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: timeout},
		pollInterval: pollInterval,
		timeout:      timeout,
	}, nil
}

func (c *Client) Download(ctx context.Context, query string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return "", "", err
	}

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path.Join(c.baseURL.Path, "/v1/tracks")})
	result, statusCode, err := c.request(ctx, http.MethodPost, endpoint.String(), body)
	if err != nil {
		return "", "", err
	}

	if statusCode == http.StatusOK && result.Status == "ready" {
		return result.Filename, result.Title, nil
	}
	if statusCode != http.StatusAccepted || result.JobID == "" {
		return "", "", responseError(statusCode, result)
	}

	jobEndpoint := c.baseURL.ResolveReference(&url.URL{Path: path.Join(c.baseURL.Path, "/v1/jobs/", result.JobID)})
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-ticker.C:
			result, statusCode, err = c.request(ctx, http.MethodGet, jobEndpoint.String(), nil)
			if err != nil {
				return "", "", err
			}
			if statusCode != http.StatusOK {
				return "", "", responseError(statusCode, result)
			}
			switch result.Status {
			case "ready":
				return result.Filename, result.Title, nil
			case "failed":
				return "", "", responseError(statusCode, result)
			}
		}
	}
}

func (c *Client) request(ctx context.Context, method, endpoint string, body []byte) (response, int, error) {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return response{}, 0, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return response{}, 0, fmt.Errorf("downloader request failed: %w", err)
	}
	defer httpResponse.Body.Close()

	var result response
	if err := json.NewDecoder(httpResponse.Body).Decode(&result); err != nil {
		return response{}, httpResponse.StatusCode, fmt.Errorf("invalid downloader response: %w", err)
	}
	return result, httpResponse.StatusCode, nil
}

func responseError(statusCode int, result response) error {
	if result.Error != "" {
		return errors.New(result.Error)
	}
	return fmt.Errorf("downloader returned HTTP %d with status %q", statusCode, result.Status)
}
