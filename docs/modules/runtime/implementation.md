# Runtime 模块实现文档（配置与 Redis）

## 文件清单
- `configs/grpcserver.yaml`
- `internal/pkg/config.go`
- `internal/pkg/redis.go`

## 实现说明
- 配置文件承载运行参数：
  - gRPC 监听地址
  - Redis 地址/密码/DB/Hash Key
  - 固定 3 话题
  - 请求超时
  - 日志级别（`logging.level`）
- `config.go` 负责：
  - 读取 YAML
  - 默认值兜底
  - 配置合法性校验（包含“固定 3 话题”约束）
- `redis.go` 负责：
  - Redis 客户端创建
  - 连接健康检查
  - 固定话题初始化

## 函数定位
- `LoadConfig(path string)`  
  - 文件：`internal/pkg/config.go`  
  - 职责：读取并解析配置文件，返回配置对象。
- `(c *AppConfig) RequestTimeout()`  
  - 文件：`internal/pkg/config.go`  
  - 职责：解析请求超时配置，异常时回退默认值。
- `defaultConfig()`  
  - 文件：`internal/pkg/config.go`  
  - 职责：构造默认配置（包括默认 3 话题）。
- `validateConfig(cfg)`  
  - 文件：`internal/pkg/config.go`  
  - 职责：校验必填项、话题数量与唯一性。
- `NewRedisClient(cfg RedisConfig)`  
  - 文件：`internal/pkg/redis.go`  
  - 职责：创建 Redis 客户端实例。
- `PingRedis(ctx, client)`  
  - 文件：`internal/pkg/redis.go`  
  - 职责：验证 Redis 连通性。
- `EnsureTopicsInitialized(ctx, client, hashKey, topics)`  
  - 文件：`internal/pkg/redis.go`  
  - 职责：将固定话题初始化为 0（若字段不存在）。

## 维护要求
- 新增配置字段时：同步更新 `configs/grpcserver.yaml`、`AppConfig` 结构与本文件函数/字段说明。
