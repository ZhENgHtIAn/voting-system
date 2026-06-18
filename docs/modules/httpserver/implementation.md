# HttpServer 模块实现文档（Phase 3）

## 文件清单
- `configs/httpserver.yaml`
- `internal/httpserver/config.go`
- `internal/httpserver/server.go`
- `cmd/httpserver/main.go`

## 实现说明
- `httpserver` 作为 API 网关，向前端暴露 REST 接口：
  - `GET /api/results`
  - `POST /api/vote`
- 对后端通过 gRPC 客户端调用 `VoteService`，并配置 `round_robin` 负载均衡策略。
- 同时托管 `web/` 静态资源，访问 `http://localhost:8080/` 可直接打开投票页。
- 通过中间件记录请求链路日志（request_id、method、path、status、duration），并向 gRPC 透传请求 ID。

## 函数定位
- `LoadConfig(path string)`  
  - 文件：`internal/httpserver/config.go`  
  - 职责：读取并校验 HTTP 服务配置。
- `(c *Config) RequestTimeout()`  
  - 文件：`internal/httpserver/config.go`  
  - 职责：解析请求超时。
- `NewServer(voteClient, requestTimeout)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：构建 HTTP 业务处理器。
- `(s *Server) NewHandler(webDir)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：注册 API 路由与静态文件服务。
- `(s *Server) handleGetResults(w, r)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：转发 `GetResults` 并返回 JSON。
- `(s *Server) handlePostVote(w, r)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：解析请求体、转发 `CastVote` 并返回 JSON。
- `(s *Server) loggingMiddleware(next)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：记录 HTTP 请求开始与完成日志。
- `main()`  
  - 文件：`cmd/httpserver/main.go`  
  - 职责：初始化 gRPC 连接、创建 HTTP Handler 并启动服务。

## 运行命令
- `go run ./cmd/httpserver -config configs/httpserver.yaml`
