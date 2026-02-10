package bili

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"PinkTide/internal/origin"
)

// Client 负责调用 B 站直播 API 获取可用流地址。
type Client struct {
	originClient *origin.Client
}

// NewClient 注入回源客户端用于复用超时与请求头。
func NewClient(originClient *origin.Client) *Client {
	return &Client{originClient: originClient}
}

// FetchPlayURL 根据房间号获取可播放 URL，失败返回错误。
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

func (c *Client) FetchRoomStatus(ctx context.Context, roomID string) (RoomStatus, error) {
	if roomID == "" {
		return RoomStatus{}, fmt.Errorf("room id is empty")
	}

	apiURL := fmt.Sprintf(
		"https://api.live.bilibili.com/room/v1/Room/room_init?id=%s",
		url.QueryEscape(roomID),
	)

	data, status, err := c.originClient.Get(ctx, apiURL)
	if err != nil {
		return RoomStatus{}, err
	}
	if status != 200 {
		return RoomStatus{}, fmt.Errorf("api status %d", status)
	}

	var result roomInitResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return RoomStatus{}, fmt.Errorf("decode response failed: %w", err)
	}
	if result.Code != 0 {
		msg := strings.TrimSpace(result.Message)
		if msg == "" {
			msg = strings.TrimSpace(result.Msg)
		}
		if msg == "" {
			msg = "api error"
		}
		return RoomStatus{}, fmt.Errorf("%s", msg)
	}

	return RoomStatus{
		RoomID:     result.Data.RoomID,
		ShortID:    result.Data.ShortID,
		UID:        result.Data.UID,
		LiveStatus: result.Data.LiveStatus,
		IsHidden:   result.Data.IsHidden,
		IsLocked:   result.Data.IsLocked,
	}, nil
}

// apiResponse 对齐 B 站 API 返回结构，仅保留必要字段。
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

type roomInitResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
	Data    struct {
		RoomID     int  `json:"room_id"`
		ShortID    int  `json:"short_id"`
		UID        int  `json:"uid"`
		LiveStatus int  `json:"live_status"`
		IsHidden   bool `json:"is_hidden"`
		IsLocked   bool `json:"is_locked"`
	} `json:"data"`
}

type RoomStatus struct {
	RoomID     int
	ShortID    int
	UID        int
	LiveStatus int
	IsHidden   bool
	IsLocked   bool
}
