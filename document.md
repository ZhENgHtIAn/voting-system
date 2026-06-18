
# 📝 在线投票系统 (Voting System) 开发指导文档

## 一、 项目概述

本项目旨在开发一个供公司内部使用的在线技术话题投票系统。系统采用微服务架构，基于 Go 语言开发，使用 Redis 作为底层数据存储以应对并发场景，并最终容器化部署至 Kubernetes 集群中。

**核心技术栈：**

* **编程语言：** Go (建议版本 >= 1.18)
* **通信协议：** HTTP/REST (前端与网关) + gRPC (内部微服务通信)
* **数据存储：** Redis (处理高并发计数)
* **前端技术：** HTML + 原生 JavaScript
* **部署环境：** Docker + Kubernetes (K8s)

---

## 二、 架构设计与微服务划分

系统包含以下三个核心组件，数据流向为：`Web Frontend -> HttpServer -> GrpcServer -> Redis`

1. **Web 前端 (Frontend):**
* 展示至少 3 个技术话题（如：Golang, Kubernetes, Rust）。
* 提供投票按钮，通过 AJAX/Fetch 向 HttpServer 发送 HTTP POST 请求。
* 实时（或在每次投票后）更新页面上的各选项得票数。


2. **httpserver (HTTP 接入层 / API 网关):**
* **无状态服务**（K8s 部署 3 个 Pod）。
* 负责托管前端静态文件（HTML/JS）。
* 接收前端的 HTTP 投票请求，将其解析并转换为 gRPC 请求。
* **负载均衡：** 内部配置 gRPC 客户端负载均衡策略（Round Robin），将请求均匀分发给后端的 GrpcServer。


3. **grpcserver (核心业务逻辑层):**
* **无状态服务**（K8s 部署 3 个 Pod）。
* 暴露 gRPC 接口供 HttpServer 调用。
* 负责处理并发投票逻辑，直接与 Redis 交互。


4. **Redis (存储层):**
* 采用 **Hash** 数据结构存储投票数据。Key 为 `voting:topics`，Field 为话题名称，Value 为票数。



---

## 三、 核心难点与解决方案

1. **并发投票导致的数据不一致（Race Condition）：**
* **❌ 错误做法：** 先从 Redis `GET` 当前票数，加 1，再 `SET` 回去。高并发下会导致数据互相覆盖。
* **✅ 正确做法：** 使用 Redis 的原子自增命令 `HINCRBY voting:topics <topic_id> 1`。Redis 核心是单线程执行命令的，这保证了并发修改的绝对安全。


2. **Kubernetes 中的 gRPC 负载均衡失效：**
* **问题：** gRPC 基于 HTTP/2 长连接。K8s 默认的 ClusterIP Service 是基于连接的四层负载均衡。一旦 HttpServer 和 GrpcServer 建立连接，后续所有请求都会打到同一个 GrpcServer Pod 上，导致 3 个 Pod 闲置 2 个。
* **✅ 解决方案：** 将 GrpcServer 的 K8s Service 配置为 **Headless Service** (`clusterIP: None`)。并在 HttpServer 的 gRPC 客户端代码中配置 `round_robin` 负载均衡策略，通过 DNS 轮询解析到所有 Pod 的 IP，实现客户端负载均衡。



---

## 四、 项目目录规范 (Standard Go Project Layout)

请 Agent 严格按照以下目录结构生成文件：

```text
voting-system/
├── api/
│   └── pb/
│       └── vote.proto             # gRPC 接口定义文件
├── cmd/
│   ├── httpserver/
│   │   └── main.go                # HTTP 服务入口
│   └── grpcserver/
│       └── main.go                # gRPC 服务入口
├── internal/
│   ├── httpserver/                # HTTP 路由与 gRPC Client 封装
│   ├── grpcserver/                # gRPC 服务逻辑实现
│   └── pkg/                       # 公共组件 (如 redis 初始化)
├── web/
│   ├── index.html                 # 投票页面 UI
│   └── app.js                     # 页面交互脚本
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile.http
│   │   └── Dockerfile.grpc
│   └── k8s/
│       ├── redis.yaml             # 单机版 Redis 部署
│       ├── grpcserver.yaml        # Headless Service + Deployment (3 replicas)
│       └── httpserver.yaml        # NodePort Service + Deployment (3 replicas)
├── go.mod
└── README.md                      # 项目说明文档

```

---

## 五、 接口定义契约 (API Design)

### 1. Protobuf 定义 (`api/pb/vote.proto`)

定义两个 RPC 方法：获取所有话题及票数、进行投票。

```protobuf
syntax = "proto3";
package vote;
option go_package = "./pb";

service VoteService {
  rpc CastVote (VoteRequest) returns (VoteResponse);
  rpc GetResults (Empty) returns (VoteResponse);
}

message Empty {}

message VoteRequest {
  string topic_name = 1;
}

message VoteResponse {
  map<string, int64> results = 1; // 返回所有话题的最新票数
}

```

### 2. HTTP RESTful API (由 `httpserver` 暴露给前端)

* **获取结果:** `GET /api/results` -> 返回 JSON 格式的所有话题及票数。
* **提交投票:** `POST /api/vote` -> Body `{"topic_name": "Golang"}` -> 返回更新后的所有话题票数。

---

## 六、 Agent 执行步骤指南 (Phase-by-Phase)

*请让 Agent 按以下阶段顺序编写代码，每完成一个阶段进行一次本地测试。*

### Phase 1: 基础环境与契约生成

1. 初始化项目 `go mod init voting-system`。
2. 安装依赖库 (注意兼容性，推荐指定版本)：
* gRPC: `go get google.golang.org/grpc@v1.54.0`
* Protobuf: `go get google.golang.org/protobuf@v1.30.0`
* Redis: `go get github.com/redis/go-redis/v9@v9.0.5`


3. 编写 `vote.proto` 并使用 `protoc` 生成 Go 源码 (`vote.pb.go` 和 `vote_grpc.pb.go`)。

### Phase 2: 开发 GrpcServer (后端核心)

1. 在 `internal/pkg/` 下封装 Redis 客户端连接逻辑。
2. 在 `internal/grpcserver/` 下实现 `VoteServiceServer` 接口。
* `CastVote` 方法：调用 Redis `HIncrBy` 增加对应话题票数，随后调用 `HGetAll` 返回最新完整数据。
* `GetResults` 方法：调用 Redis `HGetAll` 返回当前完整数据。


3. 在 `cmd/grpcserver/main.go` 中启动 TCP 监听并注册 gRPC 服务。

### Phase 3: 开发 HttpServer 与 Web 前端

1. **Web 前端**：编写简单的 HTML，包含三个按钮（例如：Golang, K8s, Rust）。使用 `fetch` 实现 GET 初始化列表和 POST 提交投票。
2. **HttpServer**：
* 初始化 gRPC 客户端连接（使用 `grpc.DialContext`），**注意配置 Round Robin 负载均衡策略**。
* 编写 HTTP Handler：拦截 `/api/results` 和 `/api/vote`，提取参数并调用 gRPC 客户端。
* 配置静态文件服务代理 `web/` 目录，使得打开 `http://localhost:8080/` 即可访问前端（解决跨域问题）。



### Phase 4: 单元测试编写 (覆盖率目标 > 30%)

1. 对 HttpServer 的 Handler 使用 `net/http/httptest` 进行测试。
2. 通过接口（Interface） Mock掉后端的 gRPC 客户端或 Redis 客户端，编写不依赖真实数据库的业务逻辑测试。
3. 指导生成查看测试覆盖率的命令 (`go test -coverprofile=coverage.out ./...`)。

### Phase 5: 容器化与 Kubernetes 部署配置

1. 编写基于多阶段构建的 Dockerfile (`Dockerfile.http`, `Dockerfile.grpc`)，保持镜像轻量（Alpine）。
2. 编写 K8s YAML 清单：
* **Redis**：Deployment (1 Pod) + Service (ClusterIP)。
* **GrpcServer**：Deployment (`replicas: 3`) + Service (**ClusterIP: None**, 端口如 50051)。
* **HttpServer**：Deployment (`replicas: 3`) + Service (Type: **NodePort**，供外部浏览器访问)。



---

## 七、 交付物清单对照 (作业验收标准)

Agent 开发完成后，请对照以下要求确认是否全部满足：

* [ ] 1. Go 语言开发的后端服务。
* [ ] 2. 包含 httpserver (网关) 和 grpcserver (微服务)。
* [ ] 3. Kubernetes YAML 中明确写明了 `replicas: 3`。
* [ ] 4. 数据流转完整：Web -> HTTP -> gRPC -> Redis，且返回结果。
* [ ] 5. 使用 Redis `HINCRBY` 解决了并发投票计数问题。
* [ ] 6. Web 页面能展示至少三个话题，且点击投票后数字能实时刷新。
* [ ] 7. 编写了单元测试，覆盖率超过 30%。
* [ ] 8. 撰写了最终的架构文档，并准备了符合要求的代码仓库。

---

**给你的提示**：接下来，你可以将这个文档的前半部分和“Phase 1”单独喂给 Agent，例如告诉它：“*根据文档的 Phase 1 和目录结构，帮我生成 protobuf 文件和对应的生成脚本*”，这样能够最稳定地推进开发！