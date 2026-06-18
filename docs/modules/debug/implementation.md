# Debug 模块实现文档

## 文件清单
- `cmd/grpcclient/main.go`
- `scripts/debug_grpc.sh`

## 实现说明
- `cmd/grpcclient/main.go` 提供内置 gRPC 调试客户端：
  - `-action results` 调用 `GetResults`
  - `-action vote -topic <name>` 调用 `CastVote`
- `scripts/debug_grpc.sh` 对常用调试命令做了薄封装，便于快速调用。

## 函数定位
- `main()`  
  - 文件：`cmd/grpcclient/main.go`  
  - 职责：解析参数、建立 gRPC 连接并按动作调用 RPC。
- `printResults(results map[string]int64)`  
  - 文件：`cmd/grpcclient/main.go`  
  - 职责：按话题名排序输出结果，便于观察增量变化。

## 用法示例
- 查询结果：`go run ./cmd/grpcclient -action results`
- 投票：`go run ./cmd/grpcclient -action vote -topic Golang`
- 脚本方式：`./scripts/debug_grpc.sh vote Golang`
