# 技术分享会投票系统 (Voting System)

一家科技公司内部使用的技术话题投票系统。本项目基于 **Go 语言** 与 **微服务架构** 开发，旨在使用户能够实时、高并发地票选出最受欢迎的技术分享话题。

## 🎯 项目要求与特性

- **语言**: 后端完全基于 Golang 开发。
- **微服务架构**:
  - `httpserver`: 无状态 HTTP 网关，负责托管前端页面并接收 Web 端的 RESTful 投票请求。
  - `grpcserver`: 无状态 gRPC 服务，负责处理核心的投票业务逻辑。
- **并发安全**: 使用 Redis Hash 结构及原子自增命令 (`HINCRBY`) 处理并发投票，避免数据覆盖问题。
- **实时刷新**: 前端页面可实时获取并显示各选项的最新投票数。
- **云原生部署**: 采用 Docker 容器化，并部署于 Kubernetes (K8s) 集群。其中 `httpserver` 与 `grpcserver` 配置 3 个副本 (Pod)，`Redis` 维持单实例部署，并解决 gRPC 的负载均衡问题。

## 📂 目录结构规范

```text
voting-system/
├── api/pb/                 # Protobuf IDL 定义文件
├── cmd/                    # 微服务的主程序入口 (main.go)
│   ├── httpserver/
│   └── grpcserver/
├── internal/               # 核心业务逻辑实现
│   ├── httpserver/         # HTTP 路由、Handler 及 gRPC Client
│   ├── grpcserver/         # gRPC 服务端实现
│   └── pkg/                # 公共组件 (如 Redis 客户端封装)
├── web/                    # 前端原生 HTML/JS 静态文件
├── deployments/            # 容器化与 K8s 部署清单
│   ├── docker/
│   └── k8s/
└── scripts/                # 构建及自动生成代码的脚本


