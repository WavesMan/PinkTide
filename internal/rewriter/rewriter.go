package rewriter

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Rewriter 将原始 M3U8 中的切片地址改写为 CDN 可回源地址。
type Rewriter struct {
	cdnPublicURL     string
	cdnPublicURLs    []string
	cdnPublicHostMap map[string]string
}

// New 校验并归一化 CDN 域名，支持多域名用于本地与公网切换。
func New(cdnPublicURL string) (*Rewriter, error) {
	cdnPublicURL = strings.TrimSpace(cdnPublicURL)
	if cdnPublicURL == "" {
		return nil, fmt.Errorf("cdn public url is empty")
	}
	parts := strings.Split(cdnPublicURL, ",")
	urls := make([]string, 0, len(parts))
	hostMap := make(map[string]string, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		normalized, host, err := normalizePublicURL(part)
		if err != nil {
			return nil, err
		}
		if _, exists := hostMap[host]; exists {
			continue
		}
		hostMap[host] = normalized
		urls = append(urls, normalized)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("cdn public url is empty")
	}
	return &Rewriter{cdnPublicURL: urls[0], cdnPublicURLs: urls, cdnPublicHostMap: hostMap}, nil
}

// Rewrite 保留原有换行风格并重写切片 URL，按请求 Host 选择回源地址。
func (r *Rewriter) Rewrite(content string, originBase string, requestHost string) (string, error) {
	if originBase == "" {
		return "", fmt.Errorf("origin base is empty")
	}
	publicURL := r.selectPublicURL(requestHost)

	newline := "\n"
	if strings.Contains(content, "\r\n") {
		newline = "\r\n"
	}

	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		resolved, err := resolveURL(originBase, line)
		if err != nil {
			return "", err
		}
		payload := base64.URLEncoding.EncodeToString([]byte(resolved))
		lines[i] = publicURL + "/seg?payload=" + payload
	}

	return strings.Join(lines, newline), nil
}

// normalizePublicURL 统一补全协议并提取主机，用于多域名匹配。
func normalizePublicURL(raw string) (string, string, error) {
	value := strings.TrimSpace(raw)
	value = strings.TrimRight(value, "/")
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") {
		value = "https://" + strings.TrimPrefix(value, "http://")
	} else if !strings.HasPrefix(lower, "https://") {
		value = "https://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", "", fmt.Errorf("cdn public url is invalid")
	}
	return value, strings.ToLower(parsed.Host), nil
}

// selectPublicURL 根据请求 Host 选择匹配的公开地址，未命中则回退默认。
func (r *Rewriter) selectPublicURL(requestHost string) string {
	if requestHost == "" {
		return r.cdnPublicURL
	}
	host := strings.ToLower(requestHost)
	if urlValue, ok := r.cdnPublicHostMap[host]; ok {
		return urlValue
	}
	hostOnly := stripHostPort(host)
	if hostOnly != host {
		if urlValue, ok := r.cdnPublicHostMap[hostOnly]; ok {
			return urlValue
		}
	}
	if hostOnly != "" {
		for candidateHost, urlValue := range r.cdnPublicHostMap {
			if stripHostPort(candidateHost) == hostOnly {
				return urlValue
			}
		}
	}
	return r.cdnPublicURL
}

// stripHostPort 去除端口以便进行域名匹配。
func stripHostPort(value string) string {
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	return value
}

// resolveURL 将相对切片地址补齐为可回源的绝对地址。
func resolveURL(baseURL string, ref string) (string, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("parse ref failed: %w", err)
	}
	if u.IsAbs() {
		return ref, nil
	}
	b, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base failed: %w", err)
	}
	return b.ResolveReference(u).String(), nil
}
