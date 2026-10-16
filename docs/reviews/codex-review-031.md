# Codex Review 031：PR #12 首次正式独立安全代码审查

- 审查日期：2026-10-16（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #12](https://github.com/XKHoshizora/specmetry/pull/12)
- 目标提交：`db81f5a20ddc30dac692f80aa9db8f16539afd15`
- 基准提交：`a91016ef788eeed3122170b5c2176da3b43c2227`
- CI：[GitHub Actions 38873707154](https://github.com/XKHoshizora/specmetry/actions/runs/38873707154)
- 范围：全局终端文本转义、Writer 短写入、JSON 旁路、人工确认完整性、Source 确认变化及直接回归；不重新全面审查已验收的 WP1.1～WP1.7 或 PR #11 的 Cobra 安全修复。
- 方法：实际源码与契约静态审查、三平台 CI 日志核查、隔离副本中的基准与目标提交对照测试，以及 Writer 故障注入。危险控制字符仅在内存或隔离日志中检查，未向真实终端发送。

**结论：NO-GO。** 发现 **1 项 BLOCKER**：新增 Writer 包装会掩盖短写入，使未完整显示的确认提示被当作成功，实际允许业务写入。

## BLOCKER

### B1：Writer 吞掉短写入，破坏 Guard 的 fail-closed 保证

位置：

- [termtext.go:106](../../internal/termtext/termtext.go#L106)：忽略底层写入字节数，直接返回 `len(p), nil`。
- [root.go:92](../../internal/cli/root.go#L92)：将该 Writer 接入全部命令 stdout。
- [channel.go:124](../../internal/channel/channel.go#L124)：完整输出检查因此得到错误的成功结果。

违反 [architecture.md:149](../architecture.md#L149)：提示只输出一部分时，必须返回 exit 6，不读取输入、不写入数据。

**独立反例：**

底层 Writer 每次只接收一半字节，并返回 `nil` 错误。对同一个提示和确认输入：

| 路径 | 退出码 | 输入读取次数 | 结果 |
|---|---:|---:|---|
| 直接传入 Guard | 6 | 0 | 正确拒绝 |
| 经 `termtext.NewWriter` 包装后传入 | 0 | 1 | 错误确认成功 |

独立 Writer 测试还观察到：输入 5 字节、转义后 8 字节、底层仅写入 4 字节，包装器却返回 `5, nil`。

进一步通过隔离环境中的 Question CLI 命令入口验证：

```text
question answer Q-001 -m review031-confirmation
```

使用同样的短写入 stdout 和等待中的正确确认行，实际结果为：

```text
exit = 0
confirmation reads = 1
before: seq=7, Q-001=open
after:  seq=8, Q-001=answered
```

修复前基准使用同一反例时返回 **exit 6、读取次数 0、数据不变**，证明这是本 PR 引入的直接回归。

**最小修复：**

保存转义结果，检查底层返回的 `n`：

- `err == nil && n != len(escaped)` 时，返回 `io.ErrShortWrite`。
- 仅在完整写入且无错误时返回 `len(p), nil`。
- 保留 Guard 现有的完整输出检查。

补充 Writer 短写入、包装后的 Guard，以及公开 CLI 不写入数据的组合回归测试。

现有绿色测试没有覆盖这个组合：Guard 的短写入测试直接使用底层 Writer；`TestWriter` 的失败用例已经返回非空错误，无法发现“短写入但 err=nil”。

## NON-BLOCKER

### N1：首次登记特殊路径的非文档来源，确认体验仍有缺口

位置：[source.go:263](../../internal/cli/source.go#L263)。

尚未登记时没有 Source ID，`confirmText` 返回空值后又回退到原始路径。

独立验证了包含双向控制字符的非文档文件：

- 输入屏幕上的可见转义文本：exit 3，不写入。
- 输入实际原始路径：可以成功登记。
- 输出本身没有危险控制字符。

这没有造成错误放行。对于 POSIX 允许的含换行文件名，单行读取逻辑无法完成这种确认；本轮未在真实 POSIX 文件系统动态复验该情况。

建议对无法单行确认的路径提供明确的处理提示，或另行采用可输入的确认标识，不作为本轮 BLOCKER。

### N2：人工交互取消后的 JSON 混流是既有问题

位置：[question.go:207](../../internal/cli/question.go#L207)、[root.go:74](../../internal/cli/root.go#L74)。

Human 通道执行：

```text
question answer Q-001 -m answer --json
```

取消确认后，stdout 包含确认提示和随后输出的 `error.v1`，整个 stdout 不是完整 JSON。

目标与基准提交均独立复现，取消时均未写入，因此不是本 PR 引入的回归。建议独立处理 JSON 模式下提示与机器输出的分流，不将其描述为已接受风险。

## PRODUCT DECISION

没有必须新增产品决策才能修复本轮 BLOCKER 的事项。

## ACCEPTED RISK

没有新增安全风险接受。其他已登记事项保持原安排，不因本轮审查视为关闭。

## 其余验证结果

- **正常完整写入时，统一转义边界有效。** 未发现普通 CLI 文本结果、错误、帮助和确认输出新增可达的直接 stdout/stderr 绕过路径。
- ESC、CSI、OSC、CR、BEL、BS、DEL、C1、双向控制字符和非法 UTF-8 均得到可见转义。
- Guard 的多行摘要续行保持缩进；独立验证了确认原值含控制字符时，输入可见转义文本不能代替原值。
- 底层显式写入错误能够传播；问题集中在短写入且错误为空。
- UTF-8、CRLF 跨 Write 时会分别显示为片段转义，存在可读性差异，但没有留下危险原始控制字节。
- 重复包装与 `Raw` 解包检查通过；JSON 旁路仅位于明确的 `writeJSON` 路径。正常 JSON、错误信封及数据库原值未发现被终端转义改写。
- 特殊已登记 Source 的 ack/remove 使用正确 ID：错误路径、错误 ID、可见转义文本均被拒绝，正确 ID 可以完成操作。
- Source 的真实 PTY、Agent 拒绝、重定向拒绝、确认期间文件变化及过期哈希检查通过。

## 测试证据

本地独立运行通过了新增文本显示测试、Line/Writer 既有测试、Guard 既有测试、Question/Review 显示与确认测试，以及六项 Source 相关回归。

独立故障注入则在 **Writer、包装后的 Guard、公开 CLI 写事务**三个层面复现 B1。

在基准副本中，新文本显示测试能检出原始缺陷；同一短写入 CLI 反例被正确拒绝，支持“转义修复有效，但引入确认完整性回归”的判断。

已核对[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38873707154)：对应目标提交，8/8 成功，三平台实际运行了新增测试。**CI 未覆盖 B1 的包装组合反例。** 本地验证在 Windows 完成，其他平台依据 CI 日志。

## 最终结论

| 问题 | 结论 |
|---|---|
| 全局终端控制字符是否修复？ | 正常完整输出路径有效，但整体安全修复尚不能验收。 |
| Writer 和 JSON 旁路是否安全？ | **Writer 存在 B1；JSON 编码旁路未发现直接回归。** |
| 人工确认完整性是否保持？ | **没有，短写入可进入写事务。** |
| Source 确认行为是否正确？ | 已登记来源的 ID 确认正确；首次特殊非文档来源存在非阻塞体验问题。 |
| 是否存在直接回归？ | **存在，B1 已独立复现。** |
| 该 M6 安全遗留问题能否关闭？ | **暂不能。** |
| PR #12 是否可以合并？ | **暂不能，修复 B1 后复验。** |

**最终结论：NO-GO。**

原仓库在审查期间保持干净，未修改、提交、推送或合并；未启动 WP1.8。本文件为审查完成后按用户要求保存的完整审查意见。
