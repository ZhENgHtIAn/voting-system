# 在线投票系统项目说明与验收文档

## 1. 项目目标

本系统用于公司内部技术话题投票，目标是让员工通过 Web 页面投票，系统实时展示票数，并用投票结果决定下一次技术分享会主题。

项目按微服务方式实现，采用 Go + gRPC + Redis + Kubernetes，满足以下关键要求：

- 后端使用 Go 开发
- 微服务拆分为 `httpserver` 与 `grpcserver`
- `httpserver` 与 `grpcserver` 在 K8s 中至少 3 副本
- 使用 Redis 存储投票计数，保证并发写入正确性
- 提供可用前端页面，支持投票后实时刷新
- 单元测试覆盖率 > 30%

---

## 2. 架构与数据流（详细）

### 2.1 组件职责

1) Web 前端 (`web/index.html`, `web/app.js`)
- 展示话题与票数
- 发送 HTTP 请求给 `httpserver`
- 投票后刷新最新结果

2) HTTP 网关 (`httpserver`)
- 提供 REST API：
  - `GET /api/results`
  - `POST /api/vote`
- 负责把 HTTP 请求转换为 gRPC 调用
- 托管前端静态资源，避免跨域

3) gRPC 业务服务 (`grpcserver`)
- 提供 `VoteService`：
  - `GetResults`
  - `CastVote`
- 执行话题校验、投票写入、结果读取
- 直接访问 Redis

4) Redis
- 使用 Hash：`voting:topics`
- Field 为话题名，Value 为票数
- 写操作采用 `HINCRBY` 原子自增，解决并发覆盖问题

### 2.2 请求链路

完整链路：

`Browser -> httpserver -> grpcserver -> Redis -> grpcserver -> httpserver -> Browser`

#### A. 页面初始化（读取结果）

1. 浏览器加载 `http://<host>:8080/`
2. `web/app.js` 调用 `GET /api/results`
3. `httpserver` 调用 gRPC `GetResults`
4. `grpcserver` 执行 Redis `HGETALL voting:topics`
5. 返回 `map<string, int64>` 到前端并渲染

#### B. 用户投票（写入并返回最新结果）

1. 浏览器点击某话题投票按钮
2. 前端发 `POST /api/vote`，Body: `{"topic_name":"Golang"}`
3. `httpserver` 调 gRPC `CastVote`
4. `grpcserver` 校验话题是否在固定三话题白名单中
5. 执行 `HINCRBY voting:topics Golang 1`
6. 再执行 `HGETALL voting:topics` 返回全量最新票数
7. 前端收到响应后直接重绘票数

### 2.3 并发一致性设计

并发风险点是“同时投票可能互相覆盖”。  
系统采用 Redis 原子命令 `HINCRBY`，避免了“先读后写”竞态：

- 错误模式：GET -> +1 -> SET（会丢写）
- 当前实现：`HINCRBY` 单命令原子更新（不会丢写）

### 2.4 可观测性与链路追踪

系统增加了统一结构化日志：

- 级别：`debug/info/warn/error`
- 字段：时间、服务名、消息、扩展字段
- HTTP 层为请求生成/透传 `X-Request-ID`
- gRPC 拦截器记录 method、request_id、code、duration

可用日志定位一次完整请求在多服务间的流转。

---

## 3. 代码组织

```text
voting-system/
├── api/pb/                        # proto 契约 + 生成代码
├── cmd/
│   ├── grpcserver/                # gRPC 服务启动入口
│   ├── httpserver/                # HTTP 网关启动入口
│   ├── grpcclient/                # 调试客户端
│   └── redistest.go               # Redis 本地验证脚本
├── internal/
│   ├── grpcserver/                # VoteService 业务实现
│   ├── httpserver/                # HTTP handler、配置、路由
│   └── pkg/
│       ├── config.go              # grpcserver 配置
│       ├── redis.go               # Redis 客户端封装
│       └── logging/logger.go      # 统一日志组件
├── configs/                       # http/grpc 运行配置
├── web/                           # 前端静态页面
├── deployments/
│   ├── docker/                    # Dockerfile.grpc / Dockerfile.http
│   └── k8s/                       # redis/grpcserver/httpserver K8s 清单
├── docs/                          # 总览与分模块实现/原理文档
└── scripts/                       # proto 生成、调试与演示脚本
```

---

## 4. 接口定义

### 4.1 gRPC 接口（`api/pb/vote.proto`）

- `CastVote(VoteRequest) returns (VoteResponse)`
- `GetResults(Empty) returns (VoteResponse)`
- `VoteRequest.topic_name`：投票话题
- `VoteResponse.results`：全部话题票数映射

### 4.2 HTTP 接口（`httpserver` 暴露）

- `GET /api/results`
  - 返回：`{"results":{"Golang":1,"Kubernetes":0,"Rust":0}}`
- `POST /api/vote`
  - 请求：`{"topic_name":"Golang"}`
  - 返回：最新全量票数

---

## 5. 配置说明

### 5.1 `configs/grpcserver.yaml`
- gRPC 监听地址
- Redis 地址/密码/DB/Key
- 固定三话题
- 请求超时
- 日志级别

### 5.2 `configs/httpserver.yaml`
- HTTP 监听地址
- gRPC 目标地址（`dns:///grpcserver-headless:50051`）
- 请求超时
- 静态资源目录
- 日志级别

---

## 6. 测试与覆盖率

### 6.1 什么是单元测试覆盖率

单元测试覆盖率表示“测试执行时，代码语句被运行到的比例”。  
常见表达是 statements coverage（语句覆盖率）：

`覆盖率 = 被测试执行过的语句数 / 总语句数 * 100%`

在 Go 中通过如下命令计算：

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### 6.2 当前项目测试现状

当前测试已迁移至统一目录 `internal/test`：

- `internal/test/test.go`（测试逻辑与公共测试基建）
- `internal/test/test_test.go`（`go test` 入口）

当前已通过的测试覆盖 HTTP、gRPC 和并发一致性：

- HTTP 配置加载/校验/回退测试
- HTTP Handler 成功与异常分支测试
- gRPC `CastVote` / `GetResults` 核心行为测试
- gRPC 并发 `CastVote` 一致性测试
- Web 入口全链路并发集成测试（`Web -> HTTP -> gRPC -> Redis`）

本地覆盖率结果（最近一次）：

- `go test -v ./internal/test` 全部通过
- `go test -v -coverprofile=coverage.out ./...`
- `go tool cover -func=coverage.out`
- 覆盖率输出为 `80.0%`（高于要求的 `>30%`）

### 6.3 覆盖率口径说明

当前 `go test -coverprofile=coverage.out ./...` 的 `80.0%`，来自 `internal/test` 包自身语句覆盖。  
若要统计“被测试调用到的业务包”的覆盖率，建议使用：

```bash
go test -v -coverpkg=./... -coverprofile=coverage.out ./internal/test
go tool cover -func=coverage.out
```

### 6.4 并发一致性结论

并发一致性已通过两层验证：

1) gRPC 并发一致性  
- 多协程并发调用 `CastVote`，最终票数与请求数完全一致

2) Web 全链路并发一致性  
- 并发请求 `POST /api/vote`（HTTP）  
- 链路经过 `HTTP -> gRPC -> Redis`  
- 最终 `GET /api/results` 结果与总请求数完全一致

### 6.5 并发投票在测试中如何实现

并发投票通过 Go 协程并行发请求实现，核心步骤如下：

1) 创建并发任务  
- 使用 `sync.WaitGroup` 计数，循环启动多个 goroutine（例如 120/240/300）。

2) 每个任务执行一次投票写入  
- HTTP 测试：每个 goroutine 发一次 `POST /api/vote`。  
- gRPC 测试：每个 goroutine 调一次 `CastVote`。

3) 等待所有任务结束  
- 调用 `wg.Wait()`，保证所有并发写入已经完成。

4) 读取最终结果并断言不变量  
- 调用 `GET /api/results` 或 `GetResults`。  
- 断言“最终票数 == 并发请求总数”，验证无丢票、无覆盖写。

### 6.6 非 K8s 本地演示脚本

项目提供 `scripts/demo_local.sh`，用于不依赖 K8s 的本地演示流程（适合录屏）：

```bash
bash scripts/demo_local.sh up
bash scripts/demo_local.sh status
bash scripts/demo_local.sh verify
bash scripts/demo_local.sh logs
bash scripts/demo_local.sh down
```

脚本能力：
- 自动启动（或复用）本地 Redis
- 编译并启动 `grpcserver` / `httpserver`
- 通过 `http://127.0.0.1:8080/` 打开前端页面
- 输出本地日志文件路径，便于演示讲解

### 6.7 K8s 演示脚本

项目提供 `scripts/demo_k8s.sh`，用于演示“构建镜像 -> 部署 K8s -> Web 验证 -> 日志观察”：

```bash
bash scripts/demo_k8s.sh up
bash scripts/demo_k8s.sh up-nobuild
bash scripts/demo_k8s.sh forward
bash scripts/demo_k8s.sh stop-forward
bash scripts/demo_k8s.sh status
bash scripts/demo_k8s.sh url
bash scripts/demo_k8s.sh verify
bash scripts/demo_k8s.sh logs http
bash scripts/demo_k8s.sh logs grpc
bash scripts/demo_k8s.sh logs redis
bash scripts/demo_k8s.sh reset-votes
bash scripts/demo_k8s.sh down
```

说明：
- `up`：`build + load + deploy`
- `up-nobuild`：跳过构建，直接 `load + deploy`（适合镜像已构建完成的二次演示）
- 脚本会自动启动端口转发，推荐通过 `http://127.0.0.1:18080/` 访问前端
- 若转发中断，可手动执行 `bash scripts/demo_k8s.sh forward`

命令描述（与脚本 help 一致）：
- `build`：构建 `grpcserver/httpserver` Docker 镜像
- `load`：导出并加载镜像到 Minikube
- `deploy`：应用 K8s YAML 并等待服务就绪
- `up`：执行 `build + load + deploy`
- `up-nobuild`：跳过构建，直接 `load + deploy`
- `forward`：启动本地端口转发（`127.0.0.1:18080 -> svc/httpserver-nodeport:8080`）
- `stop-forward`：停止本地端口转发
- `verify`：执行接口验证（`GET -> POST -> GET`）
- `url`：输出 Web 访问地址
- `logs`：查看服务日志（`http/grpc/redis`）
- `status`：查看 Deployment/Pod/Service 状态
- `reset-votes`：清空 Redis 投票数据
- `down`：删除 K8s 资源

### 6.8 单元测试与覆盖率演示脚本

项目提供 `scripts/demo_test.sh`：

```bash
bash scripts/demo_test.sh run
```

等价分步骤命令：

```bash
bash scripts/demo_test.sh unit
bash scripts/demo_test.sh cover
bash scripts/demo_test.sh report
```

---

## 7. Redis 数据重置方法

你指出的问题成立：当前没有提供单独的“重置票数”HTTP/gRPC 接口。  
当前可用的运维重置方式是直接在 Redis 执行清理命令，例如：

```bash
redis-cli DEL voting:topics
```

如果在 K8s 中：

```bash
kubectl exec -it deploy/redis -- redis-cli DEL voting:topics
```

这是当前实现下可行且最简单的重置方式。后续如需产品化，可增加管理员重置接口（需鉴权）。

---

## 8. 容器化与 K8s 部署

### 8.1 Docker

- `deployments/docker/Dockerfile.grpc`
- `deployments/docker/Dockerfile.http`

特性：

- 多阶段构建（builder + runtime）
- Alpine 运行镜像
- 非 root 用户运行

### 8.2 Kubernetes

- `deployments/k8s/redis.yaml`
  - Redis 单实例：`Deployment(1) + ClusterIP Service`
- `deployments/k8s/grpcserver.yaml`
  - `Deployment(replicas: 3)`
  - `Headless Service (clusterIP: None)`
- `deployments/k8s/httpserver.yaml`
  - `Deployment(replicas: 3)`
  - `NodePort Service (30080)`

说明：Redis 按当前策略保持单实例，不做 Redis 集群化。

---

## 9. 本地验证步骤（Minikube，完整可执行）

### 9.1 启动 Minikube

```bash
minikube start
kubectl config current-context
kubectl get nodes
```

### 9.2 构建服务镜像（本机 Docker）

如需代理，先在当前 shell 设置代理变量（示例端口 7897）：

```bash
export http_proxy=http://127.0.0.1:7897
export https_proxy=http://127.0.0.1:7897
export HTTP_PROXY=http://127.0.0.1:7897
export HTTPS_PROXY=http://127.0.0.1:7897
```

构建镜像：

```bash
docker build --network=host \
  --build-arg http_proxy=$http_proxy \
  --build-arg https_proxy=$https_proxy \
  -f deployments/docker/Dockerfile.grpc \
  -t voting-system/grpcserver:latest .

docker build --network=host \
  --build-arg http_proxy=$http_proxy \
  --build-arg https_proxy=$https_proxy \
  -f deployments/docker/Dockerfile.http \
  -t voting-system/httpserver:latest .
```

### 9.3 导入镜像到 Minikube

```bash
docker save -o /tmp/voting-grpcserver.tar voting-system/grpcserver:latest
docker save -o /tmp/voting-httpserver.tar voting-system/httpserver:latest

minikube image load /tmp/voting-grpcserver.tar
minikube image load /tmp/voting-httpserver.tar
```

### 9.4 部署 K8s 资源

```bash
kubectl apply -f deployments/k8s/redis.yaml
kubectl apply -f deployments/k8s/grpcserver.yaml
kubectl apply -f deployments/k8s/httpserver.yaml

kubectl get deploy
kubectl get pods
kubectl get svc
```

### 9.5 访问 Web 与接口验证

Web 页面由 `httpserver` 托管，不需要额外单独启动前端服务。  
获取访问地址并验证：

```bash
MINIKUBE_IP=$(minikube ip)
echo "Open in browser: http://$MINIKUBE_IP:30080/"

curl -s http://$MINIKUBE_IP:30080/api/results
curl -s -X POST http://$MINIKUBE_IP:30080/api/vote \
  -H "Content-Type: application/json" \
  -d '{"topic_name":"Golang"}'
curl -s http://$MINIKUBE_IP:30080/api/results
```

### 9.6 集群启动后 Web 联调验证（推荐）

先确认服务全部就绪：

```bash
kubectl rollout status deploy/redis
kubectl rollout status deploy/grpcserver
kubectl rollout status deploy/httpserver
kubectl get pods -o wide
```

访问 Web：

```bash
MINIKUBE_IP=$(minikube ip)
echo "Web URL: http://$MINIKUBE_IP:30080/"
```

打开浏览器进入上面 URL，连续点击不同话题投票按钮。  
同时在终端执行：

```bash
curl -s http://$MINIKUBE_IP:30080/api/results
```

若页面显示和接口返回票数持续递增，说明 Web 前端与集群后端联调正常。

---

## 10. 多 Pod 日志查看方法

### 10.1 查看某个 Deployment 的所有 Pod 名称

```bash
kubectl get pods -l app=httpserver
kubectl get pods -l app=grpcserver
kubectl get pods -l app=redis
```

### 10.2 查看 Deployment 聚合日志

```bash
kubectl logs deploy/httpserver --tail=200
kubectl logs deploy/grpcserver --tail=200
kubectl logs deploy/redis --tail=200
```

### 10.3 指定某个 Pod 实时跟踪日志

```bash
kubectl logs -f <pod-name>
kubectl logs -f <pod-name> --previous
```

### 10.4 推荐日志观察方式（3 个终端）

终端 A（HTTP 网关）：

```bash
kubectl logs -f deploy/httpserver --tail=200
```

终端 B（gRPC 服务）：

```bash
kubectl logs -f deploy/grpcserver --tail=200
```

终端 C（Redis）：

```bash
kubectl logs -f deploy/redis --tail=200
```

然后在浏览器页面投票，或执行：

```bash
MINIKUBE_IP=$(minikube ip)
curl -s -X POST http://$MINIKUBE_IP:30080/api/vote \
  -H "Content-Type: application/json" \
  -d '{"topic_name":"Rust"}'
```

即可观察三层服务日志联动：
- `httpserver`：接收请求、转发 gRPC、响应状态
- `grpcserver`：话题校验、投票写入、结果读取
- `redis`：连接与命令执行日志（镜像默认日志较少）

---

## 11. 验收对照（对应作业要求）

- [x] 1. Go 语言后端服务
- [x] 2. 包含 httpserver 与 grpcserver 微服务
- [x] 3. 两个服务均为 3 副本部署
- [x] 4. 数据流转：Web -> HTTP -> gRPC -> Redis 完整
- [x] 5. 使用 Redis `HINCRBY` 处理并发计数
- [x] 6. Web 页面至少 3 话题且投票后刷新
- [x] 7. 单元测试覆盖率 > 30%
- [x] 8. 提供架构与实现文档

---
