package segment

import (
	"context"
	"fmt"
	"net/http"

	"PinkTide/internal/origin"
	"golang.org/x/sync/singleflight"
)

type Fetcher struct {
	originClient *origin.Client
	group        singleflight.Group
}

func NewFetcher(originClient *origin.Client) *Fetcher {
	return &Fetcher{originClient: originClient}
}

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
