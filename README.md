# PinkTide

基于 Go 的 B 站直播回源与 M3U8 重写服务，支持 CDN 回源分流与 SingleFlight 合并。

## 功能

- 通过直播间 ID 获取真实直播流地址并重写 M3U8
- TS 切片回源合并，降低源站并发压力
- 支持 URL 参数 room_id 动态指定直播间
- 支持 CDN 访问与回源缓存策略
- 结构化日志与多级别控制

## 快速开始

### 启动

```bash
PT_CDN_PUBLIC_URL=http://localhost:8080 \
PT_LOG_LEVEL=info \
go run ./cmd/pt-server
```

### 访问

```text
http://localhost:8080/live.m3u8?room_id=544853
```

## 配置

环境变量：

| 名称 | 说明 | 默认值 |
| --- | --- | --- |
| PT_LISTEN_ADDR | 服务监听地址 | :8080 |
| PT_CDN_PUBLIC_URL | CDN 对外域名 | 必填 |
| PT_BILI_ROOM_ID | 默认直播间 ID | 空 |
| PT_LOG_LEVEL | 日志级别 | info |
| PT_REFRESH_INTERVAL | 默认房间刷新间隔 | 10m |
| PT_REQUEST_TIMEOUT | 回源请求超时 | 5s |
| PT_READ_TIMEOUT | 读取超时 | 10s |
| PT_WRITE_TIMEOUT | 写入超时 | 10s |
| PT_IDLE_TIMEOUT | 空闲连接超时 | 60s |

## 接口

### GET /live.m3u8

- 说明：获取重写后的 M3U8
- 参数：room_id（可选）
- 行为：
  - room_id 为空且未配置 PT_BILI_ROOM_ID 返回 400
  - room_id 为空且配置 PT_BILI_ROOM_ID 使用默认值
  - room_id 提供时优先使用该值

### GET /seg

- 说明：回源 TS 切片
- 参数：payload（Base64 编码的真实 TS 地址）

## CDN 建议

- /seg 路径保持参数不忽略，缓存 365 天
- .m3u8 后缀短缓存 1 秒

## 日志

日志为 JSON 格式，包含回源链路相关字段：

- remote_ip
- xff
- real_ip
- client_ip
- cf_ip
- true_client_ip
- via
- x_cache
- x_cache_status
- x_forwarded_proto
- cdn_request_id

## 测试

```bash
go test ./...
```
