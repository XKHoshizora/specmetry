# Codex 审查 024：WP1.5 Review 023 B1 修复最终定向独立复审

- 审查日期：2026-10-15
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #8](https://github.com/XKHoshizora/specmetry/pull/8)
- 目标提交：`de14374bbee0bf055069af9088921b89181f7c7b`
- 增量基准：`27a49af930f390ad0b4a2bef14d85ee1894bfed4`
- 上轮报告：[Codex Review 023](codex-review-023.md)
- CI：[GitHub Actions 38746116548](https://github.com/XKHoshizora/specmetry/actions/runs/38746116548)，Attempt 1，8/8 成功。
- 审查范围：仅复核 Review 023 B1，即 Trace2 事件完整性、安全分类、结果缓存及相关测试；不重新进行 WP1.5 全量审查。
- 验证方式：静态分析、既有 CI 日志核查，以及隔离源码副本中的纯 JSON 解析测试。未运行网络、外部 Helper 或攻击模拟。

**最终结论：GO。Review 023 的 B1 可以关闭，WP1.5 可以通过代码验收，PR #8 可以合并。** 本轮未发现新的实质性 BLOCKER。

## BLOCKER

无。

原始三个反例已用同一份独立纯 JSON 测试在隔离副本中对照验证：

| 反例 | 基准提交 | 目标提交 |
|---|---|---|
| A：后代 Fetch 事件存在，缺少根 `child_start` | 错误满足 `neverFetches` 条件 | 判为不可信，拒绝 |
| B：`exit` 在 `start` 之前 | 错误满足安全条件 | 判为不可信，拒绝 |
| C：先 `exit=0`，后 `exit=128`，实际退出码为 128 | 最后一次退出覆盖此前矛盾，错误满足安全条件 | 判为不可信，拒绝 |

这些独立测试没有启动 Git、Helper 或网络操作。基准提交的三项测试均因错误接受而失败，目标提交均通过，证明反例确实触达原缺陷，且本次修复改变了安全分类结果。

## 修复与测试有效性

[checkProbeEvents](../../internal/gitx/objects.go#L449) 的状态转换可靠地拒绝根进程倒序、重复启动、重复退出及状态码矛盾；`atexit` 必须位于 `exit` 之后且状态一致。合法的无 `atexit` 日志仍被允许。

子进程事件必须对应已启动、尚未退出的 `child_id`；后代 SID 始终成为子进程证据。[showsNoChild](../../internal/gitx/objects.go#L312) 同时检查根进程完整性、子进程数量、Fetch 标记和后代标记，未发现本轮引入的 false-safe 路径。

重复、缺失、矛盾、无关进程以及无法关联的后代事件不能产生可信的无子进程结论。非法 JSON、缺少必要事件字段等读取或解析异常也沿错误路径拒绝，不会变成 `neverFetches`。

缓存链路也符合要求：解析不可信时，[probeRun](../../internal/gitx/objects.go#L331) 返回错误，能力状态不会被更新为 `neverFetches`。既有[门禁测试](../../internal/gitx/probe_events_test.go#L189) 不仅检查退出码 6，还检查：

- A／B／C 后状态保持 `unprobed`；
- 下一次调用重新执行两轮探测；
- 合法无子进程日志判为 `neverFetches`；
- 合法 Fetch 日志判为 `fetches`。

新增异常测试的断言验证了可信度和分类结果，具有实际验收价值。正常及异常纯 JSON 测试在本轮独立运行通过。

本轮独立执行限于纯 JSON 测试；端到端门禁测试的运行结果取自 CI，并结合调用链静态核查，未在本轮重新启动测试替身程序。

## CI 与兼容性证据

已核对[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38746116548)：对应目标提交，8/8 成功。CI 使用的 PR 合并提交为 `bbf47e4`，合并目标提交与当时的 master `35d060f5518d7c3abcd4791377cf7b9fde719e3e`。

以下测试在 Windows、Ubuntu、macOS 均实际运行通过，没有跳过：

- `TestProbeEventsOfRealGit`
- `TestProbeEventsThatShowNothing`
- `TestProbeTrustsOnlyCompleteEvents`
- 使用当前真实 Git 的探测与独立判据对照测试

Ubuntu 启用了 Race Detector。端到端门禁中的合法无子进程、合法 Fetch 和 A／B／C 子用例均在三平台通过。

既有本地兼容性记录仍与新解析器一致：Git 2.32.0 判为会抓取，Ubuntu 2.43 和 2.56 判为不会抓取，2.31.1 在最低版本检查时拒绝。本轮未重新执行这些旧版 Git 兼容性实验。

**真实旧版 Git 测试在 CI 中按设计跳过，不能视为三平台 CI 已独立验证这些版本。** CI 实际使用的当前 Git 为 Windows 2.56.0.windows.2、Ubuntu 2.55.0、macOS 2.55.0；正常 Trace2 与新增解析器测试在这些环境中实际运行通过。

## NON-BLOCKER

保留此前 HTTPS 测试断言偏弱的记录，本轮不要求修复。未发现新的非阻塞问题。

## ACCEPTED RISK

沿用已批准的 PD-26 能力探测与可信 Git 可执行文件边界，未发现本轮新增、需要另行接受的风险。

## 验收结论

| 问题 | 结论 |
|---|---|
| Review 023 B1 是否关闭？ | 关闭。三个原始反例均已通过基准与目标的独立对照验证。 |
| 是否仍有异常或不完整事件流产生 `neverFetches` 并缓存的已知路径？ | 本轮限定范围内未发现。异常分类返回错误，保留 `unprobed`，下一次调用重新探测。 |
| 合法 Fetch 与合法无子进程对照是否仍有效？ | 有效；分类、门禁与正常 Trace2 测试均有对应证据。 |
| 是否发现本次修复直接引入的实质性 BLOCKER？ | 未发现。 |
| WP1.5 最终代码审查结论 | **GO**。 |
| PR #8 是否可以合并？ | **可以合并**。 |

该结论限于当前工作包，不代表后续里程碑已经验收。此前已通过的 PD-26、TD-26、Sparse Index、Init、Proposal 及 B2～B5 等内容未重新展开。

原仓库在复审期间保持干净，未修改、提交、推送或合并，也未开始 WP1.6。本文件为复审完成后按用户要求保存的完整报告。
