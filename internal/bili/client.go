package bili

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"PinkTide/internal/origin"
)

type Client struct {
	originClient *origin.Client
}

func NewClient(originClient *origin.Client) *Client {
	return &Client{originClient: originClient}
}

func (c *Client) FetchPlayURL(ctx context.Context, roomID string) (string, error) {
	if roomID == "" {
		return "", fmt.Errorf("room id is empty")
	}

	apiURL := fmt.Sprintf(
		"https://api.live.bilibili.com/xlive/web-room/v1/playUrl/playUrl?cid=%s&platform=h5&qn=10000&https_url_req=1&ptype=16",
		url.QueryEscape(roomID),
	)

	data, status, err := c.originClient.Get(ctx, apiURL)
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("api status %d", status)
	}

	var result apiResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("decode response failed: %w", err)
	}

	streamList := result.Data.PlayUrlInfo.PlayUrl.Stream
	if len(streamList) > 0 {
		formatList := streamList[0].Format
		if len(formatList) > 0 {
			codecList := formatList[0].Codec
			if len(codecList) > 0 {
				urlInfoList := codecList[0].UrlInfo
				if len(urlInfoList) > 0 {
					fullURL := urlInfoList[0].Host + codecList[0].BaseUrl + urlInfoList[0].Extra
					return fullURL, nil
				}
			}
		}
	}

	if len(result.Data.Durl) > 0 {
		rawURL := result.Data.Durl[0].URL
		if rawURL != "" {
			return rawURL, nil
		}
	}

	return "", fmt.Errorf("play url not found")
}

type apiResponse struct {
	Code int `json:"code"`
	Data struct {
		PlayUrlInfo struct {
			PlayUrl struct {
				Stream []struct {
					Format []struct {
						Codec []struct {
							BaseUrl string `json:"base_url"`
							UrlInfo []struct {
								Host  string `json:"host"`
								Extra string `json:"extra"`
							} `json:"url_info"`
						} `json:"codec"`
					} `json:"format"`
				} `json:"stream"`
			} `json:"play_url"`
		} `json:"playurl_info"`
		Durl []struct {
			URL string `json:"url"`
		} `json:"durl"`
	} `json:"data"`
}
