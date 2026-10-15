# Codex Review 027：WP1.7 首次正式独立代码审查

- 审查日期：2026-10-16（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #10](https://github.com/XKHoshizora/specmetry/pull/10)
- 目标提交：`78ca8826daaee8bba04a68f462e9af8bd8dbef64`
- 基准提交：`c9c9cf84d602e5c91ca8202401b0c4604edff7a5`
- CI：[GitHub Actions 38792687783](https://github.com/XKHoshizora/specmetry/actions/runs/38792687783)
- CI 合并测试提交：`cec344a0eb61355d443bfd930cc435562b536bf0`
- 范围：首次审查 WP1.7 新增的 Question 查询、回答、驳回、Schema、CLI、隐私及并发逻辑；不重新全面审计已验收的 M0、WP1.1～WP1.6。
- 方法：实际源码与契约静态审查、CI 日志核查、离线隔离源码副本测试。终端控制序列只在内存中检查，未向真实终端发送。

**结论：NO-GO。** 发现 **3 项 BLOCKER**。Question 的核心状态转换和并发保护未发现缺陷，但人工确认显示与 CLI 错误契约尚不满足验收要求，暂不建议合并 PR #10。

## BLOCKER

### B1：提案文本中的终端控制字符会进入人工确认摘要

位置：[question.go:133](../../internal/cli/question.go#L133)、[question.go:171](../../internal/cli/question.go#L171)、[question.go:315](../../internal/cli/question.go#L315)。

现有文本 Schema 和密钥扫描允许 ESC、CR、BEL 等控制字符。`indented` 只处理换行，`printQuestions`、`closeSummary` 和后续 `channel.Guard` 会直接输出剩余字符。

**独立反例：**

通过正常 `propose` 路径提交包含 CSI 清屏／光标定位、CR 和 OSC 标题序列的问题与上下文。提案通过 Schema、静态校验及密钥扫描。随后：

- `question list` 的内存输出含原始 `0x1b`、`0x0d`、`0x07`。
- `question answer` 的确认摘要也含这些原始字节。
- JSON 输出正确转义控制字符，解码后保留原值。
- 取消确认后没有业务写入。

测试仅检查内存缓冲区，没有向真实终端发送这些序列。

**影响：** 不可信提案可以改写终端显示、掩盖问题内容或伪装确认上下文，也可能触发终端支持的控制动作。仍需输入正确问题编号，并不能保证人看到的确认摘要可信。本轮没有证明任意代码执行。

**最小修复：**

在终端展示层转义控制字符，覆盖问题、上下文、回答、理由和显示名等动态文本。保留受控换行与缩进；ESC、独立 CR、退格及其他终端控制字符应显示为可见转义形式。数据库和 JSON 数据值可保持原样。

补充公开 CLI 的字节断言，分别验证列表、确认摘要和 JSON 原值保留。

### B2：未知 Question 子命令显示 Help，却返回成功

位置：[question.go:38](../../internal/cli/question.go#L38)。

`question` 命令组没有处理非空未知子命令，Cobra 落入默认 Help 路径。

**独立复现：**

```text
specmetry question unknown
specmetry question answr
```

两者均打印 Help，返回 **退出码 0**。

带 `--json` 时，当前表现为退出码 2、stderr 文本错误，stdout 没有 `error.v1`；并非所有形式都返回 0。

**影响：** 无效操作被报告为成功，违反“0 表示成功、2 表示输入错误”的契约。Agent 或脚本可能据此继续执行。未发现问题被实际关闭或发生业务写入。

**最小修复：**

为 Question 命令组增加明确的未知子命令处理，使用不回显参数的 `invalid_input`／退出码 2。裸 `question` 和显式 `--help` 可继续显示帮助并返回 0。

其他既有命令组的同类行为可以另行处理，不必为此全面重构 CLI。

### B3：错误 flag 位于 `--json` 前时，不输出 `error.v1`

位置：[question.go:45](../../internal/cli/question.go#L45)、[output.go:40](../../internal/cli/output.go#L40)。

JSON 输出模式取决于已经解析的 flag。解析在错误 flag 处提前结束时，后面的 `--json` 没有生效。

**独立复现：**

| 输入 | 实际结果 |
|---|---|
| `question list --all=not-bool --json` | exit 2，stdout 为空，stderr 为文本错误 |
| `question answer Q-001 -m x --unknown --json` | 同上 |
| `question list --json --all=not-bool` | exit 2，正确输出 `error.v1` |

**影响：** 相同 JSON 请求因参数顺序得到不同错误格式，违反 architecture §9.1 的错误输出契约。新增 flag 错误消息本身已脱敏，但结构化错误保证未兑现。

现有测试存在明确覆盖缺口：[question_test.go:504](../../internal/cli/question_test.go#L504) 在 `pointer == ""` 时跳过 JSON 断言，恰好包括未知 flag 和无效布尔 flag。因此测试绿色不能证明这些场景输出了 `error.v1`。

**最小修复：**

使错误路径能够可靠识别 JSON 输出请求，不依赖全部参数成功解析；同时尊重参数值及 `--` 边界。所有 JSON 错误用例都应断言完整 `error.v1`，Pointer 是否存在单独判断，并覆盖 flag 顺序。

## NON-BLOCKER

### N1：其他 CLI 的敏感参数回显是未关闭的遗留安全问题

有限复现确认：

```text
proposal list <合成Token>
proposal list --pending=<合成Token>
```

- 文本模式会在 stderr 回显 Token。
- 第一种带 `--json` 时，Token 进入 **`error.v1.error.message`**。
- 第二种把 `--json` 放在无效 flag 后时，仍通过 stderr 文本回显 Token。

原因包括 [root.go:44](../../internal/cli/root.go#L44) 包装原始 Cobra 错误，以及 [apperr.go:166](../../internal/apperr/apperr.go#L166) 将包含 cause 的 `e.Error()` 放入 JSON。因此“JSON 模式只输出固定消息”的既有说明不成立。

这是实际隐私缺陷，**不是 ACCEPTED RISK**。按本轮边界，作为独立安全修复排期，建议在公开发布前关闭；不要求本 PR 全面重审旧命令。

WP1.6 已有 N1、N2 保持原分类，本轮没有重新展开。

## PRODUCT DECISION

没有发现需要新增产品决策才能确定正确行为的事项。上述修复均可保持现有状态模型、数据原值及阶段边界。

## ACCEPTED RISK

仅沿用已批准边界：

- 通道识别用于防误操作，不证明身份。
- 已拒绝条目关联的问题保持 open，并显示临时 ref。
- 问题关闭后条目才接受时，通过既有变更关系显示正式编号。
- 阻塞问题推导属于 M2，Scope／Target 属于 WP1.8。

未自行接受新的安全风险。

## 已验证的正确行为与测试质量

本轮未发现以下逻辑存在缺陷：

- Question 由 Proposal 创建；关闭命令不新增问题。
- `open → answered/dismissed` 正确，每次成功关闭只消耗一个 seq。
- 已关闭返回退出码 5，不存在返回 `not_found`，拒绝不消耗 seq。
- 操作者、通道、文本和 answered_seq 正确保存；不创建 Baseline Version。
- 确认前检查 open，确认后在写锁内再次检查；闭合触发器继续提供第二道防线。
- 查询按 Q 编号排序、默认仅 open，`--all` 正确显示关闭动作。
- JOIN 包含项目约束，正常写入路径下未发现重复、错误关联或跨项目泄露。
- 独立补测确认：问题先回答、其 AC 条目后接受时，列表正确显示 `AC-001-1`。
- 两个 Question Schema 的状态／动作约束正确；SCH-131～139 的基础样例先验证合法，再施加反例变异，前置条件有效。

并发测试确实覆盖确认期间的变更：

- 同进程测试在读取确认输入时，通过另一连接提交。
- PTY 测试在子进程出现确认提示后、发送确认前，由另一方提交。
- 存储测试使用两个连接同时竞争关闭，断言只有一个成功、另一个过期，且只增加一个 seq。

**独立运行结果：**

| 测试 | 结果 |
|---|---|
| 现有 7 项 Question CLI 测试，含 PTY 和跨进程场景 | 全部通过 |
| 新增 Question 包的 4 项测试 | 全部通过 |
| Schema：36 个合法示例、139 个非法反例及脱敏测试 | 通过 |
| 控制字符字节反例 | 复现 B1 |
| 未知子命令反例 | 复现 B2 |
| 错误 flag 的 JSON 顺序反例 | 复现 B3 |
| 独立 AC 关联测试 | 通过 |

已核对[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38792687783)：8/8 成功，合并测试提交 `cec344a` 对应本轮目标与基准；新增 11 项 Question 测试在 Windows、Ubuntu、macOS 实际通过，Ubuntu 启用 Race Detector。CI 没有覆盖本轮三个反例。

未发现死锁或无限等待证据，不要求修改全局测试超时。

## 最终结论

| 问题 | 结论 |
|---|---|
| Question 状态模型是否正确？ | 核心状态转换、持久化与查询关联正确。 |
| 人类专属操作与并发保护是否合格？ | 通道拦截及并发保护通过；确认显示受 B1 阻塞。 |
| 两个 Schema 是否符合契约？ | 数据约束正确；CLI 错误输出受 B3 阻塞。 |
| 三个新发现是否影响验收？ | A、B 阻塞；C 为需单独修复的遗留安全问题。另有 B3。 |
| 是否有新 BLOCKER／必要产品决策？ | **3 项 BLOCKER；无需新增产品决策。** |
| WP1.7 GO／NO-GO？ | **NO-GO。** |
| PR #10 是否可以合并？ | **暂不可以，修复后复验。** |

原仓库在审查期间保持干净，未修改、提交、推送或合并；未开始 WP1.8。本文件为审查完成后按用户要求保存的完整审查意见。
