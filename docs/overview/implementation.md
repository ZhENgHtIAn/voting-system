# Voting System Phase 1-5 实现总览

## 1. 范围说明
- 当前实现覆盖 `Phase 1` 到 `Phase 5`。
- 已完成内容：proto 契约、pb 代码、配置文件读取、Redis 封装、GrpcServer 核心逻辑、HttpServer 网关、Web 投票页、统一日志系统、Phase4 单元测试、Phase5 Docker 与 K8s 部署清单。

## 2. 目录与实现映射
- `api/pb/`: gRPC 契约与 Go 代码。
- `scripts/`: proto 生成脚本。
- `configs/`: GrpcServer 运行配置。
- `internal/pkg/`: 配置与 Redis 基础能力。
- `internal/grpcserver/`: gRPC 业务逻辑。
- `internal/httpserver/`: HTTP 网关配置与 Handler。
- `cmd/grpcserver/`: gRPC 服务启动入口。
- `cmd/httpserver/`: HTTP 服务启动入口。
- `web/`: 前端页面与交互脚本。
- `deployments/docker/`: 多阶段 Dockerfile。
- `deployments/k8s/`: Redis / GrpcServer / HttpServer 部署清单。

## 3. 关键运行命令
- 生成 proto：
  - `./scripts/gen_proto.sh`
  - 该脚本已固定 `source_relative` 生成策略，输出文件位于 `api/pb/`。
- 启动 gRPC 服务：
  - `go run ./cmd/grpcserver -config configs/grpcserver.yaml`
- 启动 HTTP 服务：
  - `go run ./cmd/httpserver -config configs/httpserver.yaml`
- 调试查询结果：
  - `go run ./cmd/grpcclient -action results`
- 调试发起投票：
  - `go run ./cmd/grpcclient -action vote -topic Golang`
  - 或使用 `./scripts/debug_grpc.sh vote Golang`
- 构建容器镜像：
  - `docker build -f deployments/docker/Dockerfile.grpc -t voting-system/grpcserver:latest .`
  - `docker build -f deployments/docker/Dockerfile.http -t voting-system/httpserver:latest .`
- 应用 K8s 清单：
  - `kubectl apply -f deployments/k8s/redis.yaml`
  - `kubectl apply -f deployments/k8s/grpcserver.yaml`
  - `kubectl apply -f deployments/k8s/httpserver.yaml`

## 4. 依赖版本（Phase 1 要求）
- `google.golang.org/grpc v1.54.0`
- `google.golang.org/protobuf v1.30.0`
- `github.com/redis/go-redis/v9 v9.0.5`
- `gopkg.in/yaml.v3 v3.0.1`

## 5. 函数位置索引（实现定位）

### 配置与基础设施
- `LoadConfig(path string)`  
  - 文件：`internal/pkg/config.go`  
  - 职责：读取并校验 GrpcServer 配置文件。
- `(c *AppConfig) RequestTimeout()`  
  - 文件：`internal/pkg/config.go`  
  - 职责：将配置中的超时字符串转换为 `time.Duration`。
- `NewRedisClient(cfg RedisConfig)`  
  - 文件：`internal/pkg/redis.go`  
  - 职责：创建 Redis 客户端。
- `PingRedis(ctx, client)`  
  - 文件：`internal/pkg/redis.go`  
  - 职责：执行 Redis 健康检查。
- `EnsureTopicsInitialized(ctx, client, hashKey, topics)`  
  - 文件：`internal/pkg/redis.go`  
  - 职责：初始化固定话题字段，避免首次读取为空。

### gRPC 业务服务
- `NewVoteService(redisClient, redisHashKey, topics)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：构建服务并完成固定话题校验。
- `(s *VoteService) CastVote(ctx, req)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：严格校验话题后进行 `HIncrBy` 原子加票并返回全量结果。
- `(s *VoteService) GetResults(ctx, req)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：读取并返回全量票数。
- `(s *VoteService) buildVoteResponse(ctx)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：将 Redis 数据转换为 `map[string]int64`，并对固定话题补 0。
- `(s *VoteService) isAllowedTopic(topic)`  
  - 文件：`internal/grpcserver/service.go`  
  - 职责：固定话题白名单判断。

### 服务入口
- `main()`  
  - 文件：`cmd/grpcserver/main.go`  
  - 职责：加载配置、连接 Redis、注册并启动 gRPC 服务。
- `main()`  
  - 文件：`cmd/httpserver/main.go`  
  - 职责：加载 HTTP 配置、建立 gRPC 客户端连接并启动 HTTP 网关。
- `main()`  
  - 文件：`cmd/grpcclient/main.go`  
  - 职责：作为本地调试客户端调用 `GetResults`/`CastVote`。
- `printResults(results)`  
  - 文件：`cmd/grpcclient/main.go`  
  - 职责：对投票结果做排序并格式化输出。

### HTTP 网关
- `LoadConfig(path string)`  
  - 文件：`internal/httpserver/config.go`  
  - 职责：读取并校验 `httpserver` 配置文件。
- `(c *Config) RequestTimeout()`  
  - 文件：`internal/httpserver/config.go`  
  - 职责：解析 HTTP 请求超时参数。
- `NewServer(voteClient, requestTimeout)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：创建 HTTP 业务处理器。
- `(s *Server) NewHandler(webDir)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：注册 `/api/results`、`/api/vote` 与静态文件路由。
- `(s *Server) handleGetResults(w, r)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：处理 `GET /api/results`，调用 gRPC `GetResults`。
- `(s *Server) handlePostVote(w, r)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：处理 `POST /api/vote`，调用 gRPC `CastVote`。
- `(s *Server) loggingMiddleware(next)`  
  - 文件：`internal/httpserver/server.go`  
  - 职责：记录请求开始/完成日志，输出请求路径、状态码、耗时和请求 ID。

### 日志系统
- `NewLogger(service, levelText)`  
  - 文件：`internal/pkg/logging/logger.go`  
  - 职责：创建服务级日志器，支持日志级别控制。
- `ParseLevel(levelText)`  
  - 文件：`internal/pkg/logging/logger.go`  
  - 职责：解析日志级别字符串（debug/info/warn/error）。
- `(*Logger) Debug/Info/Warn/Error()`  
  - 文件：`internal/pkg/logging/logger.go`  
  - 职责：输出结构化 JSON 日志并携带统一字段。

### Phase 4（进行中）测试入口
- `TestHandleGetResultsSuccess`  
  - 文件：`internal/httpserver/server_test.go`  
  - 职责：验证 `GET /api/results` 成功路径。
- `TestHandlePostVoteSuccess`  
  - 文件：`internal/httpserver/server_test.go`  
  - 职责：验证 `POST /api/vote` 成功路径。
- `TestHandlePostVoteBadRequest`  
  - 文件：`internal/httpserver/server_test.go`  
  - 职责：验证非法 JSON 的 400 响应。
- `TestHandlePostVoteInvalidArgumentMappedTo400`  
  - 文件：`internal/httpserver/server_test.go`  
  - 职责：验证 gRPC `InvalidArgument` -> HTTP 400 映射。

### Phase 5 部署清单定位
- `Dockerfile.grpc`  
  - 文件：`deployments/docker/Dockerfile.grpc`  
  - 职责：构建并打包 `grpcserver` 运行镜像（多阶段 + 非 root 用户）。
- `Dockerfile.http`  
  - 文件：`deployments/docker/Dockerfile.http`  
  - 职责：构建并打包 `httpserver` 运行镜像（包含 web 静态资源）。
- `redis.yaml`  
  - 文件：`deployments/k8s/redis.yaml`  
  - 职责：部署单副本 Redis 与 ClusterIP 服务（不做 Redis 集群化）。
- `grpcserver.yaml`  
  - 文件：`deployments/k8s/grpcserver.yaml`  
  - 职责：部署 3 副本 GrpcServer、Headless Service、ConfigMap。
- `httpserver.yaml`  
  - 文件：`deployments/k8s/httpserver.yaml`  
  - 职责：部署 3 副本 HttpServer、NodePort Service、ConfigMap。

## 6. 文档维护规则
- 任何函数新增/删除/重命名后，必须同步更新本文件“函数位置索引”。
- 若模块职责变化，先更新模块实现文档，再回写本总览文档。
