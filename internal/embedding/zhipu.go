package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ZhipuAPIKeyEnv        = "ZHIPU_API_KEY"
	defaultModel          = "embedding-3"
	defaultDimensions     = 2048
	defaultEndpoint       = "https://open.bigmodel.cn/api/paas/v4/embeddings"
	defaultBatchSize      = 64
	defaultRequestTimeout = 60 * time.Second
)

type Client struct {
	apiKey     string
	httpClient *http.Client
	endpoint   string
	model      string
	dimensions int
}

type requestPayload struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type responsePayload struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

type errorPayload struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
}

func NewClientFromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv(ZhipuAPIKeyEnv))
	if key == "" {
		return nil, fmt.Errorf("%s is not set", ZhipuAPIKeyEnv)
	}
	return &Client{
		apiKey: key,
		httpClient: &http.Client{
			Timeout: defaultRequestTimeout,
		},
		endpoint:   defaultEndpoint,
		model:      defaultModel,
		dimensions: defaultDimensions,
	}, nil
}

func (c *Client) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	results := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += defaultBatchSize {
		end := start + defaultBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch, err := c.embedBatch(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		results = append(results, batch...)
	}
	return results, nil
}

func (c *Client) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	payload := requestPayload{
		Model:      c.model,
		Input:      texts,
		Dimensions: c.dimensions,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build embedding request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var apiErr errorPayload
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
			if apiErr.Error != nil && apiErr.Error.Message != "" {
				return nil, fmt.Errorf("embedding API error: %s", apiErr.Error.Message)
			}
			if apiErr.Message != "" {
				return nil, fmt.Errorf("embedding API error: %s", apiErr.Message)
			}
		}
		return nil, fmt.Errorf("embedding API returned status %s", resp.Status)
	}

	var parsed responsePayload
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("embedding API returned %d vectors for %d inputs", len(parsed.Data), len(texts))
	}

	vectors := make([][]float32, len(texts))
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(texts) {
			return nil, fmt.Errorf("embedding API returned out-of-range index %d", item.Index)
		}
		vector := make([]float32, len(item.Embedding))
		for i, value := range item.Embedding {
			vector[i] = float32(value)
		}
		vectors[item.Index] = vector
	}
	return vectors, nil
}
