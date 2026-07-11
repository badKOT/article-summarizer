package connectors

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const openRouterBaseURL = "https://openrouter.ai/api/v1"

var OpenRouterApiKey string

func GetOpenRouterCredits() ([]byte, error) {
	return doOpenRouterRequest("/credits", nil)
}

func GetOpenRouterModels(filter string) ([]byte, error) {
	query := url.Values{}
	query.Set("sort", "most-popular")
	if filter != "" {
		query.Set("q", filter)
	}

	return doOpenRouterRequest("/models", query)
}

func doOpenRouterRequest(path string, query url.Values) ([]byte, error) {
	endpoint := openRouterBaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+OpenRouterApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	return body, nil
}
