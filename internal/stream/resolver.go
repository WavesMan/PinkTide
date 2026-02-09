package stream

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"PinkTide/internal/bili"
)

// Resolver 负责定时刷新直播流地址并提供缓存读取。
type Resolver struct {
	client          *bili.Client
	roomID          string
	refreshInterval time.Duration
	cache           *stringCache
	logger          *slog.Logger
}

// NewResolver 创建刷新器并注入日志，用于异常可观测。
func NewResolver(client *bili.Client, roomID string, refreshInterval time.Duration, logger *slog.Logger) *Resolver {
	return &Resolver{
		client:          client,
		roomID:          roomID,
		refreshInterval: refreshInterval,
		cache:           &stringCache{},
		logger:          logger,
	}
}

// Start 启动定时刷新，ctx 取消后退出。
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

// Get 返回当前缓存的播放地址。
func (r *Resolver) Get() string {
	return r.cache.Get()
}

// refresh 单次拉取并更新缓存，失败只记录日志。
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

// stringCache 提供并发安全的字符串缓存。
type stringCache struct {
	mu    sync.RWMutex
	value string
}

// Get 读取当前值。
func (c *stringCache) Get() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// Set 更新缓存值。
func (c *stringCache) Set(v string) {
	c.mu.Lock()
	c.value = v
	c.mu.Unlock()
}
