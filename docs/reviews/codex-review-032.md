# Codex Review 032：PR #12 最终定向独立复审

- 审查日期：2026-10-16（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #12](https://github.com/XKHoshizora/specmetry/pull/12)
- 目标提交：`d69baf9dea8ed861fee70842926008eb3fd3d5e9`
- 修复提交：`f345c2b2ee333f6cfa0c973b35a5d3be35bee3a8`
- 修复前基准：`db81f5a20ddc30dac692f80aa9db8f16539afd15`
- CI：[GitHub Actions 38915338231](https://github.com/XKHoshizora/specmetry/actions/runs/38915338231)
- 范围：仅复核 [Review 031](codex-review-031.md) 的唯一 BLOCKER B1，以及此次修复的直接回归；不重新展开此前已通过的终端安全和业务模块全面审查。
- 方法：实际源码和契约静态审查、三平台 CI 日志核查、隔离副本中的修复前后对照测试，以及独立的 Writer、Guard、公开 CLI 故障注入。控制字符仅在内存或隔离日志中检查。

**最终结论：GO。Review 031 的 B1 可以关闭，PR #12 的全局终端输出安全修复可以验收并合并。** 本轮未发现新的实质性 BLOCKER 或直接回归。

## BLOCKER

无。

### B1 关闭：Writer 正确报告短写入，Guard 恢复 fail-closed

关键位置：

- [termtext.go:114](../../internal/termtext/termtext.go#L114)：`Writer.Write`。
- [channel.go:124](../../internal/channel/channel.go#L124)：Guard 检查提示输出错误及完整长度。
- [channel.go:130](../../internal/channel/channel.go#L130)：仅在完整输出成功后读取确认行。
- [termtext_test.go:135](../../internal/termtext/termtext_test.go#L135)：`TestWriterReportsShortWrites`。
- [channel_test.go:218](../../internal/channel/channel_test.go#L218)：`TestGuardFailsWhenThePromptCannotBeShown`。
- [shortwrite_test.go:41](../../internal/cli/shortwrite_test.go#L41)：`TestConfirmationNotShownWholeWritesNothing`。

`Writer.Write` 现在先保存转义结果，再检查底层 `io.WriteString` 的返回值：

1. 底层显式错误原样传播，不能因写入长度看似完整而报告成功。
2. 返回计数为负或超过转义结果长度时，返回固定、安全的错误。
3. 底层无错误但未写完整个转义结果时，返回 `io.ErrShortWrite`。
4. 只有完整输出且无错误时，才返回原始输入长度及 `nil`。
5. 不重试部分写入，避免把首次提示未完整输出的事实隐藏起来。

判断依据是**转义后字节长度**，因此底层恰好接收原始输入长度、但转义结果更长时，也会拒绝。错误路径返回零个已完成的原始输入字节，不尝试把部分转义字节反向映射为原始输入长度。

空输入的完整写入仍可合法返回 `0, nil`；CRLF 转义为 LF 后，按实际输出长度判断成功，再返回原始输入长度。这两种情况没有被误判。底层实现 `io.StringWriter` 时也遵守同样规则。

本轮生产代码修复集中于 `Writer.Write`，未改变 `Escape`、`Line`、`Raw`、`NewWriter` 的既有语义，Guard 和业务写入逻辑也未改动。修复恢复了 Guard 原有检查的有效性：提示输出异常即返回 `environment / exit 6`，不读取确认输入，不进入后续业务写入。

## 独立验证证据

### 修复前后使用相同回归断言

在仓库外分别建立基准提交和目标提交的隔离副本，将目标提交的三组回归测试放入基准副本运行，未修改原仓库。

| 验证 | 修复前基准 `db81f5a` | 目标提交 `d69baf9` |
|---|---|---|
| Writer 短写入、非空输出零写入及异常计数 | 相关断言失败，包装器错误报告成功 | 正确返回错误，测试通过 |
| 同一故障 Writer 直接传入 Guard | 正确拒绝 | 正确拒绝 |
| 同一故障 Writer 经 `termtext.NewWriter` 包装后传入 Guard | 短写入和零写入可能读取并接受确认 | exit 6，输入读取 0 次 |
| 公开 `question answer` 原始反例 | exit 0，读取确认 1 次 | exit 6，输入读取 0 次，无业务写入 |
| 改用完整输出 Writer 后运行相同命令 | — | 正常确认并成功提交 |

基准副本的失败来自目标断言实际发现了原始问题，没有因临时 Git 环境或其他前置条件失败而代替执行被测逻辑。

### 公开 CLI 写事务反例

通过公开 CLI 命令树执行：

```text
question answer Q-001 -m review031-confirmation
```

设置 `human_cli`，预先提供正确的 `Q-001` 确认行，并让底层 stdout 每次只写入一半字节但返回 `nil` 错误。

目标提交的结果为：

- 返回 `environment / exit 6`。
- 确认输入读取次数为 0。
- Q-001 仍为 `open`，`answered_seq` 仍为 `null`。
- 全局 seq 未增加。
- Baseline、Proposal、引用及编号计数器等业务状态与执行前一致。

随后改用完整输出 Writer 执行相同命令，结果为 exit 0，读取确认 1 次，Q-001 正确变为 `answered`，回答原值正确，seq 从 7 增至 8，且其他业务事实摘要保持不变。

### 额外独立测试

独立测试只保存在隔离副本中，均通过：

- **108 组 Writer 场景**：六组独立定义的输入及预期转义结果、九类输出模式、两种底层接口。覆盖 ASCII、ESC、中文、空输入、CRLF 缩短，以及 `io.Writer` 和 `io.StringWriter`。检查显式错误身份、短写入错误、异常计数、原始输入长度返回值，并断言底层仅调用一次。预期转义字节由测试独立定义，没有调用生产 `Escape` 生成预期值。
- **10 组 Guard 场景**：部分写入且无错误、零写入且无错误、部分写入后错误、完整写入但错误、完整成功，分别直接传入和包装传入。输出异常均为 exit 6、输入读取 0 次；完整成功正常接受确认。
- **6 类公开 CLI 输出故障**：半写入、零写入、部分写入后错误、完整写入但错误、负计数、超范围计数。均拒绝且无读取、无业务写入；完整成功路径随后正常提交。

这些验证同时确认了底层错误优先传播、无部分重试，以及“写够原始输入长度但没有写够转义输出长度”的情况不会被放行。

## 直接回归核查

目标隔离副本中的相关现有测试通过，包括：

- `TestEscape`、`TestEscapeLeavesNoControl`、`TestLine`、`TestWriter`、`TestWriterReportsShortWrites`。
- Guard 的安全摘要、正常确认、取消、Agent 拒绝、读取失败、直接及包装后的输出失败，以及终端测试。
- `TestConfirmationNotShownWholeWritesNothing`。
- `TestTextOutputIsShownSafely`、`TestQuestionTextIsShownSafely`：涵盖相关文本转义、JSON 解码后的原值、数据库原值，以及人工确认路径。
- `TestReviewInTerminal`、`TestSourceAckInTerminal`：相关确认、取消和来源变化／过期前置条件。
- `TestCommandLineErrorsQuoteNothing`、`TestShellCompletionIsRefused`：参数错误脱敏及真实二进制子进程的隐藏补全拒绝。

本轮没有发现 Review、Question、Source、`propose --apply` 的相关确认行为，取消、Agent 拒绝、正常 JSON、错误脱敏或既有来源前置条件因本次修复发生实质性回归。没有重新执行或声称重新验收整个 WP1.1～WP1.7。

## CI 与平台证据边界

已读取 [GitHub Actions 38915338231](https://github.com/XKHoshizora/specmetry/actions/runs/38915338231) 的元数据及三平台测试日志：

- Run 的 `head_sha` 对应目标提交 `d69baf9dea8ed861fee70842926008eb3fd3d5e9`。
- 8/8 Job 成功。
- Windows、Ubuntu、macOS 的日志均实际出现 `TestWriterReportsShortWrites`、`TestGuardFailsWhenThePromptCannotBeShown` 及 `TestConfirmationNotShownWholeWritesNothing` 的通过记录，没有将跳过视为通过。
- CI 检出的是以目标提交为 PR Head 的合并测试提交。

本地独立执行平台为 Windows；Ubuntu、macOS 结论来自已核查的 CI 日志，不描述为本地独立实测。CI 成功是补充证据，B1 关闭判断同时依赖源码检查和修复前后的独立反例。

## NON-BLOCKER

继续保留 Review 031 的两项维护记录，本轮不要求修复，也没有认定它们已经关闭：

- **N1：首次登记特殊路径的非文档来源，确认输入可能无法完成。**
- **N2：Human 通道使用 `--json` 时，取消人工确认可能使提示与 `error.v1` 混入 stdout。**

doctor 的 GIT_TRACE、Review 029 N1 的正常 JSON 成功输出断言、WP1.6 N1/N2，以及既有 Windows 本地测试耗时维护事项，继续按原记录处理。本轮未全面重审这些独立事项，也没有把它们描述为已接受风险。

本轮没有新增 NON-BLOCKER。

## ACCEPTED RISK

本轮未新增或自行接受任何风险。

## 最终验收回答

| 问题 | 结论 |
|---|---|
| Review 031 B1 是否可以关闭？ | **可以。** 修复前后的独立反例成立，目标提交恢复正确拒绝。 |
| termtext.Writer 是否正确传播短写入和错误？ | **是。** 检查转义后长度，传播底层错误，拒绝异常计数，不重试部分输出。 |
| Guard 是否恢复“提示未完整显示，不读取输入、不写入”？ | **是。** 直接传入和包装传入的故障组合均验证通过。 |
| 原始公开 CLI 反例是否修复？ | **是。** exit 6、读取 0 次、Q-001 及 seq 不变；完整输出时正常成功。 |
| 是否存在直接引入的实质性回归？ | **未发现。** 相关回归测试和独立边界测试通过。 |
| PR #12 的全局终端输出安全修复能否正式验收？ | **可以。** 本结论不表示既有独立 NON-BLOCKER 或其他里程碑全部完成。 |
| PR #12 是否可以合并？ | **可以。最终结论：GO。** |

审查结束时原仓库工作区干净，没有修改代码、测试、配置或业务数据，没有提交、推送、合并，也没有启动 WP1.8 或其他后续任务。本报告依据用户随后明确授权单独保存；保存动作仅新增本文件。
