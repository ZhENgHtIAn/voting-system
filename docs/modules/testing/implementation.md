# Testing 模块实现文档（Phase 4+）

## 文件清单
- `internal/test/test.go`
- `internal/test/test_test.go`

## 实现说明
- 测试已从 `internal/httpserver` 迁移到统一测试目录 `internal/test`。
- `test.go` 承载测试基建和可复用测试逻辑，`test_test.go` 暴露 `go test` 入口函数。
- 覆盖范围已扩展为 HTTP + gRPC + 并发一致性三类：
  - HTTP 配置加载与校验（成功、失败、超时回退）
  - HTTP Handler 行为（`GET /api/results`、`POST /api/vote`、错误码映射、方法约束、空客户端校验）
  - HTTP `X-Request-ID` 透传到 gRPC metadata
  - gRPC `CastVote` / `GetResults`（成功、非法话题、空请求、默认零值）
  - 并发一致性验证：
    - HTTP 层并发投票后结果精确一致
    - gRPC 层并发 `CastVote` 后 Redis 中计数与返回结果一致
    - Web 入口全链路并发（`POST /api/vote` -> HTTP -> gRPC -> Redis）结果精确一致
- gRPC/Redis 测试使用 `miniredis` 构造内存 Redis，避免依赖外部 Redis 进程。

## 日志与结果可读性设计
- 每个测试函数都通过 `t.Log` / `t.Logf` 输出：
  - 测试开始日志
  - 关键中间结果（状态码、票数字段、错误码、request id）
  - 最终通过结果（便于快速判读）
- 建议执行命令：
  - `go test -v ./internal/test`
  - `go test -v -coverprofile=coverage.out ./...`

## 函数定位
- `TestHTTPLoadConfigSuccess`、`TestHTTPLoadConfigValidationFailure`、`TestHTTPRequestTimeoutFallback`  
  - 文件：`internal/test/test_test.go`（逻辑在 `internal/test/test.go`）  
  - 职责：验证 HTTP 配置加载/校验/默认回退。
- `TestHTTPGetResultsSuccess`、`TestHTTPPostVoteSuccess`、`TestHTTPPostVoteBadRequest`、`TestHTTPPostVoteInvalidArgumentMappedTo400`、`TestHTTPGetResultsMethodNotAllowed`、`TestHTTPNewServerNilClient`  
  - 文件：`internal/test/test_test.go`（逻辑在 `internal/test/test.go`）  
  - 职责：验证 HTTP API 核心路径与异常路径。
- `TestHTTPRequestIDPropagation`  
  - 文件：`internal/test/test_test.go`（逻辑在 `internal/test/test.go`）  
  - 职责：验证 HTTP 到 gRPC 的请求 ID 透传。
- `TestGRPCCastVoteSuccess`、`TestGRPCCastVoteInvalidTopic`、`TestGRPCCastVoteNilRequest`、`TestGRPCGetResultsDefaultZero`  
  - 文件：`internal/test/test_test.go`（逻辑在 `internal/test/test.go`）  
  - 职责：验证 gRPC 服务核心行为与边界条件。
- `TestHTTPConcurrentVoteConsistency`、`TestGRPCConcurrentCastVoteConsistency`  
  - 文件：`internal/test/test_test.go`（逻辑在 `internal/test/test.go`）  
  - 职责：验证并发场景下票数一致性。
- `TestWebFullChainConcurrentIntegration`  
  - 文件：`internal/test/test_test.go`（逻辑在 `internal/test/test.go`）  
  - 职责：验证 Web 入口到 Redis 的完整链路在并发下保持最终一致性。
