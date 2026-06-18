# Deployment 模块实现文档（Phase 5）

## 文件清单
- `deployments/docker/Dockerfile.grpc`
- `deployments/docker/Dockerfile.http`
- `deployments/k8s/redis.yaml`
- `deployments/k8s/grpcserver.yaml`
- `deployments/k8s/httpserver.yaml`

## 实现说明
- Docker 镜像：
  - `Dockerfile.grpc`：构建 `grpcserver` 可执行文件并打包运行镜像。
  - `Dockerfile.http`：构建 `httpserver` 并拷贝 `web/` 静态资源。
- Kubernetes 清单：
  - `redis.yaml`：Redis 单副本部署与服务发现（不做 Redis 集群化）。
  - `grpcserver.yaml`：包含 ConfigMap、3 副本 Deployment、Headless Service。
  - `httpserver.yaml`：包含 ConfigMap、3 副本 Deployment、NodePort Service。

## 资源定位索引
- `kind: Deployment`（Redis）  
  - 文件：`deployments/k8s/redis.yaml`  
  - 职责：运行 Redis 单实例。
- `kind: Service`（redis-service）  
  - 文件：`deployments/k8s/redis.yaml`  
  - 职责：为系统内服务提供 Redis 访问入口。
- `kind: ConfigMap`（grpcserver-config）  
  - 文件：`deployments/k8s/grpcserver.yaml`  
  - 职责：注入 GrpcServer 配置（Redis 地址、话题、日志级别）。
- `kind: Deployment`（grpcserver）  
  - 文件：`deployments/k8s/grpcserver.yaml`  
  - 职责：运行 3 副本 gRPC 服务。
- `kind: Service`（grpcserver-headless）  
  - 文件：`deployments/k8s/grpcserver.yaml`  
  - 职责：提供 Headless Service 供客户端 `round_robin` 发现多个 Pod。
- `kind: ConfigMap`（httpserver-config）  
  - 文件：`deployments/k8s/httpserver.yaml`  
  - 职责：注入 HttpServer 配置（gRPC 目标、web 目录、日志级别）。
- `kind: Deployment`（httpserver）  
  - 文件：`deployments/k8s/httpserver.yaml`  
  - 职责：运行 3 副本 HTTP 网关。
- `kind: Service`（httpserver-nodeport）  
  - 文件：`deployments/k8s/httpserver.yaml`  
  - 职责：通过 NodePort 暴露前端访问入口。

## 操作命令
- 构建镜像：
  - `docker build -f deployments/docker/Dockerfile.grpc -t voting-system/grpcserver:latest .`
  - `docker build -f deployments/docker/Dockerfile.http -t voting-system/httpserver:latest .`
- 部署清单：
  - `kubectl apply -f deployments/k8s/redis.yaml`
  - `kubectl apply -f deployments/k8s/grpcserver.yaml`
  - `kubectl apply -f deployments/k8s/httpserver.yaml`
