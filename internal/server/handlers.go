package server

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
)

func (s *Server) registerRoutes() {
	s.serveMux.HandleFunc("/live.m3u8", s.handleM3U8)
	s.serveMux.HandleFunc("/seg", s.handleSegment)
}

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

	roomID := r.URL.Query().Get("room_id")
	var originBase string
	if roomID == "" {
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
			if s.logger != nil {
				fields := append([]any{"path", r.URL.Path}, requestFields(r)...)
				s.logger.Warn("stream not ready", fields...)
			}
			http.Error(w, "stream not ready", http.StatusServiceUnavailable)
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
			http.Error(w, "origin error", http.StatusBadGateway)
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
		http.Error(w, "origin error", http.StatusBadGateway)
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
		http.Error(w, "origin error", http.StatusBadGateway)
		return
	}

	rewritten, err := s.rewriter.Rewrite(string(data), originBase)
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

func (s *Server) setCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
}

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

func appendField(fields []any, key, value string) []any {
	if value == "" {
		return fields
	}
	return append(fields, key, value)
}
