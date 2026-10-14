# Codex 审查 014：WP1.2 Agent / Human CLI 通道识别独立增量代码审查

- 审查日期：2026-10-14
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #5](https://github.com/XKHoshizora/specmetry/pull/5)
- 审查目标：`6d6d1abda5f3c5621289bb6fe0ba8c3b31014443`
- 基准：`origin/master = 84e6e708394c56c06c449205ea37feaece5e1fd2`
- 目标 CI：[CI 第 14 次运行](https://github.com/XKHoshizora/specmetry/actions/runs/38506756391)
- 依据：architecture §5.1、MVP-CH-01、MVP-CH-02、implementation-plan WP1.2，以及已确认的提示输出失败和取消确认退出码契约。
- 范围：WP1.2 的通道检测、Guard、CLI 接入、真实 PTY/ConPTY 测试及相关文档增量。不重新审查 M0、WP1.0、WP1.3；不要求提前实现真实人类专属业务命令或 M3 PreToolUse Hook。

**结论：GO。** 未发现实质性 BLOCKER；WP1.2 可以验收并合并 PR #5，但应修正下面的测试循环。

审查期间未修改原仓库，未提交、推送、合并或向 GitHub 发布评论。独立验证在仓库外临时副本中完成。本报告随后按照用户要求单独保存。

## BLOCKER

**无。**

### 通道识别符合已批准设计

**位置：** [channel.go:52](../../internal/channel/channel.go#L52)、[channel.go:135](../../internal/channel/channel.go#L135)、[root.go:28](../../internal/cli/root.go#L28)。

- stdin、stdout 均为终端且两个已知 Agent 变量均未设置或为空时，才识别为 `human_cli`；stderr 不参与判断。
- 已知变量为 `CLAUDECODE`、`CLAUDE_CODE_ENTRYPOINT`，与 architecture §5.1 一致。
- Agent 变量的值只用于非空判断，结果和错误仅保存变量名称，未发现值泄露路径。
- 通道在 CLI 执行时重新检测；`--json`、`--file -` 不参与判断，管道或 stdout 重定向会降级为 `agent_cli`。
- 缺少识别结果时默认 `agent_cli`。独立探针确认，预先注入 Human Context 不能覆盖实际进程的非终端识别结果。
- `Detect(nil, nil)` 同样得到保守的 `agent_cli`，不会因为未知流状态放行。

### Guard 的确认、取消和错误处理

**位置：** [channel.go:100](../../internal/channel/channel.go#L100)。

Guard 先拦截 Agent 通道，再完整输出操作名称、摘要和确认提示；只有输出成功后才读取确认。确认要求以换行结束的完整一行，内容精确匹配，首尾空白按设计忽略。没有公开的跳过确认参数。

独立验证结果：

| 场景 | 结果 |
|---|---|
| Agent 或未识别通道 | 退出码 3，不输出提示、不读取输入 |
| 完整匹配的 LF / CRLF 确认行 | 允许继续，首尾空白按设计忽略 |
| 空行、错误内容、EOF，包括确认文字后没有换行 | 退出码 3 |
| 提示写入错误、零字节或部分短写 | 退出码 6，不读取输入 |
| 返回完整字节数但同时报告写入错误 | 退出码 6，不读取输入 |
| 未取得完整确认行时发生非 EOF 读取错误 | 退出码 6 |

这些检查使用公开 `channel.Guard` 调用路径，写入失败探针同时检查实际写入次数、完整提示内容、退出码和输入读取次数。Agent 与缺少识别结果的探针检查输入和输出均未被触碰；错误字符串及 JSON 错误响应中没有 Agent 标记值。

没有发现公开的跳过确认参数，也没有复现错误放行或不安全写入风险。

## NON-BLOCKER

### N1：提示失败测试包含六次没有触发故障的运行

**位置：** [channel_test.go:190](../../internal/channel/channel_test.go#L190)。

`TestGuardFailsWhenThePromptCannotBeShown` 仍然循环 `failAt=1..4`，每次分别测试写入错误和短写。当前 Guard 把操作名称、摘要和确认提示合并为一次写入，因此后面的失败次数根本不会到达。

独立运行确认：

| 设置 | 故障是否触发 | 读取次数 | 退出码 |
|---|---|---|---|
| `failAt=1`，写入错误 | 是 | 0 | 6 |
| `failAt=1`，短写 | 是 | 0 | 6 |
| `failAt=2..4`，各两种模式 | 否 | 1 | 0 |

后六次运行通过 `if !w.failed { continue }` 跳过断言。因此，**不能将它们报告为已验证的失败分支**。正常读取确认并返回 0，并不是在输出实际失败后错误放行，而是模拟 Writer 没有触发故障。

**影响：** 测试循环和注释容易使覆盖情况被高估，未触发故障时静默跳过断言会削弱测试的可审查性。

这不构成当前实现的实质性覆盖缺口：真实存在的单次写入错误和短写分支已经被前两次测试覆盖，本轮独立探针也验证了更多返回组合。

**最小修正：** 删除 `failAt=1..4` 循环，改成“写入错误”“短写”两个明确子测试，强制断言故障已触发、退出码为 6、输入读取次数为 0。若要测试不同截断位置，应调整单次写入返回的字节数，而不是假设存在第二、第三或第四次写入调用。

## ACCEPTED RISK

- 通道识别是防误操作机制，不能证明操作者身份；清除标记并使用伪终端的 Agent 仍可能被识别为 Human，符合已批准边界。
- 其他 Agent 的环境变量在核实后才加入；当前不能把两项已知变量的覆盖范围说成所有 Agent 都已识别。
- 真实人类专属业务命令的集成验收仍随 WP1.4、WP1.6、WP1.8 完成，不作为本轮阻塞项。
- Windows 上不提供控制台的终端按既定设计降级为 `agent_cli`，不能把这一保守结果视为身份验证失败或要求提前重构终端支持。

## 测试有效性与独立执行证据

### PTY / ConPTY 覆盖了 CLI 共同执行路径

**位置：** [channel_test.go:31](../../internal/cli/channel_test.go#L31)、[channel_test.go:49](../../internal/cli/channel_test.go#L49)。

CLI 替身命令经过真实 Cobra 执行路径、通道 Context、Guard、退出码映射，并且仅在 Guard 成功后写入标记文件。它覆盖确认、取消、Agent 标记、重定向、管道及参数场景，超出了单独测试辅助函数的范围。

组件层的真实终端测试验证 `Detect` 与 `Guard`；CLI 层通过共用的 `executeRoot` 和只在测试中存在的替身命令，验证拦截和确认失败时标记文件没有被写入。测试还覆盖了只读命令 `version`、`doctor` 在不同通道中正常执行，以及不接受 `--yes`、`-y`、`--force`、`--no-confirm` 等跳过参数。

这些测试不代表真实 `source ack`、`review` 或 `scope confirm` 已实现或已完成集成验收；相应验收仍属于后续工作包。

### 本地验证

本地使用 Windows、Go 1.27.1。原始目标提交的测试和另外加入临时副本的独立探针分别执行，没有修改原仓库。

| 验证 | 结果 |
|---|---|
| `go test -tags nowebui -count=1 -v ./...` | 通过；WP1.2 相关组件和真实 ConPTY CLI 测试实际执行 |
| `go vet -tags nowebui ./...` | 通过 |
| 独立提示输出错误、零字节和部分短写、完整字节数附带错误 | 均返回 6，输入读取次数为 0 |
| 独立完整确认行、无换行确认文字、EOF、错误内容和非 EOF 读取错误 | 结果符合本轮规定的退出码和确认规则 |
| 独立 Agent、空 Detection 和无 Context 检测结果 | 均返回 3，输入输出均未触碰 |
| 独立 canary 错误输出检查 | 错误字符串和 JSON 响应没有 Agent 标记值 |
| 预先注入 Human Context，在非终端进程执行替身命令 | 实际通道检测覆盖了预置值；退出码 3，标记文件未写入 |
| 真实 ConPTY 中同时使用 `--json --file -` 并取消确认 | 仍显示确认提示；退出码 3，标记文件未写入 |
| 对原失败循环逐次记录写入、故障、读取和退出码 | 只验证到两次真实失败；后六次故障没有触发，详见 N1 |

两项既有符号链接测试因系统权限不足跳过，未记为通过。没有据此重新宣称整个 M0 已完成独立平台验收，也没有扩大本轮审查范围。

### GitHub CI

已核实 [目标 CI 第 14 次运行](https://github.com/XKHoshizora/specmetry/actions/runs/38506756391) 对应审查目标，8/8 任务成功，并读取 Windows、Linux、macOS 三个平台的 Go 测试日志，确认通道检测、Guard、PTY / ConPTY、CLI 替身命令、只读命令和确认跳过参数测试均实际执行并通过。

CI 的 `TestGuardFailsWhenThePromptCannotBeShown` 成功不能被解释为八次失败注入都有效；本报告只认可实际触发的两次故障，另外六次没有验证到失败分支。

本轮未在本地独立运行 Linux 或 macOS；其终端测试结论来自目标提交的实际 CI 日志。

## 最终回答

| 问题 | 回答 |
|---|---|
| 通道识别是否符合已批准设计？ | **是。** |
| Guard 的确认、取消和错误处理是否正确？ | **是，已验证本轮规定场景。** |
| 是否存在实际可复现的错误放行或不安全写入风险？ | **未发现。** |
| 测试是否存在假阳性或覆盖缺口？ | **有 N1：六次未触发故障；不阻塞当前实现。** |
| WP1.2 是否可以验收并合并 PR #5？ | **可以，GO。** |

该结论仅覆盖 WP1.2，不代表全部 M1 验收完成。

原仓库在审查结束时保持干净，HEAD 未变。本报告保存操作仅新增 `docs/reviews/codex-review-014.md`；未修改其他文件，未提交、推送或合并。
