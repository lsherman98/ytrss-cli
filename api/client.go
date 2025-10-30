package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type APIClient struct {
	client  *http.Client
	baseURL string
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		client:  &http.Client{},
		baseURL: baseURL,
	}
}

func (c *APIClient) do(method, path string, body io.Reader, v any) error {
	apiKey, err := GetApiKey()
	if err != nil {
		return fmt.Errorf("API key not set. Please set an API key")
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not connect to the API")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed: %s - %s", resp.Status, string(bodyBytes))
	}

	if v != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if err := json.Unmarshal(bodyBytes, v); err != nil {
			return fmt.Errorf("failed to decode JSON response (status %d): %w\nResponse body: %s", resp.StatusCode, err, string(bodyBytes))
		}
	}

	return nil
}
