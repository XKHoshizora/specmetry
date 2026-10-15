# Codex Review 025：WP1.6 首次独立代码审查

- 审查日期：2026-10-16（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #9](https://github.com/XKHoshizora/specmetry/pull/9)
- 目标提交：`fa58b4420e743521b027f395b921bc178a4d26fb`
- 基准提交：`429e94eca59f6a81a990957d3a3e14a1ef69db53`
- CI：[GitHub Actions 38768971867](https://github.com/XKHoshizora/specmetry/actions/runs/38768971867)
- 审查范围：WP1.6 新增审核、人工编辑、Baseline 应用与历史、提案读取、引用沿用、迁移和 Schema；不重新审计已验收的 M0、WP1.1～WP1.5，也不要求提前实现后续工作包。
- 审查方式：实际差异与必要调用链静态审查、三平台 CI 日志核查、仓库外隔离副本中的独立反例及定向回归测试。

**Codex Review 025：NO-GO。** 发现两个可复现的 BLOCKER。当前不建议验收 WP1.6 或合并 PR #9。

审查目标为 `fa58b4420e743521b027f395b921bc178a4d26fb`，基准为 `429e94eca59f6a81a990957d3a3e14a1ef69db53`。原仓库工作区在审查期间保持干净。

## BLOCKER

### B1：`proposal show` 会通过错误回显敏感参数

位置：[internal/proposal/read.go:360](../../internal/proposal/read.go#L360)、[internal/cli/proposal.go:96](../../internal/cli/proposal.go#L96)。

CLI 只验证参数数量，随后将未经校验的参数传入 `ShowDocument`。找不到提案时，业务错误直接拼入该参数。

**独立复现：**

在隔离项目中，用符合现有 GitHub Token 检测规则的合成字符串作为提案 ID，分别执行：

```text
proposal show <synthetic-token>
proposal show <synthetic-token> --json
```

两者均返回退出码 2，但文本 stderr 和 `error.v1` 的 message 分别包含完整合成 Token。

**影响：** 敏感输入进入 Agent 可读取的输出，绕过了已有 Schema 错误脱敏防线。未发现该反例改变业务数据，但输出泄露已经成立。

**最小修复：**

- 未找到错误使用固定消息，不拼入未经校验的参数。
- CLI 可同时增加 ULID 格式校验，错误同样不得回显原值。
- 增加公开 CLI 的文本与 JSON 回归测试，断言 Token 不出现在任何输出中。

### B2：同一目标变化时，过期审核错误返回退出码 2

位置：[internal/cli/review.go:87](../../internal/cli/review.go#L87)、[internal/cli/review.go:110](../../internal/cli/review.go#L110)、[internal/review/review.go:160](../../internal/review/review.go#L160)。

首次检查和正式提交都先执行 `review.Check`。如果目标已经变化，该函数会因缺少 `acknowledged_conflict` 返回退出码 2，导致后面的 Baseline Version 检查无法执行。

**独立复现了两种情况：**

1. 同一版本下提交两个修改 REQ-001 的提案；接受第一个后，使用旧 `seen_base_version` 审核第二个。
2. 第二个审核完成预演并进入确认阶段后，通过另一个连接接受第一个，再确认第二个。

两种情况都返回 **2**，而 MVP-BL-10 要求 **5**，并返回当前版本。

**影响：** 违反已批准的过期处理契约，客户端无法按既定方式区分“重新读取后决策”和“输入无效”。本轮确认旧审核没有被应用，也没有发生错误覆盖；这是错误分类缺陷，不是已经证明的数据损坏。

**最小修复：**

在首次检查及正式提交的事务内，先核对当前 Baseline Version，再执行条目状态、冲突和 Followup 等业务检查。保留 `baseline.Apply` 内的再次检查。补充上述两个反例，并断言退出码、当前版本和零业务写入。

## NON-BLOCKER

### N1：确认预览可能显示 `editorial`，实际应用为强制 `material`

位置：[internal/review/review.go:182](../../internal/review/review.go#L182)、[internal/cli/apply.go:108](../../internal/cli/apply.go#L108)、[internal/baseline/apply.go:555](../../internal/baseline/apply.go#L555)。

例如修改 AC 的检查命令并选择 `editorial`：预览显示选择值，应用引擎则正确强制为 `material` 并增加 revision。

建议预览显示最终生效类别及强制原因。当前持久化行为符合修订规则，因此不阻塞验收。

### N2：历史读取接口没有展示已保存的 Followup 理由

位置：[internal/baseline/show.go:135](../../internal/baseline/show.go#L135)。

`baseline_changes.rationale` 已正确保存，但 `HistoryChange` 和历史查询没有读取、输出它，文本历史也无法展示。

建议同步扩展历史 Schema、读取模型和测试。理由没有丢失，现有 MVP-BL-11 必需字段能够查询，故列为非阻塞。

## PRODUCT DECISION

没有发现必须新增产品决策才能确定正确行为的事项。

退役 Goal 时清空关联、保留 Requirement revision 并记录 editorial，与当前规定的级联和修订语义一致，本轮不要求重新决策。

## ACCEPTED RISK

仅沿用已有批准边界：

- 通道识别用于防误操作，不能认证操作者身份。
- M1 使用 `hash_only`；来源引用校验结果绑定所观测的来源哈希，代码引用绑定 `checked_head`，不构成交付验收证据。
- 审核附带 scope 当前明确拒绝，随 WP1.8 实现。
- `source_removed`、完整状态推导和拒绝条目对应问题的阻塞性处理，按既定计划在 M2 验证。

未自行接受新的安全风险。

## 已核实的实现与测试

除上述问题外，本轮检查支持以下结论：

- `review` 在读取输入前拦截 Agent／非 TTY；审核输入经过 Schema 和密钥扫描。
- `review` 与 `propose --apply` 共用 Baseline 引擎；预演通过返回专用错误整体回滚。
- 应用顺序、编号、部分审核、父子引用、混合接受／拒绝、级联和 Followup 的主要路径符合规定。
- material／editorial 修订和强制 material 字段正确落库；原提案与编辑后的应用值分别保留。
- 新迁移 0005 的约束及只追加保护通过定向测试；迁移 0004 未修改。
- hash_only 原文剥离和数据库第二道防线未发现本轮回归。

**独立并发验证通过：** 两个独立连接均完成预演后同时确认，结果为 `[0, 5]`；只生成一个版本、一个 Requirement、一个正式编号和一个新增 seq，没有双重应用。

测试证据如下：

| 证据 | 结果 |
|---|---|
| 隔离副本中现有 18 项 WP1.6 CLI 定向测试 | 全部通过，包括真实终端审核测试 |
| Baseline 与新增迁移的 6 项定向测试 | 全部通过 |
| Schema 编译、36 个合法示例、130 个非法反例及错误脱敏测试 | 通过 |
| 独立敏感参数回显测试 | 失败，复现 B1 |
| 独立过期审核测试，两种时序 | 失败，复现 B2 |
| 独立双连接并发提交测试 | 通过 |

初次沙箱测试受文件权限、锁和硬链接限制，未计作产品失败；随后在获准的仓库外隔离环境完成上述验证。

已独立核对[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38768971867)：8/8 成功，测试使用合并提交 `c376bcf`，对应本轮目标与基准。Windows、Ubuntu、macOS 的主要 WP1.6 测试实际运行通过，Ubuntu 启用了 Race Detector。CI 绿色没有覆盖本轮新增的两个反例。

## 最终结论

| 最终问题 | 结论 |
|---|---|
| WP1.6 是否满足全部核心验收条件？ | **尚未满足**，需修复 B1、B2。 |
| 有无新的 BLOCKER？ | **有，2 项。** |
| 是否需要额外产品决策？ | **不需要。** |
| WP1.6 GO／NO-GO？ | **NO-GO。** |
| PR #9 是否可以合并？ | **暂不可以，修复后定向复验即可。** |

审查期间未修改、提交、推送或合并原仓库，也未开始 WP1.7。本文件为审查完成后按用户要求保存的完整审查意见。
