package connectors

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

const jinaBaseUrl = "https://r.jina.ai"

var JinaApiKey string

func CallJina(url string) (string, error) {
	endpoint := jinaBaseUrl + "/" + url
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+JinaApiKey)

	res, err := http.DefaultClient.Do(req)
	log.Printf("Jina RS: %s", res.Status)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	log.Printf("Jina RS: Body size is %d", len(body))
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	return string(body), nil
}
