# Deployment 模块原理文档（Phase 5）

## 1. 多阶段构建
- 构建镜像时在 builder 阶段完成编译，在 runtime 阶段仅保留二进制与运行时必需文件，减小镜像体积。

## 2. 最小权限运行
- 运行镜像创建并使用非 root 用户，降低容器逃逸或误操作风险。

## 3. 服务发现与负载均衡
- `grpcserver` 使用 Headless Service，配合客户端 `round_robin`，让 HttpServer 能将请求分发到多个 gRPC Pod。

## 4. 配置外置化
- 使用 ConfigMap 覆盖服务配置（地址、日志级别、话题集），避免为环境差异频繁重建镜像。

## 5. 可运维性
- Deployment 配置了 `readinessProbe` 与 `livenessProbe`，便于 K8s 感知服务健康状态并自动恢复。

## 6. Redis 部署策略
- 当前阶段 Redis 明确采用单实例部署，仅承担投票计数存储。
- 不引入 Redis Cluster、主从或 Sentinel，降低复杂度并聚焦核心业务链路稳定性。
