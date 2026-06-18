# 文档同步维护规则

## 适用范围
- 适用于 `docs/overview/` 与 `docs/modules/` 下的所有实现文档和原理文档。

## 强制同步规则
1. 任何代码变更都必须在同一批改动中同步更新文档。
2. 若变更涉及函数新增、删除、重命名、迁移，必须同步更新：
   - `docs/overview/implementation.md`
   - 对应模块的 `implementation.md`
3. 若变更涉及架构、协议、错误语义、配置策略，必须同步更新：
   - `docs/overview/principles.md`
   - 对应模块的 `principles.md`
4. `vote.proto` 变更后必须重新生成 `api/pb/*.go`，并同步更新 proto 模块文档。

## 提交前自检清单
- [ ] 代码改动对应的函数定位是否已更新。
- [ ] 原理性变化是否已反映到原理文档。
- [ ] 运行命令与配置说明是否仍然有效。
- [ ] 文档中的路径与函数名是否与代码一致。
