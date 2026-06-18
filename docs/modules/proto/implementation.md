# Proto 模块实现文档（Phase 1）

## 文件清单
- `api/pb/vote.proto`
- `api/pb/vote.pb.go`
- `api/pb/vote_grpc.pb.go`
- `scripts/gen_proto.sh`

## 实现说明
- `vote.proto` 定义 `VoteService` 两个 RPC：
  - `CastVote(VoteRequest) -> VoteResponse`
  - `GetResults(Empty) -> VoteResponse`
- `go_package` 使用模块绝对路径：`example.com/voting-system/api/pb;pb`。
- `scripts/gen_proto.sh` 通过显式 `--plugin` 路径调用 `protoc`，并使用 `paths=source_relative` + `--proto_path` 项目根目录，确保生成文件固定落在 `api/pb/`。

## 函数定位
- `NewVoteServiceClient(cc)`  
  - 文件：`api/pb/vote_grpc.pb.go`  
  - 职责：创建 gRPC 客户端桩。
- `RegisterVoteServiceServer(s, srv)`  
  - 文件：`api/pb/vote_grpc.pb.go`  
  - 职责：向 gRPC Server 注册服务实现。
- `(*VoteRequest) GetTopicName()`  
  - 文件：`api/pb/vote.pb.go`  
  - 职责：读取请求中的话题名字段。
- `(*VoteResponse) GetResults()`  
  - 文件：`api/pb/vote.pb.go`  
  - 职责：读取响应中的投票结果映射。

## 生成与更新规则
- 修改 `vote.proto` 后必须重新执行 `./scripts/gen_proto.sh`。
- 重新生成后需要同步更新本文件中的函数定位与契约说明。
