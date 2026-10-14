# Codex 审查 018：WP1.4 Source Management 独立代码审查

- 审查日期：2026-10-15（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #7](https://github.com/XKHoshizora/specmetry/pull/7)
- 最终审查目标：`e29b42611bfee6e4636eca062673da1ddb0574e5`
- 增量基准：`568432d992d7b81b820c5741dc4ab7a6d6fccc7b`（`origin/master`）
- 目标 CI：[CI 运行 38583920485](https://github.com/XKHoshizora/specmetry/actions/runs/38583920485)
- 范围：WP1.4 来源管理、PD-25 来源集合哈希、共享 JSON Schema 错误脱敏，以及本轮 macOS 并发初始化的权限检查和 Marker 发布修复。

**结论：NO-GO。** 在目标提交 `e29b426` 中独立复现了 3 项 BLOCKER，当前不应验收 WP1.4 或合并 PR #7。

审查基准为 `568432d`。已核实指定 CI 的 8 项任务全部成功，包含三平台 Go 测试及 Ubuntu Race Detector；但现有测试未覆盖以下反例。CI 实际测试的 PR 合并提交为 `8b90209f9254d5b6b37c2d1b6dd30b4949f95c70`，对应将目标 `e29b426` 合入基准 `568432d`。

审查期间原仓库保持只读，所有反例均在隔离的临时副本、临时仓库和数据目录中运行，未提交、推送或合并。本报告随后按照用户要求单独保存。

## BLOCKER

### B1：符号链接目标中的密钥可通过错误消息泄露

位置：[internal/source/source.go:147](../../internal/source/source.go#L147)、[internal/cli/source.go:221](../../internal/cli/source.go#L221)。

在关联 Worktree 中创建安全名称 `alias.md`，让它指向文件名含合成 GitHub Token 的文档，然后执行 `source add alias.md --json`。

独立 Linux 测试确认：命令返回退出码 2，但 Token 出现在 `error.v1` 中。原因是 CLI 先扫描原始参数；`Resolve` 在拒绝关联 Worktree 文件时，直接拼入解析后的目标相对路径，提前绕过了后续路径扫描。

影响：真实敏感路径可以进入 Agent 上下文或被调用方保存的错误日志。共享 Schema 脱敏无法保护这一普通业务错误。

最小修复：拒绝消息只使用已检查的原始参数，或者在使用解析后的路径构造错误前完成扫描。增加安全别名指向敏感目标的公开 CLI 测试，覆盖文本和 JSON 错误。

### B2：Marker 的 rename 回退仍可覆盖并发创建的不同 Marker

位置：[internal/project/files.go:249](../../internal/project/files.go#L249)。

硬链接失败后，代码先检查目标不存在，再执行 `os.Rename`。其他进程可以在两步之间创建 Marker，随后被替换。

独立 Linux 测试通过限制测试子进程的 `link/linkat`，让真实 `os.Link` 返回“不支持”，再调用未经修改的公开 `WriteMarker`。16 个不同 Marker 并发发布时，第 2 轮即出现：

- 两个不同 Marker 的写入均返回成功；
- 最终文件只保留其中一个，另一个成功结果对应的 Marker 已被覆盖。

这是对硬链接不可用条件的故障注入复现，未声称已在真实 FAT 或原生 macOS 上复现。

影响：项目 Marker 可以被静默替换，与数据库登记的项目身份不一致，违反“不同内容不能覆盖”的契约。

现有[并发测试](../../internal/project/files_test.go#L325)使用相同 Marker，且没有强制进入回退路径；顺序冲突测试也不能验证此竞态。

最小修复：使用原子“不覆盖”发布机制；无法提供该保证时明确失败。不能用检查不存在后的普通 rename 代替原子保证。补充不同内容、硬链接失败条件下的并发测试。

### B3：从主 Worktree 可以登记关联 Worktree 的文件

位置：[internal/source/source.go:145](../../internal/source/source.go#L145)。

当前只排除“目标位于当前关联 Worktree”的情况。命令在主 Worktree 执行时，检查被完全跳过。

独立 Windows、Linux 公开 CLI 测试均确认：从主 Worktree 登记关联 Worktree 中的 `only-feature.md`，实际返回：

- 退出码 0；
- `created = true`；
- `location = external`；
- 新增来源并分配 seq。

这直接违反 architecture §6.1.3 中关联 Worktree 文件必须返回退出码 2 的规定，使项目共享来源绑定到分支 Worktree 的内容。

最小修复：根据目标文件所属 Worktree 及其 Common Git Directory 判断，不能仅比较命令当前所在 Worktree。补充主 Worktree→关联 Worktree、关联 Worktree A→B 的公开调用测试。

## NON-BLOCKER

### N1：`propertyNames` 错误定位仍可能退化为根节点

位置：[internal/contract/redact.go:60](../../internal/contract/redact.go#L60)。

独立构造仅违反 `propertyNames` 的输入，确认错误指针为 `/`，而不是实际映射位置。这与 jsonschema v6.0.3 的定位问题一致。

该输入仍被正确拒绝，任意属性名没有泄露，因此不阻塞。应保留后续修复记录，在修正依赖或定位处理时继续维持脱敏测试。

## ACCEPTED RISK

- WP1.5 之前没有真实 Source Ref，`--refs` 返回空列表符合当前边界；引用保留、`source_removed`、`excerpt_state` 的完整验收继续延期。
- M1 只支持 `hash_only`。当前移除行为没有宣称完成 `full` 模式物理擦除；擦除与整理仍属于 M2/WP2.11。
- 超大来源文件采用流式哈希并支持取消，没有大小上限，符合已批准行为。
- 已记录的 Git exclude 重复、Windows seq 偶发 `SQLITE_BUSY`、非 Git 扫描数量限制及独立 doctor Git Trace 任务，本轮没有获得升级为 WP1.4 BLOCKER 的新证据。doctor Git Trace 仍是发布前必须关闭的独立安全任务。

## 验证结果与最终判断

| 审查项 | 结论 |
|---|---|
| Source Add/List/Ack/Remove | List、Ack、Remove 的已实现路径符合当前要求；Add 因 B1、B3 未通过 |
| 人工操作与并发一致性 | Agent 拦截、确认、取消、过期哈希拒绝、Ack 前后复核及事务处理未发现新的放行问题；Marker 发布仍受 B2 影响 |
| 共享 Schema 脱敏 | 通过本轮独立验证，存在 N1 定位限制 |
| macOS 并发初始化 | 消失文件检查修复成立；正常硬链接发布路径的修复成立，但 rename 回退尚未关闭 |
| PD-25 | 实现正确：输入顺序不影响哈希；ID 或已确认 SHA 改变会改变哈希；未 Ack 的磁盘变化不会改变哈希 |
| 延期验收 | Source Ref 相关内容随 WP1.5/M2；`SOURCE_CHANGED` 差距及完整擦除随 M2 |
| 验收并合并 PR #7 | **暂不可以，须先修复 B1～B3** |

独立验证包括临时副本中的 Windows 全量 Go 测试，以及新增的公开 CLI、Schema 和 Linux 权限反例。Schema 测试覆盖 pattern、format、类型、数值、长度、enum、const、任意属性名、数字映射键和真实数组下标；公开 `source ack --file` 的文本及 JSON 错误未回显敏感输入，seq 未变化。

两条 macOS 诊断 Actions 的失败与修复后结果已核对，并结合确定性删除测试确认权限检查修复有效：

- [修复前诊断](https://github.com/XKHoshizora/specmetry/actions/runs/38583919277)：并发 Init 48 次通过、2 次失败；数据库创建期间权限检查 5 次通过、15 次失败；并发 Marker 测试 50 次全部失败。
- [修复后诊断](https://github.com/XKHoshizora/specmetry/actions/runs/38583919241)：并发 Init 50 次通过；数据库创建期间权限检查 20 次通过；并发 Marker 测试 50 次通过；消失文件及祖先严格检查共 60 次通过。

这些结果支持消失文件检查及正常硬链接发布路径的修复，不能替代 B2 中硬链接不可用、不同 Marker 并发发布的验收。原生 macOS 的独立本地复验未执行，依据为实际 CI 日志与代码检查。

**最终代码审查：NO-GO。** 原仓库在审查期间保持未修改，未提交、推送或合并；此后仅按用户要求新增本报告。
