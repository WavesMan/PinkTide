package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config 统一承载运行期配置，来源于环境变量并完成归一化。
type Config struct {
	ListenAddr       string
	CDNPublicURL     string
	BiliRoomID       string
	LogLevel         string
	TLSCertFile      string
	TLSKeyFile       string
	TLSCertDir       string
	HTTPRedirectAddr string
	RefreshInterval  time.Duration
	RequestTimeout   time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
}

// Load 加载并校验配置，缺失必填项或格式错误时返回错误。
func Load() (Config, error) {
	cfg := Config{
		ListenAddr:       getEnv("PT_LISTEN_ADDR", ":8080"),
		CDNPublicURL:     getEnv("PT_CDN_PUBLIC_URL", ""),
		BiliRoomID:       getEnv("PT_BILI_ROOM_ID", ""),
		LogLevel:         getEnv("PT_LOG_LEVEL", "info"),
		TLSCertFile:      getEnv("PT_TLS_CERT_FILE", ""),
		TLSKeyFile:       getEnv("PT_TLS_KEY_FILE", ""),
		TLSCertDir:       getEnv("PT_TLS_CERT_DIR", "certs"),
		HTTPRedirectAddr: getEnv("PT_HTTP_REDIRECT_ADDR", ":8081"),
		RefreshInterval:  10 * time.Minute,
		RequestTimeout:   5 * time.Second,
		ReadTimeout:      10 * time.Second,
		WriteTimeout:     10 * time.Second,
		IdleTimeout:      60 * time.Second,
	}

	if v, ok := os.LookupEnv("PT_REFRESH_INTERVAL"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse PT_REFRESH_INTERVAL failed: %w", err)
		}
		cfg.RefreshInterval = d
	}

	if v, ok := os.LookupEnv("PT_REQUEST_TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse PT_REQUEST_TIMEOUT failed: %w", err)
		}
		cfg.RequestTimeout = d
	}

	if v, ok := os.LookupEnv("PT_READ_TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse PT_READ_TIMEOUT failed: %w", err)
		}
		cfg.ReadTimeout = d
	}

	if v, ok := os.LookupEnv("PT_WRITE_TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse PT_WRITE_TIMEOUT failed: %w", err)
		}
		cfg.WriteTimeout = d
	}

	if v, ok := os.LookupEnv("PT_IDLE_TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse PT_IDLE_TIMEOUT failed: %w", err)
		}
		cfg.IdleTimeout = d
	}

	cfg.CDNPublicURL = strings.TrimSpace(cfg.CDNPublicURL)
	cfg.CDNPublicURL = strings.TrimRight(cfg.CDNPublicURL, "/")
	cfg.BiliRoomID = strings.TrimSpace(cfg.BiliRoomID)
	cfg.TLSCertFile = strings.TrimSpace(cfg.TLSCertFile)
	cfg.TLSKeyFile = strings.TrimSpace(cfg.TLSKeyFile)
	cfg.TLSCertDir = strings.TrimSpace(cfg.TLSCertDir)
	cfg.HTTPRedirectAddr = strings.TrimSpace(cfg.HTTPRedirectAddr)

	if cfg.CDNPublicURL == "" {
		return Config{}, fmt.Errorf("PT_CDN_PUBLIC_URL is required")
	}

	return cfg, nil
}

// getEnv 读取环境变量，未设置时回退默认值。
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
