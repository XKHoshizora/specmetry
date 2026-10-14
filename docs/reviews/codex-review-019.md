# Codex 审查 019：WP1.4 Review 018 修复最终定向复审

- 审查日期：2026-10-15（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #7](https://github.com/XKHoshizora/specmetry/pull/7)
- 最终审查目标：`3d5272b6020ee5dbda26d361a1542e6c819885a2`
- 修复基准：`e29b42611bfee6e4636eca062673da1ddb0574e5`
- 上一轮报告：[Codex Review 018](codex-review-018.md)
- 目标 CI：[CI 运行 38595448856](https://github.com/XKHoshizora/specmetry/actions/runs/38595448856)
- 范围：Review 018 的 B1 符号链接目标路径脱敏、B2 Marker 原子发布、B3 Worktree 归属修复，以及必要的初始化、来源管理、PD-25、隐私和 Schema 脱敏回归。不重新审查已验收的 M0、WP1.0～WP1.3，不提前要求 WP1.5 或 M2 功能。

**最终结论：GO。** Review 018 的 B1、B2、B3 均已关闭，未发现新的可复现 BLOCKER。WP1.4 可以按当前工作包范围正式验收，PR #7 可以合并。

审查目标为 `3d5272b6020ee5dbda26d361a1542e6c819885a2`，修复基准为 `e29b426`。

审查期间原仓库保持只读，故障注入和攻击性测试均在隔离临时环境中运行，未提交、推送或合并。本报告随后按照用户要求单独保存。

## BLOCKER

无。原有问题的关闭证据如下：

| 问题 | 复审结果 |
|---|---|
| **B1：符号链接目标泄露** | [Resolve](../../internal/source/source.go#L131) 的拒绝错误只使用已扫描的原始参数，不再拼入目标路径，也不转述 Git 归属识别的原始错误。独立 Linux 复验原始 `alias.md` 反例及损坏 Git 配置场景：文本和 JSON 错误均未泄露 Token 或目标路径，退出码 2，来源表及 seq 不变。合法符号链接仍可登记。**关闭。** |
| **B2：Marker 覆盖竞态** | [publishMarker](../../internal/project/files.go#L254) 已完全删除 rename 回退。Windows、WSL Linux 的真实硬链接测试中，16 个不同 Marker 并发发布、连续 20 轮均恰好一个成功。Linux 通过禁止真实 `link/linkat` 系统调用复验：全部返回环境错误，没有 Marker 或残留临时文件。已有相同、不同 Marker 的行为及字节不变性测试通过。**关闭。** |
| **B3：关联 Worktree 错误登记** | [inLinkedWorktree](../../internal/source/source.go#L186) 根据目标目录的 Git 元数据判断归属，使用 `os.SameFile` 比较公共目录及 Worktree 身份，并在路径前缀分类之前拒绝关联 Worktree。公开 CLI 测试覆盖主→A、A→B、嵌套 Worktree、符号链接、合法外部目录及独立仓库；无法判断时保守拒绝。**关闭。** |

额外独立验证了数据库登记成功、硬链接发布失败后的恢复：第一次公开 `init` 返回退出码 6；恢复硬链接后再次执行，复用原项目和 Worktree、补写 Marker，seq 不再增加。

## NON-BLOCKER

没有新增项。Review 018 的 `propertyNames` 错误定位不准仍保留记录；本轮没有修改共享校验器，相关脱敏回归测试通过。

## ACCEPTED RISK

- 不支持可靠硬链接发布的文件系统可以拒绝初始化，符合产品负责人批准的边界。已登记项目保留，修复环境后可重试。
- Source Ref 相关验收仍随 WP1.5/M2 完成；`full` 模式擦除与整理继续延期。
- Git exclude 重复、Windows seq 偶发 `SQLITE_BUSY`、doctor Git Trace 等已有事项维持原安排，本轮没有新的阻塞证据。

## 验证范围与最终回答

- Windows 临时副本中的定向回归通过：来源命令、正常及并发初始化、Marker、PD-25、`hash_only` 隐私和共享 Schema 脱敏。
- 本地 Windows 的符号链接测试因权限不足跳过；独立 Linux 复验通过，三平台 CI 日志中的对应测试均实际通过。
- 已核实[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38595448856) **8/8 成功**。实际测试的合并提交 `bfc60bc49bd0d960d2ffb602cd5cc6fb942b8a0d` 对应将最终目标 `3d5272b` 合入 `568432d`；Ubuntu 启用了 Race Detector。macOS 依据实际 CI 日志，未进行独立本地复跑。
- 新增回归测试具备有效前置条件和失败断言；硬链接故障注入结果与真实文件系统结果已分别核验。

**B1～B3 全部关闭；没有发现新的隐私泄露、Marker 覆盖或错误 Worktree 归属问题。WP1.4 代码审查通过，可以验收并合并 PR #7。此结论不代表整个 M1 已完成验收。**

原仓库在复审期间保持未修改，未提交、推送或合并；此后仅按用户要求新增本报告。
