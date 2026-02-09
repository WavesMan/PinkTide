package stream

import (
	"context"
	"sync"
	"time"

	"PinkTide/internal/bili"
)

type Resolver struct {
	client          *bili.Client
	roomID          string
	refreshInterval time.Duration
	cache           *stringCache
}

func NewResolver(client *bili.Client, roomID string, refreshInterval time.Duration) *Resolver {
	return &Resolver{
		client:          client,
		roomID:          roomID,
		refreshInterval: refreshInterval,
		cache:           &stringCache{},
	}
}

func (r *Resolver) Start(ctx context.Context) {
	r.refresh(ctx)
	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refresh(ctx)
		}
	}
}

func (r *Resolver) Get() string {
	return r.cache.Get()
}

func (r *Resolver) refresh(ctx context.Context) {
	url, err := r.client.FetchPlayURL(ctx, r.roomID)
	if err != nil {
		return
	}
	if url == "" {
		return
	}
	r.cache.Set(url)
}

type stringCache struct {
	mu    sync.RWMutex
	value string
}

func (c *stringCache) Get() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

func (c *stringCache) Set(v string) {
	c.mu.Lock()
	c.value = v
	c.mu.Unlock()
}
