package rewriter

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// Rewriter 将原始 M3U8 中的切片地址改写为 CDN 可回源地址。
type Rewriter struct {
	cdnPublicURL string
}

// New 校验并归一化 CDN 域名，空值时返回错误。
func New(cdnPublicURL string) (*Rewriter, error) {
	cdnPublicURL = strings.TrimSpace(cdnPublicURL)
	cdnPublicURL = strings.TrimRight(cdnPublicURL, "/")
	if strings.HasPrefix(strings.ToLower(cdnPublicURL), "http://") {
		cdnPublicURL = "https://" + strings.TrimPrefix(cdnPublicURL, "http://")
	}
	if cdnPublicURL == "" {
		return nil, fmt.Errorf("cdn public url is empty")
	}
	return &Rewriter{cdnPublicURL: cdnPublicURL}, nil
}

// Rewrite 保留原有换行风格并重写切片 URL，解析失败返回错误。
func (r *Rewriter) Rewrite(content string, originBase string) (string, error) {
	if originBase == "" {
		return "", fmt.Errorf("origin base is empty")
	}

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
		lines[i] = r.cdnPublicURL + "/seg?payload=" + payload
	}

	return strings.Join(lines, newline), nil
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
