package stream

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"PinkTide/internal/bili"
)

type Resolver struct {
	client          *bili.Client
	roomID          string
	refreshInterval time.Duration
	cache           *stringCache
	logger          *slog.Logger
}

func NewResolver(client *bili.Client, roomID string, refreshInterval time.Duration, logger *slog.Logger) *Resolver {
	return &Resolver{
		client:          client,
		roomID:          roomID,
		refreshInterval: refreshInterval,
		cache:           &stringCache{},
		logger:          logger,
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
		if r.logger != nil {
			r.logger.Warn("fetch play url failed", "room_id", r.roomID, "error", err)
		}
		return
	}
	if url == "" {
		if r.logger != nil {
			r.logger.Warn("empty play url", "room_id", r.roomID)
		}
		return
	}
	r.cache.Set(url)
	if r.logger != nil {
		r.logger.Debug("play url updated", "room_id", r.roomID)
	}
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
