# Codex Review 029：独立安全修复 PR #11 首次正式代码审查

- 审查日期：2026-10-16（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #11](https://github.com/XKHoshizora/specmetry/pull/11)
- 目标提交：`9879c56e9c506088122bc1a248008cc714a1dfb9`
- 基准提交：`867bd60b7c0c40e408a74b2752d2d12dcb192f0b`
- CI：[GitHub Actions 38845384417](https://github.com/XKHoshizora/specmetry/actions/runs/38845384417)
- 范围：全局 CLI 命令行参数错误脱敏、通用命令解析和错误处理逻辑，以及直接回归；不重新全面审查已验收的 WP1.1～WP1.7 业务实现。
- 方法：实际源码与契约静态审查、三平台 CI 日志核查、隔离副本中的旧版与新版对照测试，以及真实 CLI 子进程输出检查。Canary 在运行时生成，未使用真实凭据。

**结论：NO-GO。** 常规 CLI 路径的参数脱敏有效，但仍存在 **1 项 BLOCKER**：Cobra 动态加入的隐藏补全命令可以泄露敏感参数，Review 027 N1 尚未完全关闭。

原仓库在审查期间保持干净，未修改、提交、推送或合并。

## BLOCKER

### B1：隐藏补全命令绕过统一脱敏，将参数写入真实 stderr，且可能落盘

位置：

- [cmdline.go:28](../../internal/cli/cmdline.go#L28)：`secureCommandLine` 只遍历当时已经注册的命令。
- [root.go:75](../../internal/cli/root.go#L75)：`DisableDefaultCmd` 只关闭默认的可见补全命令。
- [root.go:36](../../internal/cli/root.go#L36)：后续 `ExecuteContextC` 仍会触发 Cobra 的隐藏命令初始化。

Cobra v1.10.2 在执行期间动态加入 `__complete`／`__completeNoDesc`。它们没有经过前面的安全遍历；内部直接调用 `ParseFlags`，再通过 `CompErrorln` 输出包含原始参数的错误。这个输出直接写入 `os.Stderr`，不会经过 `executeAs` 的错误脱敏。

**独立复现：**

使用目标提交构建的真实 CLI 二进制、运行时生成的合成 Secret Canary，在隔离目录执行：

```text
specmetry __complete proposal list --pending=<CANARY> ""
specmetry __completeNoDesc review --<CANARY> ""
```

实际结果：

- 退出码为 **0**。
- stdout 为 `:0`。
- 真实 stderr 包含完整 Canary，例如：

```text
[Debug] [Error] Error while parsing flags from args [--pending=<CANARY>]:
invalid argument "<CANARY>" for "--pending" flag ...
```

增加根级 `--json` 后仍能复现。

此外，将 `BASH_COMP_DEBUG_FILE` 指向隔离目录中的文件后，Cobra 创建了该文件，并将包含 Canary 的错误写入其中。没有创建 Specmetry 业务数据目录。

**影响：** 未经过密钥扫描的命令行参数仍可进入 stderr 和补全调试文件，直接违反本 PR 的全局参数错误脱敏目标。该路径还将解析失败报告为成功。

**最小修复：**

当前已禁用默认补全功能，可以在隐藏补全处理器运行前，明确、安全地拒绝这两个入口。若保留补全，则必须接管其内部错误输出和调试写入，不能仅依赖返回错误后的统一处理。

补充真实子进程回归测试，捕获进程级 stdout/stderr，并检查隔离的补全调试文件。仅检查传给 `Execute` 的缓冲区不足以覆盖直接写入 `os.Stderr` 的路径。

## NON-BLOCKER

### N1：正常 JSON 输出测试的断言偏弱

[cmdline_test.go:166](../../internal/cli/cmdline_test.go#L166) 只检查成功输出以 `{` 开头，不能证明符合对应业务 Schema。

本轮独立补测已确认根级 JSON 下的 Source、Proposal、Question、Baseline 读取输出符合对应 Schema，因此没有发现当前功能缺陷。建议将现有测试改为校验具体 Schema。

其他终端转义、doctor Git Trace、WP1.6 N1/N2 等事项继续保持独立任务，本轮未将其视为已修复或已接受风险。

## PRODUCT DECISION

无。修复上述遗漏不需要新增产品决策。

## ACCEPTED RISK

没有新增风险接受。B1 属于实际隐私缺陷，不能作为补全功能的默认行为予以豁免。

## 已验证的正确行为

- 当前静态注册的根命令、命令组和叶子命令均受到统一控制。
- `quotingNothing` 保留现有参数个数判断，仅替换不安全错误消息。
- 当前命令组没有被 `secureGroup` 覆盖的有意义业务校验或执行逻辑。
- 普通未知命令和未知帮助主题返回退出码 2；建议只使用已定义命令名。
- 常规 JSON 错误模式能区分真正的 flag、字符串取值、重复布尔值、简写和 `--` 边界。
- 业务 `apperr.Error` 的退出码、Details 和 Current 保持有效。
- 未发现普通命令、人工通道或确认流程的直接回归。

## 测试证据

- 指定的四项测试均独立运行通过：
  - `TestCommandLineErrorsQuoteNothing`
  - `TestCommandLineStillWorks`
  - `TestQuestionCommandLineErrors`
  - `TestProposalShowDoesNotEchoItsArgument`
- 补充的 **51 组参数组合**通过。
- 根级 JSON 的 **6 类成功读取输出**通过对应业务 Schema 校验。
- 人工通道、确认显示、真实 PTY 及过期错误相关回归通过。
- 在隔离基准提交上运行同组 Canary 断言，实际复现原有泄露，确认新测试能检出旧问题。
- 已核对[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38845384417)：对应目标提交，8/8 成功；三平台日志显示上述四项测试实际通过。**CI 没有覆盖 B1。**

本地独立测试在 Windows 完成；Ubuntu、macOS 的验证依据为 CI 日志。

## 最终结论

| 问题 | 结论 |
|---|---|
| Review 027 N1 是否关闭？ | **未完全关闭，隐藏补全入口仍泄露参数。** |
| 是否存在遗漏的 Cobra 错误路径？ | **存在，见 B1。** |
| JSON 与正常命令语义是否正确？ | 常规路径通过；隐藏补全绕过统一错误处理。 |
| 是否发现其他直接业务回归？ | 未发现。 |
| PR #11 是否可以合并？ | **暂不可以，修复 B1 后复验。** |

**最终结论：NO-GO。**

未启动下一项安全任务，未开始 WP1.8。本文件为审查完成后按用户要求保存的完整审查意见。
