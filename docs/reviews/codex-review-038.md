# Codex Review 038：M1 最终退出条件与 PR #15 文档一致性独立审查

- 审查日期：2026-10-17（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #15](https://github.com/XKHoshizora/specmetry/pull/15)
- 基准提交：`5aad555cc8293e3fb28d82aff7844a3112fcd502`
- 目标提交：`9b8ea5de46e14542b6e2d04a4f323d9386fa29ce`
- 主报告：`docs/m1-exit-evidence.md` 第 2 版
- master CI：[39019706495](https://github.com/XKHoshizora/specmetry/actions/runs/39019706495)
- PR CI：[39020671974](https://github.com/XKHoshizora/specmetry/actions/runs/39020671974)
- 范围：M1 最终退出条件的证据完整性、提交适用性、PR #15 文档一致性，以及新披露的验证边界。不重新全面审查已获得 GO 的九个工作包。
- 方法：固定提交差异和必要调用链静态审查、既有审查证据核对、三平台 CI 元数据及日志核查，以及目标提交隔离副本中的定向独立测试。

**结论：PR #15 GO；固定代码快照的 M1 退出条件 GO。**

59 项 P0 最终为 **59 PASS、0 INCOMPLETE**。没有发现阻塞合并或建议负责人验收的问题。存在几处非阻塞的证据表述问题，详见下文。

这不是代替产品负责人宣布 M1 验收，也不授权开始 M2。

## BLOCKER

无。

## NON-BLOCKER

### N1：PASS 的定义未准确涵盖独立审查证据

[主报告第 102 行](../m1-exit-evidence.md#L102) 将 PASS 定义为“M1 要求的每一条都有断言，并在 CI 中运行通过”。这一表述过强：SRC-07 的“不解析文档语义”来自 Review 036 的独立调用链审查，部分其他条款也组合使用测试和源码证据。

建议改为：“M1 各项要求均有适用的测试、独立审查或规定的其他验证证据；自动化测试注明对应提交和 CI 结果。”

不阻塞的原因是：SRC-07 行及第 8、10 节已经明确披露证据来源，实际证据充分，无须新增功能或测试才能成立。

### N2：CH-01／CH-02 的结论依据应写成证据链

[主报告第 327～328 行](../m1-exit-evidence.md#L327) 提到“两格不是第一版缺口”及“按‘或’可以接受”。这两句话本身不能证明验收成立。

本轮已通过统一识别机制、真实命令接线、既有进程证据及独立复验确认结论成立。建议用下面的 CH-02 裁定替换上述论据，不改变验收标准。

### N3：历史审查存在失效锚点，保留原文即可

[Review 007 第 15 行](codex-review-007.md#L15) 使用 `#112-数据库迁移`，当前对应标题是 [architecture §11.2“迁移”](../architecture.md#112-迁移)。

该链接在早期归档中已经存在，不是 PR #15 引入。文档、章节号、审查提交仍可追溯，不影响 M1 结论。后续可在活跃索引补充定位，**不改写历史审查原文**。

## 提交与 CI 适用性

| 对象 | 核查结果 |
|---|---|
| PR #15 HEAD | 仍为 `9b8ea5de46e14542b6e2d04a4f323d9386fa29ce`，未变化 |
| 基准 | `5aad555cc8293e3fb28d82aff7844a3112fcd502`，是目标祖先 |
| 实际差异 | 仅 8 个文档及审查归档文件；生产代码、Schema、迁移、测试均未改变 |
| PR #16 | 合并提交 `01bd50f` 与 Review 036 目标 `e1d2ea7` 的树相同 |
| PR #17 | 合并提交 `5aad555` 与 Review 037 目标 `a106f9e` 的树相同 |
| master CI | [39019706495](https://github.com/XKHoshizora/specmetry/actions/runs/39019706495)，实际检出 `5aad555`，Attempt 1，8/8 成功 |
| PR CI | [39020671974](https://github.com/XKHoshizora/specmetry/actions/runs/39020671974)，元数据 HEAD 为目标提交；实际检出合并快照 `a3b43c2`，Attempt 1，8/8 成功 |

PR CI 合并快照的父提交是基准和目标，树为 `0f97617…`，与审查目标完全相同。没有将 PR HEAD、合并快照和 master 提交混用。

九个工作包及相关安全修复的合并提交，均已确认属于基准的历史。

本 PR 的 8 个变更文件为：

- `AGENTS.md`
- `docs/acceptance-criteria.md`
- `docs/architecture.md`
- `docs/implementation-plan.md`
- `docs/m1-exit-evidence.md`
- `docs/review-resolution.md`
- `docs/reviews/codex-review-035.md`
- `docs/reviews/codex-review-037.md`

## 退出条件集合与六项旧缺口

按 acceptance-criteria 的里程碑列独立提取，并排除明确标注的 P1：

| 类别 | P0 数量 |
|---|---:|
| INIT | 7 |
| WT | 1 |
| SRC | 8 |
| BL | 27 |
| PRV | 2 |
| TGT | 10 |
| CH | 2 |
| CX-14、CX-17 | 2 |
| **合计** | **59** |

与报告逐项匹配，**无遗漏、无重复、无第 60 项**。项目名称修复是独立前置条件，不计入这 59 项。

第一版六项 INCOMPLETE 的关闭依据成立：

| 项目 | 关闭依据 |
|---|---|
| CH-01 | PR #16 补充真实 PTY／正式二进制的 Agent 环境变量拒绝证据 |
| CH-02 | PR #16 补充真实子进程重定向证据，加上本轮下面的组合审查 |
| BL-21 | 明确断言环境变量建议、字段指针、拒绝及不泄露 |
| BL-27 | 真实冲突前置条件；缺少 change_class 拒绝，合法对照成功 |
| PRV-07 | Review 四类字段的统一扫描、准确指针、确认前拒绝及无写入 |
| SRC-07 | **Review 036 的独立源码调用链审查 PASS**，不是 CI 语义测试 |

SRC-07 相关的来源处理、片段匹配、项目规则及迁移文件，相对 Review 036 没有变化，原独立审查证据仍适用。

P1 的 INIT-08、INIT-10 有对应断言且三平台通过；INV-05 未实现，按既定规则不阻塞 M1。

## CH-02 两个场景的正式裁定：PASS

`review` 和 `question answer` 的 stdin 非终端场景，现有仓库证据可以组合成立，理由如下：

1. [executeRoot](../../internal/cli/root.go#L29) 在命令执行前，从进程的 `os.Stdin`、`os.Stdout` 统一检测通道。
2. [channel.Decide](../../internal/channel/channel.go#L60) 对非终端 stdin 一律产生 `agent_cli`；WP1.2 已有真实子进程证据验证该操作系统边界。
3. [review](../../internal/cli/review.go#L44) 和 [question answer](../../internal/cli/question.go#L170) 都在读取输入、打开数据库及人工确认之前调用相同的 `RequireHuman`，没有命令专属旁路。
4. 两条真实命令已有 stdout 重定向的进程级证据。其进程内非终端测试调用生产 `Execute`，**没有注入 human/agent 通道**；它们证明命令接线和拒绝后的业务状态，单独不能替代真实进程证明。
5. `TestBuiltBinary` 另行验证正式 `main` 入口、真实退出码及关键通道边界。

已批准的三类证据定义没有要求“每条命令的每种组合都必须使用正式二进制”。上述证据链满足其证明要求，而不是因为第一版未列缺口便自动判为 PASS。

本轮还在隔离副本补做了直接验证：

- 正式二进制的 stdin 接到保持写端打开、没有数据的管道；stdout 保持真实 ConPTY。
- 对 `review --file -`、合法审核文件、`question answer --file -` 和正常 `-m` 参数，分别验证文本和 JSON 模式。
- 全部 exit 3、无确认提示、业务状态不变；没有因读取管道而阻塞。
- JSON stdout 完整解码、检查 EOF，并通过 `error.v1` 校验。

这些是**本轮 Windows 独立审查证据**，未加入仓库，也没有冒称已在三平台 CI 中运行。

`question answer` 只在原测试中使用 CLAUDECODE，不构成缺口：两个变量通过同一识别循环产生同一 Detection，随后执行相同防护。本轮也用正式二进制补验了 CLAUDE_CODE_ENTRYPOINT：拒绝且 canary 不泄露；无 Agent 变量、正确人工确认的对照能够成功。

## 实际运行与未验证边界

本机环境为 Windows amd64、Go 1.27.1、Git 2.56.0.windows.2。在目标提交隔离副本中，以下 **10 项测试全部通过，未跳过，总耗时 105.492 秒**：

- `TestBuiltBinary`
- `TestHumanOnlyCommandRefusesWithoutTerminal`
- `TestAgentVariablesRefuseTheRemainingHumanOnlyCommands`
- `TestRedirectionRefusesTheRemainingHumanOnlyCommands`
- `TestProposeSecretInACheckNamesAnEnvironmentVariable`
- `TestReviewConflictItemNeedsAChangeClass`
- `TestReviewRefusesASecretAnywhereInTheDocument`
- `TestQuestionInTerminal`
- `TestReviewInTerminal`
- 本轮新增的 `TestReview038RealStdinRefusal`

另核查两组 CI 的三个平台日志：

- Linux 主测试步骤启用 Race Detector。
- 两项真实旧 Git 测试按设计跳过；没有将其写成三平台实际通过。
- Linux 四项跨用户测试在普通步骤跳过，随后由同一 job 的 root 步骤实际通过。
- Windows 不可读来源测试按设计跳过，不是任何 M1 项目的唯一证据。

本轮没有重跑真实 Git 2.31.1／2.32.0 的安全实验。第二版记录的重跑对象为 `5aad555` 的代码树，提交适用性成立；这些最新重跑属于**作者提供的本地证据**，不能描述成本轮独立复现或三平台 CI 结果。

PR #16 的变异检验也准确保留为作者辅助证据。`nowebui` 证明生产 CLI 入口，不证明完整发布包。Windows `check.sh` 未完整通过、延长超时后的 CLI 包通过、GitHub CI 通过，报告已分别记录。

## 文档、产品决定与风险

AGENTS、architecture、implementation-plan、acceptance-criteria、review-resolution §45～§48 与主报告总体一致：

- 九个工作包已验收，M1 仍等待最终核查及负责人验收。
- 未新增 M2 延期，也没有移走 M1 必须完成的模型或校验。
- M-3、M-11 的关闭有证据；M-12 的修复与兼容性限制分别记录。
- M-1、M-2、M-4～M-10、M-13 均保持未关闭，没有被归为已接受风险。

### PRODUCT DECISION

无新增待决事项。三类证据定义和方案 1 已有负责人明确批准。

### ACCEPTED RISK

仅沿用方案 1 的局部兼容性限制。已有围栏名称的旧项目暂时不能通过 init 修改阶段、补写 marker 或登记关联 worktree。其保证是业务事实、marker、exclude 和仓库文件不变；不是“数据库未打开”或“数据目录绝对零变化”，也不放宽 doctor、版本保护或其他存储契约。

## 最终结论

| 最终判断 | 结论 |
|---|---|
| PR #15 是否可以合并 | **GO** |
| 固定快照的 M1 退出条件 | **GO：59 PASS，0 INCOMPLETE** |
| 是否可以建议负责人正式验收 | **可以，在合并 PR #15、核对合并结果及 master CI 后决定** |
| 是否需要新增产品决策或阻塞性补证 | **不需要** |

审查结束时，原仓库工作区仍然干净；未修改、提交、推送或合并，未启动 M2。

归档说明：本文件在审查完成后，按用户明确要求保存。此次归档只新增 `docs/reviews/codex-review-038.md`，不修改其他文件，不提交或推送代码。文中的审查结论、证据范围及未验证边界保持不变。
