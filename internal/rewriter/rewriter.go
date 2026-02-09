package rewriter

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

type Rewriter struct {
	cdnPublicURL string
}

func New(cdnPublicURL string) (*Rewriter, error) {
	cdnPublicURL = strings.TrimSpace(cdnPublicURL)
	cdnPublicURL = strings.TrimRight(cdnPublicURL, "/")
	if cdnPublicURL == "" {
		return nil, fmt.Errorf("cdn public url is empty")
	}
	return &Rewriter{cdnPublicURL: cdnPublicURL}, nil
}

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
