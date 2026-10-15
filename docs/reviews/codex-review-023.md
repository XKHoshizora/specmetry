# Codex 审查 023：WP1.5 PD-26 / TD-26 最终定向独立审查

- 审查日期：2026-10-15
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #8](https://github.com/XKHoshizora/specmetry/pull/8)
- 最终目标提交：`27a49af930f390ad0b4a2bef14d85ee1894bfed4`
- PD-26 实现基准：`96be9963e3ca20a066d0bb31b053f445d584bbbc`
- TD-26 增量基准：`25a93dfda21c3123428832593607b48e56d6bdbb`
- 前轮报告：[Codex Review 022](codex-review-022.md)
- CI：[Actions run 38735281951](https://github.com/XKHoshizora/specmetry/actions/runs/38735281951)
- 范围：PD-26 运行时安全能力探测、Sparse Index、TD-26 SHA-256 拒绝，以及直接影响的 WP1.1 和 WP1.5 调用路径；不全面重审已关闭事项，不提前要求 WP1.6 或 M2 功能。
- 操作记录：审查期间原仓库保持只读；独立验证使用隔离源码副本和无网络测试。本报告随后按用户要求单独保存，未修改其他仓库文件，未提交、推送或合并。

**最终结论：NO-GO。** PD-26 的门禁结构、Sparse Index 和 TD-26 的拒绝逻辑基本符合批准方案，但确认了 **1 项 BLOCKER：异常 Trace2 事件流可能被错误分类为 `neverFetches`**，违反探测失败时必须拒绝的要求。

目标提交已核对为 `27a49af9`。审查期间原仓库工作区干净，未修改、提交、推送或合并。

## BLOCKER

### B1：Trace2 解析器会把存在子进程证据的不完整日志判为安全

位置：

- [objects.go:428](../../internal/gitx/objects.go#L428)：后代 SID 的事件被直接忽略。
- [objects.go:423](../../internal/gitx/objects.go#L423)：重复退出事件覆盖此前结果，未验证生命周期顺序。
- [objects.go:285](../../internal/gitx/objects.go#L285)：上述结果可以进入 `neverFetches` 分支。
- [objects.go:90](../../internal/gitx/objects.go#L90)：错误结论随后可被缓存。

**独立验证：**在隔离源码副本中，仅运行纯 JSON 解析测试，没有启动 Git、Helper 或网络操作。以下三种事件流都被接受：

1. 根进程有正常 `start`、`exit`；保留了 `root/child` 的 `fetch` 启动和退出事件，但缺少根进程的 `child_start`。
2. 根进程的 `exit` 出现在 `start` 之前。
3. 根进程先记录退出码 0，再记录退出码 128，实际退出码为 128。

三者均得到：

```text
trusted=true
started=true
exited=true
children=0
fetched=false
```

这满足当前选择 `neverFetches` 的条件。第一种尤其明确：**日志已经包含抓取子进程的证据，解析器仍回答“没有子进程”。**

**实际影响：**异常或不完整事件流可能解除对象读取门禁，并保存错误的安全结论。现有空协议列表仍能阻断传输，但不足以替代旧版 Git 的仓库只读保护。

本轮证明的是**异常事件输入的错误分类**；没有证明官方 Git 2.32、2.43 或 2.56 在正常完整日志下被错误放行，也没有声称复现了新的网络访问或用户仓库写入。

**最小修复：**

- 测试组出现任何后代进程事件，都不能判为“没有子进程”；父子事件缺失或无法关联时返回无法判定。
- 根进程 `start`、`exit` 必须唯一且顺序正确；重复、矛盾或退出状态不一致应拒绝。
- 增加上述纯解析反例，并确认无法判定的结果不缓存。

修复范围可以限制在事件解析与分类，无需改变 PD-26 方案或存储层。

## NON-BLOCKER

此前 HTTPS 测试仅凭非零退出判断阻断的证据缺口仍存在。本轮没有发现新的实际绕过证据，维持非阻塞分类。

## ACCEPTED RISK

以下边界与批准方案一致：

- 不具备安全对象读取能力的 Git，即使达到最低版本 2.32，也拒绝对象读取。
- 不认识 `ls-files --sparse` 时，在用户仓库执行前保守拒绝。
- 缺失 `.gitignore` Blob 可能使候选来源列表变宽；不会因此确认客户需求。
- 能力缓存依赖 Git 路径、大小和修改时间；刻意替换程序并保持这些元数据不变，属于文档明确排除的边界。

## 其余审查结果

| 审查项 | 结论 |
|---|---|
| **能力探测机制** | 正反对照确实检查实际抓取行为，不再依赖版本号或 `--no-lazy-fetch` 选项。超时、取消、无事件等普通失败路径保守拒绝；异常事件完整性受 B1 阻塞。 |
| **统一门禁** | `CommitOf`、`TreeFiles`、`CountLines`、`ListFiles` 均经过能力门禁。未发现当前调用链绕过。 |
| **原配置 TOCTOU** | 已从结构上消除：安全性不再依赖读取前检查用户仓库是否为部分克隆。`PartialClone` 仅用于 Code Ref 拒绝提示。 |
| **Sparse Index** | `--sparse` 支持在私有临时仓库探测；用户仓库中不试探未知参数。目录占位项由项目文件发现层排除。完整与缺失 Tree、缺失 `.gitignore` 的现有回归通过。 |
| **TD-26** | `ObjectFormat` 使用受控命令；SHA-256 和未知格式在数据库打开前拒绝。有提交、无提交、Detached HEAD 三种状态均有真实仓库测试。 |
| **Snapshot** | SHA-1 未出生分支保留 `null`；存在 Ref 但提交不可读取时返回退出码 6。非 SHA-1 快照不会静默转为 `null`。 |
| **事务与 Schema** | 已登记项目目标读取失败会回滚事务，关联 Worktree 不被登记、seq 不变；`init.v1` 未修改。 |

## 验证证据与兼容性

已核对[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38735281951)：8/8 成功，测试合并提交包含最终目标。SHA-256、Sparse Index、能力探测及直接回归在三平台实际运行通过；Ubuntu 启用了 Race Detector。

CI 日志中的实际 Git 版本为：

- Windows：`2.55.0.windows.5`
- Ubuntu：`2.55.0`
- macOS：`2.56.0`

真实 Git 2.32.0 和 Ubuntu 2.43 的结论来自[既有本地兼容性记录](../review-resolution.md#L1466)：前者拒绝对象读取，后者按实际行为通过。真实 Git 2.31.1 的最低版本拒绝也有既有记录。**对应真实旧版测试在三平台 CI 中按设计跳过，不能描述为 CI 已独立验证这些版本。**

本轮还在隔离 Windows 副本重跑了对象格式、Sparse Index、缺失提交、SHA-256 拒绝及初始化回滚测试，均通过。新增纯解析反例则暴露了 B1。

## 最终结论

因此，**TD-26 和正常调用链可通过本轮检查；PD-26 的 fail-closed 要求尚未全部满足。WP1.5 暂不能正式验收，PR #8 暂不建议合并。**

原仓库在审查期间保持不变，未提交、推送、合并，也未开始 WP1.6。本文件为审查完成后按用户要求保存的完整报告。
