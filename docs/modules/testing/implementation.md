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

## 并发投票实现机制（测试层）

### 1) HTTP 并发投票测试：`TestHTTPConcurrentVoteConsistency`
- 位置：`internal/test/test.go`
- 实现方式：
  - 使用 `sync.WaitGroup` 同时启动多个 goroutine（当前为 120 个）。
  - 每个 goroutine 通过 `httptest` 向 `POST /api/vote` 发送一次投票请求。
  - 所有请求完成后，再调用 `GET /api/results` 读取最终票数。
  - 断言 `results["Golang"] == 并发请求总数`，验证并发下无丢票。

### 2) gRPC 并发投票测试：`TestGRPCConcurrentCastVoteConsistency`
- 位置：`internal/test/test.go`
- 实现方式：
  - 使用 `sync.WaitGroup` + `errCh` 并发调用 `CastVote`（当前为 240 次）。
  - 等待所有协程结束后，调用 `GetResults` 和 Redis `HGET` 双重校验。
  - 断言 gRPC 返回值和 Redis 存储值都等于请求总数，验证服务层与存储层一致性。

### 3) Web 全链路并发测试：`TestWebFullChainConcurrentIntegration`
- 位置：`internal/test/test.go`
- 实现方式：
  - 在测试进程中启动真实 gRPC Server（注册 `VoteService`）。
  - 用 gRPC Client 初始化 `httpserver`，再通过 `httptest.NewServer` 暴露 HTTP 入口。
  - 使用 goroutine 并发请求 `POST /api/vote`（当前为 300 次），完整经过 `HTTP -> gRPC -> Redis`。
  - 最后请求 `GET /api/results`，断言 `results["Kubernetes"] == 并发请求总数`。

### 4) 为什么这种方式能验证并发一致性
- 并发是通过“多个 goroutine 同时发起写请求”实现的，模拟多用户同时点击投票。
- 一致性通过“最终值 == 请求总数”这一不变量校验。
- 核心依赖 Redis `HINCRBY` 的原子自增语义，避免并发下覆盖写导致的丢票。

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
