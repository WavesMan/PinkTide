package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"PinkTide/internal/bili"
	"PinkTide/internal/config"
	"PinkTide/internal/origin"
	"PinkTide/internal/rewriter"
	"PinkTide/internal/segment"
	"PinkTide/internal/stream"
)

// Server 负责路由注册、依赖组织与 HTTP 生命周期管理。
type Server struct {
	cfg        config.Config
	httpServer *http.Server
	origin     *origin.Client
	biliClient *bili.Client
	rewriter   *rewriter.Rewriter
	resolver   *stream.Resolver
	segFetcher *segment.Fetcher
	serveMux   *http.ServeMux
	logger     *slog.Logger
}

// New 按配置构建服务依赖，必要参数无效时返回错误。
func New(cfg config.Config, logger *slog.Logger) (*Server, error) {
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Referer":    "https://live.bilibili.com/",
	}
	originClient := origin.NewClient(cfg.RequestTimeout, headers)
	rewriterInstance, err := rewriter.New(cfg.CDNPublicURL)
	if err != nil {
		return nil, err
	}
	biliClient := bili.NewClient(originClient)
	var resolver *stream.Resolver
	if cfg.BiliRoomID != "" {
		resolver = stream.NewResolver(biliClient, cfg.BiliRoomID, cfg.RefreshInterval, logger)
	}
	fetcher := segment.NewFetcher(originClient)

	mux := http.NewServeMux()
	srv := &Server{
		cfg:        cfg,
		origin:     originClient,
		biliClient: biliClient,
		rewriter:   rewriterInstance,
		resolver:   resolver,
		segFetcher: fetcher,
		serveMux:   mux,
		logger:     logger,
	}
	srv.registerRoutes()
	srv.httpServer = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	return srv, nil
}

// Start 启动 HTTP 服务并在必要时启动后台刷新任务。
func (s *Server) Start(ctx context.Context) error {
	if s.resolver != nil {
		go s.resolver.Start(ctx)
	}
	if s.logger != nil {
		s.logger.Info("server start", "addr", s.cfg.ListenAddr)
	}
	if err := s.httpServer.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen failed: %w", err)
	}
	return nil
}

// Shutdown 尝试在超时内关闭服务并释放资源。
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	if s.logger != nil {
		s.logger.Info("server shutdown")
	}
	return s.httpServer.Shutdown(ctx)
}
