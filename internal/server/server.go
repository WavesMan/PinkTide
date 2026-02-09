package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"PinkTide/internal/bili"
	"PinkTide/internal/config"
	"PinkTide/internal/origin"
	"PinkTide/internal/rewriter"
	"PinkTide/internal/segment"
	"PinkTide/internal/stream"
)

type Server struct {
	cfg        config.Config
	httpServer *http.Server
	origin     *origin.Client
	biliClient *bili.Client
	rewriter   *rewriter.Rewriter
	resolver   *stream.Resolver
	segFetcher *segment.Fetcher
	serveMux   *http.ServeMux
}

func New(cfg config.Config) (*Server, error) {
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
		resolver = stream.NewResolver(biliClient, cfg.BiliRoomID, cfg.RefreshInterval)
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

func (s *Server) Start(ctx context.Context) error {
	if s.resolver != nil {
		go s.resolver.Start(ctx)
	}
	if err := s.httpServer.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen failed: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
