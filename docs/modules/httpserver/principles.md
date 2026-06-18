# HttpServer 模块原理文档（Phase 3）

## 1. 网关职责
- HttpServer 只做协议转换与页面托管，不承载投票核心计数逻辑。
- 投票业务统一下沉到 GrpcServer，避免逻辑重复。

## 2. 协议转换
- 前端使用 HTTP/JSON，后端核心使用 gRPC。
- 网关将 `/api/*` 请求映射到 `VoteService` RPC，隔离前端与后端服务实现细节。

## 3. 负载均衡准备
- gRPC 客户端启用 `round_robin` 策略，为后续 K8s Headless Service 多副本调用做兼容。

## 4. 错误映射
- 业务参数错误（如非法话题）映射为 HTTP 400。
- 后端服务/网络错误映射为 HTTP 502，方便前端区分输入问题与后端异常。
