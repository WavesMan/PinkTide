package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// registerRoutes 统一注册对外路由，便于后续扩展。
func (s *Server) registerRoutes() {
	s.serveMux.HandleFunc("/api", s.handleRoot)
	s.serveMux.HandleFunc("/api/", s.handleRoot)
	s.serveMux.HandleFunc("/api/status", s.handleRoomStatus)
	s.serveMux.HandleFunc("/api/watch", s.handleRoomWatch)
	s.serveMux.HandleFunc("/ui", s.handleUI)
	s.serveMux.HandleFunc("/ui/", s.handleUI)
	s.serveMux.HandleFunc("/live.m3u8", s.handleM3U8)
	s.serveMux.HandleFunc("/seg", s.handleSegment)
	s.serveMux.Handle("/", http.FileServer(http.Dir("ui")))
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		if s.logger != nil {
			fields := append(
				[]any{"path", r.URL.Path, "method", r.Method},
				requestFields(r)...,
			)
			s.logger.Warn("method not allowed", fields...)
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	s.setCors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		if s.logger != nil {
			fields := append(
				[]any{"path", r.URL.Path, "method", r.Method},
				requestFields(r)...,
			)
			s.logger.Warn("method not allowed", fields...)
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := map[string]string{
		"name":    "PinkTide",
		"version": BuildVersion,
		"repo":    "https://github.com/WavesMan/PinkTide",
		"author":  "WavesMan",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleRoomStatus(w http.ResponseWriter, r *http.Request) {
	s.setCors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		if s.logger != nil {
			fields := append(
				[]any{"path", r.URL.Path, "method", r.Method},
				requestFields(r)...,
			)
			s.logger.Warn("method not allowed", fields...)
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	roomID, ok := s.resolveRoomID(r)
	if !ok {
		http.Error(w, "missing room_id", http.StatusBadRequest)
		return
	}

	state, code := s.inspectStreamState(r.Context(), roomID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(state)
}

func (s *Server) handleRoomWatch(w http.ResponseWriter, r *http.Request) {
	s.setCors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		if s.logger != nil {
			fields := append(
				[]any{"path", r.URL.Path, "method", r.Method},
				requestFields(r)...,
			)
			s.logger.Warn("method not allowed", fields...)
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	roomID, ok := s.resolveRoomID(r)
	if !ok {
		http.Error(w, "missing room_id", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		state, code := s.inspectStreamState(r.Context(), roomID)
		payload, _ := json.Marshal(state)
		_, _ = fmt.Fprintf(w, "event: status\ndata: %s\n\n", payload)
		flusher.Flush()

		if state.State == "ready" {
			_, _ = fmt.Fprintf(w, "event: ready\ndata: %s\n\n", payload)
			flusher.Flush()
			return
		}
		if code >= 400 {
			_, _ = fmt.Fprintf(w, "event: stop\ndata: %s\n\n", payload)
			flusher.Flush()
			return
		}

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

// handleM3U8 根据房间号获取并重写播放列表。
func (s *Server) handleM3U8(w http.ResponseWriter, r *http.Request) {
	s.setCors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		if s.logger != nil {
			fields := append(
				[]any{"path", r.URL.Path, "method", r.Method},
				requestFields(r)...,
			)
			s.logger.Warn("method not allowed", fields...)
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	roomID, ok := s.resolveRoomID(r)
	if !ok {
		if s.logger != nil {
			fields := append([]any{"path", r.URL.Path}, requestFields(r)...)
			s.logger.Warn("missing room id", fields...)
		}
		http.Error(w, "missing room_id", http.StatusBadRequest)
		return
	}

	state, code := s.inspectRoomState(r.Context(), roomID)
	if code != http.StatusOK {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(state.Message))
		return
	}

	roomIDParam := r.URL.Query().Get("room_id")
	var originBase string
	if roomIDParam == "" {
		if s.resolver == nil {
			if s.logger != nil {
				fields := append([]any{"path", r.URL.Path}, requestFields(r)...)
				s.logger.Warn("missing room id", fields...)
			}
			http.Error(w, "missing room_id", http.StatusBadRequest)
			return
		}
		originBase = s.resolver.Get()
		if originBase == "" {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("等待加载"))
			return
		}
	} else {
		var err error
		originBase, err = s.biliClient.FetchPlayURL(r.Context(), roomID)
		if err != nil {
			if s.logger != nil {
				fields := append(
					[]any{"room_id", roomID, "path", r.URL.Path, "error", err},
					requestFields(r)...,
				)
				s.logger.Error("fetch play url failed", fields...)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("等待加载"))
			return
		}
	}

	data, status, err := s.origin.Get(r.Context(), originBase)
	if err != nil {
		if s.logger != nil {
			fields := append(
				[]any{"room_id", roomID, "path", r.URL.Path, "error", err},
				requestFields(r)...,
			)
			s.logger.Error("fetch m3u8 failed", fields...)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("加载中"))
		return
	}
	if status != http.StatusOK {
		if s.logger != nil {
			fields := append(
				[]any{"room_id", roomID, "path", r.URL.Path, "status", status},
				requestFields(r)...,
			)
			s.logger.Error("fetch m3u8 failed", fields...)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("加载中"))
		return
	}

	rewritten, err := s.rewriter.Rewrite(string(data), originBase, r.Host)
	if err != nil {
		if s.logger != nil {
			fields := append(
				[]any{"room_id", roomID, "path", r.URL.Path, "error", err},
				requestFields(r)...,
			)
			s.logger.Error("rewrite failed", fields...)
		}
		http.Error(w, "rewrite error", http.StatusInternalServerError)
		return
	}
	if s.logger != nil {
		fields := append(
			[]any{"room_id", roomID, "path", r.URL.Path},
			requestFields(r)...,
		)
		s.logger.Debug("m3u8 served", fields...)
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "public, max-age=1")
	_, _ = w.Write([]byte(rewritten))
}

// handleSegment 拉取切片并返回，便于 CDN 长缓存。
func (s *Server) handleSegment(w http.ResponseWriter, r *http.Request) {
	s.setCors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		if s.logger != nil {
			fields := append(
				[]any{"path", r.URL.Path, "method", r.Method},
				requestFields(r)...,
			)
			s.logger.Warn("method not allowed", fields...)
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload := r.URL.Query().Get("payload")
	if payload == "" {
		if s.logger != nil {
			fields := append([]any{"path", r.URL.Path}, requestFields(r)...)
			s.logger.Warn("missing payload", fields...)
		}
		http.Error(w, "missing payload", http.StatusBadRequest)
		return
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		if s.logger != nil {
			fields := append(
				[]any{"path", r.URL.Path, "error", err},
				requestFields(r)...,
			)
			s.logger.Warn("payload decode failed", fields...)
		}
		http.Error(w, "decode error", http.StatusBadRequest)
		return
	}
	target := string(decoded)
	if target == "" {
		if s.logger != nil {
			fields := append([]any{"path", r.URL.Path}, requestFields(r)...)
			s.logger.Warn("payload empty", fields...)
		}
		http.Error(w, "decode error", http.StatusBadRequest)
		return
	}

	data, err := s.segFetcher.Fetch(r.Context(), target)
	if err != nil {
		if s.logger != nil {
			fields := append(
				[]any{"path", r.URL.Path, "error", err},
				requestFields(r)...,
			)
			s.logger.Error("fetch segment failed", fields...)
		}
		http.Error(w, "fetch error", http.StatusBadGateway)
		return
	}
	if s.logger != nil {
		fields := append(
			[]any{"path", r.URL.Path, "bytes", len(data)},
			requestFields(r)...,
		)
		s.logger.Debug("segment served", fields...)
	}

	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

type streamState struct {
	RoomID     string `json:"room_id"`
	LiveStatus int    `json:"live_status"`
	State      string `json:"state"`
	Message    string `json:"message"`
}

func (s *Server) resolveRoomID(r *http.Request) (string, bool) {
	roomID := r.URL.Query().Get("room_id")
	if roomID != "" {
		return roomID, true
	}
	if s.cfg.BiliRoomID == "" {
		return "", false
	}
	return s.cfg.BiliRoomID, true
}

func (s *Server) inspectRoomState(ctx context.Context, roomID string) (streamState, int) {
	status, err := s.biliClient.FetchRoomStatus(ctx, roomID)
	if err != nil {
		return streamState{RoomID: roomID, State: "error", Message: "获取直播状态失败"}, http.StatusBadGateway
	}

	state := streamState{RoomID: roomID, LiveStatus: status.LiveStatus}
	if status.IsLocked {
		state.State = "locked"
		state.Message = "直播间已封禁"
		return state, http.StatusLocked
	}
	if status.IsHidden {
		state.State = "hidden"
		state.Message = "直播间不可访问"
		return state, http.StatusForbidden
	}
	if status.LiveStatus == 0 {
		state.State = "offline"
		state.Message = "直播间未开播"
		return state, http.StatusConflict
	}
	if status.LiveStatus == 2 {
		state.State = "loop"
		state.Message = "直播间轮播中"
		return state, http.StatusConflict
	}
	state.State = "live"
	state.Message = "直播中"
	return state, http.StatusOK
}

func (s *Server) inspectStreamState(ctx context.Context, roomID string) (streamState, int) {
	state, code := s.inspectRoomState(ctx, roomID)
	if code != http.StatusOK {
		return state, code
	}

	originBase, err := s.biliClient.FetchPlayURL(ctx, roomID)
	if err != nil || originBase == "" {
		state.State = "waiting"
		state.Message = "等待加载"
		return state, http.StatusAccepted
	}

	data, status, err := s.origin.Get(ctx, originBase)
	if err != nil || status != http.StatusOK || len(data) == 0 {
		state.State = "loading"
		state.Message = "加载中"
		return state, http.StatusAccepted
	}
	state.State = "ready"
	state.Message = "直播中"
	return state, http.StatusOK
}

// setCors 统一跨域响应头，避免 CDN 命中时缺失。
func (s *Server) setCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
}

// requestFields 采集回源链路相关字段用于日志分析。
func requestFields(r *http.Request) []any {
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remoteIP = host
	}

	fields := make([]any, 0, 18)
	fields = append(fields, "remote_ip", remoteIP)
	fields = appendField(fields, "xff", r.Header.Get("X-Forwarded-For"))
	fields = appendField(fields, "real_ip", r.Header.Get("X-Real-IP"))
	fields = appendField(fields, "client_ip", r.Header.Get("X-Client-IP"))
	fields = appendField(fields, "cf_ip", r.Header.Get("CF-Connecting-IP"))
	fields = appendField(fields, "true_client_ip", r.Header.Get("True-Client-IP"))
	fields = appendField(fields, "via", r.Header.Get("Via"))
	fields = appendField(fields, "x_cache", r.Header.Get("X-Cache"))
	fields = appendField(fields, "x_cache_status", r.Header.Get("X-Cache-Status"))
	fields = appendField(fields, "x_forwarded_proto", r.Header.Get("X-Forwarded-Proto"))
	fields = appendField(fields, "cdn_request_id", r.Header.Get("X-Request-ID"))
	return fields
}

// appendField 仅在字段非空时写入日志键值对。
func appendField(fields []any, key, value string) []any {
	if value == "" {
		return fields
	}
	return append(fields, key, value)
}
