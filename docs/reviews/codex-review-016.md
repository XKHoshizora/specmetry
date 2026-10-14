# Codex 审查 016：WP1.1 安全修复独立定向代码复审

- 审查日期：2026-10-14
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #6](https://github.com/XKHoshizora/specmetry/pull/6)
- 审查目标：`66d5948ba3af293642ee1bc63220620ee3766144`
- 增量基准：上一实现提交 `86ac50dc64fa1ccf7f5fe96b783cacba4ded6f5b`
- 上一轮报告：[Codex Review 015](codex-review-015.md)
- 目标 CI：[CI 第 17 次运行](https://github.com/XKHoshizora/specmetry/actions/runs/38526504232)
- 依据：AGENTS.md、architecture §6.1.2、§10、implementation-plan WP1.1、对应初始化验收项、`init.v1` / `error.v1` 契约，以及审查 015 的 B1～B3、N1～N2。
- 范围：Git 子进程隔离、默认交付 Ref 扫描、Git 错误脱敏、Marker 冲突与失败恢复、扫描取消，以及必要的测试有效性检查。单独调查 Linux SQLite 偶发测试失败和 doctor 的遗留 Trace 写入问题；不重新审查 M0 存储架构、WP1.0、WP1.2、WP1.3，不扩展到 WP1.4～WP1.8。

**结论：NO-GO。** B2、B3、N1、N2 已修复；**B1 仍有一个真实可复现的缺口：旧版 Git 的版本探测会先泄露环境变量，再返回拒绝。**

审查目标为 `66d5948ba3af293642ee1bc63220620ee3766144`。审查期间原仓库保持只读，反例仅在隔离的临时仓库和数据目录中执行，未提交、推送或合并。本报告随后按照用户要求单独保存。

## BLOCKER

### B1 未完全关闭：版本探测本身没有安全隔离旧版 Git

**位置：** [gitx.go:242](../../internal/gitx/gitx.go#L242)、[gitx.go:301](../../internal/gitx/gitx.go#L301)。

当前先运行 `git --version`，再判断是否达到 2.32。但这个探测仍继承 HOME；用于隔离全局配置的 `GIT_CONFIG_GLOBAL` 恰好不受旧版 Git 支持。

经产品负责人授权，在 Linux 临时目录构建真实的官方 Git **2.31.1** 后，通过公开初始化调用链复现：

1. 临时 HOME 的 `.gitconfig` 配置 `trace2.eventTarget`，并用 `trace2.envVars` 指定合成环境变量。
2. 使用 Git 2.31.1 执行 `init --json --no-exclude`。
3. Init 返回退出码 **6**，提示需要 Git 2.32。
4. **但拒绝之前，已经创建外部 Trace2 文件，并写入合成环境变量的完整值。**
5. 数据库和 Marker 未创建。

复现产物为仓库外临时目录中的 `old-git/result-True.json` 和 `old-git/external-trace.json`：前者记录退出码 6、Trace 文件存在、数据库与 Marker 未创建；后者记录 `git --version` 的启动事件，以及包含合成环境变量值的 `def_param` 事件。临时验证产物未加入仓库。

根因与旧版 Git 的初始化顺序一致：Trace2 在处理版本命令前初始化，并读取全局配置。[Git 2.31.1 初始化代码](https://github.com/git/git/blob/v2.31.1/common-main.c)、[Trace2 配置读取](https://github.com/git/git/blob/v2.31.1/trace2/tr2_sysenv.c)。

这是本轮新增版本探测路径的安全缺陷，违反环境变量不得持久化、Git 调用不得产生额外文件副作用的边界，不能归入 doctor 的遗留问题。

**最小修复建议：** 在版本探测的受控环境中显式设置：

```text
GIT_TRACE2=0
GIT_TRACE2_EVENT=0
GIT_TRACE2_PERF=0
```

已用真实 Git 2.31.1 单独验证：加入这三个受控值后，同一全局配置不再创建 Trace 文件，结果记录在临时产物 `old-git/trace-disabled-probe.json`。还应补充旧版本拒绝路径的回归测试，检查退出码、外部文件及 canary 未落盘。无需重构 Git 或存储层。

## 原有问题的关闭情况

| 项目 | 结论与证据 |
|---|---|
| B1 Git 子进程隔离 | **未完全关闭。** 支持版本中的已知注入路径已关闭，但旧版本探测仍存在上述泄露。 |
| B2 默认 Ref 扫描 | **关闭。** [分支扫描](../../internal/project/init.go#L169) 位于数据库打开之前。新数据目录下返回 2、无 Token 输出、无数据库或 Marker；另以安全路径中的敏感主分支验证，已有数据库的 `seq` 和 Target 保持不变。 |
| B3 Git 错误脱敏 | **关闭。** [执行错误处理](../../internal/gitx/gitx.go#L342) 丢弃原始 stderr。仓库本地配置含合成 Token 时，JSON 解码后的错误消息和文本错误均不包含 Token 或路径；环境注入被忽略后正常成功也符合要求。 |
| N1 Marker 冲突 | **修复。** [数据库打开前检查](../../internal/cli/init.go#L119) 正确拒绝孤立 Marker；无数据目录及仅有配置文件两种情况均未增加文件。独立强制 Marker 写入失败后，重试仍恢复同一项目。 |
| N2 扫描取消 | **修复。** [扫描检查](../../internal/project/files.go#L27) 覆盖递归遍历和 Git 列表处理。独立触发处理中途的真实 Context 取消，均返回错误及空结果；公开 Init 的取消路径不登记项目。 |

## NON-BLOCKER

### 1. doctor 的 Trace 写入属于独立遗留安全问题

**位置：** [doctor.go:157](../../internal/cli/doctor.go#L157)。

Windows、Linux 均复现：设置临时 `GIT_TRACE` 路径后执行公开 `doctor`，会创建外部 Trace 文件，尽管 Specmetry 数据目录没有变化。

这不符合非破坏性诊断的整体承诺，应单独修复其 Git 版本查询环境。本轮未修改该调用路径，因此**不作为 WP1.1 新增 BLOCKER**，也不应标记为已接受的产品限制。

### 2. 新增安全测试仍有两处可收紧

- [init_security_test.go:146](../../internal/cli/init_security_test.go#L146) 对多个 Trace 副作用的前置验证只要求至少一个生效。审查已逐项独立验证五种 Trace 变量均生效、且被修复后的 Init 隔离；建议原测试也逐项断言。
- [init_security_test.go:358](../../internal/cli/init_security_test.go#L358) 的 `seq` 测试把 Token 放入 Worktree 路径，会先触发路径扫描，不能单独证明 Ref 扫描有效。审查已补做安全路径下的敏感主分支验证，结果正确。

这些测试改进本身不构成新的实现 BLOCKER。

## ACCEPTED RISK

- 大型非 Git 项目仍完整遍历，没有数量上限；取消支持已经补齐。
- TD-25 的迁移能力、SHA-256 快照为 Unknown，以及后续工作包的业务功能，继续沿用既定范围。
- Linux 原先那一次偶发失败仍**未定位**。本次没有证据将其归因于并发、数据完整性或资源竞争，也不能宣称它已经消失。

## 独立验证与边界

- Windows：目标提交原有 Go 全套测试、`go vet` 通过。
- Windows、Linux：独立验证混合大小写注入、FSMonitor、配置注入、逐项 Trace、Hook、对象替换、Ref 扫描、错误脱敏、Marker 恢复和取消路径。混合大小写的实际环境变量攻击在 Windows 验证。
- Linux SQLite：正确包目录下两组各 **40 轮**均未复现失败；其中第二组的 **40 份完整日志已持久保存**，逐轮结果记录在仓库外临时产物 `linux-logs/summary.log`，对应日志为 `linux-logs/sqlite-run-01.log` 至 `linux-logs/sqlite-run-40.log`。
- WSL 重复测试未启用 Race，四项需要 root 的双用户测试跳过。已核实目标提交的[三平台 CI 8/8 成功](https://github.com/XKHoshizora/specmetry/actions/runs/38526504232)，Ubuntu 测试使用 Race Detector；本机没有另行执行 macOS 测试。
- 版本替身验证了旧版、未知版本在仓库命令前被拒绝；真实 Git 2.31.1 则证明，仅有正确退出码仍不足以保证安全拒绝。

## 最终答复

| 最终问题 | 答复 |
|---|---|
| B1、B2、B3 是否关闭？ | **B1 未关闭；B2、B3 关闭。** |
| N1、N2 是否修复？ | **是。** |
| 是否有新的安全或一致性问题？ | 有版本探测导致的环境变量落盘；未发现新的数据库一致性问题。 |
| Git 2.32 最低版本约束是否正确？ | 版本判断和拒绝码正确，**安全拒绝路径不完整**。 |
| Linux 偶发失败是否定位？ | **未定位，本轮未复现。** |
| WP1.1 是否满足代码验收条件？ | **尚未满足，需关闭剩余 B1。** |
| 是否可以合并 PR #6？ | **暂不可以。最终结论：NO-GO。** |
