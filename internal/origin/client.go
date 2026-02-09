package origin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 封装回源请求，统一超时与头部伪装策略。
type Client struct {
	httpClient *http.Client
	headers    http.Header
}

// NewClient 创建回源客户端，timeout 控制整体请求超时。
func NewClient(timeout time.Duration, headers map[string]string) *Client {
	hdr := make(http.Header)
	for k, v := range headers {
		hdr.Set(k, v)
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		headers:    hdr,
	}
}

// Get 执行回源请求并返回响应体与状态码，请求失败返回错误。
func (c *Client) Get(ctx context.Context, target string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create request failed: %w", err)
	}

	for k, v := range c.headers {
		req.Header[k] = append([]string(nil), v...)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response failed: %w", err)
	}

	return data, resp.StatusCode, nil
}
