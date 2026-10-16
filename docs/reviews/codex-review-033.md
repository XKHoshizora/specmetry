# Codex Review 033：PR #13 首次正式独立代码审查

- 审查日期：2026-10-17（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #13](https://github.com/XKHoshizora/specmetry/pull/13)
- 目标提交：`511403dbb5f68c7e30bcb171918efe901f0f4b9f`
- 基准提交：`26c787ba386dab40090147ac773c91e48f300a46`
- CI：[GitHub Actions 38925900884](https://github.com/XKHoshizora/specmetry/actions/runs/38925900884)
- 范围：[Review 031](codex-review-031.md) N2 的修复，即 human_cli 使用 `--json` 时，人工确认提示与机器可读 stdout 的分流，以及该修复的直接回归。
- 方法：实际源码及契约静态审查、三平台 CI 日志核查、修复前后隔离副本对照、公开 CLI 命令入口测试、真实 ConPTY 测试，以及 stderr 故障 Writer 注入。
- 范围限制：不重新全面审查已验收的 WP1.0～WP1.7、PR #11 或 PR #12；不启动 WP1.8。

**最终结论：GO。Review 031 N2 已解决，PR #13 可以合并。** 未发现阻塞当前正式 CLI 使用路径的缺陷。

## BLOCKER

无。

### 统一确认入口覆盖完整

七个人工确认操作全部调用 `cli.confirm`：

1. `question answer`。
2. `question dismiss`。
3. `review`。
4. `propose --apply`。
5. `source ack`。
6. `source remove`。
7. `source add --allow-any-type`。

对应生产调用点：

- [question.go:207](../../internal/cli/question.go#L207)：answer 和 dismiss 共用关闭命令实现。
- [review.go:128](../../internal/cli/review.go#L128)。
- [propose.go:153](../../internal/cli/propose.go#L153)。
- [source.go:268](../../internal/cli/source.go#L268)：允许非文档类型的首次登记。
- [source.go:485](../../internal/cli/source.go#L485)：ack。
- [source.go:664](../../internal/cli/source.go#L664)：remove。

生产命令中未发现遗漏的直接 `channel.Guard` 调用。测试中用于早期通道验收的独立替身命令不属于生产命令路径。

### 分流和错误输出符合契约

核心实现位于 [confirm.go:24](../../internal/cli/confirm.go#L24)：

- 不带 `--json` 或指定 `--json=false` 时，提示仍在 stdout。
- JSON 模式先拒绝 Agent，返回 `human_required / exit 3`，不显示提示、不读取确认。
- human_cli 的 stderr 不是终端时，返回 `environment / exit 6`，不向 stderr 输出确认、不读取确认、不写入业务事实。
- 可以交互时，提示通过 `cmd.ErrOrStderr()` 输出；取消、错误输入、EOF 和读取失败的最终错误仍由统一错误处理在 stdout 输出。
- 确认函数和 `executeAs` 的业务错误输出均使用 `wantsJSON(cmd)`，未发现两者选择不一致而重新混流的路径。

取消、错误输入和 EOF 返回 exit 3；非 EOF 读取失败返回 exit 6。相关 stdout 是单个完整、Schema 合法的 `error.v1`，没有通过截去提示或只检查首字符来掩盖混流。

已有成功输出契约保持有效：`propose --apply --json` 输出 `proposal-receipt.v1`，`source add --json` 输出 `source-add.v1`。原本成功输出文本的命令继续输出文本，不能把本次分流理解为这些命令新增了成功 JSON 响应。

### 人工确认完整性保持有效

[channel.Guard](../../internal/channel/channel.go#L124) 的完整输出检查未弱化；stderr 同样经过终端文本 Writer。输出短写入、零写入或显式错误时，仍返回 exit 6，不读取确认，不进入业务写入。

确认内容继续按原值比较；`--json` 不改变通道分类，也不跳过 RequireHuman。确认后的事务、哈希和版本前置条件检查仍然执行。

## NON-BLOCKER

### N1：自定义 Writer 与进程 stderr 的终端属性可能不一致

位置：

- [root.go:21](../../internal/cli/root.go#L21)：`Execute` 接受调用者提供的 stdout/stderr Writer。
- [root.go:31](../../internal/cli/root.go#L31)：终端检测针对进程的 `os.Stderr`。
- [confirm.go:34](../../internal/cli/confirm.go#L34)：实际提示使用 `cmd.ErrOrStderr()`。

**独立反例：** 在真实 ConPTY 子进程中，进程的 stdin、stdout、stderr 都保持为终端，通过公开 Execute 调用：

```go
var hidden bytes.Buffer
cli.Execute(ctx, args, os.Stdout, &hidden)
```

执行 JSON 模式的 `question answer Q-001 -m review033`。提示进入 `hidden`，终端中没有确认摘要；预先输入正确的 Q-001 确认行后，操作仍返回 exit 0，问题变为 answered。此验证没有注入 Detection，没有替换 Guard，也没有直接调用 `executeAs` 来伪造 human_cli。

**影响及本轮分类依据：** 自定义 Writer 并不因进程 stderr 为终端，就必然可被人看到。这是当前接口组合的真实行为限制。不过，当前正式程序唯一的生产调用位于 [main.go:14](../../cmd/specmetry/main.go#L14)，直接传入 `os.Stdout/os.Stderr`，检测对象与实际输出目标一致；没有命令行参数能制造上述 Writer 替换。反例需要可信的进程内调用者主动更换输出目标，不能据此声称正式二进制存在可利用的确认绕过，因此本轮不列为 BLOCKER。

**最小建议：** 明确 Execute 的交互流使用约束。未来若支持嵌入或重定向包装，应将终端能力绑定实际确认输出目标；不能把进程 stderr 为终端推广为任意传入 Writer 都可见。此项作为接口维护记录，不自行认定为已接受风险。

## 测试有效性与独立验证

### 修复前后对照

在仓库外分别建立基准和目标提交的隔离副本，运行同一原始混流反例：

```text
question answer Q-001 -m review033 --json
```

使用 human_cli 并取消确认：

| 提交 | 实际结果 |
|---|---|
| 基准 `26c787b` | exit 3；stdout 是“确认提示＋错误 JSON”，stderr 为空；对整个 stdout 解码失败。 |
| 目标 `511403d` | exit 3；提示位于 stderr；整个 stdout 可以解码为单个文档，随后读取为 EOF，并通过 `error.v1` 校验。 |

两边取消后都检查了业务状态不变。基准失败来自实际混流，而不是前置条件失败或被测逻辑未执行。

### JSON flag 独立矩阵

16 组场景通过，覆盖：

- 无 flag、`--json=false`、`--json=true`。
- 根级、命令组级、叶子命令级的 `--json`。
- 跨级别及同一级别重复 flag，最终值覆盖。
- `-m`、`--message` 的值恰好是 `--json`，包括长参数赋值及简写连接形式。
- 字符串值为 `--json` 后，再提供一个真正的 JSON flag。

JSON 场景对整个 stdout 解码、验证 EOF，并校验 `error.v1`；文本场景检查提示只位于 stdout。

另检查不存在的问题，以及 `--file` 的字符串值为 `--json` 并同时指定真正 JSON flag 的场景。前置条件失败返回 exit 2，stdout 是完整错误文档，stderr 为空，没有显示确认。所有拒绝场景结束后，业务状态与 seq 保持不变。

### stderr 故障及短写入

独立注入以下四种输出行为：

1. 只写入一半字节但返回 nil。
2. 零字节写入但返回 nil。
3. 部分写入后返回错误。
4. 完整写入但同时返回错误。

均得到 exit 6、确认输入读取 0 次、底层提示写入只尝试一次、stdout 为完整 `error.v1`，业务状态不变。Review 032 已关闭的短写入安全问题没有回归。

### 真实终端验证

既有 `TestConfirmationWithJSONInTerminal` 和额外独立 ConPTY 验证均通过，覆盖：

- 正确确认后成功。
- 取消确认后返回 exit 3。
- Agent 环境变量存在时返回 exit 3，无提示、无写入。
- stderr 实际重定向到文件时，JSON 模式返回 exit 6；文件保持零字节，无提示、无写入。
- stderr 重定向但不带 JSON 时，仍可在 stdout 显示提示并正常确认。

额外独立测试保留进程真实终端检测，分别捕获输出；取消和拒绝场景对完整 stdout 做 JSON 解码、EOF 检查及 Schema 校验，不依赖合并后的终端文本里出现某个 JSON 子串来判定成功。

### 新增源码扫描测试的能力边界

[TestHumanOnlyCommandsConfirmThroughConfirm](../../internal/cli/confirm_test.go#L26) 会检查当前 CLI 目录的非测试 Go 文件，拒绝出现当前拼写形式的 `channel.Guard(`，能防止常规新增命令直接重复旧写法。

它仍是字符串扫描：不能证明未来命令必定执行确认，也不能覆盖导入别名、函数别名等所有语义等价写法。因此它是有效的辅助检查，不能替代 WP1.8 自身的人类专属操作和输出契约验收。当前七个入口已通过实际调用链及业务测试核实，没有以该扫描测试单独代替完整性判断。

[wholeJSON](../../internal/cli/review_helpers_test.go#L309) 已由跳过提示前缀改为解码整个 stdout，并检查之后为 EOF，能够发现前置提示、追加文本或第二份文档。

## 直接回归结果

隔离目标副本中以下相关测试实际通过：

- `TestHumanOnlyCommandsConfirmThroughConfirm`。
- `TestConfirmationWithJSON`：逐一覆盖七个人工操作的取消、错误输入、EOF、读取失败、非终端 stderr、短写入、文本模式、Agent 拒绝及正确确认。
- `TestConfirmationWithJSONInTerminal`。
- `TestConfirmationNotShownWholeWritesNothing`。
- `TestWriterReportsShortWrites`、`TestGuardFailsWhenThePromptCannotBeShown`。
- `TestQuestionInTerminal`、`TestReviewInTerminal`、`TestSourceAckInTerminal`。
- `TestSourceRemoveInTerminal`、`TestSourceAddAnyTypeInTerminal`、`TestProposeApply`。
- `TestQuestionClosedWhileConfirming`、`TestReviewIsStaleAfterTheConfirmation`。

取消和拒绝不写入；成功后的 Question 状态、Baseline 版本、Source 确认哈希及移除状态符合预期。确认期间另一方先提交的过期场景仍正确拒绝，没有发现覆盖或额外业务写入。此次生产改动没有改变事务应用引擎、Schema 业务字段或既有命令行脱敏及补全拦截实现。

## CI 与证据边界

已核查 [CI 38925900884](https://github.com/XKHoshizora/specmetry/actions/runs/38925900884) 元数据及三平台测试日志：

- `head_sha` 为目标提交 `511403dbb5f68c7e30bcb171918efe901f0f4b9f`。
- 8/8 Job 成功。
- CI 检出以目标提交为 Head、基准提交为 Base 的 PR 合并测试提交。
- Windows、Ubuntu、macOS 日志均实际出现三个新增确认测试及原短写入 CLI 回归测试的通过记录，没有把跳过当作通过。
- Ubuntu 的 Go 测试按工作流启用 `-race`。

本地独立执行平台为 Windows；Ubuntu、macOS 及 Linux Race Detector 结论来自已核查的 CI，不描述为本地独立执行。CI 绿色没有代替源码审查或修复前后反例。

核查时 PR #13 仍为 Open，Head 与审查目标一致，尚未合并。

## PRODUCT DECISION

无新增必要产品决策。

## ACCEPTED RISK

本轮未新增或自行接受风险。

以下既有事项继续独立保留，不要求本 PR 一并修复，也不认定为已接受风险：Review 031 N1 的特殊路径首次登记确认、Review 029 N1 的部分成功 JSON 测试断言、doctor 的 GIT_TRACE 隔离、WP1.6 N1/N2，以及 Windows 本地测试耗时维护。

## 最终回答

| 问题 | 结论 |
|---|---|
| Review 031 N2 是否已解决？ | **是。** 正式 CLI 路径的确认提示与 JSON 错误 stdout 已分流。 |
| 七个人工确认入口是否全部覆盖？ | **是。** 两个 Question 操作共用一个调用点，其余逐一核实。 |
| JSON stdout 是否与交互提示完全分流？ | **是，符合既有成功输出契约。** 成功原本为文本的命令仍输出文本。 |
| stderr 终端检测及输出目标是否安全？ | 正式二进制路径正确；自定义 Writer 的接口限制见 NON-BLOCKER N1。 |
| 是否存在正常命令、人工确认和数据写入回归？ | **未发现。** 相关成功、取消、拒绝、短写入和确认期间过期测试通过。 |
| PR #13 是否可以合并？ | **可以。最终结论：GO。** |

审查结束时原仓库工作区干净，未修改代码、测试、配置或业务数据，未提交、推送或合并，未启动 WP1.8。本报告根据用户随后明确授权单独保存，保存动作仅新增本文件。
