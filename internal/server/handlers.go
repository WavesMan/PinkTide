package server

import (
	"encoding/base64"
	"fmt"
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	roomID := r.URL.Query().Get("room_id")
	var originBase string
	if roomID == "" {
		if s.resolver == nil {
			http.Error(w, "missing room_id", http.StatusBadRequest)
			return
		}
		originBase = s.resolver.Get()
		if originBase == "" {
			http.Error(w, "stream not ready", http.StatusServiceUnavailable)
			return
		}
	} else {
		var err error
		originBase, err = s.biliClient.FetchPlayURL(r.Context(), roomID)
		if err != nil {
			http.Error(w, "origin error", http.StatusBadGateway)
			return
		}
	}

	data, status, err := s.origin.Get(r.Context(), originBase)
	if err != nil {
		http.Error(w, "origin error", http.StatusBadGateway)
		return
	}
	if status != http.StatusOK {
		http.Error(w, "origin error", http.StatusBadGateway)
		return
	}

	rewritten, err := s.rewriter.Rewrite(string(data), originBase)
	if err != nil {
		http.Error(w, "rewrite error", http.StatusInternalServerError)
		return
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload := r.URL.Query().Get("payload")
	if payload == "" {
		http.Error(w, "missing payload", http.StatusBadRequest)
		return
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		http.Error(w, "decode error", http.StatusBadRequest)
		return
	}
	target := string(decoded)
	if target == "" {
		http.Error(w, "decode error", http.StatusBadRequest)
		return
	}

	data, err := s.segFetcher.Fetch(r.Context(), target)
	if err != nil {
		http.Error(w, "fetch error", http.StatusBadGateway)
		return
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
