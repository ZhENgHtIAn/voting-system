# GrpcServer 模块实现文档（Phase 2）

## 文件清单
- `internal/grpcserver/service.go`
- `cmd/grpcserver/main.go`

## 实现说明
- `service.go` 是核心业务实现：
  - 按固定话题白名单校验 `topic_name`
  - 使用 `HIncrBy` 完成原子投票
  - 使用 `HGetAll` 返回全量结果
  - 输出统一的 gRPC 错误语义
  - 记录投票与查询链路日志（开始/完成/异常）
- `main.go` 是服务启动编排：
  - 读取配置
  - 初始化 Redis 并检查可用性
  - 初始化固定话题字段
  - 注册 gRPC 服务并监听端口
  - 注册 gRPC unary interceptor 记录访问日志（方法、请求 ID、状态码、耗时）

## 函数定位
- `NewVoteService(redisClient, redisHashKey, topics)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：构建服务对象并进行启动期参数校验。
- `(s *VoteService) CastVote(ctx, req)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：处理投票请求，执行原子加票并返回最新全量结果。
- `(s *VoteService) GetResults(ctx, req)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：读取并返回当前投票结果。
- `(s *VoteService) buildVoteResponse(ctx)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：从 Redis 读取结果并转换为 `map[string]int64`。
- `(s *VoteService) isAllowedTopic(topic)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：执行固定话题白名单判断。
- `unaryLoggingInterceptor(logger)`  
  - 文件：`cmd/grpcserver/main.go`  
  - 职责：统一记录 gRPC 访问日志并串联请求 ID。
- `main()`  
  - 文件：`cmd/grpcserver/main.go`  
  - 职责：完成 GrpcServer 生命周期初始化与启动。

## 调用关系
- `main()` -> `pkg.LoadConfig()` -> `pkg.NewRedisClient()` -> `pkg.PingRedis()` -> `pkg.EnsureTopicsInitialized()` -> `grpcserver.NewVoteService()` -> `pb.RegisterVoteServiceServer()`。
- `CastVote()`/`GetResults()` -> `buildVoteResponse()` -> Redis `HGetAll`。

## 维护要求
- 若调整错误码或请求校验规则，必须同步更新本文件和 `docs/overview/principles.md`。
