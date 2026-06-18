# Voting System Phase 1-5 原理总览

## 1. 架构边界
- 当前系统在后端核心上采用分层结构：`cmd`（启动）-> `internal/grpcserver`（业务）-> `internal/pkg`（基础设施）-> Redis。
- Phase 1/2 聚焦 gRPC + Redis；Phase 3 新增 `internal/httpserver` 与 `web`，形成 `Web -> HTTP -> gRPC -> Redis` 闭环。

## 2. 并发一致性原理
- 投票写入必须使用 Redis `HINCRBY`，避免“先读后写”导致并发覆盖。
- `CastVote` 在同一个请求流程中先写后读，保证返回结果是当前投票后的最新聚合视图。

## 3. 固定话题严格校验原理
- 话题集合由配置文件定义且要求固定 3 项。
- `CastVote` 对 `topic_name` 做白名单校验，禁止动态扩展话题字段。
- `GetResults` 对固定话题补零，确保响应结构稳定。

## 4. 配置驱动原理
- gRPC 监听地址、Redis 连接信息、投票话题、请求超时均由配置文件提供。
- 代码内保留默认值用于兜底，配置缺失时仍有可预测行为。

## 5. 错误语义原理
- 参数错误（空请求、空话题、非法话题）返回 `codes.InvalidArgument`。
- 基础设施失败（Redis 访问失败、票数字段异常）返回 `codes.Internal`。

## 6. 分级日志与链路可观测性原理
- `grpcserver` 与 `httpserver` 共用统一日志组件，日志字段包含时间、级别、服务名、消息与扩展字段。
- HTTP 网关通过请求 ID（`X-Request-ID`）串联“请求进入 -> 调用 gRPC -> 响应返回”全过程。
- gRPC 侧通过拦截器记录方法名、请求 ID、状态码、耗时，便于排查跨服务问题。

## 7. 契约稳定性原理
- `vote.proto` 定义是系统内通信契约，服务端实现必须严格遵守。
- proto 生成脚本统一参数，确保本地与后续 CI 生成结果一致。

## 8. 测试保障原理（Phase 4）
- `httpserver` 的核心路由采用 mock 驱动的单元测试，不依赖真实 gRPC/Redis。
- 当前覆盖率已显著高于目标值，保证接口行为和错误映射稳定。

## 9. 容器化与部署原理（Phase 5）
- Docker 采用多阶段构建：构建阶段编译二进制，运行阶段使用轻量 Alpine 镜像。
- `grpcserver` 与 `httpserver` 运行镜像都以非 root 用户启动，降低运行时风险。
- K8s 拓扑：
  - Redis：`Deployment(1)` + `ClusterIP Service`
  - GrpcServer：`Deployment(3)` + `Headless Service(clusterIP: None)`
  - HttpServer：`Deployment(3)` + `NodePort Service`
- Redis 在当前方案中明确保持单实例，不做 Redis Cluster/主从/Sentinel 集群化。
- 通过 ConfigMap 注入服务配置，避免把集群内地址写死在镜像内。
