package segment

import (
	"context"
	"fmt"
	"net/http"

	"PinkTide/internal/origin"

	"golang.org/x/sync/singleflight"
)

// Fetcher 使用 SingleFlight 合并相同切片请求，降低源站并发。
type Fetcher struct {
	originClient *origin.Client
	group        singleflight.Group
}

// NewFetcher 复用回源客户端以统一超时与请求头。
func NewFetcher(originClient *origin.Client) *Fetcher {
	return &Fetcher{originClient: originClient}
}

// Fetch 拉取切片内容并返回字节数据，回源失败返回错误。
func (f *Fetcher) Fetch(ctx context.Context, target string) ([]byte, error) {
	value, err, _ := f.group.Do(target, func() (interface{}, error) {
		data, status, err := f.originClient.Get(ctx, target)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("origin status %d", status)
		}
		return data, nil
	})
	if err != nil {
		return nil, err
	}

	result, ok := value.([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid response type")
	}
	return result, nil
}
