# Codex 审查意见处理记录

> 状态：2026-10-14 · 对应设计版本 Draft v0.3
> 原始审查：[reviews/codex-review-001.md](reviews/codex-review-001.md)（第一轮，§1 ~ §4）、[reviews/codex-review-002.md](reviews/codex-review-002.md)（冻结审查，§5）、[reviews/codex-review-003.md](reviews/codex-review-003.md)（最终设计审查，§6）、[reviews/codex-review-004.md](reviews/codex-review-004.md)（M0 代码审查，§7）、[reviews/codex-review-005.md](reviews/codex-review-005.md)（M0 第二次代码审查，§8）、[reviews/codex-review-006.md](reviews/codex-review-006.md)（M0 第三次代码审查，§9）、[reviews/codex-review-007.md](reviews/codex-review-007.md)（M0 最终定向复审，§10）、[reviews/codex-review-008.md](reviews/codex-review-008.md)（M0 代码审查 GO，§11）、[reviews/codex-review-009.md](reviews/codex-review-009.md)（M1 设计审查 GO，§14）、[reviews/codex-review-010.md](reviews/codex-review-010.md)（WP1.3 代码审查，§15）、[reviews/codex-review-011.md](reviews/codex-review-011.md)（WP1.3 最终复审 GO，§16）、[reviews/codex-review-012.md](reviews/codex-review-012.md)（WP1.0 代码审查，§17）、[reviews/codex-review-013.md](reviews/codex-review-013.md)（WP1.0 最终复审 GO，§18）
> 处理原则：逐条分析，不默认全盘接受；只要会改变核心模型或交换协议，就在本轮设计清楚；只涉及实现细节的，安排在对应模块开工前关闭。不用"开发时再说"来掩盖当前的数据模型矛盾。

**状态术语**（本文和 implementation-plan 统一使用）：

| 维度 | 取值 | 含义 |
|---|---|---|
| 设计状态 | 未解决 | 设计中还存在已知缺口 |
| | 已解决（待冻结复审） | 文档、schema 和验收项已经修订，但还没有经过审查方确认 |
| | **已冻结** | 审查方复审通过，契约冻结；之后只能通过变更流程修改 |
| 实现状态 | 未开始 / 进行中 | — |
| | **已验证** | 对应的验收项和 CX 反例已经在实现中通过 |

**当前总体状态**（2026-10-14 更新）：codex-review-003 结论为 **GO**，BLOCKER 为 0。本文所有条目的设计状态均为**已冻结**（包括 codex-review-003 要求的两处同步修正，见 §6）；冻结后的修改必须走 implementation-plan §1 的变更流程。**没有任何条目整体处于"已验证"**：验证需要等到对应里程碑实现完成，并通过验收项和 CX 反例；目前只有 #6、#12 中属于 M0 的验收项已验证（§12）。M0 的代码审查经过五轮（codex-review-004 ~ 008），第五轮（codex-review-008）结论为 **GO**，BLOCKER 为 0（§11）；GitHub Actions 三平台 CI 通过后，产品负责人于 2026-10-14 正式验收 M0（§12）。M1 开工准备中的产品决策与设计收尾见 §13，已于 2026-10-14 最终确认，G2 已满足。

## 1. 总览（codex-review-001）

| # | 问题 | 结论 | 设计状态 | 实现验证（里程碑） | 主要验收项 |
|---|---|---|---|---|---|
| 1 | 隐私约束与采集设计冲突 | **采纳**（有调整） | 已冻结 | 未开始；M2、M3、M6 | MVP-EV-05、EV-11、EV-12、COL-05、PRV-02、BL-21 |
| 2 | 验收没有绑定交付代码 | **采纳**（不做工作区指纹） | v0.2 解决主体；**剩余缺口 B1 已在 v0.3 解决**，待冻结复审 | 未开始；M2（观测级）、M7（隔离级） | CX-01、CX-02、CX-19 ~ 22、CX-26 ~ 28、ACC-09 |
| 3 | worktree 上下文没有隔离 | **采纳** | 已冻结 | 未开始；M1、M2 | WT-01 ~ 05、CX-03、CX-04 |
| 4 | 部分确认或空范围也可能 `ready` | **采纳** | 已冻结 | 未开始；M1、M2 | TGT-01 ~ 09、CX-05、CX-06、CX-18 |
| 5 | 需求语义变化可以绕过 AC 失效 | **采纳** | 已冻结 | 未开始；M1、M2 | BL-18 ~ 20、CX-07 |
| 6 | 缺少读取版本前置条件；时间戳排序有歧义 | **采纳**（范围扩大到所有关键写入） | v0.2 解决主体；**剩余缺口 B2 已在 v0.3 解决**，待冻结复审 | M0 部分已验证（STO-04，§12）；M1、M2、M5 未开始 | STO-04、BL-03、BL-10、TGT-05、ACC-11、CX-08、CX-09、CX-13 ~ 15 |
| 7 | 检查成功的含义与验收前置条件不明确 | **部分采纳** | 已冻结 | 未开始；M2 | ACC-04 ~ 06、ACC-09、ACC-10、CX-19 ~ 21 |
| 8 | Agent 可以自行消除来源告警 | **采纳** | 已冻结 | 未开始；M1 | SRC-03、SRC-04、CX-17 |
| 9 | "盘点完成"的定义过弱 | **采纳**（不作为交付前置条件） | 已冻结 | 未开始；M2、M4 | INV-01 ~ 04、CX-10、COL-11、SK-06 |
| 10 | Hook 规格与运行时有缺口 | **采纳**（事件细节在 M3 前核实） | 模型已解决；细节在 M3 开工前确定 | 未开始；M3 | COL-01 ~ 15、ACC-03 |
| 11 | 跨 Agent 兼容性 | **部分采纳** | 措辞与矩阵结构已解决；其他 Agent 的核实在 M4 开工前完成 | 未开始；M3、M4、M5 | SK-03、UI-14、CX-23、COL-15 |
| 12 | 本地数据生命周期没有闭合 | **采纳**（不承诺物理擦除） | 模型已解决；ACL 细节在 M0 开工前、恢复命令在 M2 开工前确定 | M0 部分已验证（PRV-05、PRV-06、STO-01 ~ 04，§12）；M2 未开始 | PRV-03、PRV-05、PRV-06、STO-01 ~ 05、CX-11 |
| 中-1 | 引用的历史归属 | **采纳** | 已冻结 | 未开始；M1、M5 | BL-04、BL-22、UI-06 |
| 中-2 | 可选项与里程碑退出条件矛盾 | **采纳** | 已冻结 | — | 结构调整 |
| 中-3 | 离线承诺没有区分边界 | **采纳** | 已冻结 | 未开始；M6 | PRV-01 |
| 结论段 | 八类反例测试 | **采纳**，并补充"自述完成但缺乏有效证据"等反例 | 已冻结 | 未开始；M1 ~ M5 | CX-01 ~ CX-28 |

修改的文档：`data-model.md`（重写）、`architecture.md`（重写）、`prd.md`（重写）、`mvp.md`（重写）、`acceptance-criteria.md`（重写）、`vision.md`、`AGENTS.md`；新增 `schemas/*.json`（14 个）、`schemas/examples/`（20 个合法示例，53 条 schema 反例）、`implementation-plan.md` 和本文。

本轮修订之后，我们又做了一次独立的跨文档自检，发现并修正了 18 处问题，记录在 §4。

## 2. 逐条处理

### 1. 隐私约束与采集设计冲突（严重）

**结论：采纳，有三处调整。**

**分析**：审查指出的冲突确实存在。v0.1 中，C0 禁止保存工具输出，但检查却保存完整输出和输出尾部；Hook 保存命令文本。审查认为脱敏不能消除这个冲突，这一判断成立：脱敏只能处理已知格式的密钥，而客户数据、内联代码、普通格式的秘密都会漏过；替换还可能改变命令语义。

**采纳**：

- [architecture.md §10](architecture.md#10-隐私与数据保留契约) 成为隐私与保留的**唯一规范来源**，分为 C0 禁止、C1 仅本地、C2 同步候选三级，并明确了本地持久化与未来同步的边界。
- 检查执行**不保存任何输出**（也不保存输出哈希），只保存结果码、失败类型、时长和代码上下文。
- Hook **不采集命令文本和文件路径**。白名单由 [`spool-event.v1`](../schemas/spool-event.v1.schema.json) 规范定义，其中没有任何自由文本字段。
- 检查命令中出现密钥字面量时，**拒绝写入**，并要求改用环境变量名。
- 提交的标题和正文不保存。

**调整（没有原样采纳的部分）**：

1. 审查建议保留"必要哈希"。我们评估后认为输出哈希没有用途，所以连哈希也不保存。
2. 来源引用片段**保留**，上限从 1000 字符降到 500 字符，并且只能来自已登记的来源。理由：来源可追溯是核心价值；审查要求的是"明确边界"，没有要求取消。
3. 新增**输出实时透传**：检查输出直接显示在调用者（人或 Agent）的终端中，不经过 Specmetry 存储，以保留可调试性。
4. 所有自由文本的处理从"脱敏替换"改为"命中即拒绝"。审查关于"替换会改变语义"的论证，同样适用于 Claim 摘要和理由。

**修改的文档**：architecture §10、§12；data-model §4.9、§4.11、§4.12、§9；prd F13、PD-20；AGENTS.md；`schemas/spool-event`、`common`（`shortSummary`、`checkSpec`）。

**剩余风险**：

- C1 内容（引用片段、路径、Claim 摘要、人填写的理由）仍然可能含有客户的敏感信息。缓解措施：只在本地；有权限控制；有 `purge`；未来同步默认排除 C1。
- 密钥规则存在漏报。缓解措施：可以追加规则；Hook 白名单中没有自由文本，所以漏报只会影响人或 Agent 主动提交的文本。

**验证**：MVP-EV-05（输出 canary 不入库）、MVP-EV-11（提交标题 canary）、MVP-COL-05（Hook canary + schema 校验）、MVP-PRV-02（全链路 C0 canary）、MVP-BL-21、MVP-CLM-03、MVP-PRV-07。

### 2. 验收没有可靠地绑定交付代码（严重）

**结论：采纳。MVP 只对干净的提交快照给出正式判定，不实现工作区指纹。**

**分析**：审查给出的反例（在 A 上验收，B 删除功能后仍然 `ready`）在 v0.1 的决策表下确实成立，根源是验收只绑定了 AC 的修订，没有绑定代码对象。

**采纳**：

- 新增 **Delivery Target**：它跟踪一个分支或标签；每次推导时解析出快照 `S`。
- **合格执行**：开始和结束时 HEAD 都等于记录的提交、工作区都干净。只有 `S` 上的合格执行，才参与交付判定。（v0.2 曾经表述为"执行期间 HEAD 没有变化"，这是只观测两端所无法支持的保证；codex-review-002 B1 指出后，在 v0.3 中改为分级证据，见 §5。）
- **历史验收与当前适用性分开**：决定是只追加的历史事实，记录当时的快照；它是否适用于当前的 `S`，由推导决定（[data-model.md §6.2](data-model.md#62-决定的适用性)）。
- 统一交付状态，并给出保守规则（§6.8、§6.9）。

**没有采纳的部分**：工作区指纹（审查把它列为"支持 dirty workspace 时"的可选项）。MVP 中，有未提交修改的执行一律不合格。理由：指纹需要定义哪些文件纳入、如何处理被忽略的文件，复杂度高，而且 MVP 的价值在于正式交付，正式交付本来就应该针对提交。

**本轮新增的设计选择（需要确认，PD-16）**：**自动化 AC 的验收可以沿用**。如果新快照上的合格执行使用了同一检查配置、同一输入对象 ID 并且通过，验收就继续适用。否则，目标每前进一个提交，人都要逐条重新验收，在实际中不可行。限制条件：

- 手动 AC 和例外验收不沿用；
- 同一快照上结果不一致时，不沿用；
- 声明的 `inputs` 发生变化时，不沿用。

**修改的文档**：data-model §2、§4.2、§4.11、§4.13、§6、§7.3；architecture AD-12、§6.5；prd §3.2、F4、F9；mvp §1、§3。

**剩余风险**：

- 执行环境（被忽略的文件、环境变量、外部服务）不属于代码对象。同一提交在不同环境中，结果可能不同。缓解措施：`inconsistent_results` 标记，且出现时不沿用。
- 沿用依赖人对"检查是否足以验证 AC"的一次性判断。没有声明 `inputs` 时，测试被削弱可能检测不到（见 #7）。

**验证**：CX-01、CX-02、CX-19、CX-20、CX-21、CX-22、MVP-ACC-09、E2E-6；不变量 #2（由独立的检查器复核每一次 `ready`）。

### 3. worktree 上下文没有隔离（严重）

**结论：采纳。**

**采纳**：

- 新增 `worktrees` 表；项目身份由 git 公共目录确定，各 worktree 共享这个身份。
- Claim 批次、检查执行、会话和事件都记录 `worktree_id`、HEAD 和工作区状态。
- 检查在**调用者所在的 worktree**中执行（v0.1 是相对主 worktree 执行，这是个缺陷）。
- 扫描游标按 ref 维护，取代项目级的 `last_scanned_head`。
- `status` 同时给出交付视图和工作区视图；Dashboard 明确显示正在观察的目标和快照。

**说明**：不同 worktree 在**同一个 SHA** 上的合格执行，被视为等价。这正是 CX-03 中"快进合并后，feature 上的结果生效"的依据：代码对象相同，结果就相同。这一点与 #2 的剩余风险（执行环境的差异）一并说明。

**没有采纳的部分**：在 Dashboard 中提供每个 worktree 的视图。交付判定只针对目标；worktree 只出现在健康页和 CLI 的工作区视图中。这是为了控制复杂度。

**修改的文档**：data-model §1、§3、§4.1、§4.12；architecture AD-14、§6.3、§6.7；prd F5。

**剩余风险**：已删除的 worktree 中的历史事实保留，并标记为 `missing`；同一 SHA 上的环境差异（同上）。

**验证**：MVP-WT-01 ~ 05、MVP-COL-10、CX-03、CX-04、E2E-4。

### 4. 部分确认甚至空范围都可能 `ready`（严重）

**结论：采纳。**

**采纳**：

- 新增人工确认的**交付范围**：明确列出需求清单，绑定交付目标、基准版本和来源集合哈希。
- 确认范围时需要声明：范围完整；排除了多少条待审条目（必须与服务端的计数一致）；`incomplete_docs` 项目已补充代码中看不到的交付目标。
- 范围不能为空；范围全部豁免时，判定为 `not_assessed`（`ALL_WAIVED`）。
- 基准或来源变化之后，范围过期，交付状态降为 `needs_reverification`。
- 审核提交可以附带范围确认，避免反复操作。

**与审查意见一致的取舍**：待审提案**不阻塞**交付，只作为警告。"范围是否完整"由人在确认范围时明确声明，而不是由"有没有待审提案"推断。

**刻意的保守选择**：Agent 新登记一个来源，会改变来源集合哈希，使范围过期。也就是说，Agent 可以"迫使人重新确认"，但无法让交付提前变为 `ready`。这符合保守原则。

**PD-01 已关闭**："`must` 是否足够"不再由系统推断，而是由范围确认决定（默认选中 `must`）。

**修改的文档**：data-model §4.2、§5.3、§6.8、§6.10；prd F4、PD-01、PD-18、PD-21；schemas `scope`、`review`。

**剩余风险**：人可能未经思考就确认范围。缓解措施：确认界面列出待审条目、来源状态、交付目标和快照，并要求填写各项声明（FR-UI-04、MVP-UI-07）。

**验证**：MVP-TGT-01 ~ 08、CX-05、CX-06、CX-18、E2E-1、E2E-5。

### 5. 需求语义变化可以绕过 AC 失效（严重）

**结论：采纳。**

**采纳**：

- 每个 AC 记录 `requirement_revision`，表示它被确认适用于父需求的哪个修订。
- 父需求实质修订后，它的 AC 立即变为 `parent_revised`（`needs_reverification`）。这是一个推导出的"需要复核"的事实，界面无法忽略。
- 审核时，可以对每个 AC 选择 `reaffirm`、修订、退役或暂缓；提案中新增 `reaffirm` 操作。

**设计选择（并入 PD-04）**：`reaffirm` 保留 AC 已有的执行和验收。理由：人已经明确确认"这条 AC 在新的需求语义下仍然适用"，如果需要新的验证，应该新增 AC，或者对 AC 做 `material` 修订。审查给出的例子（"只能查看自己的订单"），正确的做法是新增一条 AC；没有新增之前，范围内的需求虽然可能 `ready`，但审核界面会明确提示这一变化。

**修改的文档**：data-model §4.4、§5.2、§6.3（规则 2）、§7.2；prd FR-BL-11；schemas `proposal`（`reaffirm`）、`review`（`criterion_followups`）。

**剩余风险**：审核者可能草率地 `reaffirm`。这属于人工判断，系统已经强制要求作出明确的选择。

**验证**：MVP-BL-18 ~ 20、CX-07、E2E-5。

### 6. 交换格式缺少读取版本前置条件（严重）

**结论：采纳，并扩大到所有关键写入。**

**采纳**：

- 全局 `seq` 是判断先后的唯一依据；时间戳只用于展示。
- 提案携带 `base_version` 和 `target_revision`；审核携带 `seen_base_version`。
- Claim 携带 `base_version` 和 `target_revision`。
- 验收携带目标、快照、AC 修订、需求修订和证据 ID。
- 范围确认携带基准版本和来源集合哈希；来源确认携带看到的哈希。
- 检查在开始时固定修订、配置和上下文，之后**不会重新绑定**。
- 不一致时：CLI 返回退出码 5，API 返回 409，并给出当前值。

**设计选择**：提案提交对 `base_version` 采用**严格匹配**（即使提案只有新增条目）。理由是规则简单、可预测；代价是人审核之后，Agent 需要重新读取。审核阶段另外有冲突检测作为第二层保护。

**修改的文档**：data-model §1、§4.6、§4.10、§4.11、§4.13、§7.1、§7.4；architecture AD-13、§9；全部写入类 schema。

**验证**：MVP-STO-04、MVP-BL-03、MVP-BL-10、MVP-TGT-05、MVP-SRC-03、CX-08、CX-09、CX-13、CX-14、CX-15、MVP-ST-05、MVP-SK-07。

### 7. 检查成功的含义与验收前置条件不明确（高）

**结论：部分采纳。**

**采纳**：

- 收窄"验证"的含义：v0.1 的 `verified` 改为 `ready_for_review`，含义是"已确认的检查在快照上合格执行并通过"，**不声称**语义覆盖。
- 自动化 AC 的检查**必须**附带理由，由人审核。
- 明确正常、手动、例外三种验收的前置条件。例外验收必须填写理由，只适用于当前快照，并带警告。
- 检查失败时不能接受。
- "自动恢复验收"（即沿用）受代码对象、配置和输入的约束（见 #2）。
- 可以声明 `inputs`，记录其对象 ID，作为测试变化的信号。

**没有采纳的部分**：任何试图证明"检查确实覆盖了 AC"的机制，例如覆盖率分析、测试代码审查。理由：在通用场景下无法可靠实现，而且属于范围外的"复杂代码影响分析"。最终判断由人作出。

**修改的文档**：data-model §4.4、§4.13、§6.2、§6.3、§7.3；prd F9；schemas `common.checkSpec`、`decision`。

**剩余风险**：

- 没有声明 `inputs` 的 AC，测试被削弱时无法检测（文档建议声明）。
- 不稳定的测试会触发 `inconsistent_results`，增加人工工作量。

**验证**：MVP-ACC-04 ~ 06、MVP-ACC-09、MVP-BL-21、CX-19、CX-20、CX-21。

### 8. Agent 可以自行消除来源变化告警（高）

**结论：采纳。**

**采纳**：

- `source ack` 和 `source remove` 只能由人执行。确认来源变化时，必须绑定看到的哈希（不一致时返回退出码 5），并填写理由。
- 来源移除后，引用保留，并显示为 `source_removed`。
- 来源集合哈希参与范围的绑定。
- Agent 仍然可以登记新来源、提交处理提案。

**修改的文档**：data-model §4.3；prd §3.3、F2；architecture §5 拦截清单；schemas `source-ack`；AGENTS.md。

**验证**：MVP-SRC-03、MVP-SRC-04、CX-17、E2E-7。

### 9. "盘点完成"的定义过弱（高）

**结论：采纳；但盘点不作为交付的前置条件。**

**采纳**：

- `audit` 批次必须声明范围。
- 按 AC 计算覆盖率：当前修订至少有一条 `audit` Claim，才算盘点过。
- 只有全量覆盖才算 `inventoried`；没有盘点到的 AC，进度为 `unknown`。
- `sync` 的比较起点有明确的优先级和 `none` 状态；变化路径包括未跟踪的文件。
- `incomplete_docs` 项目确认范围时，必须声明已补充交付目标。

**没有采纳的部分**：把"盘点完成"作为交付判定的前置条件。理由：交付判定本来就要求范围内的每个 AC 在快照上有证据和人工决定，盘点只影响"进度"这一参考视图，再加一道闸门是重复的。

**说明**："30 分钟建立基准"在 vision 中改为"以夹具评测衡量的目标"，不再作为承诺。

**修改的文档**：data-model §4.10、§6.5、§6.12；prd F6、FR-COL-05、FR-SK-06；architecture §6.3；vision §8。

**剩余风险**：audit 的质量取决于 Agent；Agent 可能滥用 `unknown` 断言（评测中会观察）。

**验证**：MVP-INV-01 ~ 04、CX-10、MVP-COL-11、MVP-SK-06、MVP-TGT-06。

### 10. Hook 规格与真实运行时存在缺口（高）

**结论：采纳。模型层面本轮关闭；事件名和字段的核实，以及版本固定，在 M3 开工前完成（TD-12）。**

**采纳**：

- 会话起点 HEAD **只通过读文件**获取（`HEAD`、`refs`、`packed-refs`），失败时记为 `unknown`，**绝不**用入库时的 HEAD 代替。
- 覆盖 PostToolUseFailure；拦截覆盖 Bash 和 PowerShell。
- 输入上限 1 MiB，内部预算 200 ms，外部超时 5 s。
- 丢弃的事件可以计数、可以看到。
- 把"永不阻塞"改写为可以验证的延迟预算。
- `doctor` 报告 Claude Code 版本是否在经过测试的范围内。
- 保持原有分工：Hook 只记录活动，语义映射由 Skill 完成。

**为什么可以推迟细节**：spool 记录的格式是我们自己定义的（`spool-event.v1`）。Hook 事件名或字段如果有出入，只影响适配器中的映射，不影响数据模型和交换协议。

**修改的文档**：architecture §12；data-model §4.9；prd F11；schemas `spool-event`。

**剩余风险**：Claude Code 接口变化；在使用 reftable 的仓库中，会话起点更常为 `unknown`。

**验证**：MVP-COL-01 ~ 15、MVP-ACC-03。

### 11. "能执行 shell"不足以保证跨 Agent 兼容（高）

**结论：部分采纳。**

**采纳**：

- 兼容矩阵新增 Shell 工具、沙箱与数据目录写入、`.git/info/exclude`、会话标识、`sync` 比较起点、人工操作路径等维度（architecture §12.4）。
- 如实声明能力边界：通道识别是启发式的，Dashboard 令牌不能证明操作者是人，"人类专属"的含义是防止误执行（architecture §5、AD-08）。
- Skill 明确禁止通过 CLI、HTTP、浏览器自动化或写文件代做人工操作。
- 数据目录不可写时，返回退出码 6 并给出指引；`.git/info/exclude` 不可写时只给出警告。

**没有采纳的部分**：由 Specmetry 在 `setup` 时**自动配置**其他 Agent 的沙箱权限。理由：修改其他工具的安全配置，与"不接管 Agent"的原则冲突，也有安全风险。我们只检测并给出指引。

**修改的文档**：architecture §5、§12.4；prd §3.3；mvp §4；AGENTS.md。

**剩余风险**：使用伪终端的 Agent 可能被识别为人；读取了运行时文件的 Agent 可以调用 API（CX-23 已记录这一已知边界）。

**验证**：MVP-SK-03、MVP-UI-14、CX-23、MVP-COL-15；M4 开工前核实其他 Agent 的实际约束，并更新兼容矩阵。

### 12. 本地数据生命周期没有闭合（高）

**结论：采纳。明确不承诺物理擦除。**

**采纳**：

- Windows 使用受保护的 DACL（只授予当前用户），令牌文件和备份的权限与数据目录相同。
- 只用 `VACUUM INTO` 生成一致性备份，保留最近 3 份。
- 迁移锁、`min_binary_version`：旧程序拒绝打开新 schema。
- `purge` 的步骤：先写墓碑，再删除行，然后 `wal_checkpoint` 和 `VACUUM`，最后按用户的选择处理备份。
- **墓碑在每次打开数据库时生效**，防止从备份恢复后数据复活。

**没有采纳的部分**：物理擦除保证（文件系统或 SSD 上的残留、用户自己制作的副本），只如实声明。

**推迟的细节**：Windows ACL 的具体授权对象（是否包含 SYSTEM，TD-22）在 M0 开工前确定；`backup restore` 命令属于 P1，在 M2 开工前细化。墓碑机制不依赖恢复命令，手动复制备份文件同样受保护。

**修改的文档**：architecture §8、§11；data-model §9；prd F13、F14。

**验证**：MVP-PRV-03、MVP-PRV-05、MVP-PRV-06、MVP-STO-01 ~ 05、CX-11、E2E-8。

### 中-1. 来源与代码引用的历史归属

**结论：采纳。**

- `source_refs` 和 `code_refs` 都只追加，并绑定 `entity_revision`。
- 实质更新时，沿用旧的引用，重新校验后写成新行。
- `code_refs` 只保存路径和行号。
- 界面如实说明：校验只证明"片段存在、路径存在"，不证明语义正确。

**验证**：MVP-BL-04、MVP-BL-22、MVP-UI-06。

### 中-2. MVP 的可选项与里程碑退出条件矛盾

**结论：采纳。**

- 里程碑的退出条件只包含 P0 项和 CX 项；P1 项不阻塞。
- 每个验收项都标注了里程碑。
- 交换契约（JSON Schema）在本轮交付，M0 起在 CI 中校验。
- P1 中的趋势、完整活动页、报告与导出、Stop 提醒、粘贴来源，降为 P2，以便优先完成可信的交付闭环。

### 中-3. 离线承诺没有区分边界

**结论：采纳。**

- 网络边界分为核心、Agent、检查命令三部分（architecture §10.5）；NFR-01 已改写。
- 保留人工或录制输入的离线流程。

**验证**：MVP-PRV-01。

## 3. 审查结论段的要求

| 要求 | 处理 |
|---|---|
| 开发前先确定第 1 ~ 6 项 | v0.2 解决了主体；codex-review-002 指出 #2、#6 仍有缺口（B1、B2），已在 v0.3 解决（§5），待冻结复审。仍有产品决策需要确认：PD-16、PD-17、PD-19 ~ PD-22（[prd.md §9](prd.md#9-产品待决策事项)）。 |
| MVP 必需范围和交换契约的安排 | mvp.md v0.2；`schemas/` 已交付 |
| 第 7 ~ 12 项在对应模块开发前关闭 | 模型层面已在本轮关闭；实现细节的截止时间见 [implementation-plan.md §3](implementation-plan.md#3-设计细化截止点) |
| M0 和只读技术验证可以先推进 | 采纳：implementation-plan 中，M0 只依赖技术决策 |
| 八类反例测试 | CX-01（代码改变未重跑）、CX-03（worktree 相反）、CX-05（部分基准）、CX-07（父需求变更）、CX-08（旧页面）、CX-09（同一毫秒）、CX-10（局部 audit）、CX-11（`purge` 后恢复备份） |
| 新增：Agent 自述完成但缺乏有效证据 | CX-12（五种变体） |
| 其他补充的反例 | CX-02、CX-04、CX-06、CX-13 ~ 25 |

## 4. 第二轮自检发现并修正的问题

v0.2 初稿完成后，由一个独立的审查代理通读全部文档和 schema，共发现 18 处缺陷，均已修正。按严重程度列出如下：

| # | 缺陷 | 修正 | 验证 |
|---|---|---|---|
| 1 | 未被排除的 marker 会让主 worktree 永远是 dirty，因而永远无法 `ready` | 计算 `tree_state` 时始终排除 `.specmetry/`；项目级安装只写入已被 git 忽略的设置文件（data-model §1、architecture §10.6） | CX-24 |
| 2 | 首次审核时，附带的范围无法引用刚新增的需求 | 审核中附带的范围可以使用提案条目的 `item_id`，在事务内解析为新编号（scope.v1、data-model §4.2） | MVP-TGT-07、SCH-45 |
| 3 | 拦截规则会误拦 `dashboard --page review` 和 heredoc 正文 | 改为按参数位置匹配，跳过 heredoc，并递归检查 `-c` 字符串（architecture §5） | MVP-ACC-03 中的"必须放行"用例 |
| 4 | Agent 在同一提交上重跑检查，就能解除人的拒收 | 只有"代码变化后的新合格通过"或人的新决定才能解除拒收（data-model §6.3 第 5 条） | CX-25 |
| 5 | 多个 JSON 输出和 API 请求体没有 schema；`status` 中没有目标（goal）和来源 | 新增 `init.v1`、`check-run.v1`、`gaps.v1`、`question-answer.v1`；`status.v1` 加入 `goals` 和 `sources`；面向人的读取模型的 schema 列入开工前的截止点 | MVP-ST-10、SCH-46 ~ 51 |
| 6 | 差距可能在 `ready` 时仍然阻塞；同一个 AC 产生重复差距 | 条目级差距只由交付状态产生，每个条目至多一个，并限定在范围内（data-model §6.10）；新增不变量"`ready` ⇔ 没有阻塞差距" | MVP-ST-13、不变量 #8 |
| 7 | 推导依赖读取时的磁盘状态，不是纯函数；`--diff-since` 的语义不明 | 来源观测哈希作为显式输入，并写进输出；明确 `--diff-since` 的含义（data-model §6） | MVP-ST-14、MVP-ST-11 |
| 8 | 同一次审核内的应用顺序没有定义 | 定义 11 步的固定顺序（data-model §5.5） | MVP-BL-24 |
| 9 | 立即应用的人工编辑无法声明 `material` / `editorial` | `proposal.v1` 的更新条目新增 `change_class`；人工编辑时必填（V-P09） | MVP-BL-23、SCH-53 |
| 10 | 恢复旧备份后再迁移，迁移前备份可能把已删除的项目带回来 | 打开数据库时，先执行墓碑，再生成迁移前备份；删除方式与 schema 版本无关（architecture §11.2、§11.4） | CX-11（加强） |
| 11 | 部分 M1 退出项依赖 M2 才有的推导 | 拆分为"M1 / M2"：写入侧在 M1 验证，推导侧在 M2 验证 | implementation-plan M1、M2 |
| 12 | 提案文本中可以包含代码；来源片段可以逐段拼出全文；源代码可以被登记为来源 | 所有自由文本禁止代码围栏；每个来源的片段总量有上限；来源只能是文档类文件 | MVP-BL-23、BL-26、SRC-07、SCH-43 |
| 13 | 部分验收项与规则矛盾（ACC-08、ST-01、CX-12(c)、E2E-1/2/5） | 逐条修正 | — |
| 14 | 问题与提案条目的关联没有回填；`BLOCKING_QUESTION` 的统计口径不一致 | 定义回填规则和"关联条目被拒绝"的处理；统一口径（data-model §4.8、§6.8） | MVP-BL-25 |
| 15 | `pending_items_excluded` 和豁免的前置条件不完整 | 补充到 data-model §7.1；`target_id` 始终必填 | MVP-TGT-06 |
| 16 | 实施计划与验收表中的里程碑标注不一致 | 已对齐 | — |
| 17 | 部分 P0 需求没有被直接引用 | 新增 SRC-07，并在追溯表中补全映射 | §6 追溯表 |
| 18 | 若干措辞和细节问题：`not_found` 的退出码；无法产生的原因码；原因码的优先级；重复执行 `source add`；Hook 如何从关联 worktree 找到项目；兼容矩阵缺少的行 | 逐条补充（architecture §9、§12.2、§12.4；data-model §4.3、§6.3） | — |

## 5. 冻结审查（codex-review-002）的处理

审查结论为 **NO-GO**，原因只有两个阻塞项 B1、B2。另有三个非阻塞项，以及一组被接受为 MVP 边界的风险。

| 编号 | 问题 | 结论 | 设计状态 | 实现依赖 | 验证 |
|---|---|---|---|---|---|
| B1 | 只观测执行前后两端，不能证明检查针对的是所记录的快照 | **采纳**：分级证据 + 收窄保证 | 已冻结 | M2：观测级执行器、证据等级、观测级证据不沿用、风险确认、`--isolation required` 返回退出码 6。M7（MVP 之后）：隔离执行器。 | MVP-EV-13、ACC-09、ACC-10、CX-26 ~ 28 |
| B2 | 旧页面可以覆盖刚发生的人工拒收 | **采纳** | 已冻结 | M2：CLI 与写事务；M5：API 与界面 | MVP-ACC-08、ACC-11、UI-08、UI-16、CX-08 (e)(f) |
| NB-1 | 新的通过证据解除拒收时，没有要求发生在拒收之后 | **采纳**：增加 `V.seq > D.seq` | 已冻结 | M2 | CX-25 (b) |
| NB-2 | Claim 对交付状态影响的表述不一致 | **采纳**：统一表述 | 已冻结 | — | data-model §2；不变量 #3 |
| NB-3 | `check run --json` 的输出流不明确 | **采纳**：stdout 只输出 JSON，检查输出转到 stderr | 已冻结 | M2 | MVP-EV-12 |

### B1：证据与快照的一致性

**设计**（[data-model.md §4.11.1](data-model.md#4111-证据等级)、§6.2、§6.8、§7.3、§7.4；[architecture.md §6.4](architecture.md#64-检查执行与证据等级)）：

- **`observed`（观测级）**：在发起执行的 worktree 中执行。只保证开始和结束两个时刻，HEAD 等于记录的提交，并且工作区干净；**不保证**执行期间没有变化。它不能被称为"快照已验证"。
- **`snapshot_isolated`（隔离级）**：只有同时验证了以下四点，才记为隔离级——私有快照物化、由外部强制只读（文件权限位不算）、执行前写入探测失败、执行后树比对一致。保证范围仅限于"检查读到的受跟踪文件等于 `S`"，执行环境不在保证范围内。

**对交付判定的影响**：

- 两种等级的合格执行都可以让 AC 进入 `ready_for_review`，都可以支撑人在同一快照上的直接验收；
- **只有隔离级才能支撑沿用**，也只有隔离级才能让交付判定标为 `snapshot_isolated`；
- 人依据观测级证据验收时，必须确认 `observed_evidence` 风险，协议会校验这一点（V-D01）。

**无法取得隔离环境时**：

- `--isolation required`：不执行，不写记录，返回退出码 6；
- `--isolation auto`（M7）：降级为观测级，并记录原因；
- 系统不存在任何把观测级记为隔离级的路径。

**MVP 尚未实现严格执行器时，必须保守保持待验证的状态**：

- 目标前进后，自动化 AC 不会自动恢复为 `accepted`，而是保持 `ready_for_review`（`observed_evidence`），需要人重新验收；
- `delivery.evidence_grade` 只能是 `observed` 或 `none`。

**人工例外验收**：

- 只能在 `S` 上没有任何等级的合格执行、且检查没有失败时使用；
- 必须确认 `no_automated_evidence` 风险，并填写理由；
- 只适用于 `S`，永不沿用，在交付判定中单独列出；
- 不能用来把观测级"升级"为隔离级。

**易用性**：普通项目的初始化和日常使用都不依赖 Docker 或其他隔离后端。

**最小技术验证**（本轮，用 git 2.56 在临时仓库中进行）：

- "检查期间修改受跟踪文件后再恢复"和"HEAD A → B → A"两种情况，开始和结束时的观测结果完全相同。这证实了观测级无法发现中途变化，所以必须收窄保证。
- 用 `chmod` 设为只读后，同一用户可以自行恢复写权限。这证实了文件权限位不能作为隔离机制。
- 用 pathspec 排除 `.specmetry/` 后，工作区状态判定正确。

**剩余风险**：

- MVP 中的交付判定依据观测级证据，执行期间被临时修改的代码，仍可能让检查通过并被人验收。缓解措施：这是已声明的边界；不沿用；必须确认风险；交付判定标为观测级。
- 隔离执行器的后端与平台覆盖尚未确定（TD-24）。

**产品确认**：PD-22（MVP 只提供观测级证据），以及修订后的 PD-16（沿用只接受隔离级证据）。

### B2：防止旧页面覆盖人工决定

**设计**（[data-model.md §7.1](data-model.md#71-写入前置条件)、[architecture.md §6.5](architecture.md#65-验收)、`schemas/decision.v1`）：

- `decision.v1` 新增**必填**的 `seen_decision_id`，第一次决定时为 `null`；读取模型返回 `current_decision_id`。
- 在同一个写事务内，比较"该 AC 当前修订下的最新决定"与 `seen_decision_id`；不一致时返回 409 或退出码 5，不写入，并在 `error.current` 中给出最新决定。
- 客户端不得自动重试；重新读取后，可以明确改判。

**剩余风险**：同一个人在两个页面中都基于最新状态各自操作，依然会按 `seq` 依次生效；这是明确的人工行为，不属于过期操作。

### 被接受为 MVP 边界的风险

codex-review-002 列出的 ACCEPTED RISK 全部接受，已经分别写在对应文档的剩余风险或能力边界中：

- 检查成功不能证明语义覆盖；
- 同一 SHA、不同 worktree 的执行被视为等价；
- 范围确认、`reaffirm` 和例外验收依赖人的判断；
- 通道识别不证明身份；
- Hook 是尽力采集；
- 删除不承诺物理擦除。

### 修改的文件

`schemas/common.v1`、`decision.v1`、`status.v1`、`check-run.v1`；`schemas/examples/`（新增 1 个合法示例；更新 5 个合法示例；新增反例 SCH-54 ~ SCH-62）；`docs/data-model.md`、`architecture.md`、`prd.md`、`mvp.md`、`acceptance-criteria.md`、`implementation-plan.md`、本文；`AGENTS.md`。

## 6. 最终审查（codex-review-003）与产品决策记录

**审查结论**：GO，BLOCKER 为 0。B1、B2 的核心设计关闭，设计冻结。

**审查要求的同步修正**（已完成）：

| 事项 | 修正 | 验证 |
|---|---|---|
| `gaps.v1` 缺少证据等级 | 新增必填的 `evidence_grade`（与 `status.v1` 共用 `common.v1#/$defs/deliveryEvidenceGrade`）；新增合法示例 `gaps.ready-observed.json`，以及反例 SCH-63、SCH-64 | Schema 自检通过 |
| CX-08 的预期 | 从没有决定开始，最终恰好有两条决定（D1 拒收、D2 接受）；被拒绝的过期接受不留下记录 | 实现阶段：M2 / M5 |
| CX-26 / CX-27 的夹具与预期 | 明确了"较早的快照 P 上已经有验收"这一前置条件；人工验收之前 `evidence_grade = none`，人确认风险并验收之后为 `observed` | 实现阶段：M2 |
| 实现提示：并发比较 | data-model §7.1 写明：必须以 `BEGIN IMMEDIATE` 先取得写锁，再比较；MVP-ACC-11 用两个独立连接验证 | 实现阶段：M2 |

**产品与技术决策**（2026-10-13，产品负责人确认）：

| 决策 | 结论 | 记录位置 |
|---|---|---|
| PD-22 | 批准：MVP 只提供观测级证据，隔离执行器放在 MVP 之后 | prd §9 |
| PD-20 | 批准建议的 C1 内容，并附加以下约束：允许保存片段不等于允许自动持久化客户文档原文；C1 默认只存在本地，未来同步必须独立授权；`observed` 不得被描述为 snapshot-verified。由此新增 FR-SRC-07、FR-PRV-08、FR-PRV-09 和对应验收项（M1、M2、M5 实现）。 | prd §9、architecture §10.1 和 §10.7 |
| G0：TD-02、TD-04、TD-10、TD-16 | 按建议默认确认；TD-04 的 spike 已通过 | architecture §16 |
| TD-22 | 当前用户 + SYSTEM | architecture §11.1、§16 |

按照要求，这些补充没有引发新的整体架构审查。

## 7. M0 代码审查（codex-review-004）的处理

**审查结论**：NO-GO，4 项 BLOCKER、4 项 NON-BLOCKER。审查针对提交 `ae195b9`、`23c4479`。

**处理方式**：先把审查中的复现条件写成自动化测试，确认它们在原代码上失败，再做最小修改，然后运行 M0 的全部检查。没有修改冻结的产品目标，也没有删除或放宽任何验收约束；验收项只增不减（MVP-PRV-05、PRV-06、STO-02、EV-07 的描述加严，新增 MVP-STO-06、STO-07 并纳入 M0 退出条件）。

**实现状态**：已修复，本地验证通过（Windows 原生；Linux 通过 WSL 运行交叉编译的测试），**待 Codex 复审**。macOS 只在 CI 中验证，CI 尚未运行。

### 阻塞项

| 编号 | 根因 | 修复 | 反例测试 |
|---|---|---|---|
| B1 已有目录或数据库权限不安全时仍然写入 | `Ensure` 对已存在的目录直接返回；Windows 上的文件保护依赖继承，没有任何写入前检查；`doctor` 只能事后发现 | `Ensure` 对已存在的目录做完整检查，不满足即在写入前拒绝（退出码 6）；检查范围是目录本身和其中所有条目。新增检查：符号链接与联接（含数据目录本身）、Windows 属主与可继承性、macOS 扩展 ACL（architecture §11.1） | `TestOpenRefusesInsecureExistingDirectory`、`...InsecureExistingDatabase`、`...InsecureLeftoverWAL`、`...SymlinkedDatabase`；datadir 包内对目录、文件、链接、联接、不可继承 DACL、macOS ACL 的测试。全部断言拒绝后数据目录逐字节不变 |
| B2 `doctor` 隐式改写数据库 | `query_only` 只限制 SQL，不限制 SQLite 的恢复与 checkpoint；遗留的 WAL 被合并并删除 | 新的只读快照读取：只读打开文件，在内存中按 SQLite 规则应用已提交的 WAL 帧，经只读 VFS 查询，读后校验并重试；无法只读读取时报告限制（architecture §11.5）。实测了驱动在各种文件状态下的行为，并以表征测试 `TestSQLiteReadOnlyModesModifyFiles` 固定下来（SQLite 自身的只读模式在常见状态下都会改动文件） | `TestInspectLeaves...`（崩溃遗留 WAL 与 SHM、只有 WAL、干净关闭、另一进程正在写入、恢复的备份）、`TestDoctorLeavesCrashedDatabaseUntouched`；与 SQLite 自身恢复结果比较的 `TestSnapshotIgnores...`（损坏或截断的最后一帧、未提交的帧、WAL 重新开始后的旧帧）；`TestSnapshotRetriesWhenDatabaseChanges`、`TestSnapshotGivesUpOnContinuousChanges`、`TestHotJournalIsReportedNotRecovered` |
| B3 版本保护晚于写入，也不保护已打开的连接 | 连接初始化先执行 `journal_mode(WAL)` 再检查版本；`WriteTx` 不重新检查 | 打开前用只读快照完成版本保护，通过后才建立读写连接（之后在真实连接上再检查一次）；`WriteTx` 在取得写锁后、执行回调前检查 `schema_version`、`min_binary_version` 和 `recursive_triggers`；迁移在排他事务内再确认一次版本；无法解析的 `min_binary_version` 一律拒绝 | `TestOlderBinaryLeavesRestoredNewerBackupUntouched`、`...CrashedNewerDatabaseUntouched`、`TestWriteTxRefusesAfterConcurrentSchemaUpgrade`、`...AfterMinBinaryVersionRaised`、`TestUnreadableMinBinaryVersionIsRefused`；断言回调未执行，事实、`seq` 和文件都不变 |
| B4 `INSERT OR REPLACE` 绕过只追加约束 | REPLACE 通过删除旧行实现，只有开启 `recursive_triggers` 才触发删除触发器；驱动默认关闭 | 每个连接通过 DSN 开启 `recursive_triggers`（连接池新建的连接同样适用），打开和每次写事务都确认；生产代码和迁移禁止 REPLACE 冲突处理，由静态测试检查 | `TestAppendOnlyRejectsReplace`（主键冲突、UNIQUE 冲突、`REPLACE INTO`、UPSERT、`UPDATE OR REPLACE`）、`...DeclaredReplaceConflict`、`TestEveryPoolConnectionEnablesRecursiveTriggers`、`TestWriteTxRefusesConnectionWithoutRecursiveTriggers`、`TestProductionSQLNeverUsesReplace` |

### 非阻塞项

| 编号 | 处理 | 状态 |
|---|---|---|
| N1 墓碑清理的崩溃恢复与外键顺序 | 删除时延迟外键检查；`projects` 按 `id` 删除；整理前写入标记文件，崩溃后下次打开时完成整理（architecture §11.4）。M1 建表约束写入 data-model §1.1，并由 `TestEveryTableHasProjectID` 对全部迁移强制；同时为 `scope_items` 补上 `project_id` | 已修复；完整的 `purge` 命令仍在 M2 实现 |
| N2 非法墓碑测试是假阳性 | 用相同的迁移集合重新打开，断言错误来自非法 `project_id`，且数据目录不变；墓碑文件改为在写入任何数据之前校验 | 已修复 |
| N3 `doctor` 忽略 `min_binary_version` | `doctor` 复用打开数据库时的版本保护规则 | 已修复（`TestDoctorReportsMinBinaryVersion`） |
| N4 公开文档包含本机绝对路径 | 外部审查原文中的链接目标改为仓库相对路径，文字内容不变；CI 和 `scripts/check.sh` 增加检查，Markdown 链接中不得出现本机绝对路径 | 已修复 |

### 其他发现

- **Linux 上的 lint 失败**：复核三平台时发现，`perm_unix.go` 对目录 `chmod 0700` 会被 gosec（G302）报告，而 CI 的 lint 任务运行在 Linux 上，M0 原提交的首次 CI 会因此失败。目录需要执行位，`0700` 是正确的；已加注说明并排除。现在对 Windows、Linux、macOS 三个目标分别运行 golangci-lint，结果均为 0。
- **测试本身的有效性**：对 WAL 解析做了变异测试（关闭校验和、关闭盐值与校验和、应用未提交的帧、关闭读后校验），确认对应的测试会失败。只关闭盐值检查时测试仍通过，因为累积校验和已经能拒绝旧帧；盐值检查与 SQLite 保持一致，作为第二道检查保留。

### 尚未验证

- MVP-PLT-01：GitHub Actions 三平台 CI 尚未运行（需要推送）。
- macOS：权限、ACL 检测（`TestDarwinCheckReportsACL`）和全部测试只在 CI 中运行；本地只做了交叉编译和 vet、lint。
- Linux 上的 race detector：本机没有 C 编译器，只在 CI 中运行。
- 只读检查不加锁；它保证不修改文件，读到的是某一时刻已提交的一致状态。另一个进程在检查期间持续写入时会重试，仍然失败则报告限制。
- 残余竞态：只读快照通过版本检查之后、建立读写连接之前，如果另一个进程恰好把数据库升级到更新的版本，读写连接上的再次检查会拒绝打开；但如果这时本进程是最后一个连接，关闭连接时 SQLite 会把对方已提交的 WAL 帧 checkpoint 进数据库。逻辑内容不变，文件字节会变化。驱动的公开接口没有提供关闭时不做 checkpoint 的设置（`SQLITE_DBCONFIG_NO_CKPT_ON_CLOSE`），因此记录为已知边界。

## 8. M0 第二次代码审查（codex-review-005）的处理

**审查结论**：NO-GO。B4 关闭；B1、B2 未关闭，B3 部分修复仍阻塞；N1 部分修复；N2 ~ N4 关闭。审查针对提交 `9fcda14`。

**处理方式**：先把审查中的三个反例写成通过公开调用路径（`sqlite.Open`、`sqlite.Inspect`、`specmetry doctor`）的自动化测试，确认它们在 `9fcda14` 上失败；然后**删除**第二轮自行实现的 WAL 快照读取器，改用 SQLite 原生的读取与锁；最后运行全部检查。B4 的实现与测试未改动；N2 ~ N4 未改动。

### 阻塞项

| 编号 | 根因 | 最终解决方式 | 反例测试（修复前失败，修复后通过） |
|---|---|---|---|
| B1 路径替换 | 只检查了数据目录本身，之后按路径重新访问；能写入上级目录的其他用户可以在检查之后把数据目录换成链接 | 从根目录到数据目录逐级检查（Unix 按内核方式跟随路径中的链接）：每一级目录只能由当前用户或 root / 管理员改动，否则在创建或写入任何内容之前拒绝。MVP 对这类位置保守拒绝，不做防御（architecture §11.1） | Linux 真实跨 UID：受害者 uid 4242、攻击者 uid 65534 不断把数据目录换成指向自己数据库的链接，3 秒内的每次打开都被拒绝，攻击者目录中找不到 canary（修复前 canary 出现在攻击者的 WAL 中）；上级目录属于其他用户时拒绝；安全父目录和 sticky 父目录下，攻击者的改名被内核拒绝且 specmetry 正常工作。Windows：上级目录授予 Everyone 完全控制、DELETE 或 FILE_DELETE_CHILD 时拒绝，路径中有联接时拒绝 |
| B2 读到未提交数据 | 自行实现的读取器不持有 SQLite 的锁，无法识别读取期间开始、尚未提交的回滚日志事务 | 删除该读取器。改为 SQLite 只读连接（`mode=ro`、`readonly_shm=1`），只在实测确认不修改文件的状态下读取，由 SQLite 的锁和事务隔离保证只读到已提交的数据；其他状态报告版本未知。另外让每个连接保留 WAL 文件（`SQLITE_FCNTL_PERSIST_WAL`，WAL 在关闭时截断为 0 字节），使 specmetry 关闭后的数据库始终处于可读状态（architecture §11.5） | 回滚日志模式的进行中事务把 `schema_version = 9` 写进主库（日志在内存中，没有 `-journal` 文件）：修复前读取器返回 9，修复后为版本未知（SQLITE_BUSY）。WAL 模式的进行中事务：只报告已提交的版本。各文件状态下 `Inspect` 与 `doctor` 前后目录逐字节一致 |
| B3 打开与升级之间的竞态 | 只读预检与读写连接之间没有协调；升级可以在这个窗口内发生，旧程序随后在读写连接上才发现不兼容，关闭时把新数据库合并 | 把 `.migrate.lock` 改为操作系统排他锁（Unix `flock`，Windows `LockFileEx`），并把整个打开过程（只读版本检查、建立读写连接、墓碑、迁移）放进锁内。升级只能在锁内进行，所以会等旧程序打开完成；旧程序打开后的写入由 `WriteTx` 的复查拒绝。锁由操作系统持有，进程崩溃时自动释放 | 在旧程序判断兼容之后、建立连接之前暂停它，另一进程尝试升级：修复前升级立即完成，旧程序拒绝时主库从 20 480 字节变为 24 576 字节、WAL 和 SHM 被删除；修复后升级等待旧程序打开完成，旧程序随后的写入被拒绝（回调未执行，`seq` 不变），旧程序关闭最后一个连接后，升级后的数据库完整可用 |

### 新方案为什么更简单可靠

- **一致性交给 SQLite**：不再自行解析 WAL、重建页面、做读后校验和重试；"什么是已提交的数据"由 SQLite 的锁与事务隔离决定，这正是审查指出旧方案无法替代的部分。
- **不修改文件的保证来自实测的状态表**：只在两类实测零修改的状态下读取，其余一律报告未知；表征测试 `TestReadOnlyConnectionByFileState` 在两个平台上固定驱动的行为。
- **竞态由一把锁消除**：版本判断、连接建立和迁移在同一个锁内，不再依赖"读后再检查"。操作系统锁还去掉了旧锁文件在崩溃后需要人工删除的运维问题。
- **代码量**：删除 `snapshot.go`（约 460 行）及其专用测试（约 360 行），新增的只读读取约 170 行、锁约 100 行。

### 删除的实现

- `internal/store/sqlite/snapshot.go`：WAL 解析（文件头、盐值、校验和、提交帧）、页面映像、只读 VFS 接入、读后校验与重试。
- `internal/store/sqlite/snapshot_test.go`：针对上述解析器的测试（损坏帧、截断帧、未提交帧、WAL 旧代、读取期间变化、持续变化），以及只为它们服务的崩溃辅助进程。其中与解析器无关的两项（墓碑整理的崩溃恢复、兼容程序恢复崩溃数据库）已移到 `tombstone_test.go`、`inspect_test.go`。
- 旧的 `O_EXCL` 锁文件实现（`acquireMigrateLock`）。

### 与上一轮行为不同之处（未放宽验收）

- `MVP-STO-07` 按审查意见与本轮要求改写："只有 WAL、没有 SHM"等状态下，上一轮报告版本，本轮报告版本未知。审查已确认"无法安全读取时报告限制、保持版本未知"可以接受。
- 留下的锁文件不再是问题，`doctor` 改为报告"锁是否正被持有"（信息），不再因锁文件存在而报告失败。相应测试按新机制重写：用真实持锁进程验证超时（退出码 6、文件不变），并新增"持锁进程崩溃后不阻塞"。
- 在 Windows 上，测试使用 `%LOCALAPPDATA%\specmetry-test` 下的目录：本机的 `%TEMP%` 授予了其他本地账户（如编码代理的沙箱账户）修改权限，按 B1 的规则会被拒绝，这是正确行为。

### 产品决策：无法只读确认版本的数据库如何恢复（已确认，PD-23）

**决定**（2026-10-13，产品负责人）：采用方案 B 并调整：恢复是独立命令 `specmetry db recover`（不放进 `doctor`）；普通打开与 `doctor` 保持严格；恢复必须由用户明确授权，并实际检查 schema 版本、`min_binary_version` 和数据库一致性，不能仅凭用户声明；恢复前必须已有一致性备份，不得直接删除 `-wal` / `-shm`，不得绕过版本兼容限制。现在不开发，`3e0b0bc` 保持不变，先完成复审。依赖与排期评估见 [implementation-plan.md §8](implementation-plan.md#8-已决策待排期的功能)。

以下是提交决定时的问题与选项（保留作记录）。

**问题。** 被其他程序以普通方式关闭过的数据库（`-wal`、`-shm` 被删除），或只剩 `-wal` 没有 `-shm` 的数据库，本轮任何版本的 specmetry 都会拒绝打开（退出码 6），以保持 MVP-STO-02 的严格标准。specmetry 自己关闭的数据库不会进入这些状态。最小复现：用 `sqlite3` 命令行打开再退出 `specmetry.db`，随后任何 specmetry 命令都会被拒绝。可选方案：

| 方案 | 做法 | 代价 |
|---|---|---|
| A 保持现状（当前实现） | 一律拒绝，错误信息说明原因 | 用其他工具打开过数据库的用户无法继续使用，需要手动处理 |
| B 增加显式恢复命令 | 例如 `specmetry doctor --recover`：用户确认使用的是最新的 specmetry 后，由它以读写方式打开一次，恢复 WAL 文件 | 新增一个命令；用户确认错误时，旧程序可能合并新数据库的 WAL（逻辑内容不变） |
| C 对"缺少 `-wal` 与 `-shm`"放宽为"最终状态不变" | 此状态下数据库文件是完整的；用读写连接读取，若不兼容则关闭（SQLite 会删除它临时创建的 `-wal`、`-shm`），数据库字节和目录项最终不变 | 需要把 MVP-STO-02 对这一状态的要求改为"最终状态不变"；读取期间短暂出现辅助文件 |

### NON-BLOCKER

| 编号 | 状态 |
|---|---|
| N1 | **部分修复**：崩溃恢复和外键顺序已修复；checkpoint 返回 busy 时仍可能误报整理完成。最晚在 M2 的 `purge` 验收前解决（修复方向见 architecture §11.4） |
| N2 ~ N4 | 已关闭，本轮未改动 |

### 尚未验证与残余边界

- MVP-PLT-01：GitHub Actions 三平台 CI 尚未运行。
- macOS：全部测试（包括路径与 ACL 检查）只在 CI 中运行；本地只做了交叉编译、vet、lint。
- Linux 上的 race detector 只在 CI 中运行。CI 另加一步以 root 运行跨用户测试。
- Windows 上没有跨用户的真实测试（需要第二个账户）；Windows 的路径规则以 ACL 测试验证。
- 操作系统层面的竞态边界：路径检查保证的是"其他用户无法改变检查结果"；当前用户本人和 root / 管理员仍可以在检查之后改变目录，属于已声明的信任边界。网络文件系统、绑定挂载等的权限语义不在 MVP 范围内。
- 检查期间另一程序以普通方式关闭数据库（删除 `-wal`、`-shm`）时，只读连接可能留下空的辅助文件；读取后的检查会发现变化并报告版本未知，但无法撤销。这只涉及非 specmetry 的程序。

## 9. M0 第三次代码审查（codex-review-006）的处理

**审查结论**：NO-GO。B1 关闭，B4 保持关闭；B2 的未提交数据问题和 B3 的原有升级竞态已解决，但严格的文件不变验收分别被 R2、R1 两个反例阻断。审查确认可以保留原生 SQLite、路径检查、操作系统打开锁和写事务复核的整体实现。

### 产品决策（2026-10-13，产品负责人）

对 MVP-STO-02 和 MVP-STO-07 的保证边界做**有限调整**，不再为非核心的文件系统绝对不变性持续增加复杂实现。

| 决策 | 内容 | 理由 |
|---|---|---|
| 调整 MVP-STO-02 | 版本不兼容而拒绝时，仍禁止修改 schema、业务数据、事实、`seq` 和已有数据库 / WAL 的内容；**允许**创建权限正确、不含业务数据的协调文件 `.migrate.lock`。该例外只适用于协调文件，不适用于任意辅助文件，也不能用来放宽版本保护或数据完整性约束 | 协调文件是打开锁的载体，不含数据，权限私有；为在锁文件不存在时也做到零副作用而增加的复杂度，与它保护的价值不相称。数据与版本保护不变 |
| 调整 MVP-STO-07 | `doctor` 定位为非破坏性诊断：不执行 SQL 写入、迁移、恢复或 checkpoint；不把未提交数据当作已提交数据报告；无法保证一致性时报告 Unknown；没有外部竞争操作时保持文件级不变性检查；外部 SQLite 进程并发关闭导致的 WAL / SHM 辅助文件变化列为受限的已知行为；不得借此允许修改主数据库或业务数据；向用户说明它不提供取证级的绝对只读保证 | 驱动的 `mode=ro` 本身允许在目录可写时创建辅助文件，要完全阻止需要受限 VFS 或操作系统访问边界，属于存储层重构；而这一竞态只改变空的辅助文件，不影响数据和诊断结论的正确性（此时报告 Unknown） |
| PD-23 排期 | `db recover` 在 M2 实现，必须在 M4 自测前完成 | 见 implementation-plan §8.1 |

### 验收契约的前后差异

| 验收项 | 调整前 | 调整后 |
|---|---|---|
| MVP-STO-02 | 拒绝时"数据库、WAL、SHM 和目录项都不变"；并发升级时"旧程序拒绝且文件不变" | schema、业务数据、事实、`seq` 和已有数据库 / WAL / SHM 内容不变，不删除文件；**唯一**允许的副作用是创建空、私有、拒绝后不被持有的 `.migrate.lock`（目录修改时间随之变化）；协调文件已存在时拒绝不改变任何文件；并发升级时"旧程序在上述限度内拒绝"。其余（写事务复核、升级等待打开完成、关闭后数据库完整）不变 |
| MVP-STO-07 | "`doctor` 和打开数据库前的版本检查都只读"；"检查前后数据库、WAL、SHM 的内容和目录项完全一致"（无条件） | `doctor` 是非破坏性诊断（不写 SQL、不迁移、不恢复、不 checkpoint）；未提交数据永不报告、无法一致读取时报告未知（不变）；文件级完全不变只在**没有外部竞争操作时**要求；外部 SQLite 并发关闭时可能新建空的 `-wal` / `-shm`，列为受限的已知行为，此时必须报告未知且主数据库、业务数据、已有文件内容不变；说明不提供取证级保证。打开前的版本检查改由 MVP-STO-02 约束文件副作用 |

### 阻塞项的处理

| 编号 | 处理 | 代码 | 测试 |
|---|---|---|---|
| R1 / B3：拒绝不兼容数据库时新建锁文件 | 按调整后的 MVP-STO-02，这是唯一允许的副作用 | 现有代码满足新契约：锁文件由 `CreatePrivateFile` 以私有权限创建、为空，拒绝时释放；**未修改业务代码** | 新增 `TestIncompatibleRefusalMayOnlyCreateCoordinationFile`：审查中的状态（新目录、只有恢复的备份、没有锁文件），按新 schema 和 `min_binary_version` 两种原因各拒绝一次。允许的副作用：只新增空的 `.migrate.lock`（及目录修改时间），权限检查通过，锁未被持有；禁止的修改：数据库字节不变，schema、`min_binary_version`、`seq`、事实不变；锁文件已存在时再次拒绝，目录完全不变 |
| R2 / B2：外部关闭与只读连接之间的辅助文件创建 | 按调整后的 MVP-STO-07，列为受限的已知行为；打开前检查的同一行为按方案 A 纳入 MVP-STO-02（见下文） | 现有代码满足新契约：此时报告未知，不报告版本，不写 SQL、不 checkpoint；**未修改业务逻辑**。为让审查的跨进程调度反例在仓库内可重复，在只读读取中加入测试钩子 `readOnlyHook`（生产中为空）；`doctor --help` 改为说明它是非破坏性、非取证级的诊断 | 新增 `TestInspectDuringExternalCloseReportsUnknown`（保留审查反例）：判断文件状态之后，真实的独立子进程用普通 SQLite 连接读取并关闭数据库（确认 `-wal`、`-shm` 被删除）。断言：报告未知且版本为空；从外部关闭完成之后起，诊断只新增空的 `-wal` / `-shm`（Windows 两个都新增，Linux 只新增 `-wal`），没有改写或删除任何文件，权限检查通过；数据库中的 schema、`seq`、事实正确 |

### 打开前检查的同一竞态：产品决策（方案 A，已批准）

本轮复核时发现：**普通打开**前的版本检查使用同一只读读取，在相同的外部关闭竞态下同样会新建空的辅助文件。两个平台上的表现：

- Linux：打开拒绝（"读取期间文件变化，无法确认"），新增空的 `-wal`；
- Windows：SQLite 重新建出了被删除的 `-wal`、`-shm`，存在情况的比较看不出变化，打开按读到的已提交版本判断（不兼容则拒绝），新增空的 `-wal`、`-shm`。

提交决定时的选项：A 把 MVP-STO-07 的外部关闭例外有限扩展到打开前检查；B 改用受限 VFS 或操作系统访问边界（存储层改动）；C 维持现状，作为已知缺陷到 M2 处理。

**决定**（2026-10-13，产品负责人）：**方案 A**。边界：

1. 数据安全与版本保护不变：不修改 schema、业务数据、事实、`seq` 或已提交数据；不绕过 `schema_version`、`min_binary_version` 检查；不执行 checkpoint，不删除有效 WAL 事务，不覆盖已有有效数据；无法可靠判断兼容性时拒绝打开或报告未知。
2. 文件副作用严格限定：除私有的 `.migrate.lock` 外，只允许外部 SQLite 进程并发关闭期间，原生只读连接产生的必要 `-wal` / `-shm`；新增文件符合本地数据的权限与保密要求；不扩大为一般性的文件修改许可；区分 Specmetry 自身的副作用与外部进程造成的变化；没有外部竞争操作时，继续执行原有的文件不变性要求。

**是否仍有数据完整性、安全或错误版本接受问题**（按要求复核，未发现）：

- 只读连接以只读方式打开数据库文件；SQLite 只在能写数据库时才 checkpoint，只在 checkpoint 成功后才删除 WAL，所以它不能合并、删除或覆盖任何数据。
- Linux 上竞态被发现，版本未确认，打开拒绝。Windows 上竞态没被发现，但只读连接是在外部关闭之后、持有 SQLite 的锁读取的，读到的是已提交的版本；兼容时打开随后还会在读写连接上再检查一次版本。因此不会接受错误的版本。
- 新增的辅助文件由 SQLite 按数据库文件的权限创建（Unix 0600；Windows 继承受保护的 DACL），测试用权限检查确认。
- 可用性：外部关闭本身已使数据库进入 PD-23 的"缺少辅助文件"状态，普通打开会拒绝；Linux 上只读连接留下的空 `-wal` 使它变为"只有其一"，同样拒绝，需要 M2 的 `db recover`。Windows 上留下的空 `-wal`、`-shm` 恰好使数据库回到可读状态。两种情况都没有让数据变得更难恢复。

**验收契约的前后差异**（本次，在 §9 前文调整的基础上）：

| 验收项 | 调整前 | 调整后 |
|---|---|---|
| MVP-STO-02 | 唯一允许的副作用是创建 `.migrate.lock`；该例外不适用于 WAL、SHM 等其他辅助文件 | Specmetry 自身只允许两种副作用：① `.migrate.lock`；② 仅在外部 SQLite 进程于版本检查期间并发关闭数据库时，原生只读连接新建的空 `-wal` / `-shm`（权限合规）。增加"无法确定版本时一律拒绝，不判为兼容""不执行 checkpoint""外部进程造成的变化不算 Specmetry 的副作用"；没有外部竞争操作时文件不变的要求保留 |
| MVP-STO-07 | 外部关闭的已知行为只针对 `doctor`；打开前检查的副作用"以 MVP-STO-02 为准" | 已知行为写明与 MVP-STO-02 的例外 ② 相同，两项契约对同一行为的边界一致 |

**测试**（受控的多进程同步：外部进程在只读读取判断文件状态之后、打开连接之前同步运行完毕，并确认它删除了辅助文件；不依赖固定等待）：

- `TestOpenDuringExternalCloseOnlyAddsEmptyAuxiliaryFiles`：
  - 不兼容的数据库仍被拒绝（Windows 按已提交的新版本拒绝，Linux 按无法确认拒绝）；
  - 兼容的数据库不被误判（Windows 读到正确版本后正常打开；Linux 报告无法确认并拒绝）。
  - 两种情况下都确认：外部关闭之后，Specmetry 只新增空的 `-wal` / `-shm`，权限检查通过；schema、`min_binary_version`、`seq`、事实不变；锁已释放。
- `TestOpenRefusesWhenVersionIsUnknown`：没有竞态时，"被其他程序关闭"和"只有 `-wal`"两种版本无法确定的状态下，打开一律拒绝，文件完全不变。
- 无竞争场景的原有文件不变性测试全部保留并通过（如 `TestOlderBinaryLeavesCrashedNewerDatabaseUntouched`、`TestIncompatibleRefusalMayOnlyCreateCoordinationFile`、`TestInspectLeaves...`）。

代码：只修改了 `readonly.go`、`inspect.go`、`store.go` 中不再准确的注释（原文称"不修改任何文件"），没有改动逻辑。

### 非阻塞项

| 编号 | 状态 |
|---|---|
| N1 墓碑整理 | 保持**部分修复**，M2 的 `purge` 验收前必须解决（architecture §11.4） |
| N2 PD-23 排期 | 已同步：prd PD-23、implementation-plan §8.1、M2 的 WP2.10、§3 设计细化截止点 |
| N3 并发升级测试的启动握手 | 已改进：升级进程在等待打开锁时（测试钩子 `lockWaitHook`，生产中为空）写出信号文件，测试据此确认它确实被锁阻塞，不再依赖固定的 2 秒等待；用"锁总是成功"的变异确认测试会失败 |
| B1、B4 | 已关闭，本轮未改动 |

### 修改的代码

只有测试钩子和用户可见的说明，没有改动业务逻辑：`readonly.go` 的 `readOnlyHook`、`lock.go` 的 `lockWaitHook`（均在生产中为空），`doctor` 的命令帮助。

### 尚未验证

- MVP-PLT-01：GitHub Actions 三平台 CI 尚未运行。
- macOS 上的全部测试；Linux race detector；Windows 真实双用户测试。

## 10. M0 最终定向复审（codex-review-007）的处理

**审查结论**：NO-GO，只剩一项 BLOCKER（B2）；B1、B3、B4 已关闭，本轮未改动。审查接受产品负责人批准的方案 A 及其文件副作用边界。验收契约（MVP-STO-02、MVP-STO-07）本轮**没有修改**。

### B2：Windows 上连续外部关闭竞态时 `doctor` 漏报版本未知

**根因**：`doctor` 读取前后只比较数据库文件的存在情况和大小。在 Windows 上，第一次外部关闭竞态之后，只读连接留下空的 `-wal` 和空的 `-shm`。第二次同样的竞态中，外部进程删除这两个文件，只读连接又以相同的文件名、相同的大小（都是 0）重建它们，前后比较看不出变化，于是 `doctor` 报告了版本（`[ok] database schema v1`），违反 MVP-STO-07 中"此时必须报告版本未知"。Linux 上只读连接不重建 `-shm`，第一次之后只剩 `-wal`，属于"只有其一"的状态，本来就报告未知，因此不会进入这一重复状态。

**修复**（`internal/store/sqlite/readonly.go`，只影响 `doctor`）：`-shm` 为空时，`doctor` 不打开任何连接，直接报告版本未知。理由：

- 只读连接在外部关闭之后重建的 `-shm` 总是空的。读取前 `-shm` 不为空时，重建会表现为大小变化，原有比较能发现；读取前 `-shm` 为空时，重建无法发现，所以不读取。
- specmetry 自己的连接关闭后会留下 32 KB 的 `-shm`（持久 WAL），只有只读连接在竞态中才会留下空的 `-shm`。下一次正常打开会重建它，之后无竞争的 `doctor` 又能报告版本。
- 在打开连接之前就返回，不产生任何文件副作用；不读取数据，不可能报告未提交的数据。
- 打开前的版本检查不适用这条规则：审查已确认 Windows 上普通打开在这一状态下读到的是可靠的已提交版本，并在读写连接上再检查一次。因此不会引入新的不兼容版本接受问题，也不降低可用性。
- 没有重新引入自行解析 WAL，没有新增 VFS。

**测试**（真实 SQLite 子进程，判断文件状态之后同步完成外部关闭，不依赖固定等待；修复前在 Windows 上失败，修复后通过）：

| 测试 | 修复前 | 修复后 |
|---|---|---|
| `TestInspectRepeatedExternalCloseReportsUnknown`（公开 `Inspect`） | 第 1 轮未知；第 2 轮报告 schema v2，失败 | 两轮都报告未知。第 1 轮确实发生竞态，并在 Windows 上自然留下空的 `-wal`、`-shm`（审查反例的前置状态）；第 2 轮在读取前因为 `-shm` 为空而报告未知，目录完全不变。每轮确认权限合规，schema、`min_binary_version`、`seq`、事实不变；之后正常打开（Windows）重建 `-shm`，无竞争的检查重新报告 schema v2 且不改变文件；Linux 上正常打开按 PD-23 状态拒绝 |
| `TestDoctorRepeatedExternalCloseReportsUnknown`（公开 `doctor` 命令） | 第 2 轮输出 `[ok] database schema v1`，失败 | 两轮都输出 `[warn] database version unknown`，不出现版本；主数据库文件不变 |

`doctor` 命令测试位于存储包的外部测试包中，通过 `export_test.go` 使用已有的测试钩子；生产代码没有为此新增导出。

### NON-BLOCKER：兼容打开成功分支的断言

已补齐：`TestOpenDuringExternalCloseOnlyAddsEmptyAuxiliaryFiles` 的成功分支现在断言打开后主数据库文件未变，schema、`min_binary_version`、`seq`、事实内容正确，权限检查通过，打开锁已释放，后续写入取得 seq 3。

### N1

保持部分修复，最晚在 M2 的 `purge` 验收前解决；本轮未改动。

### 是否还有同类漏报

在审查描述和本轮测试覆盖的场景中没有。理论上，如果在读取期间有**另一个**程序恰好以与读取前完全相同的大小重新建出这两个文件，基于大小的比较仍然看不出来；这需要多个外部进程交错操作。即使发生，读取仍在 SQLite 的锁内进行，报告的是已提交的版本，不会读到未提交的数据，也不会接受错误的版本。

### 修改的文件

- `internal/store/sqlite/readonly.go`：`doctor` 的空 `-shm` 规则；参数 `sameSizes` 更名为 `diagnostic`，注释说明两种模式的差异。
- 测试：`externalclose_test.go`（两轮回归、成功分支断言）、`doctor_race_test.go`（新增）、`export_test.go`（新增，只在测试中编译）。
- 文档：architecture §11.5、本节、implementation-plan 状态、development.md。

### 尚未验证

- MVP-PLT-01：GitHub Actions 三平台 CI 尚未运行。
- macOS 上的全部测试；Linux race detector；Windows 真实双用户测试。

## 11. M0 代码审查结论（codex-review-008）

**审查结论**：**GO**，BLOCKER 为 0。审查提交 `d212524`。B2 正式关闭；B1、B3、B4 沿用此前的关闭结论。可以保留现有实现，不需要存储层重构。

审查同时确认：

- 第二轮的两项新测试证明的是"读取前保守拒绝"：`-shm` 为空时不进入读取，不能描述为"检测到了第二次删除和重建"。
- **已接受的风险**：两个真实外部 SQLite 进程先后关闭、重建辅助文件且大小相同时，基于大小的比较不能记录这类外部活动（审查已复现）。此时读取的仍是已提交的真实版本，诊断自身没有创建或修改文件，最低二进制版本的拒绝仍然有效。当前实现不提供完整捕获外部活动历史的保证。
- 稳定但 `-shm` 为空的有效数据库会保守报告版本未知；正常打开一次之后恢复。
- N1 仍在 M2 的 `purge` 验收前处理。

**代码审查 GO 不等于 M0 完整验收通过**。仍需完成：GitHub Actions 三平台 CI（MVP-PLT-01）、macOS 实际运行、Linux race detector；Windows 上因缺少符号链接权限而跳过的两项测试不能算作通过；Windows 真实双用户测试尚未执行。

## 12. M0 验收（GitHub Actions CI 与产品负责人决定）

**结论**：产品负责人于 2026-10-14 正式验收 M0。依据：codex-review-008 结论为 GO（§11）；GitHub Actions 第二轮 CI 8/8 通过；Windows、Linux、macOS 都在实际平台上运行测试，没有用交叉编译代替；M0 的九项退出条件（MVP-PLT-01、PRV-05、PRV-06、STO-01 ~ 04、STO-06、STO-07）全部满足。

### CI 运行记录

PR：<https://github.com/XKHoshizora/specmetry/pull/1>（`feat/m0-foundation` → `master`）。

| 轮次 | 提交 | 结果 | 说明 |
|---|---|---|---|
| [run 38425973237](https://github.com/XKHoshizora/specmetry/actions/runs/38425973237) | `f730ec3` | 6/8 | macOS、Windows 的 Go 测试在 `TestGitRepoCommitsAndWorktrees` 失败；其余各包（包括存储、权限、ACL）通过 |
| [run 38427328911](https://github.com/XKHoshizora/specmetry/actions/runs/38427328911) | `85ce2ea`（CI 测试的是 PR 合并提交 `acd42a4`，代码与 `85ce2ea` 相同） | **8/8** | — |

**第一轮失败的原因与修复**：失败在测试辅助代码，与存储层和业务代码无关。`NewGitRepo` 直接使用 `t.TempDir()`：它在 macOS 上是 `/var/...`（指向 `/private/var` 的符号链接），在 Windows runner 上含 8.3 短名（`RUNNER~1`）；git 输出规范路径，两者比较不相等。修复前在本地复现了两种情形（Windows 使用短名 TEMP，Linux 使用经过符号链接的 TMPDIR），修复后通过。

- `8ce753a`：`NewGitRepo` 用 `filepath.EvalSymlinks` 解析临时目录。只改测试辅助代码。
- `85ce2ea`：CI 的 Go 测试加 `-v`，日志列出每项测试，能看出各平台跳过了哪些测试。

这两个提交在 codex-review-008 的审查范围（截至 `d212524`）之外，都不改动生产代码。

### 各平台的实际执行情况（第二轮）

| 平台 | 执行与结果 | 跳过 |
|---|---|---|
| Linux（ubuntu-latest，linux/amd64） | 开启 race detector：95 项顶层测试通过；另一步以 `sudo` 运行 4 项真实跨用户测试，全部通过 | 主步骤中的 4 项跨用户测试（不是 root 时按设计跳过，在 `sudo` 步骤中运行） |
| Windows（windows-latest，windows/amd64） | 96 项通过，包括 DACL、上级目录、联接测试和两项符号链接测试 | 0 |
| macOS（macos-latest，darwin/arm64） | 96 项通过，包括 `TestDarwinCheckReportsACL` 和 Unix 权限、上级目录测试 | 0 |

三个平台共有的 88 项测试都实际运行；平台之间只差平台专属测试。其余任务全部通过：三平台带前端构建、`go test ./web/` 和冒烟测试（`version` 报告 schema v1、内嵌前端；`doctor` 之后数据目录没有被创建）；前端的格式、lint、类型检查、测试、构建；`go mod tidy`、`go vet`、Markdown 链接检查、golangci-lint v2.14.0。

§11 提到的"Windows 上因缺少符号链接权限而跳过的两项测试"：CI 的 Windows runner 有创建符号链接的权限，这两项在第二轮中实际运行并通过。

### 退出条件

| 验收项 | 结果 | 主要证据 |
|---|---|---|
| MVP-PLT-01 | 通过 | 第二轮 8/8 |
| MVP-PRV-05 | 通过 | Linux、macOS 的权限与上级目录测试；macOS ACL 检测；Linux 以 root 运行的真实跨 UID 攻击测试 |
| MVP-PRV-06 | 通过 | Windows 的 DACL、上级目录、联接、符号链接测试 |
| MVP-STO-01 ~ 04 | 通过 | 三平台；Linux 开启 race detector |
| MVP-STO-06 | 通过 | 三平台 |
| MVP-STO-07 | 通过 | 三平台，包括 codex-review-007 的两项回归测试 |

据此，§1 中 #6、#12 属于 M0 的验收项为**已验证**；这两项中属于其他里程碑的部分仍未开始。

### 证据的边界与后续事项（尚未完成，不属于 M0 退出条件）

- **Windows 真实双用户验证**：未执行，不标为已验证。MVP-PRV-06 没有要求跨用户攻击测试（真实跨 UID 测试只在 MVP-PRV-05 中要求在 Linux 上做）；Windows 上的测试由当前用户给其他账户授权来构造场景。
- **race detector**：只在 Linux 上运行。CI 日志不打印展开后的命令，依据是 CI 配置（Linux 上加 `-race`）和耗时（存储包在 Linux 上 38 秒，其他平台 7 ~ 10 秒）；日志中没有数据竞争报告。
- **macOS**：只覆盖 arm64，没有 Intel；没有跨用户测试（只在 Linux 上做）。
- **N1**：部分修复，最晚在 M2 的 `purge` 验收前完成。
- **PD-23 `specmetry db recover`**：未实现，在 M2（WP2.10）实现，最晚在 M4 自测前完成。在此之前，无法确认版本的数据库只能拒绝（退出码 6），`doctor` 报告版本未知。
- §11 中已接受的风险，以及方案 A 的文件副作用边界，保持不变。

## 13. M1 开工准备：产品决策与设计收尾（2026-10-14）

**背景**：M0 验收之后，产品负责人对截止在 M1 的待决策事项作出决定，并要求在不实现 M1 业务功能的前提下完成设计收尾。本节记录决定、对补充条件的核对，以及据此作出的修改。**本节的全部内容已于 2026-10-14 最终确认**（见本节末尾的"最终确认"）。

### 第一轮决定（之后的最终确认见本节末尾）

| 事项 | 决定 | 记录位置 |
|---|---|---|
| PD-03、PD-07、PD-12、PD-13、PD-14、PD-16、PD-19 | 已确认，采纳建议默认 | prd §9 |
| PD-04 | 已确认，方案 A：审核时接受 `update` 条目，审核者必须明确选择 `material` 或 `editorial`，否则拒绝；强制 `material` 的字段继续优先 | prd §9；data-model §7.2、V-R01 |
| PD-17 | 有条件批准：例外验收可以计入 `ready`；必须显示例外、理由、适用快照和证据等级；`evidence_grade = none` 不得描述为已通过自动化验证；全部豁免的范围不得为 `ready` | prd §9；data-model §6.8、§7.3 |
| PD-21 | 有条件批准：范围以需求为单位；豁免绑定 AC 的 revision，实质修订后重新决定；重新确认范围时展示已有豁免，由人确认继续适用 | prd §9；data-model §6.2、§7.1、V-T01 |
| TD-01、TD-05、TD-14、TD-18 | 原则批准 | architecture §16 |
| TD-20 | 待确认：先提供原规则和兼容性策略 | architecture §16（未修改） |
| PD-10 | 倾向 Apache-2.0，**尚未确认** | prd §9；[licensing.md](licensing.md) |

### 补充条件与现有设计的核对

结论：补充条件与现有的推导规则、schema 和验收项**没有实质冲突**。其中两项现有设计已经满足，其余三项通过**只做加法**的最小修改实现，不改变任何推导规则。

| 条件 | 现有设计 | 处理 |
|---|---|---|
| PD-17：显示例外、理由、适用快照和证据等级 | `status.v1` 中每个 AC 只有 `current_decision_id`、原因码和标记，没有理由、快照和等级 | `status.v1` 的 AC 增加 `decision` 摘要（`current_decision_id` 不为空时必有）。schema 强制：`exception` 和 `manual` 的等级只能是 `none`；`exception` 必须有理由；`accepted` 必须有快照；`normal` 的等级只能是 `observed` 或 `snapshot_isolated`。FR-ACC-05、MVP-ACC-05 同步 |
| PD-17：`none` 不得描述为已通过自动化验证 | FR-PRV-09 只约束 `observed` | 扩展 FR-PRV-09、MVP-PRV-09、data-model §6.8，以及 `common.v1` 中 `deliveryEvidenceGrade` 的说明；PD-22 增加补充说明 |
| PD-17：全部豁免的范围不得为 `ready` | **已满足**：data-model §6.8 第 5 条，以及阻塞差距 `ALL_WAIVED` | 不修改。这与 PD-18 的建议默认一致（PD-18 已在最终确认中一并批准） |
| PD-21：豁免绑定 AC 的 revision | **已满足**：当前决定只取当前 revision 下的（§5.4），所以实质修订后豁免不再适用；父需求修订先进入 `parent_revised`（§6.3 第 2 条先于第 3 条） | 只在 §6.2 补充说明 |
| PD-21：重新确认范围时展示并确认已有豁免 | 不存在 | `scope.v1` 增加可选字段 `acknowledged_waivers`（省略视为空集合）；新规则 V-T01 要求它恰好等于服务端计算的有效豁免集合，否则 `stale_precondition`。这沿用了 `pending_items_excluded` 的现有模式。新增读取模型 `target-scope.v1` 列出有效豁免及其理由。要停止某项豁免，先对该 AC 作出新的决定 |
| PD-04 方案 A | `review.v1` 的 `change_class` 可选，没有对应规则 | 新增服务端规则 V-R01，`review.v1` 的结构不变，只改说明；合法示例中缺少类别的更新条目已补上 |

### M1 设计收尾

| 项 | 结果 | 位置 |
|---|---|---|
| 1. TD-21 密钥识别规则集 | 14 条内置规则（正则加附加条件），每条 2 个命中、2 个不命中样本，用运行时拼接的方式书写；扫描范围是写入输入中的每个字符串值；命中时的错误格式（不回显命中文本）；`config.toml` 中 `[secrets]` 的追加方式（只能追加）。已在本地用一次性程序验证：每个命中样本都先命中自己的规则，每个不命中样本都不命中任何规则，另有 8 个常见文本没有误报 | architecture §10.8 |
| 2. M1 读取模型 schema | `source-list.v1`、`proposal-list.v1`、`proposal-show.v1`（三方对比 = `base` / `current` / `proposed`）、`baseline-show.v1`、`baseline-history.v1`、`question-list.v1`、`target-scope.v1` | `schemas/`；data-model §8 |
| 3. M1 Agent 调用接口的 JSON 输出 | `source-add.v1`、`proposal-receipt.v1`（不回显片段）；已有的 `init.v1`、`error.v1` 不变。人类专属写命令在 MVP 中只输出人类可读文本 | `schemas/`；prd §6；architecture §9.1 |
| 4. 通道识别的已知边界 | 规则、已核实的 Agent 环境变量（`CLAUDECODE`、`CLAUDE_CODE_ENTRYPOINT`；只检查是否存在，从不读取或保存值）、交互式确认（不提供跳过参数）、六种已知边界 | architecture §5.1；MVP-CH-01、CH-02 |
| 5. 阶段识别与来源发现 | 只读取路径和文件大小；扫描范围与排除规则；候选来源的排序和截断；三种阶段的判定表 | architecture §6.1.1；MVP-INIT-03、INIT-04 |
| 6. `excerpt_storage` | 新增 PD-24：默认 `hash_only`；`full` 只能由人开启，M2 起可用；`hash_only` 下原文不写入任何位置；沿用引用的重新校验规则；`project set` 加入人类专属操作和拦截清单 | prd FR-PRV-08、PD-24；architecture §10.7；data-model §4.1、§4.5 |
| 7. M1 夹具与依赖 | 新增 WP1.0：把 `fx-docs-only`（含录制的 `proposal.json`）、`fx-in-dev` 的文件与分支脚本、`fx-incomplete` 从 WP2.9 提前到 M1 | implementation-plan M1；acceptance-criteria §1.1 |
| WT-01 依赖 M2 的 `status` | 拆成两部分：M1 用 `init --json` 验证项目相同、worktree 被登记；M2 用 `status --json` 验证 `worktree_id` | MVP-WT-01 |
| SRC-08 依赖 N1 | M1 只有 `hash_only` 引用，没有原文需要擦除，因此只验证命令和通道；`full` 模式、擦除后的整理和字节级验证一起移到 M2（新增 WP2.11），与 N1 的修复、`purge` 共用整理步骤。没有降低擦除保证：在保证可以兑现之前，不保存任何原文 | MVP-SRC-08；implementation-plan WP2.11 |

### 文档与一致性修正

- 术语："验收标准"改为唯一的验收单位（PD-03）；"沿用"补充隔离级证据和 `inputs` 条件（PD-16）；PD-16 的"输入哈希"改为"输入对象 ID"。
- PD-22 增加补充说明：证据等级 `none` 的含义，以及不得描述为已通过自动化验证。
- `claims.v1` 的说明与 data-model 中 Claim 的统一表述保持一致：手动 AC 的 `done` 自述只会让它进入"等待人工查看"。
- G2 与 G2b：G2 列出截止在 M1 的全部 PD 和 TD；`decision` 契约在 G2b 冻结，G2 只确认它的语义前提（PD-03、PD-16、PD-17）。
- data-model §7.3 中断开的表格已修复，`rejected` 和 `waived` 两行回到表中。
- PD-14 的覆盖：AGENTS.md 的人类专属操作清单、不变量 #1、E2E-7、MVP-ACC-03 的拦截样本、MVP-BL-17（增加驳回）。
- MVP-BL-12 增加：修改 `verification` 时强制按 `material` 记录。
- data-model §10 更新各项决策的状态。

### Schema 变更

- **修改**：
  - `status.v1`：AC 增加 `decision`；
  - `scope.v1`：增加 `acknowledged_waivers`（最终确认时改为必填，并增加 `seen_scope_confirmation_id`）；
  - `review.v1`、`claims.v1`、`common.v1`：只改说明。
- **新增**：9 个 schema、10 个合法示例（包括 `status.exception.json`），以及反例 SCH-65 ~ SCH-92。`internal/contract` 的测试只更新了 schema 清单和示例前缀映射。
- **与 TD-20 原规则的关系**：
  - 全部是新增：新的 schema、新的可选字段，以及只出现在输出中、在条件满足时才必有的 `decision`。
  - 没有删除或改变已有字段的含义。
  - 这些 schema 都还没有实现，也没有发布过的使用方。

### 最终确认（2026-10-14）

产品负责人正式批准 PD-17、PD-18、PD-21、PD-24，以及 TD-01、TD-05、TD-14、TD-18、TD-20（调整后）、TD-21；此前已批准的 PD 和 TD 保持不变。G2 因此满足。最终批准附带的要求，以及据此所做的修改：

| 要求 | 修改 |
|---|---|
| PD-17：采用 `decision` 摘要；例外验收显示理由、适用快照和证据等级；`none` 不得表述为自动化验证通过；**不能因此删除或隐藏已有的实际测试证据**；保留全部豁免不得 `ready` | `status.v1` 中 `snapshot_run` 的说明写明：无论作出什么决定，执行都照常输出；`decisionSummary` 说明决定之后的执行不改变它的等级。data-model §6.8 写明：例外验收之后的执行照常显示，不会把这次例外验收升级为 `observed`。MVP-ACC-05 增加对应的断言 |
| PD-21：`acknowledged_waivers` 必须覆盖范围内全部有效豁免；写入时校验用户读取的基准、范围和相关 AC 的 revision；已失效的历史豁免不进入确认列表 | `scope.v1`：`acknowledged_waivers` 改为**必填**（空数组表示没有豁免）；新增必填的 `seen_scope_confirmation_id` 和规则 V-T02，与 `seen_decision_id` 的做法相同，用来防止旧页面覆盖刚写入的范围确认。`target-scope.v1` 给出需要回传的值。AC 的 revision 由两处保证：任何 revision 变化都产生新的基准版本；每条豁免都绑定 AC 的 revision。`scope_confirmations` 保存确认的豁免和读取到的范围确认。新增 MVP-TGT-11 和不变量 #11，修改 MVP-TGT-10 |
| TD-20：调整后的六条版本策略 | architecture §9.3；TD-20 已确认。本轮的 schema 修改发生在首次正式发布之前，属于完善 v1，已同步 schema、示例和测试 |
| TD-21：批准 14 条规则；命中即拒绝持久化，不回显；规则集只能降低风险 | architecture §10.8 的"已知限制"、FR-PRV-07 写明这一点；TD-21 已确认 |
| PD-24：默认 `hash_only`；M1 不提供原文持久化；`full`、人工授权和删除保证在 M2 | 与 §13 中的设计一致；PD-24 已确认 |

新增的 schema 反例为 SCH-93 ~ SCH-97，覆盖缺少 `acknowledged_waivers` 或 `seen_scope_confirmation_id` 的情况。

**仍未确认、但不阻塞 M1 的事项**：

- PD-10（许可证，接受外部贡献之前）：确认项见 [licensing.md §4](licensing.md#4-pd-10-的确认项)。
- PD-02（M2）。
- TD-17（M2）、TD-24（M7）。
- 其他 Agent 的环境变量与约束（M4 开工前）。

## 14. M1 设计审查（codex-review-009）的处理

**审查结论**：**GO**，BLOCKER 为 0。审查目标 `e834e83`（PR #2），基准 `65e1d0a`。PR #2 已合并（合并提交 `08eb951`）。产品负责人接受全部三项 NON-BLOCKER，并授权开始 M1 开发，从 WP1.3 开始。

| # | 问题 | 处理 | 位置 |
|---|---|---|---|
| 1 | MVP-BL-26（片段累计总量上限）仍在 M1，而该上限只计算 `full` 模式保存的原文，M1 只有 `hash_only` | BL-26 移到 M2，前置条件写明 `excerpt_storage = full`；M1 继续验证单个片段的长度（BL-04）和原文不落盘（PRV-08）。V-P11 写明只计算已保存的原文；WP2.11 纳入这项上限 | acceptance-criteria MVP-BL-26；data-model V-P11；implementation-plan WP2.11 |
| 2 | 部分输出 schema 缺少字段之间的语义约束（`proposal-receipt`、`question-list`、`proposal-show`、`status.v1` 中拒收决定的快照） | 记为实现相应输出时必须完成的验收任务：补充 schema 条件约束和反例；schema 无法表达的关系由输出一致性测试验证。不提前实现 | implementation-plan M1"输出契约的语义约束"；MVP-BL-14、BL-17、BL-28、ACC-07 |
| 3 | 密钥检测的边界验收不够具体 | 纳入 WP1.3 的自动化测试：环境变量引用与真实密钥在同一字段；多个候选匹配中前一个豁免、后一个不豁免；追加规则编译失败时不回显含 canary 的 pattern。architecture §10.8 写明豁免只作用于对应的匹配，配置错误不回显 pattern 或配置原文 | MVP-PRV-07；architecture §10.8 |

**已接受的风险**（审查列出，与已批准的边界一致）：例外验收可以计入 `ready`（PD-17）；密钥检测是启发式保护，短的纯字母密码、命令替换中的普通密码字面量可能漏报（TD-21）；通道识别不是身份认证（architecture §5.1）。

审查原文中链接到本机路径的地方，入库时改为仓库内的相对链接，正文未作修改。

## 15. WP1.3 代码审查（codex-review-010）的处理

**审查结论**：NO-GO，1 项 BLOCKER、1 项 NON-BLOCKER。审查目标 `563fe8f`（PR #3）。审查未发现 M0 回归，错误输出没有回显敏感内容。

### B1：`$(` 前缀豁免了整个密码捕获值

**根因**：环境变量引用的判断包含一个分支，接受任何以 `$(` 开头的值。`url_credentials` 和 `generic_assignment` 把这个判断用在整个捕获的密码上，于是 `password=$(true)s3cr3tPass9`、`postgres://app:$(true)s3cr3tPass9@db/app`、`password=$(s3cr3tPass9` 都被放行。这超出了 TD-21 接受的"命令替换内部的普通密码可能漏报"的范围。

**修复**（最小修改，不引入 shell 解析器）：

- 环境变量引用只保留四种完整写法 `$NAME`、`${NAME}`、`%NAME%`、`$env:NAME`，被判断的值必须整体匹配，不再按前缀判断。命令替换和相邻的字面量照常扫描。
- architecture §10.8 同步收窄；测试中把 `$(…)` 列为环境变量引用的旧预期已删除。

**测试**（全部经过公开的 `New → JSON → Reject` 路径，先确认在修复前失败）：

| 测试 | 修复前 | 修复后 |
|---|---|---|
| Codex 的三个反例 | 三个"应拒绝"的断言失败，拒绝结果为 nil；正向对照通过 | 全部拒绝，错误只含字段位置和规则编号，不含密码 |
| 四种完整引用（赋值和 URL 密码两种位置） | 通过 | 通过，仍然豁免 |
| `${NAME}` 后接明文密码 | 拒绝 | 拒绝 |
| 环境变量引用与密钥在同一字段；多个候选匹配 | 拒绝 | 拒绝 |

### N1：无条件规则收集全部匹配

**修复**：没有附加条件的规则改用 `MatchString`，有条件的规则仍然逐一检查每处匹配。检测结果和错误输出不变。新增测试：用一条匹配任意字符的追加规则扫描 4000 个字符，修复前每次调用分配 4017 次，修复后不超过限值。

**接受的风险**沿用审查结论：检测是启发式的；写入集成在 WP1.5 起验证。

## 16. WP1.3 最终复审（codex-review-011）与验收

**审查结论**：**GO**，BLOCKER 为 0，没有新增 NON-BLOCKER。审查目标 `00eb479`。B1 关闭：环境变量引用只认四种完整写法，审查方在父提交上确认三个反例的回归测试失败、在目标提交上通过；N1 已合理解决。目标提交的三平台 CI 8/8 通过。

**验收**：产品负责人于 2026-10-14 验收 WP1.3，PR #3 已合并（合并提交 `cee87b4`）。

**WP1.3 的验收范围**：MVP-PRV-07 在模块层面已验证，包括 14 条规则及其样本、环境变量引用的边界、不回显命中文本的拒绝错误、追加规则、无效配置的退出码 6 和三个边界反例。

**仍未完成、随后续工作包验证**：

- MVP-PRV-07 的 CLI 写入层：写入被拒绝时退出码 2、不写入任何数据、输出和日志中没有命中的文本，并覆盖全部写入入口。这部分从 WP1.5 开始，随各写入命令验证。
- MVP-BL-21、MVP-BL-02 在 WP1.5 验证，MVP-CLM-03 在 M2 验证。

## 17. WP1.0 代码审查（codex-review-012）的处理

**审查结论**：NO-GO，1 项 BLOCKER、2 项 NON-BLOCKER。审查目标 `0e58c62`（PR #4）。审查确认三个夹具的内容符合 WP1.0，三平台 CI 的固定提交 ID 一致，录制的提案与需求说明一致。产品负责人决定一次修复全部三项，不扩大工作包范围。

### B1：继承的 git 环境变量能把夹具操作重定向到其他仓库

**根因**：`NewGitRepo` 继承整个 `os.Environ()`，只覆盖了部分变量。`GIT_DIR`、`GIT_WORK_TREE` 等定位变量不受 `cmd.Dir` 影响；`GIT_DEFAULT_HASH`、`GIT_CONFIG_*`、`GIT_TEMPLATE_DIR` 等也能改变提交 ID 或执行钩子。

**修复**（`internal/testutil/git.go`）：

- 测试用 git 子进程不继承任何 `GIT_*` 变量，也不继承 `HOME`、`XDG_CONFIG_HOME`（变量名不区分大小写）。
- 只设置受控的变量：关闭系统配置；空的全局配置；隔离的 HOME 和 XDG 目录；空的模板目录（不复制钩子等文件）；`GIT_CEILING_DIRECTORIES`（不会向上找到别的仓库）；固定身份；禁止交互提示。提交日期仍由 `CommitAt` 单独传入。
- `git init` 固定 `--object-format=sha1`。
- 第一次修改仓库配置之前，以及每次提交之前，确认 git 解析出的仓库目录就是本仓库的 `.git`，否则测试立即终止。

**回归测试**（全部经过公开的 `BuildFixture` 路径；另建一个临时的"外部"仓库，注入的变量都指向它）：

| 注入 | 修复前（Windows 与 Linux 相同） | 修复后 |
|---|---|---|
| `GIT_DIR` + `GIT_WORK_TREE` | 外部仓库被修改，夹具没有自己的 `.git` | 外部仓库完全不变（快照比对 HEAD、索引、配置、对象和文件）；夹具建在自己的目录中；提交 ID 等于清单中的值 |
| `GIT_COMMON_DIR` | 外部仓库被修改 | 同上 |
| `GIT_INDEX_FILE` | 外部仓库被修改 | 同上 |
| `GIT_OBJECT_DIRECTORY` | 外部仓库被修改 | 同上 |
| `GIT_DEFAULT_HASH=sha256` | 生成 64 位的提交 ID | 提交 ID 等于清单中的值 |
| `GIT_CONFIG_COUNT`（`core.hooksPath`） | 注入的钩子被执行 | 钩子没有执行 |
| `GIT_CONFIG_PARAMETERS`（`core.hooksPath`） | 注入的钩子被执行 | 钩子没有执行 |
| `GIT_TEMPLATE_DIR`（模板中的钩子） | 注入的钩子被执行 | 钩子没有执行 |
| 以上全部，加上 `GIT_AUTHOR_DATE`、`GIT_COMMITTER_NAME` | 外部仓库被修改 | 全部满足 |

另有一个直接测试：把测试仓库的 `.git` 换成指向外部仓库的 gitfile，提交在写入之前被拦下，外部仓库不变。去掉这项检查后测试失败。

### N1：录制提案的检查不够严格

**修复**：检查改为对照一个独立的 Oracle。Oracle 直接根据需求说明写下预期，不从提案推导：

- 1 个目标、8 条需求、9 条 AC，每个需求小节应有的 AC 数（R8 为 0）；
- 4 个问题：R3、R7 的歧义问题必须引用歧义原文，R8 的验收方式问题，以及"待确认"小节中唯一的项目级阻塞问题。

每个条目引用的片段在需求说明中的位置，决定它属于哪个小节；AC 必须挂在同一小节的需求上，问题的 `about` 必须指向对应小节的需求。审查提出的三种变异作为自动化测试：删除 `c3`、删除三个非阻塞问题、把 `c9` 挂到 R8。修复前三种都检测不到，修复后都被检出，并且各自是因为正确的原因。

### N2：`fx-in-dev` 的需求说明注记为空

**修复**：注记与需求说明一致（R8 没有可判定的验收描述，R3、R7 各有一处歧义）。新增检查：需求说明字节相同的夹具，清单中的注记也必须相同（修复前失败）。清单不属于被构造的仓库，三个夹具的固定提交 ID 不变。

审查原文按原样入库（它本来就是仓库内的相对链接）。

## 18. WP1.0 最终复审（codex-review-013）与验收

**审查结论**：**GO**，BLOCKER 为 0，没有新增 NON-BLOCKER。审查目标 `a206a3a`。B1（git 环境隔离）、N1（提案 Oracle）、N2（清单注记一致性）均已关闭：审查方在外部临时仓库上独立复现了原始攻击，包括 Windows 混合大小写的变量名，确认修复后外部仓库的文件和 `.git` 内容不变。目标提交的三平台 CI 8/8 通过，三个夹具的固定提交 ID 一致。

**验收**：产品负责人于 2026-10-14 验收 WP1.0，PR #4 已合并（合并提交 `84e6e70`）。

**已接受的风险**：提案 Oracle 只针对当前固定的需求说明，不承担通用的自然语言需求判定。

**仍未完成、随后续工作包补齐**：

- `fx-in-dev` 中故意失败的测试和录制的 Claim、其余夹具和期望结果（WP2.9）。
- 使用这些夹具的验收项随对应工作包验证：MVP-INIT-03、INIT-04 和 WT-01 的 M1 部分在 WP1.1，MVP-BL-01 在 WP1.5。

## 19. WP1.2 代码审查（codex-review-014）与验收

**审查前的修正**：CI 首轮通过之后、送审之前，产品负责人指出 `Guard` 的两处问题，以独立提交 `6d6d1ab` 修正（architecture §5.1 同步更新）：

- 输入在确认内容之后直接结束（没有回车）时，原实现会放行。现在只有以回车结束的完整一行、且内容匹配才继续；输入结束一律视为取消，退出码 3。
- 原实现忽略了提示输出的错误。现在操作名称、摘要和确认提示一次写出；写入出错或只写入一部分时，不读取输入，返回退出码 6（`environment`）。

**审查结论**：**GO**，BLOCKER 为 0，1 项 NON-BLOCKER。审查目标 `6d6d1ab`。审查方用独立探针确认：通道识别符合 §5.1；Agent 变量的值不出现在结果、错误和 JSON 错误响应中；预先放入 context 的 `human_cli` 会被实际识别结果覆盖；确认、取消、输入结束、输出失败和读取错误的退出码都符合规定。目标提交的三平台 CI 8/8 通过。

**验收**：产品负责人于 2026-10-14 验收 WP1.2，PR #5 已合并（合并提交 `4c5a038`）。

**N1：提示输出失败的测试有六次没有触发故障**。`Guard` 只写一次，`failAt=2..4` 的六次运行根本没有触发故障，而是被 `continue` 跳过了断言，不能算作已验证的失败分支。实际有效的只有第一次写入出错和短写两次。这不影响实现，按产品负责人的安排在下一个开发分支（`feat/m1-init`）中以单独提交修正测试，不改动 `Guard`。

**N1 的修正**：删除 `failAt` 循环，改为四个明确的子测试：写入出错、写出全部内容但同时报告错误、短写、什么也没写入。测试用的 Writer 每次写入都失败，所以故障必然发生；每个子测试都断言 Writer 确实被调用、退出码为 6、确认输入一次也没有读取。检验：换回送审前的 `Guard`（`2a9bd59`），四个子测试全部失败；把 `Guard` 改成不检查写入字节数，短写和什么也没写入两个子测试失败。

**已接受的风险**（与 §5.1 一致）：

- 通道识别是防误操作措施，不能证明操作者身份。
- 目前只核实了 Claude Code 的两个变量，其他 Agent 在 M4 核实后加入。
- Windows 上没有控制台的终端按设计识别为 `agent_cli`。

**仍未完成、随后续工作包验证**：MVP-CH-01、CH-02 对真实人类专属命令的验证：`source ack`、`source remove` 在 WP1.4，`review`、`propose --apply` 在 WP1.6，`target set`、`scope confirm` 在 WP1.8。WP1.2 只用测试中的替身命令验证了拦截与确认组件和 CLI 的共同路径。

审查原文按原样入库（它本来就是仓库内的相对链接）。

## 20. WP1.1 代码审查（codex-review-015）的处理

**审查结论**：NO-GO，3 项 BLOCKER、2 项 NON-BLOCKER。审查目标 `86ac50d`（PR #6）。审查确认项目身份、Worktree 归属、幂等、并发登记和失败恢复正常；问题集中在安全边界。产品负责人决定一次修复全部五项，不改变已批准的架构，不扩大工作包范围。

### B1：git 读取命令能执行外部程序并修改 Ref

**根因**：`gitx` 只过滤了仓库定位变量。配置注入变量（`GIT_CONFIG_COUNT`、`GIT_CONFIG_PARAMETERS`、`GIT_CONFIG_GLOBAL`）、跟踪变量、用户的全局配置和仓库的本地配置都照常生效；而 `ls-files` 读取索引时会运行 `core.fsmonitor` 指定的程序。`GIT_OPTIONAL_LOCKS=0` 挡不住这些。

**修复**（`internal/gitx`，architecture §6.1.2）：

- 不传递任何继承的 `GIT_*` 变量（按前缀，不区分大小写），而不是只列出已知的几个。
- 只重新设置受控变量：系统和全局配置指向空文件（`GIT_CONFIG_NOSYSTEM`、`GIT_CONFIG_SYSTEM`、`GIT_CONFIG_GLOBAL`），`GIT_OPTIONAL_LOCKS=0`，`GIT_TERMINAL_PROMPT=0`。用户的配置文件不读取，也不修改。
- 仓库本地配置中唯一会让读取命令执行程序的 `core.fsmonitor`，在命令行上用空值关闭（2.32 以来的各个版本都把空值当作关闭）；另加 `--no-pager`、`--no-replace-objects`。
- 要求 git 2.32 或更高（第一个支持 `GIT_CONFIG_GLOBAL` / `GIT_CONFIG_SYSTEM` 的版本；Ubuntu 22.04 为 2.34）；更旧时返回退出码 6，而不是在无法隔离的情况下继续运行。

**回归测试**（全部经过公开的 `init --json --no-exclude`；每种注入先用普通 git 确认确实生效，避免测试本身无效）：

| 注入 | 修复前（Windows） | 修复后（Windows、Linux） |
|---|---|---|
| 仓库本地配置 `core.fsmonitor` | 外部文件被写入，新 Ref 被创建，`.git` 改变 | 都不发生 |
| `GIT_CONFIG_COUNT` 注入 `core.fsmonitor` | 同上 | 同上 |
| `GIT_CONFIG_PARAMETERS` 注入 `core.fsmonitor` | 同上 | 同上 |
| 全局配置中的 `core.fsmonitor` 和 `trace2.normalTarget` | 同上，并写入 trace2 文件 | 同上 |
| `GIT_CONFIG_GLOBAL` 指向的配置 | 同上 | 同上 |
| `GIT_TRACE`、`GIT_TRACE2`、`GIT_TRACE2_EVENT`、`GIT_TRACE_SETUP`、`GIT_TRACE_PERFORMANCE` | 五个跟踪文件都被写入 | 都不写入 |
| 仓库通过 `core.hooksPath` 配置的 Hook | 未执行（读取命令不触发 Hook） | 未执行（回归防护） |

每种情况都比较 `.git` 下的全部文件及内容，以及全部 Ref。

**保证边界**：隔离针对 Specmetry 自己运行的 git 命令。仓库本地配置仍然生效（除 `core.fsmonitor` 外），所以仍依赖"这些读取命令本身不执行程序、不触发 Hook"这一 git 行为；已用 Hook 和 FSMonitor 反例验证。`doctor` 的 `git --version` 检查不属于本轮范围。

### B2：默认交付目标的 Ref 绕过密钥扫描

**根因**：主 worktree 的分支在扫描之后才读取，随后直接写入 `delivery_targets.ref` 并进入输出。

**修复**：读取分支后立即按 WP1.3 的规则扫描，命中时返回退出码 2，错误只给出字段（`default delivery target ref`）和规则编号。扫描在打开数据库之前完成，不论这次是否会新建目标（打开数据库之前无法知道）。

**回归测试**：分支名含合成的 GitHub token（运行时拼接）→ 退出码 2；`error.v1` 中只有字段与 `secret_detected: github_token`；stdout、stderr 中都没有该 token；数据目录（数据库与日志）未创建，没有 marker，没有分配 `seq`。修复前：退出码 0，token 写入数据库和输出。

### B3：git 错误原文泄露

**根因**：git stderr 的第一行和运行目录被拼进错误信息，并进入 `error.v1`；`Locate` 在路径扫描之前运行，所以错误中的目录也未经扫描。

**修复**：git 的 stderr 直接丢弃，不保存。错误只包含 git 子命令名和退出状态（退出码 6）；查找仓库时的错误也不包含目录，只描述错误类别。

**回归测试**（公开 CLI，canary）：

- 环境变量注入无效的 `core.bare`：修复前退出码 6，`error.v1` 中出现 canary；修复后该变量不再传给 git（B1），`init` 成功，输出、数据目录和 marker 中都没有 canary。
- 仓库本地配置中写入无效的 `core.bare`（不依赖 B1）：修复前 `error.v1` 中出现 canary 与仓库路径；修复后退出码 6，`--json` 与文本输出中都没有 canary 和仓库路径，数据目录未创建。
- 单元测试：`rev-parse` 失败时，错误中没有 git 的信息和目录。

### N1：marker 冲突时的文件副作用

**修复**：数据目录中还没有数据库、却有 marker 时，在创建数据库之前拒绝。错误改为"No project was registered and the marker was left as it is"，与实际一致。数据库已登记项目但 marker 缺失时，仍然补写 marker。

**回归测试**：没有数据目录时，拒绝后数据目录仍不存在；有数据目录（只有 `config.toml`）时，拒绝前后数据目录的文件与内容完全相同；marker 逐字节不变。修复前（在审查目标上复现）：退出码 6，数据库已被创建，错误却声称 "Nothing was changed"。原有冲突测试改为在任何查询之前检查数据目录。

### N2：非 git 扫描不响应取消

**修复**：文件列举在开始前、遍历每个条目时，以及 git 列举结果的处理循环中检查取消；取消时返回退出码 6，不返回部分结果，不登记任何内容。没有引入扫描数量限制。

**回归测试**：调用前已取消 → 立即返回错误；遍历中途取消（第 10 次检查时）→ 立即停止，不再检查；git 列举在已取消时返回错误；公开 CLI 在已取消的 context 下，非 git 和 git 项目都返回错误，不创建数据目录和 marker。修复前：已取消时仍返回完整列表，公开路径登记了项目。

审查原文按原样入库（它本来就是仓库内的相对链接）。

## 21. WP1.1 定向复审（codex-review-016）的处理

**审查结论**：NO-GO，只剩 1 项 BLOCKER。审查目标 `66d5948`。B2、B3、N1、N2 经独立复审关闭（N1、N2 修复有效），本轮不再改动。目标提交的三平台 CI 8/8 通过。

### B1（剩余部分）：旧版 git 的版本探测会泄露环境变量

**根因**：`checkVersion` 先运行 `git --version`，再判断是否达到 2.32。探测本身在受控环境中运行，但低于 2.32 的 git 不认识 `GIT_CONFIG_GLOBAL`，仍读取 `HOME` 中的全局配置；而 trace2 在处理命令之前就按全局配置的 `trace2.*Target` 初始化，于是在 Specmetry 以退出码 6 拒绝之前，跟踪文件已经写出，其中包含 `trace2.envVars` 指定的环境变量值。审查方用真实的 git 2.31.1 复现。

**修复**（最小修改）：在每个 git 进程都使用的受控环境中加入 `GIT_TRACE2=0`、`GIT_TRACE2_EVENT=0`、`GIT_TRACE2_PERF=0`。第一个 git 进程（版本探测）启动时就已生效。继承的 `GIT_*` 变量照旧全部过滤；2.32 的最低版本要求不变；旧版 git 仍以退出码 6 拒绝。gitx 没有其他改动。

**是否还有其他探测阶段的副作用**：用从官方源码构建的 git 2.31.1 和 2.32.0 验证。全局配置中同时设置 trace2 的三个目标、`trace2.envVars`、`trace2.configParams`、`core.fsmonitor`、`core.pager`、`core.hooksPath`（含 Hook），以及经 `include.path` 引入的 `core.sshCommand`，各自指向哨兵文件。不加控制时，`git --version` 只写出三个 trace2 文件，其中含有环境变量的值；其余设置都不生效。使用受控环境时，什么都不写，HOME 和当前目录中也没有新文件。因此只需关闭 trace2。

**回归测试**：

| 测试 | 修复前 | 修复后 |
|---|---|---|
| `TestFilterEnv`（受控环境的完整列表） | 缺少三个 trace2 变量 | 通过 |
| `TestVersionProbeRunsInTheControlledEnvironment`（gitx；**替身**：报告 2.31.1 并记录调用的假 git） | 探测时没有 trace2 变量 | 只运行一次 `--version`，环境中的 `GIT_*` 恰好是受控集合，注入的 `GIT_TRACE2*` 被替换；随后以退出码 6 拒绝 |
| `TestInitRefusesOldGitBeforeAnythingElse`（公开 `init --json --no-exclude`，新进程，**替身**） | 同上 | 退出码 6，提示需要 2.32；版本探测是唯一运行的 git；没有数据目录和 marker |
| `TestInitRefusesRealOldGit`（**真实 git 2.31.1**，由 `SPECMETRY_TEST_GIT_2_31` 指定，CI 中跳过） | 写出 3 个 trace 文件 | 退出码 6；没有 trace 文件；stdout、stderr 中没有 canary；没有数据目录和 marker；`.git` 下的文件与 Ref 不变 |

真实版本测试在 WSL（Ubuntu 24.04）中运行，git 2.31.1 和 2.32.0 从 kernel.org 的官方源码包构建，并对照同一目录的 `sha256sums.asc` 核对了哈希（未验证 GPG 签名）。测试先用这个 git 2.31.1 证明全局配置确实会写出含 canary 的跟踪文件，避免测试本身无效。CI 中没有旧版 git，只运行替身测试。替身测试不能代替真实旧版验证，报告中两者分开列出。

**2.32 及更新版本仍正常工作**：gitx、project、cli、testutil 四个包的全部测试，分别在 git 2.32.0（自行构建）、2.43.0（WSL）和 2.56.0（Windows）下通过；其中包括 codex-review-015 的全部反例。

### NON-BLOCKER

1. **`doctor` 的跟踪写入**：未修改，作为独立的安全修复任务保留。它违反 `doctor` "非破坏性诊断" 的承诺，不是已接受的产品限制，已列入 implementation-plan M6 "发布前必须关闭的遗留安全问题"。
2. **两处安全测试可以收紧**：按产品负责人的要求，本轮只修复剩余的 BLOCKER，保留现有回归测试不变，留作后续改进：
   - Trace 用例的前置检查只要求至少一个变量生效（审查方已逐项验证五个变量都生效）；
   - `TestInitRefusalsRegisterNothing` 中，Token 位于 worktree 路径，会先触发路径扫描，不能单独证明 Ref 扫描（审查方已用安全路径和敏感主分支另行验证）。

### Linux SQLite 偶发失败

推送 `ef8cad3` 之后，CI 的 Linux 测试（带 race detector）失败了一次，这次留下了测试名：M0 的 `TestOpenLockTimeout`。另一个进程持有打开锁时，`Open` 本应等待超时，却直接成功了。

**根因（测试辅助进程的缺陷，不在产品代码中）**：

- `lockHolderHelper` 调用 `acquireOpenLock` 后丢弃了返回的锁对象，锁文件的 `*os.File` 因此不再被引用。Go 的 finalizer 会在下一次垃圾回收时关闭这个文件，flock（Windows 上是 LockFileEx）随之释放。
- 所以只有在测试等待期间恰好发生垃圾回收时，这个测试才会失败；race detector 增加了内存分配，让这种情况更容易出现。
- 产品代码不受影响：`Open` 持有锁时一直以 `defer lock.release()` 保持引用，`acquireOpenLock` 的其他调用方只有这个辅助进程。

**证实**：在辅助进程的等待循环中强制垃圾回收，同时仍然丢弃锁。Windows 上 3/3 失败，Linux 上 3/3 失败，锁都在测试检查之前就被释放了。

**修复**：辅助进程保留锁的引用（`defer lock.release()`，放在模拟崩溃的分支之后）。等待循环中保留强制垃圾回收：以后如果再丢掉引用，测试会立刻失败，而不是偶发失败。修复后在 Windows 和 Linux 上都通过，Linux 上整个存储包又连续运行了 8 次，全部通过。

**仍然不能确定的部分**：本会话本地那一次失败（37 次中的 1 次）丢失了测试名，它与这个机制相符，但不能断定就是同一个测试。审查方保存的 80 次运行日志中没有复现。

审查原文按原样入库（它本来就是仓库内的相对链接）。

## 22. WP1.1 最终复审（codex-review-017）与验收

**审查结论**：**GO**，BLOCKER 为 0。审查目标 `7bb4b84`。

- **B1 关闭**：受控环境在第一个 git 进程（版本探测）启动之前就已生效。审查方用真实的 git 2.31.1 独立复验：只运行了 `--version`，退出码 6；没有跟踪文件、数据库或 marker；`.git` 的文件与引用不变。仓库中的真实旧版测试 3/3 通过，与替身测试分开验证。
- **`TestOpenLockTimeout` 的根因与修复成立**：用强制 GC 对照，恢复"丢弃引用"后 Windows、Linux 都 3/3 失败；修复后 Windows 连续 10 轮、Linux 连续 20 轮通过；崩溃后由操作系统释放锁的情况，两个平台各 3/3 通过。
- 最终提交的三平台 CI 8/8 通过（Ubuntu 带 race detector）。

**验收**：产品负责人于 2026-10-14 验收 WP1.1，PR #6 已合并（合并提交 `568432d`）。

**WP1.1 的验收范围**：MVP-INIT-01、02、03；INIT-11 和 WT-01 的 M1 部分；P1 的 INIT-08、INIT-10。

**仍未完成、随后续工作包验证**：

- MVP-INIT-04 中 `source list --json` 的 `sources` 为空一项：随 WP1.4 验证。
- MVP-INIT-07（M2）；INIT-09 的推导侧，以及 INIT-11、WT-01 的 M2 部分。
- 审查 016 建议的两处测试收紧，尚未处理。

**发布前必须关闭的独立安全任务**：`doctor` 的 git 版本查询没有隔离环境（implementation-plan M6）。这项任务单独处理，不并入 WP1.4。

**仍然不能确定的部分**：本地 37 次运行中那一次未记录名称的失败，不能断定就是 `TestOpenLockTimeout`。

审查原文按原样入库（它本来就是仓库内的相对链接）。

## 23. WP1.4 代码审查（codex-review-018）的处理

**审查结论**：NO-GO，3 项 BLOCKER、1 项 NON-BLOCKER。审查目标 `e29b426`（PR #7）。审查确认 `source list`、`ack`、`remove` 的既定功能、通道拦截与确认、Schema 输入错误脱敏、PD-25 来源集合哈希、数据目录消失条目的检查，以及正常硬链接环境中的 Marker 原子发布；这些不重新设计。产品负责人决定一次修复全部三项 BLOCKER，不扩大业务范围。

### B1：符号链接目标路径经错误消息泄露

**根因**：CLI 先扫描原始参数，再调用 `source.Resolve`；`Resolve` 拒绝关联 worktree 中的文件时，把**解析后的**相对路径拼进错误消息，而解析后的路径要到 `Resolve` 返回之后才扫描。名称无害的符号链接（`alias.md`）指向文件名含密钥的文件时，密钥进入 `error.v1`。

**修复**：`Resolve` 的所有错误只给出原始参数（已扫描）和固定说明，不给出解析后的路径；解析后的路径仍在使用之前扫描（architecture §6.1.3）。

**回归测试**（公开 CLI，Linux；Windows 没有创建符号链接的权限时跳过）：在关联 worktree 中创建 `alias.md`，指向文件名含合成 GitHub token 的文件，执行 `source add alias.md` 与 `--json`。修复前：退出码 2，但 stdout / `error.v1` 中出现 token 和目标目录名。修复后：两种输出都是退出码 2、`invalid_input`，不含 token 与目标目录名；不登记来源、不分配 `seq`。从主 worktree 经符号链接指向同一文件，同样拒绝且不泄露。合法的符号链接照旧解析：指向主 worktree 文档的链接登记为 `in_repo`（目标路径），指向外部文档的链接登记为 `external`。

### B2：Marker 的 rename 回退可覆盖并发创建的不同 Marker

**根因**：`os.Link` 失败（非"已存在"）后，代码先确认 Marker 不存在，再 `os.Rename`。另一个进程可以在两步之间创建 Marker，随后被替换（TOCTOU）。

**产品决定**（2026-10-15）：不支持安全原子、不可覆盖发布的文件系统，直接返回环境错误（退出码 6），不使用有覆盖风险的 rename 回退。不为 FAT 等文件系统开发新的原子发布机制。

**修复**（architecture §10.6）：保留"临时文件 + `os.Link`"的正常路径；硬链接失败时读取已有 Marker：内容相同视为成功，内容不同（或硬链接报告已存在）拒绝，不存在则返回退出码 6，说明硬链接失败、不使用 rename 回退。临时文件在任何结果下都被删除。错误消息带上原因，`init` 原样转述。

**回归测试**（`os.Link` 经测试钩子替换为返回"不支持"或"权限不足"）：

| 测试 | 修复前 | 修复后 |
|---|---|---|
| 硬链接失败，16 个不同 Marker 并发写入，10 轮 | 第 0 轮即有 6 个写入方返回成功，最终只保留其中一个（writer 10） | 全部返回退出码 6，消息说明硬链接；没有 Marker，也没有临时文件 |
| 硬链接失败，已有 Marker，16 个写入方（一半内容相同、一半不同） | 通过（回归防护） | 内容相同的成功，不同的退出码 6；Marker 字节不变，目录中只有 Marker |
| 正常硬链接，16 个不同 Marker 并发写入，20 轮 | 通过（回归防护） | 每轮恰好一个成功，Marker 是它写入的内容；其余退出码 6 |
| 正常硬链接，相同 Marker 并发写入（已有测试） | 通过 | 通过 |

**平台限制**：硬链接不可用的文件系统（FAT、exFAT，部分网络或 FUSE 挂载）上，`init` 无法写入 Marker，返回退出码 6；项目此时已登记（登记先于 Marker），修复问题后重新执行 `init`。NTFS、APFS、ext4 等常见文件系统不受影响。

### B3：从主 worktree 可以登记关联 worktree 的文件

**根因**：只检查"目标位于**当前**关联 worktree 中"，命令在主 worktree 或另一个关联 worktree 中执行时，检查被跳过；关联 worktree 放在主 worktree 目录之内时，路径前缀判断还会把它当作 `in_repo`。

**修复**（architecture §6.1.3）：在目标文件所在的目录调用 `gitx.Locate`（已验收的受控执行环境），得到所属 worktree 与 git 公共目录。公共目录与项目的相同（`os.SameFile`，按文件身份）而 worktree 不是主 worktree 时，返回退出码 2；不在任何 git 仓库中、或属于其他独立仓库的文件，照常判定。git 无法判断时（文件在 git 目录或裸仓库中，git 拒绝读取所在仓库等），保守拒绝（退出码 2），不当作 `external`。错误只给出原始参数。

**回归测试**（公开 CLI，Windows、Linux；每个拒绝都检查文本与 `--json` 两种输出的退出码和 `error.v1`，并在全部拒绝之后检查 `sources` 表为空、`seq` 不变）：

| 情况 | 修复前 | 修复后 |
|---|---|---|
| 主 worktree → 关联 worktree A 的文件 | 退出码 0，登记为 `external` | 退出码 2 |
| 关联 worktree A → B 的文件；B → A 的文件 | 退出码 0 | 退出码 2 |
| 关联 worktree A → 自己的文件 | 退出码 2 | 退出码 2 |
| 主 worktree → 放在主 worktree 目录之内的关联 worktree 中的文件（绝对路径、相对路径） | 退出码 0，登记为 `in_repo` | 退出码 2 |
| 符号链接（主 worktree 中）→ 关联 worktree 的文件（Linux） | 退出码 0 | 退出码 2，不泄露目标路径 |
| 裸仓库目录中的文件（git 无法判断） | 退出码 0，`external` | 退出码 2 |
| 主 worktree → 主 worktree 的文档；关联 worktree → 主 worktree 的文档 | `in_repo` | `in_repo`（不变） |
| 无关目录中的文档；另一个独立 git 仓库中的文档 | `external` | `external`（不变） |

### NON-BLOCKER

- **N1**：只违反 `propertyNames` 的输入，错误指针可能退化为 `/`。原因是 jsonschema v6.0.3 把 `propertyNames` 错误的 `InstanceLocation` 指向内部的位置栈而不复制（`validator.go:296`）。输入仍被正确拒绝，任意属性名不会泄露。继续记录；修正依赖或定位处理时，保持现有脱敏测试。本轮不修改 Schema 校验器。

### 接受的风险与延期

与审查原文一致：Source Ref 相关验收随 WP1.5 / M2；M1 只支持 `hash_only`，`full` 模式的擦除与整理在 WP2.11；超大来源文件流式哈希、可取消、无大小上限；Git exclude 重复追加、Windows `seq` 偶发 `SQLITE_BUSY`、非 Git 扫描数量限制保持原安排；`doctor` 的 git 版本查询隔离仍是发布前必须关闭的独立安全任务。

审查原文按原样入库。

## 24. WP1.4 最终复审（codex-review-019）与验收

**审查结论**：**GO**，BLOCKER 为 0，没有新增 NON-BLOCKER。审查目标 `3d5272b`，修复基准 `e29b426`。

- **B1 关闭**：拒绝错误只使用已扫描的原始参数，也不转述 git 归属识别的原始错误。审查方在 Linux 独立复验了 `alias.md` 反例和损坏的 git 配置：文本与 JSON 错误都不含 token 或目标路径，退出码 2，来源表和 `seq` 不变；合法符号链接仍可登记。
- **B2 关闭**：rename 回退已完全删除。Windows、WSL Linux 的真实硬链接环境中，16 个不同 Marker 并发发布，连续 20 轮都恰好一个成功；Linux 上禁止真实的 `link` / `linkat` 系统调用后，全部返回环境错误，不留 Marker 或临时文件。审查方另外验证了"登记成功、Marker 发布失败"之后的恢复：第一次 `init` 退出码 6，恢复硬链接后再次执行，复用原项目与 worktree，补写 Marker，`seq` 不再增加。
- **B3 关闭**：按目标目录的 git 元数据判断归属，用 `os.SameFile` 比较公共目录与 worktree，并在路径前缀分类之前拒绝关联 worktree；无法判断时保守拒绝。
- 最终提交的三平台 CI 8/8 通过（运行 38595448856，Ubuntu 带 race detector）。

**验收**：产品负责人于 2026-10-15 验收 WP1.4，PR #7 已合并（合并提交 `35d060f`）。

**WP1.4 的验收范围**：MVP-SRC-01、02、03、05、06（按已确认的 PD-25）；SRC-04、SRC-07、SRC-08 的 M1 部分（见下）；MVP-CH-01、CH-02 对 `source ack`、`source remove`、`source add --allow-any-type` 的验证；CX-17 的 CLI 部分；MVP-INIT-04 中 `source list` 为空一项。另外包括验收前修复的三项检查（A、B、C）、共享 Schema 校验错误的脱敏（含数字映射键）、macOS 并发初始化的根因修复（数据目录消失条目、Marker 只创建不替换）。

**仍未完成、随后续工作包验证**：

- MVP-SRC-04 中"已有引用显示为 `source_removed`"：WP1.5 写入引用之后，随 WP1.5 / M2 验证。
- MVP-SRC-08 中引用的 `excerpt_state = not_stored`、`locator` 与 `excerpt_sha256` 保留：随 WP1.5 验证；`full` 模式与擦除后的整理在 WP2.11（M2）。
- MVP-SRC-07 中 `SOURCE_CHANGED` 作为差距：M2。
- MVP-PRV-07 的 CLI 写入集成：随 WP1.5 验证（§16）。
- MVP-CH-01、CH-02 对 `review` 与 `scope confirm` 的验证：随 WP1.6、WP1.8。

**继续记录的事项**：`propertyNames` 错误指针定位不准（审查 018 N1，jsonschema v6.0.3 的定位问题），不修改共享校验器；Git exclude 重复追加、Windows `seq` 偶发 `SQLITE_BUSY` 维持原安排；`doctor` 的 git 版本查询隔离仍是发布前必须关闭的独立安全任务。

**接受的风险**：硬链接不可用的文件系统上，`init` 无法写入 Marker 而返回退出码 6（产品负责人批准，2026-10-15）；项目已登记，修复环境后可以重试。

**macOS 诊断记录**：两次临时诊断运行（只在 macOS、只由 `diag/**` 分支触发）的结果。Actions 日志有保留期限，关键结果记录如下：

| 运行 | 代码 | 并发 `init`（50 次） | 创建数据库期间检查数据目录（20 次） | 并发写 Marker（50 次） | 消失条目测试（60 次） |
|---|---|---|---|---|---|
| [38583919277](https://github.com/XKHoshizora/specmetry/actions/runs/38583919277) | `9cceb51` 加诊断（`da6a57f`） | 48 通过，2 失败：`lstat …/specmetry.db-journal: no such file or directory`；`cannot list access control lists with /bin/ls: exit status 1` | 5 通过，15 失败（14 次 `/bin/ls`，1 次 `lstat`） | 0 通过，50 失败（`mkdir …/.specmetry: file exists`） | — |
| [38583919241](https://github.com/XKHoshizora/specmetry/actions/runs/38583919241) | `e29b426` 加诊断（`220c607`） | 50 通过 | 20 通过 | 50 通过 | 60 通过 |

两个诊断分支 `diag/macos-before`、`diag/macos-after` 已于验收后删除；运行记录仍在 Actions 中。

审查原文按原样入库。

## 25. WP1.5 代码审查（codex-review-020）的处理

**审查结论**：NO-GO，5 项 BLOCKER、2 项 NON-BLOCKER。审查目标 `2abca00`（PR #8）。审查确认 V-P01 ~ V-P08 的主要校验、`propose` 的原文剥离与输出脱敏、事务与并发、版本前置条件、`proposal-receipt.v1` 的条件与反例，以及六项实现细化。产品负责人决定一次关闭全部五项，不扩大范围，不开始 WP1.6。

### B1：读取对象可能触发按需抓取

**根因**：部分克隆缺少的对象，git 在读取时会从 promisor remote 抓取（启动 `git fetch`，运行远端的 upload-pack）。受控环境没有禁止这一点。

**修复**（`internal/gitx`，architecture §6.1.2）：

- 受控环境增加 `GIT_NO_LAZY_FETCH=1` 和禁止所有传输的 `GIT_ALLOW_PROTOCOL`。
- 版本检查时探测一次 git 是否支持 `--no-lazy-fetch`。不支持时，`CommitOf`、`TreeFiles`、`CountLines` 在读取对象之前识别部分克隆（只读配置），是部分克隆就返回退出码 6。
- 代码引用检查在任何 git 版本下都先识别部分克隆并拒绝（退出码 6），提示改用完整克隆。
- 最低版本仍为 2.32。

**为什么旧版 git 需要"识别后拒绝"**：用真实的 git 2.32.0 验证，`GIT_ALLOW_PROTOCOL` 确实让 upload-pack 不再运行，但 git 仍会启动 `git fetch --filter=blob:none`，它在因传输被禁止而失败之前，向仓库配置写入 `remote.origin.partialclonefilter`。只靠环境变量不能保证只读，所以对不支持 `--no-lazy-fetch` 的 git 一律拒绝部分克隆中的对象读取。

**回归测试**（upload-pack 替身只向仓库外的标记文件追加一行，不连接网络；每个用例先用普通 git 做对照，证明替身确实会被执行）：

| 场景 | git 2.56（Windows） | git 2.43（Linux） | git 2.32.0（Linux，官方源码构建） |
|---|---|---|---|
| 修复前：缺失 blob、树、提交 | 替身执行，`.git` 改变 | 同左 | 同左 |
| 修复后：缺失 blob（`CountLines`）、树（`TreeFiles`）、提交（`CommitOf`） | 替身不执行，`.git` 逐文件不变，不报告为存在 | 同左 | 同左（经由识别后拒绝） |
| 强制走旧版路径（任何 git） | 三种读取都返回退出码 6，替身不执行，`.git` 不变；完整克隆照常读取 | 同左 | 同左 |
| 公开 `propose` 带代码引用（部分克隆） | 修复前：替身执行后才退出 6；修复后：退出 6，替身不执行，`.git` 不变，不写入、不分配 `seq`；不带代码引用的提案照常保存 | 通过 | 通过 |

另外在 git 2.32.0 上运行了全部 18 个 `propose` / `source` 相关的 CLI 测试，全部通过。macOS 与 Windows、Ubuntu 的其他版本由 CI 验证。

### B2：`final_payload_json` 缺少 `hash_only` 防线

**根因**：触发器只在插入时检查 `sources_json`。

**修复**（修改尚未合并的迁移 0004）：`proposal_items` 在插入和更新时，用 `json_tree` 检查 `sources_json` 与 `final_payload_json` 中任何层级、取值不为 null 的 `excerpt` 键；`source_refs` 在插入之外也检查更新；这些触发器在维护模式下同样生效。

**回归测试**：

- 数据层：插入、更新待审条目、更深层级、审核时带原文，都被拒绝且原因正确；维护模式下给引用写入原文同样被拒绝；不含原文的审核结果（`excerpt` 为 null）照常写入。
- 公开路径：`propose` 提交含 canary 的片段之后，用普通 `WriteTx` 尝试更新与插入。修复前三次都成功、并新增一行；修复后都被拒绝，业务记录与 `seq` 不变，数据目录（数据库与 WAL）中找不到 canary；不含原文的审核结果更新成功。

### B3：`goal: null` 被当作空编号

**根因**：`json.Unmarshal` 把 null 解码为空字符串且不报错。

**修复**：读取字符串字段时，缺失与 null 都表示"没有编号"；null 原样保留在 payload 中。

**回归测试**（公开 CLI）：新增需求 `goal: null`、更新需求清除 `goal`。修复前退出码 2；修复后保存为待审条目，payload 中保留 `"goal": null`。不存在的目标编号、类型不对的编号仍为退出码 2。

### B4：行号溢出被当作"没有行号"

**根因**：`strconv.Atoi` 溢出时返回 `ok = false`，调用方把它解释为未填写行号，因而 `exists = true`。

**修复**：静态检查与代码引用检查共用一个解析函数，区分"未填写"与"无效"。行号范围为 1 ~ 2147483647（32 位上限），超出或倒置都返回退出码 2；即使绕过静态检查，无效范围也只会得到 `exists = false`。

**回归测试**：公开 CLI 中 30 位数字、溢出的起点或终点、2147483648、`1-2147483648`、64 位上限与溢出，修复前都以退出码 0 写入（`exists = true`），修复后都返回退出码 2、指针正确、不写入；2147483647、`1-2147483647`、空文件（有、无行号）给出正确的 `exists`；单元测试覆盖边界。

### B5：特殊文件使提案阻塞

**根因**：先 `os.Open` 再检查类型；没有写端的 FIFO 会让打开本身阻塞，取消无从生效。WP1.4 的 `source.HashFile` 有同样的问题。

**修复**：共用的 `source.OpenRegular`。Unix 以 `O_NONBLOCK|O_NOCTTY` 打开，再对描述符 `fstat`，不是普通文件就关闭并拒绝，普通文件恢复阻塞模式；Windows 打开设备或管道本身不等待，再通过句柄确认类型。`HashFile`、片段校验和非 git 项目的代码引用都改用它。

**回归测试**：Linux 上，`HashFile`、`CheckExcerpts`（含 25 ms 超时）、公开 `propose` 与 `source list` 遇到 FIFO 都立即返回（修复前四项都阻塞）；普通文件与 FIFO 在检查期间反复互换，2 秒内 10692 次检查全部立即返回，结果只有 `verified` 或 `source_missing`；提案照常保存，只使用一个 `seq`。Windows 上，`NUL` 设备和目录被拒绝。macOS 由 CI 验证。

### NON-BLOCKER

- **N1**：增加真正超过 1 MiB 的测试文件：跨块边界的多字节字符、匹配之后出现的 NUL、文件末尾截断的字符、读取过程中（两块之间）取消；随机对照测试中每三轮有一轮超过 1 MiB，并断言这一点。
- **N2**：符号链接与子模块测试改为直接提交索引，不再经过 `git add -A`，并在检查前断言提交树中确实有 `120000`、`160000`、`100755` 条目。

### 接受的风险

与审查原文一致：引用结果绑定读取时的 `source_sha256` 与 `checked_head`；正式审核与编号分配属于 WP1.6；`verified` 与 `exists` 只是标记。新增的平台限制：部分克隆中不能检查代码引用；不支持 `--no-lazy-fetch` 的 git 在部分克隆中不读取任何对象（例如 `init` 读取 HEAD 提交时返回退出码 6）。

审查原文按原样入库。

## 26. WP1.5 定向复审（codex-review-021）的处理

**审查结论**：NO-GO。审查目标 `0af9d19`。审查 020 的 B2 ~ B5 关闭，N1、N2 的测试改进有效；**B1 仍有三个残余缺口**，可用真实的 git 2.32.0 绕过。本轮只修复 B1，不改动已关闭的事项。

### B1-1：promisor 的布尔值未按 git 的语义解析

**根因**：`PartialClone` 手工把空值、`true`、`yes`、`on`、`1` 当作真值。git 2.32.0 把 `promisor = 2` 解释为真，Specmetry 却判为完整克隆；空值（`promisor =`）在 git 中为假，却被判为真；git 无法解析的值被当作"不是部分克隆"。

**修复**：`extensions.partialClone` 单独判断，有任何取值即为部分克隆（它是远端名称，不按布尔值解释）；`remote.<name>.promisor` 用 `git config --bool --get-regexp` 由 git 自己解析，任一远端为真即为部分克隆。git 无法解析或无法读取时返回退出码 6，不读取对象。仍在受控环境中只读配置，错误不出现配置值。

### B1-2：`GIT_ALLOW_PROTOCOL` 实际允许了一个协议

**根因**：取值 `specmetry-no-transport` 是允许列表中的一项，同名的 remote helper（`git-remote-specmetry-no-transport`）因此被允许运行。

**修复**：`GIT_ALLOW_PROTOCOL=`，设置为空串：允许列表中没有任何协议。变量必须存在，未设置时按 git 的默认策略允许大多数传输。

### B1-3：部分克隆的否定结果被缓存

**根因**：旧版 git 路径按目录缓存识别结果；仓库在第一次读取之后变成部分克隆时，仍沿用"不是部分克隆"。

**修复**：取消缓存，每次读取对象之前都重新识别。识别与读取之间的并发配置修改不在保证范围内，已在 architecture §6.1.2 写明边界（见下）。

### 回归测试与证据

每个用例都先在旧代码（`0af9d19`）上失败；涉及外部程序的用例先用普通 git 做对照，证明替身确实会被执行。remote helper 替身是测试程序自身的副本（Windows 上同样能被 git 运行），只向隔离目录中的标记文件追加一行，不连接网络。

| 反例 | 修复前 | 修复后（git 2.56 Windows、2.43 Linux、真实 2.32.0 Linux） |
|---|---|---|
| `PartialClone` 布尔值表：`true`、`yes`、`on`、`1`、`2`、`TRUE`、无值的键为真；`false`、`no`、`off`、`0`、空值为假；两个远端其一为真；`maybe`；`extensions.partialClone` 取远端名、`false`、`2` | `2` 与"其一为真"判为假，空值判为真，`maybe` 判为完整克隆（三个版本相同） | 全部符合 git 的解析；`maybe` 返回退出码 6，错误中没有该值 |
| `ls-remote` 经由 Specmetry 的受控环境：同名 helper、另一名称的 helper、file（upload-pack 替身）、ext（`protocol.ext.allow=always`）、ssh（`core.sshCommand` 替身）、https | 同名 helper 运行（三个版本），其余被拒绝 | 全部被拒绝，替身都不运行 |
| 旧版 git 路径：完整克隆读取一次后，改为 `promisor = true` 或 `2` 并删除 blob，再调用 `CountLines`、`TreeFiles`、`CommitOf` | 只有 `CountLines` 出错（读取了对象），另两个照常读取；真实 2.32.0 上 `.git/config` 被追加 32 字节 | 三个都在读取对象之前返回退出码 6；替身不运行，`.git` 逐文件不变；去掉 promisor 设置后照常读取 |
| 公开 `propose --json` 组合反例：`promisor = 2`、URL `specmetry-no-transport::unused`、PATH 中有同名 helper、删除已提交的 blob | 真实 2.32.0 上 helper 运行之后才退出 6；2.56 上读取了对象（`git cat-file` 失败）而不是先拒绝 | 退出码 6 并说明是部分克隆；helper 不运行；`.git` 逐文件不变；不写入、不分配 `seq`；JSON 与文本错误中都没有 helper 名称和仓库路径 |

另外：`gitx` 全部 19 个测试在 2.32.0、2.43 和 2.56 上通过；真实 git 2.31.1 的拒绝测试仍通过。

### 并发边界

（此节描述的机制已被 §27 / PD-26 取代。）识别与随后的对象读取是两个独立的 git 进程。两者之间仓库配置仍可能被并发修改，本轮没有关闭这个窗口，测试也没有覆盖它。窗口内即使启动抓取，空的 `GIT_ALLOW_PROTOCOL` 仍使它无法连接或运行任何程序；但旧版 git 的抓取可能已改写仓库配置。需要完全排除这种情况时，使用支持 `--no-lazy-fetch` 的 git，此时 git 根本不启动抓取。

审查原文按原样入库。

## 27. WP1.5 静态定向复审（codex-review-022）与 PD-26 的处理

**审查结论**：NO-GO。审查目标 `96be996`。B1-1 ~ B1-3 的修复通过了静态审查。

- **BLOCKER**：旧版 git 的只读保证尚未闭合。识别部分克隆与读取对象是两个 git 进程，配置可能在两者之间被并发修改，旧版 git 仍可能改写仓库配置。
- **NON-BLOCKER**：HTTPS 测试的断言弱于其说明。

审查原文按原样入库。

**评估与产品决策**：针对这项 BLOCKER，先评估了以下几点：

- git 的按需抓取（Lazy Fetch）能力，即 `--no-lazy-fetch` 与 `GIT_NO_LAZY_FETCH` 在 2.32、2.43、2.56 上的实际行为；
- 按能力拒绝会带来的影响；
- 私有元数据视图的可行性。

产品负责人据此确认 **PD-26（方案 A）**：运行时探测安全能力，不具备能力时拒绝读取对象。最低版本仍为 2.32；不开发私有视图。本轮只做以下几件事：

- 能力探测与对象读取保护；
- `ListFiles` 门禁；
- `init` 的前置检查与错误传递；
- 直接相关的测试和文档。

已验收的 WP1.5 B2 ~ B5、提案数据模型、Source Ref、问题编号和 schema 脱敏都不改动。

### 评估中确认的事实

以下结果都在隔离的临时仓库中测得：

| git | `--no-lazy-fetch` | `GIT_NO_LAZY_FETCH` | 读取部分克隆缺失的对象 |
|---|---|---|---|
| 2.32.0（官方源码构建） | 不认识 | 不遵守 | 启动抓取。空的允许列表使 upload-pack 不运行，但 `.git/config` 被改写 |
| 2.43.0（Ubuntu `1:2.43.0-1ubuntu7.3`，带 CVE-2024-32465 的修复） | 不认识（退出码 129） | 遵守 | 不抓取，直接失败 |
| 2.56.0（Git for Windows） | 认识 | 遵守 | 不抓取，直接失败 |

旧实现按"认识 `--no-lazy-fetch`"推断"遵守 `GIT_NO_LAZY_FETCH`"。这个推断对 Ubuntu 的 2.43 不成立：它不认识选项，却遵守变量，于是被错误地归入旧版路径，即带并发窗口的部分克隆识别。反过来，一个认识选项、却不遵守变量的 git（测试中用替身模拟），会被旧实现当作安全的 git，直接读取对象。

评估中还发现两个问题：

- **稀疏索引上的 `ListFiles` 没有防护**：`ls-files` 由树对象展开稀疏索引，这是对象读取。git 2.32.0 在这里会启动抓取并改写配置。
- **`init` 静默输出 `snapshot = null`**：快照解析失败时，`init` 把它当作"分支还没有提交"，输出 `snapshot = null`。

### 修复

1. **能力探测**（`internal/gitx/objects.go`；architecture §6.1.2）：
   - 每个进程在第一次需要读取对象时，在 Specmetry 新建的临时目录中手工写出一个裸的部分克隆：promisor remote 指向不存在的本地路径，不含任何对象。
   - 然后让 git 读取一个不存在的对象，运行两次，用 trace2 的结构化事件（JSON）记录。
   - 对照运行不设置 `GIT_NO_LAZY_FETCH`。事件中必须出现 git 自己启动的 `fetch` 子进程。
   - 测试运行设置该变量。只有同时满足以下条件，才判定为"不会抓取"：git 自己的 `start` 和 `exit` 事件齐全；退出状态与进程实际的退出状态一致，并且非零；没有任何子进程。
   - 测试中出现 `fetch` 子进程，判定为"会抓取"。其他情况都是"无法判定"，按不具备能力处理。
   - 判断依据是结构化事件，并有正反对照，不依据"某段文本没有出现"。
   - 临时目录在成功、失败、超时（30 秒）和取消后都删除。git 的输出全部丢弃。
2. **统一入口**：可能读取对象的命令只通过 `runObjects` / `runObjectsCode` / `runObjectsTo` 运行，入口先取得经探测确认的 git。
   - 涉及 `CommitOf`、`TreeFiles`、`CountLines` 和 `ListFiles`。
   - 不具备能力时，在任何仓库中都于运行之前返回退出码 6。
   - 原来的"每次读取前识别部分克隆"（`objectsReadable`）被删除。现在的门禁是 git 的能力，不是仓库的状态。读取进程自己就遵守 `GIT_NO_LAZY_FETCH`，所以审查 022 指出的"识别与读取之间"的窗口不再存在。
   - 包级文档写明：M2 起新增的对象读取（如 `status`）必须经过同一入口。
   - `PartialClone` 保留，只用于代码引用检查的提示。代码引用检查在它之前先做能力检查。
3. **受控复用**：只保存明确的结论，并且只在 PATH 找到的仍是同一个 git 文件（路径、大小、修改时间都不变）时复用。git 变化后重新检查版本并重新探测；实际运行的就是这个文件。失败、取消、超时和"无法判定"都不保存。
4. **`CommitOf`**：
   - 先解析 ref（`rev-parse -q --verify <ref>`，不读取对象）。ref 不存在或分支还没有提交时，返回空。
   - 再读取它指向的提交（`<oid>^{commit}`）。读取失败时返回退出码 6：对象缺失，或者不是提交。不再返回空。
5. **`init`**：
   - git 项目在定位仓库之后、列出文件之前，先做能力检查。
   - 主 worktree 分支的快照在打开数据库之前解析。
   - 已登记项目的目标 ref 与当前分支不同时，这个 ref 在写事务中解析，失败则整个事务回滚。
   - 读取失败一律返回退出码 6。`init.v1` 不变。
6. **`ListFiles` 不展开稀疏索引**：`ls-files` 带 `--sparse` 运行，稀疏检出之外的目录以目录项列出。目录项不是常规文件，不参与候选来源和阶段识别，所以结果与展开后相同。

### 实施中发现、待确认的事项

以下四项已在 2026-10-15 由产品负责人决定，见 §28。

1. **展开稀疏索引会写对象文件。**
   - git 2.56 与 2.43 实测：不带 `--sparse` 的 `ls-files` 展开稀疏索引时，会重新写入已存在的树对象，改写对象文件的修改时间。树对象缺失时，还会静默漏掉其中的文件。git 2.32.0 的源码中，`ensure_full_index` 同样调用 `cache_tree_update`。`--sparse` 可以避免展开。
   - git 2.32.0 在解析参数之前就读取并展开了索引：带 `--sparse` 时，它在报告"选项未知"（退出码 129）之前，已经抓取并改写了配置。所以是否认识这个选项不能在用户仓库中试探，只能在临时目录中探测。
   - **本轮实现**：不认识 `--sparse` 的 git，列出文件时返回退出码 6（fail-closed），依据是 PD-26 ④"不能安全完成文件发现"。影响范围：遵守 `GIT_NO_LAZY_FETCH`、但 `ls-files` 不认识 `--sparse` 的 git，例如可能把该修复回移到较旧版本的发行版。本轮没有实测确认是否真有这样的发行版。在这类 git 上，即使仓库没有稀疏索引，Git 项目的 `init` 也会被拒绝。
   - **备选方案**：索引不是稀疏索引时，仍用不带 `--sparse` 的 `ls-files`。但这需要先判断索引是否稀疏，判断与读取之间同样存在并发窗口。
   - **请产品负责人确认**采用哪一种。
2. **稀疏检出中缺失的 `.gitignore`。** git 从索引读取工作区之外的 `.gitignore` 时，如果 blob 缺失，具备能力的 git 不抓取、不报错，按没有这些规则处理（2.56、2.43 实测）。这只影响候选来源列表，不影响安全，记录为已知行为。
3. **SHA-256 仓库。** `init.v1` 的 `commitSha` 只接受 40 位 SHA-1。SHA-256 仓库的提交能够读取，但 `init` 仍按原有行为输出 `snapshot = null`。这是格式限制，不是读取失败，本轮未改动。是否需要新增 TD 条目，请确认。
4. **M2 的衔接。** M2 的目标解析与状态推导同样必须经过统一入口。git 不具备能力时，返回退出码 6，还是按 data-model §6 的"git 不可用"给出保守结论 `not_assessed`，需要在 M2 设计细化时确定。

### 回归测试与证据

替身是测试程序自身的副本，命名为 `git`，可在三个平台上运行。它把命令转交给真实的 git，只做以下一种改动：

- 隐藏 `GIT_NO_LAZY_FETCH`，模拟不遵守该变量的 git；
- 隐藏 trace2 事件，模拟无法判定的 git；
- 使 `ls-files --sparse` 以退出码 129 失败；
- 一直挂起。

涉及外部程序的用例，都先用普通 git 做对照，证明替身确实会被执行。

| 反例 | 修复前（`96be996`） | 修复后 |
|---|---|---|
| 用"会抓取"的替身执行 git 项目的 `init`：新项目、已登记的项目；数据目录尚不存在，或已有其他项目 | 退出码 0，项目被登记（旧实现按选项判断，认为这个 git 安全） | 退出码 6。数据目录不存在时不创建；已存在时，数据目录和仓库逐文件不变（包括修改时间）。没有运行任何可能读取用户仓库对象的命令。非 git 项目照常初始化 |
| 部分克隆缺少 HEAD 提交，执行 `init` | 退出码 0，`snapshot = null` | 退出码 6，不创建数据目录，仓库不变，promisor remote 不运行 |
| 已登记项目从关联 worktree 再次执行 `init`，目标 ref（`main`）的提交缺失 | （新增用例） | 退出码 6，`seq` 不变，关联 worktree 未被登记（事务回滚） |
| 用"会抓取"的替身提交含代码引用的 `propose` | 退出码 0，写入 1 个提案、1 个条目，`seq` 3 → 4；运行了 `rev-parse HEAD^{commit}`、`ls-tree`、`cat-file blob` | 退出码 6，不写入，`.git` 不变，没有读取对象。不含代码引用的 `propose`，以及 `source add`、`list`、`ack`、`remove` 都照常工作 |
| 稀疏索引，执行 `ListFiles`（变异测试：去掉 `--sparse`） | 对象文件的修改时间被改写；从树对象展开出 `b/y.txt` | `.git` 逐文件不变；`b/` 作为目录项列出。树对象缺失时不抓取，仓库不变 |
| `CommitOf`：分支还没有提交、ref 不存在；部分克隆或完整克隆缺失提交；ref 指向 blob | 缺失的提交返回空，被当作"没有提交" | 前两种返回空；后几种返回退出码 6，不抓取，仓库不变 |
| 探测本身 | — | 真实 git 的结论与独立判据（不设协议限制，看 upload-pack 替身是否写出标记）一致。只打印版本的替身、隐藏 trace2 的替身判定为"无法判定"，结论不保存，下次重新探测。成功、"会抓取"、"无法判定"、取消（执行前与执行中）、超时之后，临时目录都已删除。同一个 git 只探测一次；git 文件变化后重新探测（版本、对照、测试共 3 次运行）。`ls-files --sparse` 未知时拒绝列出文件，只在临时目录中试探一次 |

真实 git 的结果：

| git | 探测结论 | 独立判据 | 验证范围 |
|---|---|---|---|
| 2.32.0（WSL，官方源码构建） | 会抓取 | 一致 | `gitx` 的探测与替身测试通过。公开命令方面：git 项目的 `init` 返回退出码 6，不写入；代码引用检查返回退出码 6，不写入；非 git 项目的 `init`、不含代码引用的 `propose`，以及 `source add`、`list`、`ack`、`remove` 都照常工作（`TestARealGitThatFetches`） |
| 2.43.0（WSL，Ubuntu） | 不会抓取 | 一致 | 全部测试包通过（`SPECMETRY_TEST_GIT_OBJECTS=safe`） |
| 2.56.0（Windows） | 不会抓取 | 一致 | 全部测试包通过 |
| 2.31.1（官方源码构建） | 不进入探测：版本检查即拒绝 | — | `TestInitRefusesRealOldGit` 通过 |

被替换的测试：`TestOlderGitRefusesPartialClones` 和 `TestObjectReadsRecheckThePartialClone` 强制走旧版路径，已被下面两个测试取代：

- `TestAGitThatFetchesReadsNoObject`：覆盖完整克隆、部分克隆和稀疏索引；
- `TestARepositoryThatBecomesAPartialCloneIsNotFetchedFrom`：第一次读取之后，仓库变成部分克隆；用具备能力的 git 读取时不抓取，仓库不变。

### 未处理

审查 022 的 NON-BLOCKER（HTTPS 测试的断言）不在本轮范围内，保持未处理。

## 28. PD-26 的细化决定与 SHA-256 仓库的拒绝（2026-10-15）

`25a93df` 的三平台 CI 8/8 成功之后，产品负责人对 §27 中待确认的事项作出三项决定。本轮只修改 SHA-256 仓库的拒绝逻辑、直接相关的测试，以及这三项决定的文档。以下内容不变：

- git Lazy Fetch 的行为探测；
- `--sparse` 的安全处理；
- WP1.5 的 B2 ~ B5 修复；
- Proposal、Source Ref、Question 和 Schema 的现有实现；
- git 2.32 的最低版本要求，以及 PD-26 已确认的规则。

### 决定一：`ls-files --sparse`

批准当前实现，作为 PD-26 已批准的实施细化，不再待确认：

- git 不认识 `ls-files --sparse` 时，`ListFiles` 返回退出码 6；
- 即使普通索引也许可以读取，也不在用户仓库中试探；
- 候选来源发现只面向 worktree 中实际可访问的普通文件，不要求列举稀疏检出之外的全部文件。

已记入 PRD 的 PD-26、architecture §6.1.2 和 MVP-INIT-12。代码不变。

### 决定二：SHA-256 仓库

MVP v0.1.0 不支持 SHA-256 格式的 git 对象 ID。§27 记录的现象（64 位提交 ID 被转为 `snapshot = null`）不可接受，只做最小修复：

- **识别**：`gitx.ObjectFormat` 用受控的 `git rev-parse --show-object-format` 识别仓库的对象格式，只读配置，不读取对象。git 2.32.0、2.43、2.56 都支持这条命令；git 给出 `sha1`、`sha256` 以外的答复时，返回退出码 6。
- **拒绝**：git 项目的 `init` 在定位仓库之后、能力检查和任何写入之前识别对象格式。是 SHA-256 时返回 environment（退出码 6）：不创建数据库，不登记项目，不写 marker 和 exclude，不分配 `seq`。拒绝与是否会输出快照无关：分支还没有提交、处于分离 HEAD 时同样拒绝，不能靠省略快照绕过。
- **快照**：快照解析得到的不是 40 位 SHA-1 ID 时，同样返回退出码 6。原来的"转为 `null`"已删除。SHA-1 分支还没有提交时，仍按既有规则输出 `snapshot = null`。
- **不变的部分**：`init.v1` 不变，不提前支持 64 位提交 ID。新增 TD-26，记录完整支持需要同步调整 Schema、数据模型、Code Ref、Git Reader 与测试，不属于 v0.1.0。

**回归测试**（真实的 SHA-256 临时仓库，用 `git init --object-format=sha256` 创建）：

| 用例 | 修复前（`25a93df`） | 修复后 |
|---|---|---|
| 分支有提交、没有提交、分离 HEAD 三种 SHA-256 仓库，数据目录尚不存在 | 退出码 0，`created = true`，`snapshot = null` | 退出码 6；不创建数据目录；仓库（包括 marker、exclude 和 `.git`）逐文件不变 |
| 同上，数据目录中已有 3 个项目（非 git、SHA-1 有提交、SHA-1 没有提交） | 退出码 0，项目被登记 | 退出码 6；数据目录逐文件不变，`seq` 不变，项目仍为 3 个；仓库逐文件不变 |
| SHA-1 对照 | — | 有提交时快照等于 HEAD；分支还没有提交时 `snapshot = null` |
| `ObjectFormat` | — | SHA-1 仓库返回 `sha1`；SHA-256 仓库（没有提交）返回 `sha256`；替身 git 给出其他答复时返回退出码 6 |

### 决定三：M2 的环境失败

确认原则：正确的进度推导依赖 git 对象，而 git 无法安全读取时，返回 environment（退出码 6），不把环境失败转换为正常的 `not_assessed`。已记入 implementation-plan 的 M2；data-model §6 中"git 不可用"一行的结论，在 M2 实现时据此细化，并补齐测试。本轮不提前开发 M2。

## 29. WP1.5 最终定向审查（codex-review-023）的处理

**审查结论**：NO-GO。审查目标 `27a49af`。PD-26 的门禁结构、Sparse Index 和 TD-26 的拒绝逻辑基本符合批准方案，但有 1 项 BLOCKER。NON-BLOCKER（HTTPS 测试的断言）维持原分类，本轮仍不处理。审查原文按原样入库。本轮只修复事件完整性与安全分类，改动限于 `internal/gitx/objects.go` 和直接相关的测试。以下内容不变：

- PD-26 的方案，TD-26 的限制，git 最低版本；
- 数据库、Proposal、Source Ref、Question；
- Review 020 的 B2 ~ B5；
- Sparse Index 与 `init` 的设计。

### B1：异常的 trace2 事件流可能被判为 `neverFetches`

**根因**：`readProbeEvents` 只检查了事件的来源，没有检查生命周期：

- git 之下进程（后代 sid）的事件被直接忽略，即使它们显示有子进程在运行；
- 重复的 `exit` 会覆盖之前的结果；
- `exit` 出现在 `start` 之前也被接受。

这样，不完整或矛盾的事件流可能得出"可信、完整、没有子进程"的结论，进而被判为 `neverFetches` 并保存。

**修复**：新增 `checkProbeEvents`，按写入顺序逐条检查。只有事件流构成 git 的一次完整、一致的运行才可信（规则见 architecture §6.1.2"事件流的完整性"）：

- **git 自身的生命周期**：`version` 至多一次且在 `start` 之前；`start`、`exit` 各一次，顺序正确；`atexit` 至多一次，在 `exit` 之后，状态相同；其他事件都在 `start` 与 `atexit` 之间。
- **子进程事件**：只在 git 运行期间出现。`child_start` 的 `child_id` 互不相同；`child_exit`、`child_ready` 只对应已启动、未退出的子进程；不认识的 `child_*` 事件不可信；所有子进程都在 git 退出前退出。
- **后代进程**：后代 sid 的任何事件都是子进程的证据，只能在 git 运行期间出现，并且这样的子进程不多于 `child_start`。
- **退出状态**：`exit` 的状态必须与进程实际的退出状态一致；不一致、或出现其他进程的事件，都不可信。

分类改用 `showsNoChild`：完整的运行中，既没有 `child_start`，也没有任何后代事件，才判为 `neverFetches`。不可信的事件流仍是"无法判定"：返回错误，不保存，下次重新探测。能力探测本身不会被跳过。

### 回归测试与证据

用纯 JSON 测试，不启动 git、不运行外部程序、不联网。测试事件流按真实 git 2.56、2.43、2.32.0 写出的事件构造，事件类型与顺序以 2026-10-15 在隔离临时仓库中的实测为准。

| 测试 | 修复前（`27a49af`） | 修复后 |
|---|---|---|
| Codex 反例 A：缺少 `child_start` 的后代 sid（`fetch` 子进程的 start、exit 都在） | 被判为"完整、没有子进程" | 不可信 |
| Codex 反例 B：`exit` 在 `start` 之前 | 同上 | 不可信 |
| Codex 反例 C：先 `exit` 0，再 `exit` 128，实际 128 | 同上 | 不可信 |
| 另外 12 种异常：两次 `exit`；第二次 `start` 属于另一命令；两次 `atexit`；`atexit` 状态不同；`atexit` 之后的事件；`start` 之前的其他事件；`start` 之后的 `version`；没有 `child_start` 的 `child_exit`、`child_ready`、其他子进程事件；只有孙进程的事件；git `start` 之前的子进程事件 | 被判为"完整、没有子进程" | 不可信 |
| 另外 6 种异常：`child_start` 缺 `child_id`；缺 `child_exit`；同一 `child_id` 两次；git 退出后的 `child_exit`；两个子进程只有一个 `child_start`；git `atexit` 之后的子进程事件 | 被当作可信的完整运行（有子进程） | 不可信 |
| 13 种原本就能识别的情况：重复的 probe `start`；缺少 `exit`、`start` 或任何事件；`atexit` 没有 `exit`；退出状态不一致或缺失；git 成功退出；无效 JSON；缺 sid；其他进程；sid 只是前缀相似；两次 probe 命令 | 不可信或不完整 | 不可信或不完整（回归保护） |
| 合法事件流：2.56、2.43、2.32.0 三种形态的无子进程测试运行；没有 `atexit`；中间有空行；线程的 `data` 事件；合法的抓取对照 | 通过 | 通过（没有被过度拒绝） |
| 经由 `CheckObjectReads`：由替身 git（测试程序自身，不运行真实 git）写出合法对照和反例 A、B、C | 判为 `neverFetches`，没有错误，结论被保存 | 退出码 6（无法判定），不保存；再次检查重新运行两次探测。合法的无子进程流仍判为 `neverFetches`，抓取流判为 `fetches` |

真实 git 的回归见 PR #8 描述中的验证一节：

- 2.32.0 仍判为会抓取，2.43 和 2.56 仍判为不会抓取；
- 2.31.1 仍被版本检查拒绝；
- 临时目录照常清理；
- 对象读取、Sparse Index、SHA-256 拒绝、`init` 和 `propose` 的既有测试都通过。

三平台 CI 中真实 git 的版本为 2.55 或 2.56；真实旧版 git 的测试在 CI 中按设计跳过，只在本地验证。

## 30. WP1.5 最终定向复审（codex-review-024）与验收

**审查结论**：**GO**，BLOCKER 为 0，没有新增 NON-BLOCKER。审查目标 `de14374`，增量基准 `27a49af`。

- **审查 023 的 B1 关闭。** Codex 的三个原始反例（缺少根 `child_start` 的后代抓取事件；`exit` 在 `start` 之前；先 `exit` 0、再 `exit` 128），审查方用独立的纯 JSON 测试，在基准提交和目标提交上做了对照：基准提交错误地满足安全条件，目标提交判为不可信并拒绝。异常事件流沿错误路径返回，能力状态保持 `unprobed`，下次调用重新探测；合法的无子进程日志与合法的抓取日志仍分别判为 `neverFetches` 与 `fetches`。
- 最终提交的三平台 CI 8/8 通过（运行 38746116548，Ubuntu 带 race detector）。新增的解析器测试与使用当前真实 git 的探测测试在三个平台上实际运行，没有跳过。CI 中的 git 为 Windows 2.56.0、Ubuntu 2.55.0、macOS 2.55.0。

**验收**：产品负责人于 2026-10-15 验收 WP1.5，PR #8 已合并（合并提交 `429e94e`）。

**WP1.5 的验收范围**（逐项证据见 PR #8 描述）：

- MVP-BL-01、BL-05 的 M1 部分；BL-02、BL-03（含 CX-14）、BL-04、BL-21、BL-28；
- BL-23 中描述与标题的拒绝部分；BL-26 的 M1 部分；BL-13 的 Agent 通道部分；
- MVP-PRV-07 的 CLI 写入集成（`propose`）、MVP-PRV-08；
- MVP-SRC-04 中引用在来源移除后保留的部分，MVP-SRC-08 的 M1 部分；
- MVP-BL-29、MVP-INIT-12、MVP-INIT-13（PD-26、TD-26）。

**仍未完成、随后续工作包验证**：

- 审核、接受与拒绝条目、冲突与三方对比、`propose --apply`、为接受的条目分配正式编号、写入实体的来源引用与代码引用、基准版本（BL-06 ~ BL-12、BL-14 ~ BL-16、BL-19 ~ BL-25、BL-27，以及 BL-23 的 `change_class` 部分、BL-13 的 `--apply` 行为）：WP1.6。
- `status` / `gaps` 的推导（BL-01 的推导部分、BL-05 的状态部分、BL-18、BL-25 的阻塞部分、SRC-04 的 `source_removed`）与 `full` 片段模式（BL-26）：M2。
- PD-26 决定三（M2 的环境失败）的测试：M2。

**继续记录的事项**：

- 审查 022 起保留的 NON-BLOCKER：HTTPS 测试只凭非零退出判断传输被阻断，断言弱于说明。作为后续改进记录，不扩大 WP1.5 的验收范围。
- 此前记录的 `propertyNames` 错误指针、Git exclude 重复追加、Windows `seq` 偶发 `SQLITE_BUSY` 维持原安排；`doctor` 的 git 版本查询隔离仍是发布前必须关闭的独立安全任务。

**接受的风险**：沿用 PD-26 已批准的能力探测边界，包括可信 git 可执行文件的边界：刻意替换 git 程序并保持路径、大小和修改时间不变，不在防护范围之内。

审查原文按原样入库。

## 31. WP1.6 代码审查（codex-review-025）的处理

**审查结论**：NO-GO。审查目标 `fa58b44`，有两项可复现的 BLOCKER 和两项 NON-BLOCKER。本轮不需要新的产品决策。

本轮只修复 B1 和 B2，不重新设计 WP1.6。以下内容不变：

- 基准的应用顺序与编号规则；
- 迁移 0005；
- material / editorial 的持久化规则；
- 来源引用与 `hash_only` 防线；
- 问题关联；
- WP1.7、WP1.8、M2 的功能，以及已验收的 WP1.5。

审查原文按原样入库。

### B1：`proposal show` 回显敏感参数

**根因**：CLI 只检查参数个数，就把未经校验的参数交给 `ShowDocument`；找不到提案时，`not_found` 的消息直接拼入这个参数。因此误粘贴的密钥会进入文本错误和 `error.v1`，绕过了 schema 错误的脱敏防线。

**修复**：

- `ShowDocument` 找不到提案时，使用固定消息，不含任何外部参数。其他调用方直接调用它也不会泄露参数。审核中同样的 `not_found`（`review.Check`）也改为固定消息。
- `proposal show` 先检查参数是否为 ULID 格式（新增 `ulid.Valid`，与 `common.v1` 的 `ulid` 模式一致）。格式无效时返回退出码 2，消息和详情不含原值。
- 同类问题，本轮新增的 `baseline history --since`：它原是整数 flag，解析失败时 cobra 的错误（文本模式下会输出原因）会回显原值。现在由命令自己解析，错误使用固定消息。

**回归测试**（`TestProposalShowDoesNotEchoItsArgument`）：

- 输入：合成的 GitHub Token、合法但不存在的 ULID、`--since` 后跟 Token；每种都在文本模式和 `--json` 模式下执行。
- 期望：退出码 2；stdout、stderr、`error.v1` 的 message / details / current，以及数据目录中都找不到输入；没有业务写入；正常的 `proposal show` 照常可用。
- 修复前（`fa58b44`）：前两种在两种模式下都回显输入；`--since` 在文本中回显，`--json` 时没有输出 `error.v1`（参数解析在命令之前失败）。

### B2：过期的审核返回退出码 2

**根因**：首次检查和正式写入都先执行 `review.Check`。目标已变化时，它先报告"冲突未确认"（退出码 2），版本比较根本执行不到。第一次的版本比较还放在事务之外，无法覆盖确认期间基准被其他连接推进的情况。这违反了 MVP-BL-10。

**修复**：

- 新增 `review.CheckVersion`：在事务中读取当前基准版本，与 `seen_base_version` 比较，不一致即返回 `stale_precondition`（退出码 5，`error.current.baseline_version`，指向 `/seen_base_version`）。
- `review.Check` 第一步就调用它，因此首次读取、演练、确认后的正式写入，这三个事务都先比较版本，过期时不再做冲突、条目状态和后续处理的检查，也不分配 `seq`。
- 事务外的版本比较已删除。`baseline.Apply` 内部的版本检查保留，作为第二道防线；两处使用同一个错误构造函数 `baseline.StaleVersion`。

**回归测试**：

- **反例 A**（`TestReviewIsStaleBeforeTheConflict`）：两份提案在版本 1 修改同一个需求；接受第一份之后，以 `seen_base_version = 1` 审核第二份。不带和带 `acknowledged_conflict` 两种情况都返回退出码 5，`current.baseline_version = 2`，没有出现确认提示，没有写入。版本为当前版本、但冲突未确认时，仍然返回退出码 2。
- **反例 B**（`TestReviewIsStaleAfterTheConfirmation`）：第二份审核通过首次检查和演练，进入确认；读取确认输入时，用另一个数据库连接提交第一份审核，然后确认第二份。结果为退出码 5、版本 2；第二份没有被应用。版本、实体、编号、`seq`、引用和条目状态，都与第一份审核提交后完全一致；第二份的条目为 `conflict`，未审核。
- **修复前**（`fa58b44`）：反例 A 在不带 `acknowledged_conflict` 时，反例 B 在确认之后，都返回退出码 2。
- 另外，所有被拒绝的审核测试现在都断言：拒绝发生在确认提示之前。审查方的双连接并发测试（两个审核都完成演练后同时确认，结果为 `[0, 5]`）在本轮之前已通过，本修复没有改变写事务的提交顺序。

### NON-BLOCKER（登记并保留，不在本轮修改）

- **N1**：确认预览显示审核者选择的 `editorial`，而应用时强制字段会把它升级为 `material`。持久化正确，预览应显示最终生效的类别及原因。待后续处理。
- **N2**：`baseline history` 没有输出已保存的后续处理理由（`baseline_changes.rationale`）。理由没有丢失，扩展历史 schema、读取模型和测试待后续处理。

## 32. WP1.6 最终定向复审（codex-review-026）与验收

**审查结论**：**GO**，BLOCKER 为 0，没有新增 NON-BLOCKER。审查目标 `6f12c12`，增量基准 `fa58b44`；只复核审查 025 的 B1、B2 及直接回归。

- **审查 025 的 B1 关闭。** `proposal show` 先校验 ULID 格式，`ShowDocument` 与 `review.Check` 的 `not_found` 使用固定消息，`baseline history --since` 由命令自行解析。审查方确认：合成的 GitHub Token、合法但不存在的 ULID、敏感及越界的 `--since` 参数，在文本和 `error.v1` 中都不回显，数据目录中也找不到；直接调用内部接口同样不泄露。
- **审查 025 的 B2 关闭。** `review.Check` 首先调用 `CheckVersion`，首次检查、演练和确认后的正式写入都在事务中先比较版本。审核前已过期（不论是否带 `acknowledged_conflict`）、确认期间版本前进，都返回退出码 5；版本为当前版本而冲突未确认时仍返回退出码 2；两个审核同时提交时结果为 `[0, 5]`，只产生一个版本、一个正式编号和一个新增 `seq`。
- 审查方把三个新增回归测试移入基准提交的隔离副本，确认它们在旧代码上失败、在目标提交上通过。确认期间的测试使用同进程中的独立数据库连接，不是多进程测试，但足以覆盖这一事务间窗口。
- 最终提交的三平台 CI 8/8 通过（运行 38780643544，Ubuntu 带 race detector），三个新增测试在三个平台上实际运行。

**验收**：产品负责人于 2026-10-16 验收 WP1.6，PR #9 已合并（合并提交 `c9c9cf8`）。M1 尚未整体完成：WP1.7、WP1.8 未验收。

**WP1.6 的验收范围**（逐项证据见 PR #9 描述）：

- MVP-BL-06 ~ BL-12、BL-14 ~ BL-16、BL-19、BL-20、BL-22、BL-24、BL-27；
- BL-13 的 `--apply` 行为、BL-23 的 `change_class` 部分、BL-25 的 M1 部分（接受条目后问题关联到正式实体）；
- `review`、`propose --apply` 的 MVP-CH-01、CH-02 验证；
- `baseline_changes` 的约束（迁移 0005）、回滚、`hash_only` 下不保存摘录。

**仍未完成、随后续工作包验证**：

- 问题的回答与驳回（MVP-BL-17）：WP1.7。
- 审核中附带的范围确认（`review.v1` 的 `scope`，目前明确拒绝）、`target set` / `scope confirm`：WP1.8。
- `status` / `gaps` 的推导：BL-18、BL-25 的阻塞部分、审核之后的 `SCOPE_STALE`：M2。

**继续记录的事项**：

- 审查 025 的两项 NON-BLOCKER 保留，不在 WP1.6 中修改：
  - **N1**：确认预览显示审核者选择的 `editorial`，而强制字段在应用时会把它升级为 `material`；
  - **N2**：`baseline history` 没有输出已保存的后续处理理由（`baseline_changes.rationale`）。
- 本地 Windows 上 `internal/cli` 包的测试在并行负载下超过 Go 默认的 10 分钟超时（审查方未发现死锁或无限等待，不要求调整全局超时）。作为独立的工程维护任务处理，不在功能工作包中顺手修改测试脚本。
- 此前各节记录的事项维持原安排。

**接受的风险**：沿用既有批准边界：通道识别只防误操作，不证明身份；M1 的 `hash_only` 策略；后续工作包的既定范围。没有接受新的风险。

审查原文按原样入库。

## 33. WP1.7 代码审查（codex-review-027）的处理

**审查结论**：NO-GO。审查目标 `78ca882`，有 3 项 BLOCKER 和 1 项 NON-BLOCKER。本轮不需要新的产品决策。审查确认 Question 的状态转换、持久化、查询关联、通道拦截和并发保护正确。

本轮只修复 B1、B2、B3，以及直接相关的回归测试。以下内容不变：

- Question 的状态模型、`questions` 表、迁移 0004 和 0005；
- schema 的业务字段结构；
- 并发处理和 `seq` 的分配；
- Proposal、Baseline、Scope 的业务行为；
- WP1.8 和 M2 的功能。

审查原文按原样入库。

### B1：提案文本中的终端控制字符进入人工确认摘要

**根因**：文本 schema 和密钥扫描都允许 ESC、CR、BEL 等控制字符。`indented` 只处理换行，`printQuestions`、`closeSummary` 和 `channel.Guard` 的确认提示把其余字符原样写到终端。不可信的提案因此可以清屏、移动光标、覆盖整行、设置终端标题，伪装人看到的确认摘要。

**修复**：

- 新增 `internal/termtext`，作为统一的终端转义函数 `Escape`：
  - LF 和制表符保留，CRLF 显示为 LF；
  - 其他 C0 控制字符、DEL、C1 控制字符显示为可见的转义；
  - 双向控制字符和行、段分隔符显示为 `\uNNNN`；
  - 不是 UTF-8 的字节显示为 `\xNN`。
  - 不删除任何字符，`Escape` 是幂等的。
- `question` 各命令在写到终端的边界统一转义：
  - `question list` 的文本输出整体转义；
  - `question answer`、`question dismiss` 交给 `channel.Guard` 的确认摘要逐行转义（问题、上下文、回答或理由、显示名、关联）；
  - 结果行同样转义。
- 数据库、提案原文和 `--json` 输出不变。schema 允许的文本不会因此被拒绝。

**回归测试**：

- `TestQuestionTextIsShownSafely`：
  - 通过公开的 `propose` 提交含有 CSI、OSC、ESC、CR、BEL、BS、DEL、C1、双向控制字符、CRLF 和正常换行的问题与上下文；配置中的显示名含有 OSC 和双向控制字符；回答和理由也含有控制字符。
  - `question list`（`--all` 与否）以及 `question answer` 和 `question dismiss` 的确认摘要中，都没有任何原始控制字符，每一处都能看到明确的转义文本；换行保留在缩进之下。
  - 取消确认时不写入。
  - `question list --json` 解码后、数据库中，都是原始文本。
  - 所有检查都在内存缓冲区中进行，没有向真实终端发送控制序列。
- `internal/termtext` 的单元测试：逐项检查各类字符、幂等性，并扫描 U+0000 ~ U+3000 和全部单字节，确认转义后不留下控制字符。
- **修复前**（`78ca882`）：`question list` 和两个确认摘要都含有原始的 0x1b、0x0d、0x07 等字节，测试失败。

### B2：未知的 Question 子命令显示帮助并返回成功

**根因**：`question` 命令组既没有 `Args` 也没有 `RunE`，cobra 对未知子命令显示帮助并返回 0。带 `--json` 时，命令组没有这个 flag，返回退出码 2 的文本错误，stdout 没有 `error.v1`。

**修复**：

- 命令组的 `Args` 拒绝未知子命令：返回 `invalid_input`（退出码 2），消息中不引用原值，只可能建议已有子命令的名字。
- 命令组新增 `--json`（错误输出 `error.v1`）。
- 只输入 `question` 或使用 `--help` 时，仍显示帮助并返回 0。`question --json` 不带子命令时返回退出码 2 的 `error.v1`，不输出帮助。
- `list`、`answer`、`dismiss` 的行为不变。

**回归测试**（`TestQuestionCommandLineErrors`，经过真实的命令入口）：

- 测试的输入：`question unknown`、`question answr`、以 Token 作为子命令、`--json` 在前或在后、未知 flag 在子命令之前。
- 结果：都返回退出码 2，不回显；带 `--json` 时 stdout 是通过 schema 校验的完整 `error.v1`，stderr 为空。
- `question`、`question --help`、`question list --help` 返回 0。
- 没有写入，不分配 `seq`。
- **修复前**：前三种返回 0 并输出帮助；带 `--json` 的形式 stdout 都不是 `error.v1`。

### B3：无法解析的 flag 在 `--json` 之前时，不输出 `error.v1`

**根因**：`wantsJSON` 只看 cobra 已经解析的 `json` flag。pflag 在出错的 flag 处停止解析，后面的 `--json` 没有生效。另外，原测试在没有 JSON Pointer 的用例上跳过了 JSON 断言，恰好漏掉了未知 flag 和无效布尔值。

**修复**：

- `question` 命令的 flag 错误（`flagError`）标记为 `unparsedFlags`。
- 只有遇到这类错误时，`executeAs` 才按 pflag 的规则从参数中判断是否要求 JSON（`jsonInArgs`）：
  - `--json`、`--json=<布尔值>` 都算，以最后一个为准；
  - `--json=<非布尔值>` 不改变结果；
  - 取值 flag 的值不算（`-m x`、`-mx`、`-m=x`、`--message x`、`--file x`）；
  - `--` 之后的参数不算；
  - 未知 flag 不取值，`--unknown=--json` 不算。
- 正常解析的命令，以及其他命令（它们的 flag 错误不是 `unparsedFlags`），仍按已解析的 flag 判断，行为不变。
- 错误消息仍是固定文本，不引用 flag 名、值或位置参数。

**回归测试**：

- `TestQuestionCommandLineErrors` 覆盖了审查列出的全部输入，并补充了更多形式：
  - 应为 JSON：`--json=true`、重复的 `--json`、`-mx` 加未知 flag、`--file` 加未知 flag；
  - 应为文本：`--json --json=false`、`--json=maybe`、`--message --json`、`--file --json`、`--` 之后的 `--json`。
- 要求 JSON 时，断言退出码 2、stdout 为完整且通过 schema 校验的 `error.v1`、`error.code` 为 `invalid_input`、stderr 为空。
- 不要求 JSON 时，断言 stdout 为空、stderr 为文本错误。
- `question answer Q-001 -m --json`：`--json` 是回答文本，命令正常解析；在 Agent 通道返回退出码 3 的文本错误，不进入 JSON 模式。
- `TestQuestionInputIsChecked` 的缺口已修正：每个 `--json` 用例都断言完整的 `error.v1` 和 `error.code`（`Q-099` 为 `not_found`，其余为 `invalid_input`），JSON Pointer 单独检查。
- **修复前**：`list --all=not-bool --json`、`answer Q-001 -m x --unknown --json` 等 11 种形式的 stdout 都不是 `error.v1`；修正后的 `TestQuestionInputIsChecked` 也在 `--<token>`、`--all=<token>` 两种用例上失败。

### NON-BLOCKER 与遗留安全问题

- **N1：其他 CLI 回显不可信参数**。这是实际的隐私缺陷，不是已接受的风险。
  - PR #10 原描述中"带 `--json` 时只把固定消息放入 `error.v1`"的说法不成立：`apperr.ToEnvelope` 使用包含原因的 `Error()`。现已更正。
  - 本轮复现了受影响的入口：根命令的未知子命令、`cobra.NoArgs` 命令多出的参数、其他命令无法解析的 flag 值和未知 flag。文本模式和 `--json` 模式都会回显。
  - 已作为独立的安全维护任务登记在 implementation-plan M6"发布前必须关闭的遗留安全问题"，列出了入口、复现方式和修复方向，最晚在 v0.1.0 发布之前关闭。本轮不修改 `root.go` 的整体错误处理。
- **同类遗留问题：其他命令的文本输出没有转义控制字符**。本轮在内存中确认，`proposal list`、`proposal show`、`review` 的确认摘要、`baseline show`、`baseline history` 原样输出提案中的 ESC。同样登记在 M6，未修复。
- WP1.6 的两项 NON-BLOCKER 保留：审核预览的修订类别可能与最终类别不同；`baseline history` 没有展示后续处理理由。

**接受的风险**：沿用审查列出的既有边界，没有接受新的风险。

## 34. WP1.7 最终定向复审（codex-review-028）与验收

**审查结论**：**GO**，BLOCKER 为 0，没有新增 NON-BLOCKER。审查目标 `58659cd`，增量基准 `78ca882`；只复核审查 027 的 B1、B2、B3 及直接回归。

- **审查 027 的 B1 关闭。** `internal/termtext` 转义 ESC、其他 C0、DEL、C1、双向控制字符和非 UTF-8 字节。`question` 的列表、确认摘要和结果行都接入了转义。JSON 解码后的值和数据库中的值保持原样，取消确认时不写入。审查方另外确认：CRLF 与缩进组合时，续行仍保持缩进，没有形成新的视觉边界问题。
- **审查 027 的 B2 关闭。** 未知子命令返回 `invalid_input`（退出码 2），不回显，只建议已有的命令名；JSON 模式输出合法的 `error.v1`。只输入 `question` 或使用 `--help` 时返回 0。
- **审查 027 的 B3 关闭。** 只对 `question` 命令解析未完成的 flag 错误重新判断 JSON 请求。审查方用独立的参数矩阵（46 组公开 `Execute` 调用）覆盖了参数顺序、布尔值、重复 flag、取值 flag、简写、未知 flag 和 `--`，没有误判，也没有回显。
- 审查方在隔离的 `78ca882` 副本上重放同一组断言，复现了三项原始缺陷，失败发生在被测行为上。
- 最终提交的三平台 CI 8/8 通过（运行 38803933170，Ubuntu 带 race detector）。

**验收**：产品负责人于 2026-10-16 验收 WP1.7，PR #10 已合并（合并提交 `867bd60`）。M1 尚未整体完成，WP1.8 未开始。

**WP1.7 的验收范围**（逐项证据见 PR #10 描述）：

- MVP-BL-17（含 `question-list.v1` 的条件约束与反例 SCH-131 ~ SCH-139）；
- `question answer`、`question dismiss` 的 MVP-CH-01、CH-02 验证；
- 关闭问题的前置条件与双连接并发（data-model §7.1）；
- MVP-BL-25 中"问题先关闭、条目后接受"时的关联显示；
- `question` 命令的终端显示安全与命令行错误契约（审查 027 B1 ~ B3）。

**仍未完成、随后续工作包验证**：

- 阻塞问题的推导（`BLOCKING_QUESTION`、BL-25 的阻塞部分）：M2。
- PreToolUse 对 `question answer`、`question dismiss` 的拦截（MVP-ACC-03、E2E-7）：M3。
- `target set`、`scope confirm`、审核中附带的范围确认：WP1.8。

**继续记录的事项**：

- implementation-plan M6 中登记的两项发布前安全问题仍未关闭：其他命令回显不可信参数（审查 027 N1）；其他命令的文本输出没有转义控制字符。前者随后作为独立的安全修复处理（分支 `fix/cli-untrusted-argument-errors`）。
- WP1.6 的两项 NON-BLOCKER、`doctor` 的 git 版本查询隔离、本地 Windows 测试超时，维持原安排。

**接受的风险**：没有新的风险接受。通道识别仍沿用已批准的防误操作边界，不是身份认证。

审查原文按原样入库。

## 35. 命令行错误回显不可信参数的独立安全修复（codex-review-027 N1）

**来源**：codex-review-027 NON-BLOCKER N1，登记于 implementation-plan M6"发布前必须关闭的遗留安全问题"。这是隐私缺陷，不是已接受的风险。本修复不属于任何工作包，单独提交、单独审查。

**范围**：只处理命令行本身的错误脱敏，以及直接相关的错误输出契约（architecture §9.1"命令行错误"）。以下内容不变：

- 业务错误的语义；
- 人类专属操作的通道防护；
- 数据模型与迁移；
- 其他命令文本输出中的终端控制字符（同在 M6 登记，另行处理）；
- WP1.6 的 N1、N2；
- `doctor` 的 git 版本查询隔离；
- 本地 Windows 测试超时；
- WP1.8。

**根因**：cobra 和 pflag 的错误消息会原样引用它们无法使用的内容。`executeAs` 把这些错误作为原因包装，`printError` 和 `apperr.ToEnvelope` 都使用包含原因的 `Error()`，因此原值进入了 stderr 和 `error.v1.message`。本修复之前复现的问题：

- 根命令的未知子命令（文本和 `--json`）；
- `cobra.NoArgs` 命令多出的参数，例如 `proposal list <x>`；
- 未知 flag、未知简写、不合法的布尔值，例如 `--pending=<x>`；`--json` 写在前面时，原值进入 `error.v1.message`。
- 另外两个问题：`help <x>` 在 stdout 回显 `<x>` 并返回 0；命令组遇到未知子命令时显示帮助并返回 0。

**修复**（`internal/cli/cmdline.go` 的 `secureCommandLine`，在 `newRoot` 中对整棵命令树调用一次）：

- 所有命令的 flag 错误都使用固定消息（`flagError`），并标记为 `unparsedFlags`；`--json` 按 pflag 的规则从参数中判断（`jsonInArgs`，规则同 WP1.7）。
- 根命令和每个命令组拒绝未知子命令（退出码 2），只建议已有命令的名字；单独输入时显示帮助并返回 0；为这个错误接受 `--json`。
- 每个命令的参数个数检查保留原来的规则，cobra 的消息换成固定文本和该命令的用法。
- `help` 遇到未知主题时返回退出码 2，不回显。
- `executeAs` 不再保留 cobra 错误的文本。
- WP1.7 中 `question` 命令组自己的那一份实现已经删除，统一由 `secureCommandLine` 提供。

**行为变化**：

- 根命令和命令组接受 `--json`，所以 `--json` 也可以写在命令之前，例如 `specmetry --json question list`。以前根命令不认识这个 flag，会拒绝这种写法。
- 根命令的帮助中多了一行 `specmetry [flags]` 的用法。
- 其余正常命令、`--help` 和业务错误不变。

**回归测试**（经过真实的命令入口；合成密钥在运行时随机生成，`testutil.NewSecretCanary`，测试源码中没有形似密钥的固定字面量）：

- `TestCommandLineErrorsQuoteNothing`，63 种命令行，覆盖：
  - 根命令、`help`、四个命令组的未知子命令；
  - 全部 `NoArgs` 命令，以及参数个数固定的命令；
  - 未知 flag 和简写、不合法的布尔值、缺少值的 flag、`--since` 的非数字值；
  - `--json` 在前、在后、为 `false`、重复出现；`--` 之后的 `--json`；作为字符串 flag 值的 `--json`。

  每种都断言：退出码 2；stdout、stderr 中都没有 canary；要求 JSON 时 stdout 是通过 schema 校验的 `error.v1`、`code` 为 `invalid_input`、stderr 为空；否则 stdout 为空、只有文本错误；没有业务写入；数据目录中没有 canary。
- `TestCommandLineStillWorks`：各层帮助返回 0；`version` 和 `--json` 读取命令正常；`--json` 写在命令之前也正常；拼写建议只包含已有命令名。
- WP1.7 的 `question` 测试全部保留并通过。
- **修复前**（`4125dfe`）：`TestCommandLineErrorsQuoteNothing` 有 40 处回显、5 处错误地返回 0、11 处缺少 `error.v1`。

**状态**：已修复，等待独立审查；审查通过之前，M6 中的这一项不关闭。

## 36. 安全修复 PR #11 的代码审查（codex-review-029）的处理

**审查结论**：NO-GO。审查目标 `9879c56`，有 1 项 BLOCKER、1 项 NON-BLOCKER。审查确认：已注册的根命令、命令组和叶子命令都受到统一控制；常规 JSON 错误模式正确；业务错误、人工通道和确认流程没有回归。本轮不需要新的产品决策。

本轮只修复 B1 及直接相关的测试。数据模型、迁移、业务错误和人工通道都不变。审查原文按原样入库。

### B1：cobra 动态加入的隐藏补全命令绕过统一脱敏

**根因**：

- cobra v1.10.2 在 `ExecuteC` 中调用 `initCompleteCmd`，临时加入隐藏命令 `__complete`（别名 `__completeNoDesc`）。只有 `root.Find(args)` 解析到这个名字时，它才保留这个命令。
- 这个命令在 `secureCommandLine` 遍历之后才出现，所以不受统一处理。它自己解析剩余的命令行，用 `CompErrorln` 把原始错误直接写到 `os.Stderr`；设置 `BASH_COMP_DEBUG_FILE` 时还写入该文件。之后输出补全指令（`:0`），并以 0 退出。
- `root.go` 的 `DisableDefaultCmd` 只关闭了可见的 `completion` 命令。

**修复**：Specmetry 不提供 shell 补全，所以在 cobra 执行之前拒绝这类命令行（`completionRequest`，在 `executeAs` 调用 `ExecuteContextC` 之前执行）：

- 判断方式与 cobra 的 `initCompleteCmd` 相同，并且在相同的状态下进行：
  - 先加入 help 命令（`ExecuteC` 在 `initCompleteCmd` 之前同样会加入）；
  - 再在根命令下加入一个同名、同别名、从不运行的占位命令；
  - 然后调用 cobra 自己的 `Find`，根命令的 flag、flag 的值和 `--`，都按 cobra 的方式解析。
- 解析到占位命令时，返回 `invalid_input`（退出码 2），消息为固定文本，不调用 `ExecuteContextC`。因此 cobra 的 `initCompleteCmd`、补全处理器和 `CompDebug` 都不会运行，不输出补全结果，也不写调试文件。占位命令随即移除。
- JSON：只看补全命令之前的根级 flag（`--json`、`--json=true` 为 JSON，`--json=false` 或最后为 false 时为文本）。补全命令之后的内容是被补全的命令行，其中的 `--json` 不算。
- `executeAs` 收到 nil 参数时改为空参数，不让 cobra 改读 `os.Args`，保证判断和 cobra 使用同一组参数。

**为什么能在隐藏处理器运行前拦截所有受支持的调用形式**：

- cobra 只在 `ExecuteC` 中、且只在 `Find(args)` 解析到这个名字时，才会运行补全处理器。
- `completionRequest` 在调用 `ExecuteC` 之前，用同一组参数、同样的命令树状态和 cobra 自己的 `Find` 做这个判断。cobra 会运行补全的每一种形式都会被拒绝，包括根级 flag 在前、别名、`--json=<布尔值>`。
- 判断为否时，cobra 的 `Find` 也不会解析到补全命令，`initCompleteCmd` 会立即移除它，补全处理器不会运行。
- 没有复制 cobra 的私有代码，不修改环境变量，也不在事后过滤 stderr。

**回归测试**（真实 CLI 二进制：测试中从 `cmd/specmetry` 构建，作为独立进程运行；合成密钥在运行时随机生成）：

- `TestShellCompletionIsRefused`，14 种命令行：
  - 审查中的 A、B；
  - 根级 `--json`、`--json=true`、`--json=false`、`--json --json=false` 在前；
  - `--json` 加 `question answer -m <canary>`；
  - 补全行内部的 `--json`；
  - 只有 `__completeNoDesc`；
  - 设置 `BASH_COMP_DEBUG_FILE` 为新文件，或为已有文件。

  每种都断言：退出码 2；没有补全结果（`:0`、`Completion ended`、`[Debug]`）；进程自己的 stdout 和 stderr 中都没有 canary；要求 JSON 时 stdout 是通过 schema 校验的 `error.v1` 且 stderr 为空，否则只有文本拒绝；新调试文件不存在，已有调试文件内容不变；数据目录逐个文件哈希不变。
- `TestShellCompletionWordIsData`：业务参数恰好是 `__complete` 或 `__completeNoDesc` 时，命令照常执行并给出自己的错误，不被当作补全命令；没有写入。覆盖了 `-m` 的值、`proposal show` 和 `source add` 的参数、`--` 之后、未知 flag 之后、`help` 的主题。
- **修复前**（`9879c56`）：每种补全形式都返回 0 并输出 `:0`，canary 出现在进程的 stderr 中，调试文件被创建，已有的调试文件被追加。
- 直接回归：`TestCommandLineErrorsQuoteNothing`、`TestCommandLineStillWorks`、`TestQuestionCommandLineErrors`、`TestProposalShowDoesNotEchoItsArgument`、`TestUnknownCommandAndFlagAreInvalidInput`，以及人类专属命令的 PTY 测试，都通过。

### NON-BLOCKER（登记，不在本轮修改）

- **N1**：`TestCommandLineStillWorks` 对成功 JSON 输出只检查以 `{` 开头，断言偏弱。审查方已独立确认六类正常输出符合各自的 schema，当前没有功能缺陷。改为按具体 schema 校验，留待后续处理。
- 其他命令文本输出中的终端控制字符、`doctor` 的 git 版本查询隔离、WP1.6 的 N1 和 N2，维持各自的安排。

**状态**：B1 已修复，等待定向复审（codex-review-030）。审查通过之前，M6 中的这一项不关闭。

## 37. 安全修复 PR #11 的最终定向复审（codex-review-030）与关闭

**审查结论**：**GO**，BLOCKER 为 0。审查目标 `30be825`。审查原文未提供文件，未入库；结论按产品负责人的通知记录。

- **审查 029 的 B1 关闭。** cobra 的隐藏补全命令（`__complete`、`__completeNoDesc`）在 cobra 执行之前就被拒绝，不再把命令行写入进程的 stderr 或 `BASH_COMP_DEBUG_FILE`。
- PR #11 已合并（合并提交 `a91016e`，2026-10-16）。

**关闭**：implementation-plan M6 中"CLI 错误回显不可信参数"标记为已完成，原登记内容和关闭依据都保留在该条目中：

- 修复：`secureCommandLine`（审查 027 N1，§35）和补全拒绝（审查 029 B1，§36）；
- 证据：`TestCommandLineErrorsQuoteNothing`、`TestShellCompletionIsRefused`、`TestShellCompletionWordIsData` 在三平台 CI 中运行通过；修复前后的对照见 §35、§36。

**保留的 NON-BLOCKER**：审查 029 N1，`TestCommandLineStillWorks` 对成功的 JSON 输出只检查以 `{` 开头，待改为按具体 schema 校验。

**M1 状态不变**：M1 尚未整体完成，WP1.8 尚未开始。M6 中仍未关闭的遗留问题：其他命令的文本输出没有转义终端控制字符（随后在分支 `fix/cli-terminal-text-escaping` 中处理）；`doctor` 的 git 版本查询隔离。WP1.6 的 N1、N2 维持原安排。

## 38. 终端输出控制字符的独立安全修复（implementation-plan M6）

**来源**：codex-review-027 B1 的同类问题，在审查 027 的处理中登记于 implementation-plan M6"发布前必须关闭的遗留安全问题"。本修复不属于任何工作包，单独提交、单独审查。

**范围**：只处理命令显示给人的文本中的终端控制字符，以及直接相关的确认显示。以下内容不变：

- 数据库、提案、基准、问题和来源中的原值；
- JSON 输出及 schema 的业务字段；
- 迁移与业务状态转换；
- CLI 参数脱敏与补全拦截（§35 ~ §37）；
- `doctor` 的 git 版本查询隔离；
- WP1.6 的 N1、N2；
- 本地 Windows 测试超时；
- WP1.8、M2。

**修复前的问题**（`a91016e`，在内存中逐字节检查）：`question` 之外的命令都把动态文本原样写到终端：

- 已确认的入口：`proposal list` 和 `proposal show`（摘要、理由、来源与代码引用路径、标题）、`review` 的确认摘要与结果、`baseline show`（标题、AC 的陈述）、`baseline history`（摘要、操作者显示名）、`propose --apply` 的确认摘要、`source add`、`source list` 和 `source ack` 的确认摘要（路径、理由）、`init`（项目名中的双向控制字符）。
- 原样输出的字节：ESC 序列（CSI、OSC）、CR、BEL、BS、DEL、C1、双向控制字符。
- 确认字符串：来源路径中有控制字符时，`source ack` 和 `remove` 仍要求输入这个路径，人无法照着输入。

**修复**（规则见 architecture §9.1"终端显示"）。两处边界，使用同一个 `termtext.Escape`：

1. **命令输出**：`newRoot` 把 stdout、stderr 包装为 `termtext.Writer`。所有文本输出、错误文本和帮助都经过转义；新命令打印的文本也自动受保护。只有 `writeJSON` 通过 `termtext.Raw` 绕过这层转义，保持标准 JSON 编码。
2. **人工确认**：`channel.Guard` 对任何 writer 都转义操作名和摘要，续行缩进在该行之下，多行文本因此无法伪装成摘要中的另一行。确认文本用 `%q` 显示，并按原值比较，比较语义不变。

另外：

- 单行文本（路径、显示名、项目名）使用新增的 `termtext.Line`，LF 和制表符也显示为转义。
- 多行文本（摘要、理由、AC 的陈述）的续行缩进在字段之下。先缩进再规范化 CRLF，对齐不受影响。
- `question` 命令改为依赖这两处边界，不再单独转义。`Escape` 是幂等的，输出不变，原有测试照常通过。
- 路径中有控制字符、换行或制表符时，`source ack` 和 `remove` 改为要求输入来源的 ID（architecture §6.1.3）。这沿用了已有的规则：路径首尾有空格时也使用 ID。
- `source` 命令的确认改为从命令的 stdin 读取，与 `review` 和 `question` 一致。真实进程中这是同一个 stdin，因此确认可以完全在内存中测试。

**回归测试**（在内存中进行，不向真实终端发送控制序列）：

- `TestTextOutputIsShownSafely`：
  - 构造：通过公开命令登记一个路径含双向控制字符、中文、日文和 emoji 的来源；提交的提案中，摘要含 CSI、OSC、CR、BEL、BS、DEL、C1、双向控制字符和 CRLF 多行，标题含 OSC 和双向控制字符，AC 的陈述含 ESC、LF 和 CR，并带有来源引用和代码引用；配置中的显示名含 OSC 和双向控制字符。
  - 输出：`proposal list`、`proposal show`、`review`（确认取消后再确认）、`baseline show`、`propose --apply`（确认取消后再确认）、`baseline history`、`source add`、`source list`、`source ack`（确认取消后再用 ID 确认）、`source remove` 的确认，以及引用参数的错误文本中，都没有原始控制字符；每一处都能看到明确的转义，多行文本的续行有缩进，中文、日文和 emoji 原样显示。
  - 取消确认时不写入。
  - `proposal show --json` 解码后，以及数据库中的摘要、陈述、操作者和路径，都是原值。
- `TestInitTextIsShownSafely`：项目名中的双向控制字符被转义，`init --json` 保留原名。
- `TestGuardShowsTheSummarySafely`（channel）：写到原始缓冲区的确认提示中没有控制字符，续行有缩进，完整提示逐字比对；输入转义后的文本不能完成确认。
- `internal/termtext`：`Line`、`Writer`、`Raw`、`NewWriter` 的单元测试。
- **修复前**（`a91016e`）：`TestTextOutputIsShownSafely` 在 `source add`、`source list`、`proposal list`、`proposal show`、`review` 确认摘要、`baseline show`、`propose --apply` 确认摘要、`baseline history` 上，都检出原始控制字符（例如 `proposal list` 中有 ESC、CR、BEL、BS、DEL、U+009B、U+202E），并在 `source ack` 要求输入原始路径处失败；`TestInitTextIsShownSafely` 检出 U+202E。
- **变异检查**：去掉 writer 包装后 `TestTextOutputIsShownSafely` 失败；去掉 `Guard` 中的转义后 `TestGuardShowsTheSummarySafely` 失败。
- 直接回归：`question`、`review`、`source`（包括真实 PTY 的确认测试）、`propose`、`init`、`doctor` 以及命令行错误的测试，全部通过。

**状态**：已修复，等待独立审查；审查通过之前，M6 中的这一项不关闭。

## 39. 终端输出安全修复 PR #12 的代码审查（codex-review-031）的处理

**审查结论**：NO-GO。审查目标 `db81f5a`，基准 `a91016e`，有 1 项 BLOCKER、2 项 NON-BLOCKER。审查确认：完整写入时统一转义边界有效，JSON 旁路只在 `writeJSON`，已登记来源的 ID 确认正确，没有新的绕过路径。本轮不需要新的产品决策。

本轮只修复 B1 及直接回归。`channel.Guard` 的确认逻辑、`Escape`、`Line`、`Raw` 和 JSON 旁路的语义都不变。审查原文按原样入库。

### B1：`termtext.Writer` 吞掉短写入，破坏人工确认的 fail-closed 保证

**根因**：`Writer.Write` 把转义后的文本写到底层 writer，但忽略底层实际写入的字节数，总是返回 `len(p), nil`。底层短写入且没有错误时，`channel.Guard` 以为提示已完整显示，于是读取输入，业务操作得以执行。这违反了 architecture §5.1：提示只显示了一部分时，应返回退出码 6，不读取输入，不写入。PR #12 把所有命令的 stdout 都接入这个 Writer，所以这是本 PR 引入的回归。审查方的复现：经包装后，`question answer` 返回 0，读取一次输入，Q-001 从 `open` 变为 `answered`，seq 加一；基准提交在同一反例下返回 6，数据不变。

**修复**（只改 `Writer.Write`）：

- 先生成转义后的文本，写到底层 writer。
- 底层返回错误时照原样传递。
- 计数超出范围（负数，或大于转义后的长度）时返回错误。
- 没有错误但写入字节数不等于转义后文本的长度时，返回 `io.ErrShortWrite`。按转义后的长度判断，因为转义后的文本可能比原文长。
- 只有转义后的文本全部写入、且没有错误时，才返回 `len(p), nil`。失败时不报告原文的任何部分已写入（无法判断原文的哪一部分到达了终端），也不重试。

**回归测试**：

- **Writer 层**（`TestWriterReportsShortWrites`）：每种情况都检查返回的 `n`、错误，以及底层实际收到的字节。
  - 普通 ASCII 只写入一半；
  - 含 ESC 的文本，转义后变长、底层恰好写入 `len(p)` 字节；
  - 5 字节转义为 8 字节、底层写入 4 字节（审查中的例子）；
  - 零字节写入且没有错误；
  - 完整写入；
  - 部分写入并返回错误；
  - 全部写入但返回错误；
  - 底层多报、负数计数。
- **Guard 层**（`TestGuardFailsWhenThePromptCannotBeShown`）：每种故障 writer 都分别直接传入、以及经 `termtext.NewWriter` 包装后传入。都必须返回退出码 6，不读取任何确认输入；完整写入时（直接或包装）正确确认仍能成功。
- **公开 CLI 写事务**（`TestConfirmationNotShownWholeWritesNothing`）：通过真实命令入口执行 `question answer Q-001 -m review031-confirmation`，human 通道，stdout 每次只接收一半字节，正确的确认输入已经准备好。结果：退出码 6，确认输入读取 0 次；Q-001 仍为 `open`，`answered_seq` 不变，全局 seq 不变，基准、提案及其他业务事实不变。改为完整写入后，同一命令可以正常完成回答。
- **修复前**（`db81f5a`）：Writer 层的短写入、零字节和计数异常都报告成功；Guard 层经包装的"短写入"和"零字节写入"放行了确认；CLI 层返回 0，读取 1 次，问题被错误回答。
- 直接回归：文本安全显示、`review` 与 `propose --apply` 的确认、`source` 的真实 PTY 确认、取消与 Agent 拒绝、`error.v1` 与正常 JSON、数据库原值、命令行错误脱敏与补全拦截，都通过。

### NON-BLOCKER（登记为独立维护事项，见 implementation-plan M6 之后的"独立维护事项"）

- **N1**：首次登记特殊路径的非文档来源时，`source add --allow-any-type` 还没有来源 ID，部分路径无法用单行输入确认。没有错误放行。登记了复现证据和修复方向，本轮不重构 `source` 命令。
- **N2**：在 human 通道使用 `--json` 时，取消确认后，确认提示与 `error.v1` 混在同一个 stdout 中。基准提交同样存在，不是本 PR 引入的回归，也不是已接受的风险。登记了复现证据和修复方向。
- 其他遗留事项维持原安排：`doctor` 的 git 版本查询隔离、审查 029 N1、WP1.6 的 N1 和 N2。

**状态**：B1 已修复，等待定向复审（codex-review-032）。审查通过之前，M6 中"其他命令的文本输出没有转义终端控制字符"这一项不关闭。

## 40. 终端输出安全修复 PR #12 的最终定向复审（codex-review-032）与关闭

**审查结论**：**GO**，BLOCKER 为 0，没有新增 NON-BLOCKER。审查目标 `d69baf9`，修复提交 `f345c2b`，修复前基准 `db81f5a`；只复核审查 031 的 B1 及直接回归。审查原文按原样入库。

- **审查 031 的 B1 关闭。** `termtext.Writer` 只在底层完整接收转义后的文本、且没有错误时才报告成功；短写入、零字节写入和计数异常都返回错误，`channel.Guard` 恢复了 fail-closed。审查方在 Writer、经包装的 Guard、公开 CLI 写事务三个层面做了故障注入：修复前复现问题，修复后都正确拒绝，不读取确认输入，也不写入。
- PR #12 已合并（合并提交 `26c787b`，2026-10-16）。

**关闭**：implementation-plan M6 中"其他命令的文本输出没有转义终端控制字符"标记为已完成，原登记内容和关闭依据都保留在该条目中（修复见 §38、§39）。

**保留的事项**：

- 审查 031 N1（首次登记特殊路径的非文档来源时确认可能无法输入）、N2（human 通道 `--json` 时确认提示与 `error.v1` 混入 stdout），都是独立维护事项。N2 随后在分支 `fix/cli-json-interactive-output` 中处理。
- 审查 029 N1（成功 JSON 输出的断言偏弱）、`doctor` 的 git 版本查询隔离（M6）、WP1.6 的 N1 和 N2、本地 Windows 测试超时，维持原安排。

**M1 状态不变**：M1 尚未整体完成，WP1.8 尚未开始。

## 41. 人工确认与 JSON 输出的分流（codex-review-031 N2）

**来源**：codex-review-031 NON-BLOCKER N2，登记为独立维护事项（implementation-plan，M6 之后）。不是已接受的风险。本修复不属于任何工作包，单独提交、单独审查。

**问题**：人类专属命令带 `--json` 时，确认提示和最终的 `error.v1` 写在同一个 stdout 中。取消确认后，stdout 不是一份完整的 JSON 文档。修复前（`f0ebd58`）的复现：human 通道执行 `question answer Q-001 -m x --json` 并取消，stdout 以确认提示开头，接着是 `error.v1`，整体无法作为一份 JSON 解析，stderr 为空。已有测试用 `jsonPart` 跳过 stdout 开头的提示，正是这个问题的表现。

**设计核对**：

- architecture §5.1 规则 1：stderr 不参与通道识别。本修复不改变识别规则（`Decide` 仍只看 stdin 和 stdout），`Detection` 只是新增一个"stderr 是否为终端"的事实字段。
- §5.1 规则 2 原文是"在标准输出显示操作名称和摘要"。按产品负责人给出的规则，带 `--json` 时改为显示在 stderr，这是这条规则的直接修订，已同步修改。stderr 不是终端时返回退出码 6，与规则 2 中"确认提示无法完整输出时返回退出码 6"属于同一原则。
- §9.1"`--json` 时错误输出完整的 `error.v1`，stderr 为空"：增加一个受限的例外，只有人在交互中看到的确认提示可以出现在 stderr。
- 与已批准的 `check run --json`（stdout 只有一份 JSON，其余输出转到 stderr，MVP-EV-12）原则一致。MVP-CH-02（`--json` 不改变识别结果）不受影响。
- 没有发现需要额外产品取舍的冲突。

**修复**：

- 新增 `internal/cli` 的 `confirm(cmd, d, op)`，作为所有人类专属命令唯一的确认入口：
  - 不带 `--json`（或 `--json=false`）时，确认显示在 stdout，行为不变；
  - 带 `--json` 时，先拒绝 Agent 通道（退出码 3，不显示提示）。stderr 不是终端时，不显示、不读取，返回退出码 6，stdout 输出一份 `error.v1`；否则确认显示在 stderr，stdout 只有命令的输出。
- `channel.Guard` 不变：完整输出检查、短写入保护（stderr 同样经过 `termtext.Writer`）、确认原文比较都照旧有效。
- `question answer` 和 `dismiss`、`review`、`propose --apply`、`source ack` 和 `remove`、`source add --allow-any-type` 都改为调用 `confirm`。`TestHumanOnlyCommandsConfirmThroughConfirm` 检查 CLI 源码中没有命令直接调用 `channel.Guard`，以后新增的人类专属命令（WP1.8）也必须走这个入口。
- 成功输出的契约不变：`propose --apply --json` 输出 `proposal-receipt.v1`，`source add --json` 输出 `source-add.v1`；其他人类专属命令成功时仍输出文本（`--json` 只影响错误），没有新增成功响应的 schema。
- 根级与子命令级 `--json`、`--json=false`、重复的 flag，都按已有的解析规则决定最终效果。没有引入 `--yes` 或任何自动确认。

**回归测试**：

- `TestConfirmationWithJSON`（human 通道，在内存中）：对上述 7 个确认逐一测试，带 `--json` 时：
  - 按 Enter 取消、输入错误、输入结束：退出码 3；stdout 恰好是一份通过 schema 校验的 `error.v1`（整个 stdout 按单个 JSON 文档解析，不只看首字符）；确认提示只在 stderr。
  - 读取失败：退出码 6，stdout 为 `error.v1`。
  - stderr 不是终端、stderr 短写入：退出码 6，确认输入读取 0 次，stdout 为 `error.v1`。
  - Agent 通道：退出码 3，任何输出流中都没有确认提示。
  - 不带 flag 和 `--json=false`：确认在 stdout，stderr 是文本错误。
  - 以上全部不写入（业务状态、seq、版本、来源确认与移除都不变）。
  - 最后逐一确认：`propose --apply` 和 `source add` 的 stdout 恰好是一份通过 schema 校验的文档，其他命令输出成功文本；确认提示都只在 stderr；问题被回答或驳回、基准版本加一、来源被确认或移除、新来源被登记，与预期一致。
- `TestConfirmationWithJSONInTerminal`（真实 PTY，通道和输出流都是进程实际拥有的）：
  - `--json` 时确认能正常显示并完成，取消时为退出码 3 和 `error.v1`；
  - stderr 重定向到文件时：带 `--json` 返回退出码 6，没有显示确认，stderr 为 0 字节，`error.v1` 在 stdout，没有写入；不带 `--json` 时确认在 stdout 正常完成。
- 原来用 `jsonPart` 跳过提示的测试，改用 `wholeJSON`，要求 stdout 恰好是一份 JSON 文档；检查"拒绝发生在提示之前"的测试，改为同时检查 stdout 和 stderr。
- **修复前**（`f0ebd58`）：`question answer --json` 取消后，stdout 不是一份 JSON，提示在 stdout，stderr 为空。

**范围外，不变**：业务状态模型、迁移、schema 业务字段、CLI 参数脱敏、补全拦截、终端转义规则、审查 031 N1、`doctor` 的 git 版本查询隔离、WP1.6 的 N1 和 N2、WP1.8、M2。

**状态**：已修复，等待独立审查；审查通过之前，维护事项 N2 不关闭。

## 42. 人工确认与 JSON 输出分流 PR #13 的代码审查（codex-review-033）与关闭

**审查结论**：**GO**，BLOCKER 为 0，有 1 项 NON-BLOCKER。审查目标 `511403d`，基准 `26c787b`。审查原文按原样入库。

- **审查 031 N2 关闭。** 审查方在隔离副本中对照：基准提交上取消后 stdout 是"确认提示加 `error.v1`"，无法整体解码；目标提交上提示在 stderr，整个 stdout 是一份通过 `error.v1` 校验的文档。另外独立验证了 16 组 JSON flag 组合、4 种 stderr 故障与短写入、真实 ConPTY 中的确认与 stderr 重定向，都没有回归。
- PR #13 已合并（合并提交 `fbd6dff`，2026-10-17）。implementation-plan 中的维护事项 N2 标记为已完成。
- 审查方说明了源码扫描测试 `TestHumanOnlyCommandsConfirmThroughConfirm` 的边界：它是字符串扫描，不能覆盖导入别名等语义等价的写法，只是辅助检查，不能代替人类专属命令自身的验收。

**N1：自定义 Writer 与进程 stderr 的终端属性可能不一致。** 登记为独立维护事项（implementation-plan，M6 之后），不是已接受的风险。

- `cli.Execute` 接受调用者传入的 Writer，而终端检测针对进程的 `os.Stderr`。
- 正式二进制唯一的调用传入的就是 `os.Stdout`、`os.Stderr`，不受影响。
- 保留 `Execute` 的交互输出使用约束：交互时 stdout、stderr 必须是进程自己的终端流。

**其他事项**：审查 031 N1、审查 029 N1、`doctor` 的 git 版本查询隔离（M6）、WP1.6 的 N1 和 N2、本地 Windows 测试超时，维持原安排。

**M1 状态**：随后开始 WP1.8（M1 的最后一个工作包）；M1 尚未整体完成。

## 43. WP1.8 交付目标与交付范围的实现

**范围**：implementation-plan WP1.8，M1 的最后一个工作包。依据为 FR-TGT-01 ~ 07、data-model §4.2、§5.3、§5.5 第 11 步、§7.1。命令行为写入 architecture §6.1.7（新增）。

**设计核对**：

- 按已有决策（G2 冻结）实现，没有扩大产品语义，没有发现需要新增 PD 或 TD 的歧义。
- 以下实现选择不改变产品语义，已写入 architecture §6.1.7：
  - `target set` 允许当前解析不到提交的 ref（例如尚未创建的发布分支），摘要中如实说明；
  - 非 git 项目执行 `target set`，返回退出码 2（FR-INIT-09）；
  - ref 与活动目标相同、并且没有改名时，什么也不变（退出码 0，不分配 `seq`，保留范围）；显式改名视为新目标；
  - 确认期间目标被其他连接更改，返回退出码 5；
  - `--name` 默认为去掉前缀的分支名或标签名；
  - `scope confirm` 的确认文本为 `v<版本>`，与 `propose --apply` 一致；
  - 交互式确认自动列入所选需求下的全部有效豁免，并在摘要中逐条显示理由。`--file` 时由文件给出，按 V-T01 比较；
  - "可以修改默认选择"（MVP-TGT-08）在 CLI 中用 `--requirements` 实现，逐条勾选的界面由 M5 的 Dashboard 提供。

**实现**（提交）：

- `2fce111` schemas：`target-scope.v1` 增加 `uniqueItems` 和字段之间的条件约束，新增两个有效示例和反例 SCH-140 ~ 146。
- `e8e5ce6` 迁移 0006：新增两张表。已合并的迁移 0001 ~ 0005 没有修改。
  - `scope_confirmations`：带声明的 CHECK、自引用的 `seen_scope_confirmation_id`、`(target_id, project_id)` 外键和只追加触发器；
  - `scope_items`。
- `1c98544` `internal/scope`，以及 `gitx.ValidRefName`：
  - 目标的读取与设置：`SetTarget` 在一个 `seq` 内归档旧目标、新建目标；
  - 状态读取：`Read`；
  - 校验：`Check`，V-T01 按集合比较，V-T02 比较当前确认；
  - 写入：`Write`；
  - `target-scope.v1` 文档。
- `b87a73e` CLI：`target show`、`target set`、`scope show`、`scope confirm`。
- `101da05` 附带在审核中的范围确认：`review.ConfirmScope`。

**安全约束**：

- 两个人类专属命令都先调用 `RequireHuman`，再通过唯一的确认入口 `cli.confirm`（`TestHumanOnlyCommandsConfirmThroughConfirm` 通过）。没有第二套确认机制，没有 `--yes`。
- Agent 通道下都返回退出码 3，不读取 `--file`，不显示确认。包括：设置了 Agent 变量的伪终端、stdout 重定向、stdin 不是终端，以及带 `--json` 的情形。通道识别不是身份认证（architecture §5.1 已知边界）。
- 输入先做密钥扫描，命中时返回退出码 2，不回显；错误消息不引用输入。扫描范围包括 ref、名称、需求清单、备注，以及 `--file` 中的全部字符串。
- 文本输出经过 `termtext.Writer`。JSON 输出经 `writeJSON` 按 schema 校验后写出。

**验收覆盖**（测试先行：在旧代码 `1c98544` 上，新增和修改的 10 个 CLI 测试全部失败）：

| 验收项 | 测试 |
|---|---|
| MVP-TGT-01（M1 部分）、TGT-02 | `TestTargetShowAndSet`、`TestTargetAndScopeNeedAPerson` |
| MVP-TGT-03 | `TestScopeConfirm`、`TestTargetAndScopeNeedAPerson` |
| MVP-TGT-04 | `TestScopeConfirm`（`--requirements ","`、需求为空的 `--file`）、`TestTargetWithoutOne` |
| MVP-TGT-05、TGT-08 | `TestScopeConfirm` |
| MVP-TGT-06 | `TestScopeConfirm`（计数包括 `conflict` 条目）、`TestScopeInReview`（以审核之后的计数为准） |
| MVP-TGT-07 | `TestScopeInReview` |
| MVP-TGT-10（M1 部分） | `TestScopeConfirm`、`TestScopeInReview`；`internal/scope` 的 `TestCheckWaivers` 用非空集合验证 |
| MVP-TGT-11 | `TestScopeConfirmedMeanwhile`、`TestScopeInReview`；`internal/scope` 的 `TestConfirmRace`（两个连接） |
| MVP-CH-01、CH-02（`target set`、`scope confirm`） | `TestTargetAndScopeNeedAPerson` |
| `--json` 的确认规则、短写入 | `TestConfirmationWithJSON`（新增两个命令）、`TestTargetAndScopeConfirmationNotShownWhole` |

- `TestScopeConfirmedMeanwhile` 覆盖三种并发：两个会话先后提交、同一进程中的另一个连接、在伪终端中确认的另一个进程。
- `target show --json` 和 `scope show --json` 的每次成功输出，都按 `target-scope.v1` 整体校验。另外检查两个命令的输出完全相同，`target_id` 与 `target.id`、`seen_scope_confirmation_id` 与 `scope.confirmation_id` 一致。
- 每次拒绝都比较以下状态，验证没有写入、没有分配 `seq`：`seq`、全部目标、范围确认及其条目、基准、待审条目。
- 已有测试的调整：
  - `TestReviewRefusals` 中的范围，原先期望退出码 2（指向 `/scope`，这是 WP1.6 的拒绝）。现在它表示另一个目标的范围，期望退出码 5（指向 `/scope/target_id`）。
  - `state` 和 `confirmState` 增加了范围相关的表。

**推迟到 M2**：

- MVP-TGT-09：`status` 中的 `SCOPE_STALE`；
- MVP-TGT-01 中 `NO_SCOPE` 的推导；
- 真实豁免的生命周期，即 MVP-TGT-10 的 M2 部分；
- `init_progress` 对范围的推导（MVP-INIT-07）。

**范围外，不变**：`doctor` 的 git 版本查询隔离、审查 031 N1、审查 033 N1、审查 029 N1、WP1.6 的 N1 和 N2、M2 的推导、Claim、检查执行、验收决定、Dashboard、本地 Windows 测试超时。

**验证**（本地）：

- `scripts/check.sh` 中，`go mod tidy`、`go vet`、Markdown 链接检查、golangci-lint（0 issues）都通过。
- `go test ./...`：`internal/cli` 以外的包全部通过。
  - `internal/cli` 在本地 Windows 上触发 Go 默认的 10 分钟超时：运行到 `TestScopeInReview` 时超时，此前没有测试失败。
  - 单独以 `-timeout 30m` 运行，通过（973 秒）。
  - 脚本和全局超时都没有修改。
- web 检查（`npm ci && npm run check`）、`go build -o dist/ ./cmd/specmetry`、`go test ./web/` 都通过。
- golangci-lint 在三个平台（`GOOS=windows`、`linux`、`darwin`，`--build-tags nowebui`）都是 0 issues；`golangci-lint fmt --diff` 没有差异。
- WSL Linux（git 2.43.0）：交叉编译的 19 个测试二进制全部通过，其中 `cli` 通过 138 项、跳过 2 项。
  - `TestShellCompletion` 需要 go 工具链，因此不在其中；改用真实的 Linux 二进制重放补全拒绝：退出码 2，不输出 canary，不写调试文件，数据目录为空。
- 三平台 CI 的结果记录在 PR 中。

**状态**：已实现。首次独立代码审查为 codex-review-034（NO-GO），处理见 §44。M1 尚未整体验收。

## 44. WP1.8 代码审查（codex-review-034）的处理

**审查结论**：**NO-GO**，有 2 项 BLOCKER、1 项 NON-BLOCKER。审查目标 `90beb48`，基准 `fbd6dff`。审查原文按原样入库（`docs/reviews/codex-review-034.md`）。

- 审查方认可的部分，本轮不重新设计：Target 与 Scope 的主要事务和并发机制、V-T01、V-T02、审核附带范围的原子性（包括晚期失败时的全部回滚），以及迁移 0006。
- 两项 BLOCKER 都是新增的自由文本入口违反了已冻结的隐私校验规则（architecture §10.2"所有自由文本都不能包含代码围栏，由 schema 强制"）。没有新增产品决策。

**B1：交互式范围确认的备注绕过 schema**

- 问题：`scope confirm --file` 按 `scope.v1` 严格校验，而 `scope confirm -m` 只对备注做密钥扫描和 `scope.Check`。
  - 含代码围栏的备注被写入 `scope_confirmations.note`；
  - 1001 个字符的备注先进入人工确认，之后才因数据库长度约束返回 internal（退出码 1）。
- 修复：交互式确认按它所代表的完整 `scope.v1` 文档，使用与 `--file` 相同的 validator 校验，在确认之前完成（`scope.Input.Document`，architecture §6.1.7"确认范围"第 4 步）。
  - 文档补齐 `schema`、`target_id`、`baseline_version`、`source_set_hash`、`requirements`、`attestations`、`acknowledged_waivers`（总是数组）和 `seen_scope_confirmation_id`（首次确认为 null）。
  - `scope.Input.Note` 改为可选（`*string`，没有时省略）。没有 `-m` 时省略 `note`，所以合法的默认确认不受影响；没有 `--missing-goals-reviewed` 时省略 `missing_goals_reviewed`。
  - 没有另写备注的正则：长度、代码围栏都由 `scope.v1` 的 `plainText1000` 决定。
  - `scope.Check` 的业务前置条件和过期判定不变。文档由刚读到的状态构造，所以 schema 校验不会把过期改判为输入错误；写事务中的重新检查照旧。
  - 审核附带的范围，原本就由 `review.v1` 的 `confirmationInReview` 校验（没有 `baseline_version`），备注错误指向 `/scope/note`。本轮补了回归测试。
  - 错误只给出字段与问题类型，不回显输入（validator 的脱敏输出）。
- 一项需要说明的选择：`-m ""`（显式给出空备注）与文件中的 `"note": ""` 等价，都返回退出码 2（`/note`）。不带 `-m` 时才是"没有备注"。修复前，`-m ""` 被当作没有备注接受。
- 直接相关的测试调整：`--requirements REQ-001,REQ-001` 现在由 schema 的 `uniqueItems` 先拒绝，与 `--file` 相同：仍是退出码 2，指针由 `/requirements/1` 变为 `/requirements`。

**B2：交付目标名称允许代码围栏**

- 问题：`CheckName` 只检查 UTF-8、空值、长度和控制字符，`target-scope.v1` 的 `target.name` 也没有禁止代码围栏。代码围栏名称可以写入 `delivery_targets.name`，并能通过输出 schema 校验。
- 修复：
  - `CheckName` 拒绝含三个连续反引号的名称，原有的非空、单行、UTF-8、控制字符和 200 字符限制保持不变。
  - `SetTarget` 在写入处再调用一次 `CheckName`，作为第二道防线。
  - `target-scope.v1` 的 `target.name` 增加相同的代码围栏限制（新反例 SCH-147）。
  - 显式 `--name` 和由 ref 生成的默认名称经过同一校验，都在打开数据库之前完成：返回退出码 2，指向 `--name`，不显示确认，不写入。默认名称不合法时，消息提示用 `--name` 给出名称。
  - Git ref 的合法性规则和 `init` 的默认目标都没有改变。含代码围栏的分支用合法的 `--name` 仍可设置为目标。

**回归测试**（测试先行；修复前的结果为当前代码 `90beb48` 上的实际运行）：

| 测试 | 修复前 | 修复后 |
|---|---|---|
| `TestScopeNoteIsCheckedLikeAFile`：两个入口分别提交代码围栏备注、行内围栏、1001 字符、空备注；审核附带范围的围栏备注 | `-m` 的围栏备注返回 exit 0 并写入；1001 字符在确认之后返回 exit 1；`-m ""` 返回 exit 0；`--file` 已经是 exit 2 | 两个入口都是 exit 2（`/note`）：error.v1 合法，没有确认提示，`seq`、目标、范围、基准、提案、待审条目都不变，输出和数据目录中都没有合成内容；审核为 exit 2（`/scope/note`）。1000 字符（按字符计）、中文、日文、emoji、没有备注，两个入口都确认成功，并按原值保存（没有备注时为 NULL） |
| `TestTargetNameHasNoCodeFence`：显式 `--name` 含围栏、行内围栏、同 ref 改名为非法名称、ref 生成的默认名称含围栏；已有目标和范围确认 | 四种情形都返回 exit 0，并创建或归档目标 | 全部 exit 2（`--name`），没有提示，没有写入，`seq` 不变，原目标与范围确认不变，数据目录中没有合成内容。含围栏的 ref 加合法 `--name` 可以设置；中文、日文、emoji、单反引号、双反引号的名称都能设置并按原值读取 |
| `internal/scope` 的 `TestCheckName`、`TestSetTarget` | 代码围栏名称通过 | `CheckName` 和写入处的 `SetTarget` 都拒绝 |
| `internal/contract` 的 SCH-147 ~ 149 | SCH-147 被接受（SCH-148、149 已经被拒绝） | 全部被拒绝 |

- `scopeState` 增加提案计数，以验证拒绝时提案不变。

**直接回归**：以下测试照常通过，没有修改断言（除上面说明的一处指针变化）。

- V-T01：`TestCheck`、`TestCheckWaivers`，空集合与合成非空集合；
- V-T02：`TestConfirmRace`（两个连接）、`TestScopeConfirmedMeanwhile`；
- 审核附带范围的同事务提交与失败回滚：`TestScopeInReview`；
- 目标替换、无变更与并发保护：`TestTargetShowAndSet`、`TestTargetSetWhileConfirming`、`TestTargetWithoutOne`；
- 来源集合哈希、基准版本、待审与冲突计数：`TestScopeConfirm`；
- 人类专属命令都经过 `RequireHuman` 和 `cli.confirm`：`TestHumanOnlyCommandsConfirmThroughConfirm`、`TestTargetAndScopeNeedAPerson`；
- JSON 的 stdout 与 stderr 分流：`TestConfirmationWithJSON`；
- 终端转义与短写入保护：`TestTextOutputIsShownSafely`、`TestTargetAndScopeConfirmationNotShownWhole`；
- 迁移：没有新增或修改迁移，`internal/store/sqlite` 全部通过。

**N1：跨项目需求外键的测试可能由另一条外键先拒绝**（`internal/store/sqlite/scope_test.go`）。审查方已确认实际约束正确，不是生产缺陷。本轮只登记为独立维护事项（implementation-plan"独立维护事项"），不扩大 B1、B2 的修复范围。

**范围外，不变**：Target 与 Scope 的事务和并发设计、迁移 0001 ~ 0006、M2、Dashboard，以及已登记的遗留事项（`doctor` 的 git 版本查询隔离、审查 029 N1、031 N1、033 N1、WP1.6 的 N1 和 N2、本地 Windows 测试超时）。

**验证**（本地）：

- `scripts/check.sh`：`go mod tidy`、`go vet`、Markdown 链接检查、golangci-lint（0 issues）都通过。
- `go test ./...`：`internal/cli` 以外的包全部通过。
  - `internal/cli` 再次触发本地 Windows 上 Go 默认的 10 分钟超时，此前没有测试失败。
  - 单独以 `-timeout 30m` 运行，通过（864 秒）。
  - 脚本和全局超时都没有修改。
- web 检查、`go build -o dist/ ./cmd/specmetry`、`go test ./web/` 都通过。
- golangci-lint 在三个平台都是 0 issues；`golangci-lint fmt --diff` 没有差异。
- WSL Linux（git 2.43.0）：19 个测试二进制全部通过，其中 `cli` 140 项、`scope` 6 项。补全拒绝用真实的 Linux 二进制重放，结果不变。
- 三平台 CI 的结果记录在 PR #14 中。

**状态**：已修复。定向复审为 codex-review-035（GO），见 §45。

## 45. WP1.8 最终定向复审（codex-review-035）与验收

**审查结论**：**GO**，BLOCKER 为 0，有 2 项 NON-BLOCKER。审查目标 `7f65408`，修复提交 `f0dab52`，修复前基准 `90beb48`；只复核审查 034 的 B1、B2 及直接回归。审查原文按原样入库（`docs/reviews/codex-review-035.md`）。

- **审查 034 B1 关闭**：交互式范围确认与 `--file` 都按 `scope.v1` 校验。非法备注在确认之前拒绝；合法备注按原值保存；省略备注时保存为 SQL NULL；密钥扫描仍然有效。
- **审查 034 B2 关闭**：`CheckName`、写入处的 `SetTarget` 和输出 schema 都禁止代码围栏名称。
- 审查方在修复前的隔离副本中运行了新增的回归测试，复现了原缺陷。三平台 CI 的结论以 attempt 2 的日志为准（[run 38957016420，attempt 2](https://github.com/XKHoshizora/specmetry/actions/runs/38957016420/attempts/2)），attempt 1 的失败保留为独立事项（见下文 N2）。
**验收**：产品负责人于 2026-10-17 确认 WP1.8 通过代码验收，PR #14 已合并（合并提交 `19600ec`；master 包含审查提交 `7f65408`，两者的树相同）。合并后 master 的三平台 CI 8/8 通过（[run 39000934831](https://github.com/XKHoshizora/specmetry/actions/runs/39000934831)）。至此 WP1.0 ~ WP1.8 全部验收完成。

**N1**（沿用审查 034 N1）：跨项目需求外键的测试可能由另一条外键先拒绝。实际约束已验证正确。继续作为独立维护事项保留（implementation-plan"独立维护事项"）。

**N2：Windows 首次创建数据库期间，权限检查偶发 `Access is denied`**。登记为独立的可靠性维护事项（implementation-plan"独立维护事项"），不是 WP1.8 的 BLOCKER，也不是已接受的风险。证据：

- PR #14 的 CI [run 38957016420](https://github.com/XKHoshizora/specmetry/actions/runs/38957016420) attempt 1 中，`Go tests (windows-latest)` 失败：`internal/store/sqlite` 的 `TestCheckWhileTheDatabaseIsCreated` 在 round 17、check 11 返回 `Access is denied`。
- 只重跑失败的 job（attempt 2），8/8 通过。重跑通过不能证明根因已消失。
- 本地 Windows 上连续 30 次运行（每次 20 轮），全部通过。

**M1 状态**：九个工作包都已验收，但 **M1 里程碑尚未整体验收**。M1 退出条件的证据核对见 [m1-exit-evidence.md](m1-exit-evidence.md)：59 个 P0 项中 53 项 PASS，6 项 INCOMPLETE（都是验证证据不足，没有发现行为缺陷），因此暂不满足退出条件。是否补齐证据、何时交给 Codex 做 M1 最终退出条件核查，由产品负责人决定。（后续：6 项缺口由 PR #16 补齐，见 §46；项目名称修复见 §47、§48；证据报告第 2 版为 59 项 PASS。）

## 46. M1 退出证据补充 PR #16 的审查（codex-review-036）与合并

§45（WP1.8 的最终复审与验收）随 PR #15 合入；本节写于 PR #15 合并之前，编号排在 §45 之后。

**审查结论**：**GO**，BLOCKER 为 0。审查目标 `e1d2ea7`，基准 `19600ec`。审查原文按原样入库（`docs/reviews/codex-review-036.md`）。

- M1 退出条件证据核对中 6 项 INCOMPLETE 的 P0 验收项，现在都有充分的证据：
  - **MVP-CH-01、CH-02、BL-21、BL-27、PRV-07**：PR #16 补齐了测试。审查方核实了这些测试的前置条件与业务断言，并在隔离副本中独立运行，全部通过。
  - **MVP-SRC-07**："只为引用校验读取文本、不解析文档语义"由审查方按实际的读取、校验、持久化和输出调用链独立审查，结论为 **PASS**。
- 正式二进制测试（`TestBuiltBinary`）的证据有效。`nowebui` 构建与发布包的差异已如实说明。
- acceptance-criteria §1 的三类集成测试证据定义，产品负责人已正式批准（审查原文中记录了这次确认）。SRC-07 的 M1 / M2 划分，与审查 019 和 review-resolution §24 的既有边界一致。两者都维持现有设计。
- **合并**：PR #16 已合并，合并提交 `01bd50f`（2026-10-17），它的树与 `e1d2ea7` 相同。合并后 master 的三平台 CI 8/8 通过（[run 39012262429](https://github.com/XKHoshizora/specmetry/actions/runs/39012262429)）。
- 审查方保留的事项：
  - Windows 首次创建数据库期间偶发的 `Access is denied`：这次通过不代表已经解决；
  - 项目名称的代码围栏修复：见 §47；
  - M1 退出条件证据报告（PR #15）的更新：要引用合并后的提交和本次 SRC-07 的审查结论，不能把原来的 INCOMPLETE 无依据地改为 PASS。

  这些都不是已接受的风险。

**M1 状态**：九个工作包都已验收；6 项证据缺口已补齐。M1 里程碑尚未整体验收：项目名称的安全修复（§47）、证据报告的更新和里程碑的最终核查仍需单独完成。

## 47. 项目名称禁止代码围栏的独立安全修复

**来源**：implementation-plan M1"M1 最终验收之前必须完成的独立安全修复"。产品负责人已确定方向：项目名称与交付目标名称一样，禁止代码围栏（architecture §10.2；参照 codex-review-034 B2）。本修复单独提交、单独审查。

**修复前的行为**（测试在 `01bd50f` 上的实际运行）：

- `init --name` 中含代码围栏的名称被登记：退出码 0，写入数据库和 marker。
- 主 worktree 的目录名含代码围栏时，不带 `--name` 也会照样登记。
- `init.v1` 的 `project.name` 只限制长度，含代码围栏的名称能通过输出 schema 的校验。

**修复**：

- `project.validateName` 拒绝含三个连续反引号的名称，问题为"contains a code fence (three backticks)"。原有的非空、UTF-8、单行、控制字符与 200 字符限制保持不变，没有增加别的限制。
- 显式的 `--name` 和由目录名生成的默认名称经过同一校验，都在打开数据库之前完成，返回退出码 2：
  - `--name` 不合法时，指向 `--name`；
  - 默认名称不合法时，消息给出原因并提示使用 `--name`（这个错误与以前一样没有 details）。

  两种情形都不回显名称，不创建数据目录，不登记项目，不写 marker。
- `init.v1` 的 `project.name` 增加相同的代码围栏限制，新增反例 SCH-150。
- Git ref 的规则、交付目标名称的校验（WP1.8）、数据库结构和迁移都没有改变。

**旧项目的兼容**（产品负责人已正式批准方案 1，见 §48）：

- **问题**：早期的开发版本可能已把含代码围栏的名称登记进数据库和 marker。数据库的 CHECK 只限制非空和长度，所以这样的数据是可能存在的。`init` 不改名，`init.v1` 现在又不接受这个名称。
- **只收紧 schema 的后果**：对这样的项目再次执行 `init --json`，会以 internal（退出码 1）失败，消息为"the init.v1 output does not conform to its schema"。带 `--stage` 或在新的关联 worktree 中执行时，阶段变更和 worktree 登记会**先提交，然后才失败**。在只应用 schema 改动的隔离副本中实测，`seq` 从 2 变为 3，阶段变为 `incomplete_docs`，worktree 从 1 个变为 2 个。
- **采用的处理**：`init` 找到已登记的项目时，先用同一规则检查登记的名称；不合法时返回退出码 2（`invalid_input`，指向 `name (registered)`），说明这个名称不再被接受、`init` 不改名、什么都没有改变。
  - 不引用这个名称。修复前，"不改名"的错误会用 `%q` 原样引用登记的名称。
  - 业务事实（名称、阶段、settings、worktree、`seq`）、marker、exclude 与仓库文件都不变；不静默改名，不迁移。这一检查需要打开数据库读取登记的事实，普通连接可能更新 SQLite 的辅助文件（例如 `-shm` 的修改时间），所以不是"数据库未打开"，也不是"数据目录绝对零变化"（codex-review-037 已区分）。
  - 主 worktree 的目录名含代码围栏、又没有给出 `--name` 的旧项目，在打开数据库之前就因目录名被拒绝；给出 `--name` 时，同样因登记的名称被拒绝。
- **后果**：这样的项目暂时不能通过 `init` 修改阶段、补写 marker 或登记关联 worktree。现在没有改名的命令（`project set` 不在 M1）。
  - 其他已实现的命令不显示项目名称。`source list`、`target show` 有动态测试；审查 037 静态核查了其他命令使用的 `project.Lookup` 路径，它们不经过这项检查。这不是对所有命令的完整 E2E 兼容保证。
- **决定**：产品负责人正式批准方案 1（codex-review-037 记录，§48）。当时列出的备选方案，保留如下：
  1. 维持当前处理：有说明的拒绝，不写入，不改名。
  2. 不在 `init.v1` 的输出中禁止代码围栏，只在登记新名称时拒绝。这样旧项目可以继续执行 `init`，但 schema 不能保证输出中没有代码围栏。
  3. 把 v0.1.0 之前的开发版本数据定为不需要兼容，处理方式同第 1 种，并在文档中声明。

  不建议迁移或改写旧数据，这需要修改冻结的数据模型。

**Schema 与数据库**：

- 新登记的名称由写入前的校验保证不含代码围栏，与 `init.v1` 一致。
- 数据库没有对应的 CHECK（本次不改迁移），所以旧数据可能不符合 schema。对这样的数据，`init` 返回有说明的拒绝，而不是输出不合法的文档。
- M2 的 `status.v1` 也有 `project.name`，实现时需要同样考虑。

**回归测试**（测试先行，在 `01bd50f` 上都失败）：

- `TestInitRefusesACodeFenceInTheProjectName` 覆盖以下情形：
  - `--name` 含代码围栏（git 项目与非 git 项目）、名称中间含代码围栏；
  - 目录名含代码围栏、不带 `--name`（git 与非 git）；
  - 首次 `init`（不创建数据目录）与已有项目（数据目录逐字节不变）；
  - JSON 与文本模式。

  每种情形都是退出码 2，error.v1 合法，不回显，不写 marker，仓库不变，合成 canary 不进入输出和数据目录。
  对照：
  - 目录名含代码围栏、带合法 `--name` 时成功；
  - 单个与两个连续的反引号、中文、日文、emoji、200 个字符，都能登记，并按原值保存在数据库与 marker 中；
  - 201 个字符被拒绝；
  - 成功输出通过 `init.v1` 校验。
- `TestInitWithARegisteredNameThatHoldsACodeFence`：在数据库与 marker 中写入含代码围栏的旧名称，模拟早期版本的数据。
  - 再次执行、带另一个名称、带 `--stage`、带登记的名称、在新的关联 worktree 中执行，都是退出码 2，而不是 internal，也不回显名称；
  - 名称、marker、worktree、阶段与 `seq` 都不变；
  - `source list`、`target show` 照常工作。
- `internal/contract` 的 SCH-150。

**范围外，不变**：Source、Proposal、Review、Question、Target、Scope 的行为，SQLite 迁移与数据库结构，Windows 首次创建数据库的竞争，`doctor` 的 `GIT_TRACE`，M2，PR #15 中的 M1 退出条件报告。

**验证**（本地）：

- 两个新测试在 `01bd50f` 上失败（RED），修复后通过。
- `scripts/check.sh` 中，tidy、vet、Markdown 链接检查、golangci-lint（0 issues）都通过。
- `go test ./...`：`internal/cli` 以外的包全部通过。
  - `internal/cli` 触发本地 Windows 上 Go 默认的 10 分钟超时，此前没有测试失败。
  - 单独以 `-timeout 30m` 运行，通过（1048 秒）。
  - 脚本和全局超时都没有修改。
- web 检查、带 Dashboard 的构建、三平台 lint 与格式检查都通过。
- WSL Linux：19 个测试二进制全部通过，其中 `cli` 147 项。
- 三平台 CI 的结果记录在 PR 中。

**状态**：已完成。codex-review-037 GO，PR #17 已合并（合并提交 `5aad555`），旧项目的处理方式按已批准的方案 1，见 §48。M1 尚未整体验收。

## 48. 项目名称修复 PR #17 的审查（codex-review-037）与合并

**审查结论**：**GO**，BLOCKER 为 0，有 2 项 NON-BLOCKER。审查原文按原样入库（`docs/reviews/codex-review-037.md`）。

- **审查对象**：PR #17，审查目标 `a106f9e`，基准 `01bd50f`。CI 为 [run 39015547400](https://github.com/XKHoshizora/specmetry/actions/runs/39015547400)，8/8。CI 实际检出的合并快照是 `947e1d7`，它的树与目标相同（`13ee736`）。
- **合并**：PR #17 已合并，合并提交 `5aad555`（2026-10-17）。
  - 它的两个父提交是基准 `01bd50f` 与目标 `a106f9e`，树为 `13ee736`，与审查目标及 CI 的合并快照相同。合并内容与审查内容没有差异。
  - 合并后 master 的三平台 CI 8/8 通过（[run 39019706495](https://github.com/XKHoshizora/specmetry/actions/runs/39019706495)）。
- **项目名称修复作为 M1 最终验收前置条件，已经满足。**
  - 审查方核实：新名称在打开数据库之前拒绝；旧项目的检查位于改名、阶段、settings 与 worktree 的更新之前，失败时写事务回滚，marker 与 exclude 不写；`init.v1` 与 SCH-150 一致。
  - 审查方另在隔离副本中逐表、逐字段比较全部业务表，并核对 marker、exclude、仓库与关联 worktree 的文件，覆盖更多参数组合，结果一致。

**产品决定（方案 1，产品负责人已正式批准，无须再次确认）**：

1. 新登记的项目名称禁止三个连续反引号（`--name` 与默认的目录名）。
2. 已登记的名称违反这一规则时，`init` 返回有说明的 `invalid_input`（退出码 2）。
3. 不静默改名，不改写旧名称、marker 或其他业务事实。旧项目保证的是业务事实、marker、exclude 与仓库文件不变。这一检查要用普通连接打开数据库读取登记的事实，可能更新 SQLite 的辅助文件（审查方观察到 `-shm` 的修改时间变化，内容哈希不变）。这既不是"数据库未打开"，也不是"数据目录绝对零变化"，也没有因此放宽存储契约、版本保护或 `doctor` 的只读约束。
4. 接受的局部限制：这类旧项目暂时不能通过 `init` 修改阶段、补写 marker 或登记关联 worktree。
5. 只针对这类名称，不代表所有开发版本数据都不需要兼容。
6. 不增加改名命令，不修改数据库结构或迁移。
7. 不据此关闭其他未解决的风险。

M2 的 `status.v1` 中 `project.name` 的兼容方式，留到对应阶段设计，本轮不决定，也不推定所有读取接口都必须拒绝旧项目。

**NON-BLOCKER**：

- **N1**：产品决定的记录需要收尾。§47 与 PR #17 的描述中"待产品负责人确认"的表述，已在本轮更新为已批准的方案 1。
- **N2**：旧项目测试（`TestInitWithARegisteredNameThatHoldsACodeFence`）的无写入摘要，只比较名称、阶段、worktree 数量、`seq` 与 marker，没有直接覆盖完整的 settings、worktree 明细与 exclude。生产检查顺序正确，审查方已另行独立验证。登记为独立维护事项（implementation-plan"独立维护事项"），本轮不改测试。

**ACCEPTED RISK**：只有上述第 4 条的局部限制。其他安全、可靠性与维护问题都没有接受。以下继续保留：Windows 首次创建数据库期间偶发 `Access is denied`、`doctor` 的 `GIT_TRACE`、本地默认测试超时。

**M1 状态**：

- 九个工作包都已验收；
- 6 项证据缺口已补齐（§46）；
- M1 最终验收前的项目名称修复已通过审查并合并；
- M1 退出条件的证据报告已更新为第 2 版（[m1-exit-evidence.md](m1-exit-evidence.md)：59 个 P0 项全部 PASS）。

M1 仍等待 Codex 的最终退出条件审查（审查 038）与产品负责人的正式验收。

## 49. M1 最终退出条件审查（codex-review-038）

**审查结论**：**PR #15 GO；固定代码快照的 M1 退出条件 GO。** BLOCKER 为 0，有 3 项 NON-BLOCKER，没有新增的产品待决事项。审查原文按原样入库（`docs/reviews/codex-review-038.md`）。

- **审查目标**：PR #15 的 `9b8ea5d`，基准 `5aad555`。
  - 这一结论适用于审查目标 `9b8ea5d`，以及它所核对的代码快照 master `5aad555`（树 `13ee736`）。PR #15 只有文档，没有改变这个代码快照。
  - 本节及之后的文档收尾提交，不改变审查对象。
- **CI**：master 为 [run 39019706495](https://github.com/XKHoshizora/specmetry/actions/runs/39019706495)（检出 `5aad555`），PR 为 [run 39020671974](https://github.com/XKHoshizora/specmetry/actions/runs/39020671974)（合并快照 `a3b43c2`，树与审查目标相同），都是 8/8。
- **59 个 P0：59 PASS、0 INCOMPLETE。** 审查方按验收标准独立提取，与报告逐项匹配，没有遗漏，没有重复，也没有第 60 项。项目名称修复是独立的前置条件，不计入。第 1 版 6 项缺口的关闭依据成立；SRC-07 的依据是审查 036 的独立调用链审查，相关代码在此之后没有变化。

**CH-02 两个场景的正式裁定：PASS。**

`review` 与 `question answer` 的"stdin 不是终端"，由以下证据链成立：

1. `executeRoot` 在命令执行之前，按进程的 stdin、stdout 统一识别通道。
2. 不是终端的 stdin 一律产生 `agent_cli`。这一操作系统边界已有真实进程的证据（WP1.2）。
3. 两个命令都在读取输入、打开数据库与确认之前调用同一个 `RequireHuman`，没有专属的旁路。
4. 它们的 stdout 重定向有进程级证据。进程内的非终端测试调用生产的 `Execute`，没有注入通道，证明接线与拒绝后的业务状态，但单独不能代替真实进程的证据。
5. `TestBuiltBinary` 提供正式入口与关键进程边界的证据。

已批准的三类证据定义不要求每个命令的每种组合都用正式二进制。这一结论不是因为"第 1 版没有列为缺口"而自动成立。

**审查方的独立补验**（Windows，没有进入仓库，也没有在三平台 CI 中运行）：

- 正式二进制的 stdin 接到保持写端打开、没有数据的管道，stdout 为真实的 ConPTY。覆盖 `review --file -`、合法的审核文件、`question answer --file -` 与 `-m`，文本与 JSON 两种模式：都是退出码 3，没有确认提示，业务状态不变，没有因读取管道而阻塞；JSON 输出完整解码并通过 `error.v1` 校验。
- `question answer` 的第二个 Agent 变量：仓库测试只设置了 `CLAUDECODE`。两个变量经同一个识别循环产生同一个 Detection；审查方另用正式二进制补验了 `CLAUDE_CODE_ENTRYPOINT`，结果是拒绝且 canary 不泄露；没有 Agent 变量、正确确认的对照成功。
- 审查方在审查目标的隔离副本中运行了 10 项测试，全部通过，没有跳过，其中包括一项只在隔离副本中新增的测试（`TestReview038RealStdinRefusal`）。按要求，这项测试不移入仓库。

**证据来源的边界**：

- 真实旧版 git（2.31.1、2.32.0）的重跑，是作者在本地对 `5aad555` 代码树的运行，审查 038 没有独立重跑。
- PR #16 的变异检验，是作者提供的辅助证据。
- `nowebui` 构建只证明生产 CLI 的入口，不证明完整的发布包。
- 本地 Windows 的 `scripts/check.sh` 没有完整通过（`internal/cli` 默认超时），延长超时后的 CLI 包通过，GitHub CI 通过；报告中分别记录。

**NON-BLOCKER 的处理**（本节所在的提交）：

- **N1**：主报告中 PASS 的定义改为"M1 各项要求均有适用的测试、独立审查或规定的其他验证证据；自动化测试注明对应提交和 CI 结果"，并保留 SRC-07 来自审查 036 独立调用链审查的说明。
- **N2**：主报告中 CH-01、CH-02 的依据，改写为上述证据链与审查方的补验，并区分仓库测试、三平台 CI、作者本地证据与审查方独立验证。第 2 版原来的两句说明（"不是第 1 版的缺口""按'或'可以接受"）只保留在修订记录中，不再作为依据。
- **N3**：审查 007 原文第 15 行的锚点 `#112-数据库迁移`，对应当前的 [architecture §11.2"迁移"](architecture.md#112-迁移)。审查原文不修改，只在此补充定位，也不扩展为历史文档的全面清理。

**产品决定与风险**：没有新增的待决事项。三类证据的定义与方案 1 都已有负责人的明确批准。接受的风险只有方案 1 的局部兼容性限制（§48）。M-1、M-2、M-4 ~ M-10、M-13 都保持未关闭，不因本轮 GO 而关闭，也不归为已接受的风险。

**M1 状态**：**M1 最终退出审查 GO，等待产品负责人正式验收。** 本记录不宣布 M1 已验收，也不授权开始 M2。

- PR #15 合并之后，还需核对合并提交、内容一致性与 master CI。
- 只有产品负责人明确宣布验收之后，才登记"M1 已正式验收"。
- M2 开工需要另行明确授权。

（后续：PR #15 已合并，产品负责人已正式验收 M1，见 §50。）

## 50. M1 正式验收（产品负责人，2026-10-17）

**决定**：产品负责人于 2026-10-17（Asia/Tokyo）**正式验收 M1**（基准、来源、交付目标与范围）。

- **验收对象**：`7e77fd6`，即 PR #15 的合并提交。
- **验收依据**：
  - **codex-review-038**：PR #15 GO；固定代码快照 `5aad555` 的 M1 退出条件 GO，59 PASS、0 INCOMPLETE（§49）。
  - **codex-review-039**：最终的文档增量与合并结果 GO（原文：[docs/reviews/codex-review-039.md](reviews/codex-review-039.md)）。
  - **合并后的 master CI**：[run 39025525139](https://github.com/XKHoshizora/specmetry/actions/runs/39025525139)，提交 `7e77fd6`，首轮 8/8 成功，没有重跑。

**审查 039 的原文**：已按原样入库为 [docs/reviews/codex-review-039.md](reviews/codex-review-039.md)。§50 起初登记时还没有收到原文，只按产品负责人给出的结论登记；收到之后补入本节，结论与起初登记的一致。

- 结论：GO。BLOCKER 无，NON-BLOCKER 无新增，没有新增的待决事项；接受的风险只保留方案 1 的局部兼容性限制，没有扩大。
- 范围：只读确认审查 038 之后的文档增量（`9b8ea5d` → `98a2c34`，两个提交、五个文档文件）、归档、合并内容与 CI。没有重新审查九个工作包，也没有重跑 59 项验收。
- 确认内容：
  - 审查 038 已正确落实：审查身份仍指向 `9b8ea5d`；PASS 的定义与 CH-01、CH-02 的证据链已更正；各类证据的来源已分开。
  - 报告仍为 59 PASS、0 INCOMPLETE，验收标准没有扩大或削弱。
  - 审查 007 的原文没有修改。
  - 合并提交 `7e77fd6` 的父提交是 `5aad555` 与 `98a2c34`，它的树 `c8e99d9` 与 `98a2c34` 相同。
  - master CI 39025525139 检出的是 `7e77fd6`，首轮 8/8 成功，没有重跑。
- 审查 039 说明：之后原样归档这份报告并登记负责人的验收决定即可，不再触发同一轮的 M1 全面审查。

**合并后的核验**（验收前完成）：

- PR #15 的合并提交为 `7e77fd6`，两个父提交是 `5aad555` 与最终 HEAD `98a2c34`。
- 合并提交的树（`c8e99d9`）与 `98a2c34` 完全相同，没有差异。
- 工作区干净。

**保留的事项**：

- 历史上的审查目标与提交身份都保持原样，例如审查 038 的目标仍是 `9b8ea5d`。
- 未解决的维护事项全部保留，不因验收而关闭，也不是已接受的风险：
  - M-1：Windows 首次创建数据库期间偶发 `Access is denied`；
  - M-2：`doctor` 的 `GIT_TRACE`；
  - M-4 ~ M-9；
  - M-10：本地默认测试超时；
  - M-13：审查 037 N2。

  明细见 [m1-exit-evidence.md](m1-exit-evidence.md) 第 9 节，以及 implementation-plan 的"独立维护事项"。
- 接受的风险仍只有方案 1 已批准的局部兼容性限制（§48）：名称含代码围栏的旧项目暂时不能通过 `init` 修改阶段、补写 marker 或登记关联 worktree。验收不扩大风险接受的范围。

**M2**：这次验收不授权开始 M2。M2 开工另行决定。

## 51. M2 开工准备报告的审查（codex-review-040）

**审查结论**：**PR #19 作为当前版本的准备报告 NO-GO；完整 WP2.1 / WP2.2 的开工前提尚未具备。** 5 项 BLOCKER、3 项 NON-BLOCKER。审查原文按收到的原样入库（[docs/reviews/codex-review-040.md](reviews/codex-review-040.md)）。

- **审查目标**：PR #19 的 `a81cc79`（准备报告第 1 版），基准 `f904ab6`。本节及之后的修订不改变审查对象：审查 040 的结论只适用于 `a81cc79`。
- **CI**：[run 39031995619](https://github.com/XKHoshizora/specmetry/actions/runs/39031995619)，attempt 1，8/8。它检出合并快照 `ef80b2e`，树与审查目标相同。审查方指出：这组 CI 验证的是仓库中已有的代码，不是作者没有提交的 shell、`inputs` 与恢复实验，不能证明 M2-Q1 ~ Q3 已有三平台的实现或验证。
- **不是因为草案没有获批**，而是报告中有与既有验收不符的结果分类、所述流程无法保证的恢复承诺，以及不成立的首批开工判断。

**BLOCKER**（指准备报告的技术依据或开工结论必须修正，不是已上线代码的漏洞）：

| 编号 | 问题 |
|---|---|
| B1 | M2-Q1 的结果映射与 MVP-EV-04 不符；退出码 127 不能反向证明命令不存在（`exit 127` 同样产生 127），`cmd` 的缺失命令与主动退出都是 1 |
| B2 | M2-Q3 在复核之后没有持续排除外部活动：复核之后还要生成、发布与轮换备份，再打开原数据库；SQLite 可能在应用层版本检查之前回放热日志或切换日志模式，所以不能保证不兼容的数据库在原数据库被修改之前被拒绝 |
| B3 | M2-Q3 的清理、备份与失败阶段，与"拒绝之后不变"的统一承诺互相矛盾（先删除遗留目录、轮换备份、打开原数据库之后的失败） |
| B4 | STO-09 承诺"已提交的数据保留"，与报告自己披露的 WAL 丢失实验不一致；可以打开、结构一致与历史完整是不同的结论 |
| B5 | 完整 WP2.1 / WP2.2"不依赖任何待决事项开工"的判断不成立：M2-Q2 的比较语义、M2-Q5、M2-Q8 是推导的实际输入与输出语义；M2-Q10 尚未改变"M2 开工前"的截止点 |

**NON-BLOCKER**：

- N1：M2-Q1 的解释器选择与参数实验不足以支撑通用结论（绝对路径不等于不来自 worktree、各 shell 需要各自的适配器与测试、"统一 `sh`"的含义、进程树的启动窗口与边界）。最晚 WP2.5 开工前冻结和验证。
- N2：M2-Q2 需要区分对象缺失的层次。审查方独立复验：树条目可以列出，不等于叶 blob 可以读取；同一对象 ID 可以对应不同的 mode。路径规范化应与 M1 已保存的配置、`check_config_hash` 一致。
- N3：工作包映射需要补足集成责任与验收标签。审查方独立统计：标注涉及 M2 的常规 P0 68 项、CX 26 项；SRC-04 的里程碑列仍写 M1，但正文把状态与差距中的 `source_removed` 留到 M2；ACC-09 的 MVP 集成部分需要明确由哪些工作包闭环。

**M2-Q1 ~ Q10**：审查方逐项给出了技术与契约判断，**全部仍是待确认的草案**，审查不批准任何产品取舍。需要重新列明的决定包括：Q1 的 EV-04 保证变化；Q3 的持续排他或离线前提、失败阶段与历史完整性；Q4 的 `null` 与 `purge` 的 ID 确认；Q7 取消关闭开关；Q2、Q5、Q8 的首批冻结；Q10 的截止点调整。

**验证边界**（审查原文第六节）：

- 审查方独立完成：Windows 上的 `cmd` 退出码对照；缺失叶 blob 与同一对象 ID 不同 mode 的复验；固定提交的差异、必要的调用链、schema 与三平台 CI 的元数据和日志核查。
- 未完成：`sh` 的对照在 MSYS 启动时失败（`0xC0000022`），没有执行到命令，没有算作 127 的复验；没有重做作者的恢复实验，没有动态复现 B2 的外部修改窗口（B2、B3 是流程与源码级的论证，B4 的行数来自作者的披露）；Linux、macOS 的锁探测，Job Object 与 Git Bash 的全部后代，Windows PowerShell 的完整参数集，各类 git 安装与 LFS 环境都没有验证。

**产品决定与风险**：没有批准任何草案。没有新增接受的风险：恢复期间的外部活动、打开原数据库之后的副作用、含墓碑数据的临时副本、WAL 丢失之后的历史不完整，都不是已接受的风险。接受的风险仍只有方案 1 的局部兼容性限制（§48）。M-1、M-2、M-4 ~ M-10、M-13 保持原安排，N1 的剩余整理修复仍是对应 M2 模块的门槛。

**处理**（本节所在的提交，准备报告第 2 版，[m2-readiness.md](m2-readiness.md) 第 12 节有逐项对照）：

| 意见 | 第 2 版的处理 | 状态 |
|---|---|---|
| B1 | 结果分为六个层次并列出各自的证据；不再用普通退出码或 stderr 文本证明命令不存在；EV-04 列为未解决的契约问题，给出方案 A（改写 EV-04，推荐）、B、C；补充作者第 2 轮的退出码实验，保留审查方 `sh` 未启动成功、其他平台未验证的边界 | 已修订，待独立复审 |
| B2 | 把最终复核、备份生成与发布、打开原数据库放在同一流程中分析；持续排除外部活动列为未解决的技术问题 U1，不再声称复核消除了竞态；U1-a（跨平台协调设计，待验证）、U1-b（离线前提，降低保证，需负责人决定）、U1-c（缩小窗口，不单独充分）；WP2.10 在 U1 有结论之前不开工 | 已修订，待独立复审 |
| B3 | 按阶段列出可能变化的文件与目录项、清理、错误结果与保证；删除统一的"拒绝不变"承诺；遗留清理与备份轮换计入副作用；需要放宽的部分单列为例外 E1 ~ E5，待负责人批准，不放宽普通打开与 `doctor` | 已修订，待独立复审 |
| B4 | 区分可以打开、结构一致与历史完整；WAL 已丢失时不承诺保留无法恢复或无法确认的数据；列出可恢复、完整性未知、无法安全继续的状态与建议；STO-09 拆为 09a、09b（只改准备报告中的草案） | 已修订，待独立复审 |
| B5 | 删除"不依赖任何待决事项"的表述；列出完整实现前必须冻结的规则 F1 ~ F6，以及各项决定对推导引擎与复核器的影响；可选的受限子集只作为建议；复核器的完整谓词 R1 ~ R10、输入域、反例与正例 | 已修订，待独立复审 |
| N1 ~ N3 | 解释器查找的规范化与 worktree 排除、进程树的边界；`inputs` 对象读取的层次、只要求元数据、与 V-P04 一致的规范化；68 + 26 项的复核与验收项归属表 | 已修订，待独立复审 |

本节不宣布审查关闭，B1 ~ B5 等待定向复审。

**保留的事项**：M1 的验收结论（`7e77fd6`）、审查 038 与 039 的结论不变；所有未关闭的维护事项保持未关闭；M2 尚未授权实现。

（后续：审查 041 复审了第 2 版，B1、B2、B4 已消除报告层面的阻塞，B3、B5 各有一处残余问题，见 §52。上表 B3 一行"E1 ~ E5 … 不放宽普通打开与 `doctor`"对 E4 不准确：E4 是普通打开成功路径上的扩展，不是 `db recover` 的例外，第 3 版已单列。）

## 52. M2 开工准备报告第 2 版的复审（codex-review-041）

**审查结论**：**第 2 版准备报告 NO-GO；完整首批与受限首批都不具备开工前提。** 审查原文按收到的原样入库（[docs/reviews/codex-review-041.md](reviews/codex-review-041.md)）。

- **审查目标**：PR #19 的 `09caf95`（准备报告第 2 版），基准 `f904ab6`。审查 041 的结论只适用于 `09caf95`；本节及之后的修订不改变审查对象。
- **CI**：[run 39035652314](https://github.com/XKHoshizora/specmetry/actions/runs/39035652314)，attempt 1，8/8 成功。它检出合并快照 `2d0bf95`（父提交为 `f904ab6` 与 `09caf95`），树与审查目标相同（`93874e9`）。审查方指出：它验证的是已提交的仓库代码，不是没有提交的 M2 实验，也不表示 Windows 首次建库的间歇性问题（M-1）已经关闭。

**审查 040 的各项**（审查 041 第二节）：

| 原问题 | 准备报告层面 | 技术问题或决定 |
|---|---|---|
| B1 | **已消除阻塞** | EV-04 的调整、默认 shell、不可用时的行为仍待决定 |
| B2 | **已消除阻塞** | U1 仍未解决，不能据此开发或验收恢复模块 |
| B3 | 部分修订，未关闭（B3-R） | — |
| B4 | **已消除阻塞** | U2 仍待负责人选择 |
| B5 | 部分修订，未关闭（B5-R） | — |

"已消除阻塞"只表示准备报告已经正确披露问题与门槛，**不表示** EV-04 的取舍、U1、U2 已经解决或获批，也不表示相关技术保证已经实现。

**BLOCKER**：

| 编号 | 问题 |
|---|---|
| B3-R | E4 要求普通打开删除恢复遗留，却被归入"只适用于 `db recover`"的例外；"版本检查通过之后"也不足以限定为成功路径：其后的墓碑校验、墓碑执行、备份或迁移仍可能失败，遗留却已删除。P5 的工作目录删除失败、轮换失败或只完成一部分没有对应的结果 |
| B5-R | 受限首批只写了"同样需要 F6"，遗漏了它同样必需的复核器规格（F5-a）与事实结构；生成器"不排除坏组合"与双向对照的输入域没有区分合法事实历史与违反写入不变量的畸形组合；R6 的差别在合法事实历史中与决策表相容，首先需要澄清适用域，而不是直接认定规格冲突 |

**NON-BLOCKER**：

- N1：反例"只违反一个条件"应考虑结构依赖，不为凑单条件用例构造违反写入约束的历史。
- N2：审查 040 只记录了审查方在 Windows 上独立复验，没有证明是另一台物理机器。

**产品决定与风险**：Q1 ~ Q10 全部仍待负责人决定。审查方说明：F5 的合法事实输入域与 R6 的澄清首先是依据既有规范完成技术规格；只有改变冻结的推导规则、放宽安全保证或为畸形输入新增用户可见的语义时，才另行请求裁决。没有新增接受的风险，仍只有方案 1 的局部兼容性限制（§48）；U1、U2、恢复的临时副本与清理例外、进程树逃逸都不是已接受的风险。M-1、M-2、M-4 ~ M-10、M-13 与 M0 N1 的剩余修复保持原安排。

**验证边界**：本轮只做了静态的契约与流程审查、只读统计、Git 身份与三平台 CI 元数据和日志的核查；没有运行作者的实验，没有攻击模拟、故障注入或恢复实验。审查 040 的行号适用于 `a81cc79`。

**处理**（本节所在的提交，准备报告第 3 版，[m2-readiness.md](m2-readiness.md) 第 12 节有逐项对照）：

| 意见 | 第 3 版的处理 | 状态 |
|---|---|---|
| B3-R | E4 单列为普通打开的待批准扩展：只在打开的全部步骤成功之后、释放打开锁之前执行，此前的拒绝与失败都不清理遗留；清理失败不改变打开的结果，记入程序日志并由 `doctor` 报告；`doctor` 从不清理。不适用的状态不清理遗留。新增清理失败表与 P5 的收尾失败结果，区分"数据库恢复已完成，收尾未完成"与"恢复未完成，原数据库的状态未确认"；退出码为草案。同步总述、例外、第 5.3.5 节、STO 草案与第 6.2 节 | 已修订，待复审 |
| B5-R | 完整首批需要 F1 ~ F6；受限首批需要 F4-a、F5-a 冻结并审查，再完成 F6。定义合法事实历史、MVP 与通用合法事实域、畸形组合；合法但不能 `ready` 的历史保留在生成器中；双向对照只在合法事实域上进行，畸形组合单独测试；规定两套转换器保留的内容；说明 R6 与 §6.3 第 6 条在合法事实域上等价；给出 data-model §6 的最小澄清建议（选项 (a)、(b)，待决定，推荐 (a)） | 已修订，待复审 |
| N1、N2 | 反例以目标条件为准，允许必然的连带条件；措辞改为"审查方在 Windows 上独立复验" | 已修订，待复审 |
| 直接关联的调整 | U1、U2 按原截止点须在 M2 开工前有结论；只有负责人按 M2-Q10 批准顺延，才可以留到 WP2.10 开工前 | 已修订，待复审 |

本节不宣布审查关闭，B3-R、B5-R 等待复审。

**保留的事项**：M1 的验收结论（`7e77fd6`）、审查 038 与 039 的结论不变；所有未关闭的维护事项保持未关闭；现有截止点与待决方案的状态不变；M2 尚未授权实现。

（后续：审查 042 复审了第 3 版，B3-R 已消除报告层面的阻塞，B5-R 只剩共享 `seq` 的定义问题，见 §53。）

## 53. M2 开工准备报告第 3 版的复审（codex-review-042）

**审查结论**：**第 3 版准备报告 NO-GO，只有一项残余 BLOCKER；完整首批与受限首批都不具备开工前提。** 审查原文按收到的原样入库（[docs/reviews/codex-review-042.md](reviews/codex-review-042.md)）。

- **审查目标**：PR #19 的 `be660df`（准备报告第 3 版），基准 `f904ab6`。审查 042 的结论只适用于 `be660df`；本节及之后的修订不改变审查对象。审查 040、041 的对象与结论保持原样（`a81cc79`、`09caf95`）。
- **CI**：[run 39037731201](https://github.com/XKHoshizora/specmetry/actions/runs/39037731201)，attempt 1，8/8 成功。它检出合并快照 `8faa31e`（父提交为 `f904ab6` 与 `be660df`），树与审查目标相同（`75d319a`）。审查方指出：它验证的是已提交的仓库代码，不是没有提交的 M2 实验；维护事项不因 CI 成功而关闭。

**各项的状态**：

| 项 | 结论 |
|---|---|
| B3-R | **已消除报告层面的阻塞**：E4 已单列为普通打开的待批准扩展；阶段、清理失败、轮换失败与整理未完成已明确区分。这不表示恢复实现、E4 或新的退出结果已经获批 |
| B5-R | **部分关闭**：完整与受限首批的门槛、合法历史与畸形组合的分组、转换器约束与 R6 的论证已修正；**只剩一处**：第 3 版把"`seq` 全局唯一"列为合法历史的不变量，并把"重复的 `seq`"列为畸形 |
| N1 | 第 6.2 节的验收项映射只写了 STO-08 ~ 13，没有列出 STO-13b 与只在 E4 获批时加入的 STO-14 |

**B5-R 的残余问题**：要区分"计数器分配的序号"与"持有这个序号的事实行"。审查方以现有调用链为证：

- 公开的 `review` 路径只调用一次 `NextSeq`；
- `baseline.Apply` 用同一个值写入 `baseline_versions` 与一条或多条 `baseline_changes`（用 `change_index` 保留版本内的顺序）；
- 附带范围时，`review.ConfirmScope` 把同一个值交给 `scope.Write`，写入 `scope_confirmations`。

这些都是合法的不同事实记录，与 data-model §1 和 §5.5 一致。按第 3 版的文字，正常审核的历史会被归为畸形而退出双向对照。审查方指出，这是报告中的事实域定义错误，不需要负责人重新决定已冻结的共享 `seq` 行为。

**产品决定与风险**：没有获批的方案。Q1 ~ Q10、E1 ~ E3 与 E5、E4、"收尾未完成"的退出码、畸形输入的选项都待决定或冻结。U1、U2 的原截止点（M2 开工前）保持有效；只有负责人明确批准 Q10，才可以顺延。没有新增接受的风险，仍只有方案 1 的局部兼容性限制（§48）；M-1、M-2、M-4 ~ M-10、M-13 与 M0 N1 保持原安排。

**验证边界**：只读的文档、源码与提交核查，以及 CI 元数据与日志核查；没有运行动态实验或测试程序。共享 `seq` 由现有的公开调用链证明，没有构建复现程序。

**处理**（本节所在的提交，准备报告第 4 版，[m2-readiness.md](m2-readiness.md) 第 12 节有逐项对照）：

| 意见 | 第 4 版的处理 | 状态 |
|---|---|---|
| B5-R | 合法事实历史改为：`seq` 的分配值唯一、单调递增，同一次逻辑写入的关联事实可以共享 `seq`，不同的逻辑写入从不共享。新增"共享 `seq`"表，覆盖审核附带范围、`propose --apply`、`propose`、`target set`，列出规范与调用链的依据：`internal/cli/review.go`、`internal/baseline/apply.go`、`internal/review/scope.go` → `internal/scope/confirm.go`、`internal/cli/apply.go`、`internal/proposal/store.go`、`internal/scope/target.go`。写明范围限定：迁移中 `seq` 表内唯一的表、可有多行的表、一个事务也可以分配多个 `seq`、M2 新事实表按对应工作包写明。畸形示例只保留违反具体写入不变量的情形；生成器与转换器保留共享 `seq` 的全部关联事实；`as_of_seq` 保留截止序号内的全部事实；同一 `seq` 内的顺序沿用既有规则；时序比较各自依据现有契约。测试规划加入共享 `seq` 的审核历史 | 已修订，待复审 |
| N1 | 第 6.2 节补入 STO-09a、09b、13b，以及只在 E4 获批时加入的 STO-14，都保持草案 | 已修订，待复审 |

本节不宣布审查关闭，B5-R 的残余问题等待复审。

**保留的事项**：M1 的验收结论（`7e77fd6`）、审查 038 与 039 的结论不变；所有未关闭的维护事项保持未关闭；现有截止点与待决方案的状态不变；M2 尚未授权实现。

（后续：审查 043 复审了第 4 版，结论为 GO，B5-R 的共享 `seq` 问题已在准备报告层面关闭，见 §54。）

## 54. M2 开工准备报告第 4 版的复审（codex-review-043）

**审查结论**：**第 4 版准备报告 GO，没有 BLOCKER。** 审查 042 的共享 `seq` 阻塞与验收映射遗漏都已解决；可以由负责人合并 PR #19。审查原文按收到的原样入库（[docs/reviews/codex-review-043.md](reviews/codex-review-043.md)）。

- **审查目标**：PR #19 的 `e1d9000`（准备报告第 4 版），基准 `f904ab6`。审查 043 的结论只适用于 `e1d9000`；之后的收尾提交不改变审查对象。审查 040 ~ 042 的对象与结论保持原样（`a81cc79`、`09caf95`、`be660df`）。
- **CI**：[run 39039844812](https://github.com/XKHoshizora/specmetry/actions/runs/39039844812)，attempt 1，8/8 成功。它检出合并快照 `4d55e43`（父提交为 `f904ab6` 与 `e1d9000`），树与审查目标相同（`98d111e`）。
- **GO 的含义**：只确认准备报告的依据、待决事项与门槛已经足够准确。它**不表示** Q1 ~ Q10、局部例外或新的退出结果获批，不表示 F4、F5 已冻结，也不授权 M2 实现；完整首批与受限首批都仍不具备开工前提。

**确认的内容**：

- 序号的分配与事实行的身份已经分开；共享 `seq` 的合法历史被保留，没有把一个事务等同于一次分配。
- "同一次逻辑写入"按调用链中的一次序号分配及其明确的复用范围理解，不等于一条 SQL、一行记录、整个命令或整个事务；不需要新增"逻辑写入 ID"或全局排序字段。
- 序号的唯一与递增针对已提交的事实历史；演练或失败的事务回滚计数器，不消耗 `seq`。
- 共享表与真实写入相符；迁移中的 `UNIQUE` 列与可以多行同值的列都已核对。
- 生成器、转换器、`as_of_seq` 与同一 `seq` 内的顺序要求已经同步；时序比较仍按既有契约，没有统一放宽。
- 没有冒充 M2 新事实表的 `seq` 分配已经冻结：首批实际需要的分配方式、记录身份、时序与引用结构，仍须在 F4（F4-a）冻结时写明，不能留到推导引擎实现之后。
- 审查 042 的验收映射 NON-BLOCKER 已解决；正式的 acceptance-criteria 没有修改。

**NON-BLOCKER**：

- N1：共享范围的概括应按具体列与实际写入收紧："对同一个分配值，每个声明 `UNIQUE` 的序号列至多有一行"，这是列约束，不是事务修改行数的约束；共享表应按"本次实际写入或更新的序号字段"理解（全部拒绝的审核不创建基准版本，`editorial` 更新不刷新 `revision_seq`）。
- N2：记录身份沿用各表既有的主键、复合键或约定标识（例如 `scope_items` 的复合主键），不要求所有表新增独立的 `id` 列。

两项都可以直接落实，不需要再开一轮准备报告审查。

**产品决定与风险**：没有批准任何方案，也没有新增必须重新决定的产品问题。U1、U2 的原截止点（M2 开工前）在审查时仍然有效。没有新增接受的风险，仍只有方案 1 的局部兼容性限制（§48）；M-1、M-2、M-4 ~ M-10、M-13 与 M0 N1 保持原安排。

**验证边界**：静态的文档、规范、迁移与调用链核查，以及提交与 CI 元数据和日志核查；没有运行新的 Go 测试、生成器、恢复或锁实验、shell 实验或跨平台动态复验。共享关系以现有规范、迁移与源码为证，不表示 M2 的生成器或复核器已经通过测试。

**收尾**（本节所在的提交）：按 N1、N2 收紧了准备报告第 8.5 节的措辞（[m2-readiness.md](m2-readiness.md) 第 12 节的"收尾"表），并同步 AGENTS.md 与 implementation-plan 中 M2 的状态说明。按负责人的指示，纯归档、两处措辞修正与状态登记不再触发同一轮准备报告的全面审查。

**保留的事项**：M1 的验收结论（`7e77fd6`）、审查 038 与 039 的结论不变；所有未关闭的维护事项保持未关闭；准备报告中的方案仍是草案；M2 尚未授权实现。

## 55. 产品负责人的 M2 决定与首批规范（2026-10-18）

PR #19（M2 开工准备报告）已合并：合并提交 `258fc50`，父提交为 `f904ab6` 与收尾的最终 HEAD `92ff653`，树与 `92ff653` 相同（`7ecbc89`）。准备报告经审查 043 GO（§54）。

**决定**（产品负责人，2026-10-18）：

| 事项 | 决定 | 写入的位置 |
|---|---|---|
| 首批路线 | 完整的 WP2.1、WP2.2，不采用受限子集。只确定路线，**业务实现授权仍未给出** | implementation-plan M2 |
| M2-Q2（TD-23 的比较语义） | 采用准备报告 §5.2 的结构化输入身份：保存的路径原值、mode、类型、对象 ID；查找按 V-P04 规范化，不改写 M1 保存的值或配置哈希；只要求路径元数据，区分路径不存在、读取路径所需对象失败、叶对象缺失与 gitlink；继续遵守 PD-26 | data-model §4.11、§4.11.2、§4.13、§6.2、§6.3、§7.4；architecture §16 TD-23；acceptance-criteria MVP-EV-10、CX-19 |
| M2-Q5（PD-02） | 批准"范围内需求验收率 + AC 状态计数"：验收率为 `RS` 中 `accepted` 的需求数除以 `RS` 中不是 `waived` 的需求数，分母为零时为 null | prd §9 PD-02；data-model §6.11 |
| M2-Q8 | 批准准备报告 §5.8 的标记与不阻塞的 `SOURCE_REMOVED` 警告方案，包括范围筛选；不得替代、消除或推迟既有的 `SOURCE_CHANGED`、`SCOPE_STALE` 阻塞 | data-model §6.6、§6.10；`status.v1`、`common.v1` 与示例、反例；acceptance-criteria MVP-SRC-04、BL-05、ST-13 |
| 畸形输入 | 采用选项 (a)：推导的业务结果保证针对合法事实历史；畸形组合另测不崩溃与结果确定；既有的保守规则与错误处理保持 | data-model §6.14 |
| M2-Q10 | TD-06 / M2-Q1：WP2.5 开工前。TD-23 的 Git 读取机制：WP2.3 开工前。`db recover` 的相关设计、§8.1 第 1 ~ 4 项的调整、需求与验收的冻结：WP2.10 开工前。`backup restore`（P1）：WP2.8 中该 P1 工作开工前 | implementation-plan §3、§5 M2、§8.1；architecture §16；prd §9 PD-23 |

**Q10 调整的边界**：

- 不移除 M2 的恢复交付义务，不改变它必须在 M4 自测前完成的安排，也不批准降低任何安全保证。
- U1、U2 必须在恢复模块（WP2.10）开工前有明确结论；批准不能代替技术验证。
- M2-Q2 的比较语义、M2-Q5、M2-Q8，以及首批的 F4、F5，不顺延到实现之后。
- M0 N1 等已有的依赖门槛保持不变。

**没有批准的事项**：M2-Q1、Q3、Q4、Q6、Q7、Q9，E1 ~ E5，恢复的退出码与交互草案。它们仍是准备报告中的草案。

**首批规范**（本节所在的 PR）：

- F4（推导的输入事实）写入 data-model §6.13，并补齐 §4.10 ~ §4.13 中推导依赖的 M2 事实结构：`seq` 的分配与共享、记录身份、`as_of_seq` 时状态的取法、未知状态与跨写入的时序约束。
- 合法事实域与畸形输入写入 data-model §6.14。
- F5（独立 `ready` 复核器）写入 acceptance-criteria §3.1。
- 这些规格**已写出，尚未通过独立审查**；产品方向已批准，不等于 F4、F5 已经冻结。

**新登记的待决问题**：TD-27（architecture §16）：项目设置的变化没有 `seq`，推导把它们作为当前值输入；是否为它们分配 `seq` 待定，不改变既有行为。

**仍未满足的开工门槛**：F4、F5 通过独立审查；产品负责人明确授权完整首批的实现。

（后续：审查 044 审查了本 PR 的 `4ef79c8`，结论为 NO-GO（B1），修订见 §56。）

## 56. PR #20 首批规范的独立审查（codex-review-044）与 B1 的修订

**审查结论**：**PR #20 为 NO-GO**，有一项 BLOCKER（B1）、一项 NON-BLOCKER（N1）。审查原文按收到的原样入库（[docs/reviews/codex-review-044.md](reviews/codex-review-044.md)）。

- **审查目标**：PR #20 的 `4ef79c8`，基准 `258fc50`（PR #19 的合并提交）。审查 044 的结论只适用于 `4ef79c8`；本节的修订不改变审查对象。
- **CI**：[run 39048967831](https://github.com/XKHoshizora/specmetry/actions/runs/39048967831)，attempt 1，8/8 成功。它检出合并快照 `554d068`（父提交为 `258fc50` 与 `4ef79c8`），树与审查目标相同（`54c4d04`）。审查方指出：它证明的是已提交仓库的检查结果，不证明尚未开发的推导引擎、复核器、M2 迁移，或没有提交的 shell、恢复与 Git 实验。

**保留的结论**（审查 044）：

- F1 ~ F3 已完整落实，没有实质性缺口：M2-Q2 的输入身份、M2-Q5 的指标、M2-Q8 的标记与警告。
- F4 的 `seq` 分配与共享、记录身份，以及主要的历史重建规则，可以保留。
- F5 的 `ready` 谓词（R1 ~ R10）、独立性要求与测试域，可以保留，不需要重写。
- §55 忠实记录了授权边界；没有发现需要重开 PR #19、M1 或恢复方案的理由。
- **整体冻结仍待 B1 关闭**：F4 的完整输入与重放契约还不能冻结，F5 的测试前端随之等待。

**B1**：当前值设置输入与输出的可复现性契约不一致。

- §6.13 把 `stage`、`sources_skipped`、`inventory_skipped` 列为当前值输入；而 data-model §6 开头、architecture §7、MVP-ST-14 与 `status.v1` 的说明，仍声称凭输出中的序号、快照与观测哈希就能复现结果，`status.v1` 也只输出 `stage`。
- 审查方以 M1 `init` 的调用链为证：更新设置不分配 `seq`。所以执行 `init --skip-sources` 之后，事实、`as_of_seq`、`S`、观测哈希与 `stage` 都可以不变，而初始化进度改变。
- 最小修正：
  - 统一列出完整的显式输入；
  - 明确 `--diff-since` 的两次推导共用同一份当前设置；
  - 规定重放旧结果时设置值从哪里来；
  - 同步 Schema 的说明与验收项。
- 不要求修改 `init`，也不要求给设置增加 `seq`。如果选择放宽既有的保证，需要负责人明确确认。

**N1**：`commit_links` 没有独立的 `seq`，经 `commit_id` 归属于提交的发现序号。这个模型只在关联与提交一起写入、之后不再补写时成立。需要在 WP2.3 / M2-Q7 冻结时写明：旧提交的关联能否补写；补写时如何记录时序；没有新提交的扫描如何更新 `scanned_seq`。本项不单独阻塞。

**产品决定与风险**：审查没有批准任何草案，没有新增接受的风险；设置缺少 `seq` 带来的重放问题，没有被接受为风险。

**验证边界**：

- 静态核查：固定差异、规范、Schema、已有源码与测试断言。
- GitHub 提交与 CI 元数据、日志的核查。
- 在 Windows 本机离线运行 `go test -tags nowebui -mod=readonly ./internal/contract -count=1 -v`，通过。第一次经环境中的 Go shim 启动时，测试没有启动，不计为通过或失败。
- 没有运行完整的 M1、动态实验、恢复实验或新的 M2 程序；没有在本机复跑 Linux / macOS。

**负责人批准的修正方向**（2026-10-18）。只覆盖规范与契约的修正，不授权 M2 实现：

- 保留 M1 `init` 更新设置的行为，不增加设置的历史或 `seq`，不修改生产代码与迁移。
- 统一以下语义：
  - 正常查询：三项设置是显式的当前值输入，一次命令取一次，推导过程中不再读取数据库中的设置；
  - 重放：使用原查询的完整输入，不用重放时的当前设置替换；
  - `--diff-since`：两次推导共用本次取得的设置与其他当前值输入，只改变事实的截止序号，两次之间不重新读取；
  - 输出：利用已有的 `project.stage`，补齐其余设置的白名单值，字段组织采用最小改动。
- TD-27 分开记录本轮的语义与将来的设置历史。
- 本轮不增加重放的 CLI、API 或持久化表。
- N1 登记到 WP2.3 / M2-Q7 的冻结任务，不升级为本轮的阻塞，也不提前批准提交关联的方案。

**处理**（本节所在的提交）：

| 意见 | 处理 | 位置 | 状态 |
|---|---|---|---|
| B1：完整的显式输入 | §6 开头改为五项输入，事实以外的四项是当前值输入。§6.13 第 2 类写明三项设置，以及它们影响的规则：不影响交付状态、阻塞差距与 `ready`。`settings_json` 的其他键不是推导输入 | data-model §6、§6.13；architecture §7 | 已修订，待复审 |
| B1：一次命令的取值 | 每项当前值输入，一次命令只取一次；设置与事实在同一个读事务中读取；推导函数没有缺省值；"没有观测"与"观测到文件缺失（NULL）"区分开 | data-model §6.13.1 | 已修订，待复审 |
| B1：重放 | 定义完整输入，在同一版本的推导规则下，同样的完整输入得到同样的结果；逐项列出正常查询与重放时各输入的来源；不用当前值替换或补齐，缺少输入时报告缺少的项；输出不是事实的备份；可重放的输出是 `status.v1`、`gaps.v1`，`init.v1` 的引导提示不属于（M1 已验收，不修改） | data-model §6.13.1；architecture §7；prd NFR-11；mvp §5 第 4 项 | 已修订，待复审 |
| B1：`--diff-since` | 两次推导共用同一份当前值输入，只改变截止序号，两次之间不重新读取设置；`changes_since` 不依赖来源观测；只有设置变化时 `changes_since` 为空，不虚构序号；结果不是过去展示结果的回放 | data-model §6、§6.13.1 | 已修订，待复审 |
| B1：输出契约 | `status.v1`：`project` 增加必填的 `sources_skipped`、`inventory_skipped`；新增与 `changes_since` 成对出现的 `diff_since_seq`；改写说明。`gaps.v1` 原来没有项目身份、设置与来源观测，无法重放，所以新增必填的 `project`（`id` 与三项设置）与 `sources`（`id`、`current_sha256`）。两者都只输出白名单中的设置。合法示例同步，新增 `status.diff-since.json`；新增反例 SCH-157 ~ 165 | `schemas/` | 已修订，待复审 |
| B1：合法输入域 | 设置的任一取值，可以与任一合法事实历史组合；生成器要变化当前值输入 | data-model §6.14；acceptance-criteria §3.1 | 已修订，待复审 |
| B1：验收 | MVP-ST-11：共用设置、注入修改、只有设置变化。ST-14：相同的完整输入、设置变化后用原输入重放、缺少输入时拒绝。新增 ST-15：JSON 中的设置值等于推导使用的值，只输出白名单。E2E-6 的重放说法。F5 的测试前端由同一份当前值输入驱动；只有设置不同的两组输入，`ready` 的结论必须相同 | acceptance-criteria §2.10、§3.1、§5、§6 | 已修订，待复审 |
| B1：TD-27 | 分为两部分：（1）本轮写明、待审的语义，截止 WP2.1、WP2.2 开工前；（2）是否提供设置的历史，是独立的问题，截止 M2 结束时的设计复核 | architecture §16；data-model §10 | 已修订，待复审 |
| N1 | 登记到 implementation-plan §3 的"提交关联的生命周期"，WP2.3 开工前与 M2-Q7 一起冻结；data-model §4.12 与 §6.13 的提交行写明这一前提，不预先批准方案 | implementation-plan §3；data-model §4.12、§6.13 | 已登记，不阻塞 |

**没有重新展开的内容**：

- 输入比较（§4.11.2）、指标（§6.11）、警告（§6.10）没有改动。
- §6.13 的 `seq` 分配与共享表、状态取法表没有改动，只在提交行写明 N1 的前提。
- R1 ~ R10 没有改动。F5 只补充了测试前端与输入域的要求，这是 B1 直接要求的（审查 044 指出，不能把同一个不完整的重放假设复制到两套转换器中）。

本节**不宣布 B1 关闭，也不宣布 F4、F5 已冻结**，等待定向复审。

**保留的事项**：M1 的验收结论（`7e77fd6`）、审查 038、039 与 043 的结论不变；所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13 与 M0 N1）保持未关闭；准备报告中没有获批的草案保持原状态与调整后的截止点；M2 尚未授权实现。

（后续：审查 045 复审了 `d4ba0a9`，结论为 GO，B1 关闭，见 §57。）

## 57. 首批规范的定向复审（codex-review-045）、规范冻结与首批实现授权

**审查结论**：**GO**。审查 044 的 B1 关闭；PR #20 的首批规范与契约具备合并条件，完整的 F4、F5 具备冻结条件。没有 BLOCKER，有一项 NON-BLOCKER（N1）。审查原文按收到的原样入库（[docs/reviews/codex-review-045.md](reviews/codex-review-045.md)）。

- **审查目标**：PR #20 的 `d4ba0a9`，基准 `258fc50`，增量起点是审查 044 的目标 `4ef79c8`。审查 045 的结论只适用于 `d4ba0a9`；本节所在的收尾提交不改变审查对象。审查 044 的对象与结论（`4ef79c8`，NO-GO）保持原样。
- **CI**：[run 39056682275](https://github.com/XKHoshizora/specmetry/actions/runs/39056682275)，attempt 1，8/8 成功。它检出合并快照 `533b8d6`（父提交为 `258fc50` 与 `d4ba0a9`），树与审查目标相同（`d0b660f`）。三个 Go Job 都记录了 40 个合法示例与 165 个非法反例；Ubuntu 使用 Race Detector；四项 CrossUser 测试在 sudo 步骤中实际通过。

**确认的内容**：

- 完整的显式输入：事实、`S`、来源观测、三项项目设置，以及只用于工作区视图的上下文。参数包括截止点、diff 起点与 session 选项。
- 一次命令只取一份当前值；设置与事实在同一个读事务中读取。审查方提示：读事务只保证数据库输入的一致快照；Git 与磁盘由调用方另行观测，不能由此推出跨系统的原子时刻或持续锁定。
- 缺少观测与观测到文件缺失（NULL）已经区分。
- 重放形成闭合的契约：
  - 缺少项目、事实、设置或观测时，报告缺少的项；
  - 不用当前值补齐，不降低截止点；
  - 输出不是事实的备份。
- "同一版本的推导规则"是固定同一个纯函数的前提，没有暗中新增跨版本重放。重放测试应在同一个构建、同一套规则下计算原结果与重放结果；对外引用重放证据时，要保留版本身份。
- `--diff-since`：两个截止点共用当前值输入；`changes_since` 只需要 AC 投影，起点不需要旧来源的观测。
  - 审查方核对的合法历史：起点的来源集合为 A，之后移除 A、登记 B；输出只记录 B 的观测，起点的 AC 投影不读取 A 的观测。
  - 实现只能选择必要的 AC 投影，或满足完整入口的输入要求；**不能捕获缺项错误后忽略，也不能把缺项当作 NULL**。
  - 重放完整的 `status` / `gaps` 仍然需要全部观测。ST-11 应保留这条"来源集合变化"的用例。
- Schema 只验证结构。来源集合是否一致、观测是否遗漏、设置是否与实际计算相符、diff 两个截止点的关系与结果是否正确，由语义校验与测试负责；示例不能代替同一完整输入的语义测试。
- 审查 044 的 N1（提交关联的生命周期）仍在 WP2.3 / M2-Q7 处理。

**N1（本轮）**：data-model §6.13 与 F5 测试前端说明中的"设置不改变阻塞差距"需要收紧。

- 正确的含义：设置不改变交付状态，也不改变阻塞差距的存在性、类型与是否阻塞；但可以改变诊断原因码，不能扩张为"差距对象逐字段不变"。
- 审查方的反例：自动化 AC，有检查配置，`S` 有效，没有 Claim、关联提交、执行或决定。`stage = docs_only` 时进度为 `not_started`，`in_development` 时为 `unknown`。两种情况下 AC 都是 `in_progress`，原因码不同，差距类型都是 `NOT_VERIFIED`。
- F5 的断言只要求 `ready` 的结论相同，这是正确的；不新增"完整 gaps JSON 必须相同"的断言。

**产品决定与风险**：审查方没有发现需要负责人裁决的新事项，没有新增接受的风险。

**验证边界**：

- 静态核查：固定差异、规范与必要的调用链；GitHub 身份与 CI 日志。
- 在 Windows 本机离线运行 `go test -tags nowebui -mod=readonly ./internal/contract -count=1 -v`，通过。
- 作者本地的首次失败（`gitx` 超时、`sqlite` 的 M-1）与单独重跑的成功分开记录。审查方没有取得原始过程日志，不据此认定根因；CI 绿色与重跑成功都不关闭 M-1、M-10 或其他维护事项。
- 没有动态执行 M2 场景。

**负责人的决定**（2026-10-18）：

- 采纳审查 045 的 GO。
- **规范冻结**：F1 ~ F5 以审查 045 通过的规范（`d4ba0a9`），加上本节对 N1 的措辞澄清，作为完整首批的冻结依据。
- **首批实现授权**：PR #20 合并、master CI 核验之后，正式开始完整的 WP2.1、WP2.2。只授权这两个工作包，不授权后续的 M2 工作包。每个工作包一个 PR；实现 PR 不合并，不宣布验收。
- 纯归档、登记与 N1 的措辞收紧，不再触发同一轮规范的全面审查。

**收尾**（本节所在的提交）：

- N1：按审查原文收紧 data-model §6.13 第 2 类与 acceptance-criteria §3.1 的措辞，不改变规则，也不扩大保证。
- 状态同步：
  - data-model §6.13、§6.13.1、§6.14 与 acceptance-criteria §3.1 的状态行改为已冻结；
  - architecture TD-27 的第（1）部分、data-model §10、implementation-plan M2、AGENTS.md、README 同步；
  - §56 只追加一条后续说明；历史审查原文与准备报告不改写。

**保留的事项**：M1 的验收结论（`7e77fd6`）、审查 038、039、043 与 044 的结论不变；所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13 与 M0 N1）保持未关闭；没有获批的草案（M2-Q1、Q3、Q4、Q6、Q7、Q9，E1 ~ E5，恢复的退出码与交互）保持原状态与调整后的截止点；TD-27 的第（2）部分仍待定。

## 58. WP2.1／WP2.2 的联合代码审查（codex-review-046）与修复

**审查结论**：PR #21（WP2.1）**NO-GO**，PR #22（WP2.2）**NO-GO**。两者共用一项 BLOCKER（B1），另有两项 NON-BLOCKER（N1、N2）和一处冻结文字的窄范围澄清（P1）。审查原文按收到的原样入库（[docs/reviews/codex-review-046.md](reviews/codex-review-046.md)），只在 PR #21 的分支中保存一份，PR #22 经合并得到。

- **审查对象**：
  - PR #21：HEAD `1dc1774`，base `666efe2`；
  - PR #22：HEAD `221df88`，base `1dc1774`（叠加在 PR #21 之上）。
  - 审查 046 的结论只适用于这两个 HEAD，本节的修复提交不改变审查对象。
- **CI**：
  - PR #21：[run 39066962753](https://github.com/XKHoshizora/specmetry/actions/runs/39066962753)，attempt 1，8/8 成功；检出合并快照 `f61e346`（父提交为 `666efe2` 与 `1dc1774`），树 `e5f9290`。
  - PR #22：[run 39067048337](https://github.com/XKHoshizora/specmetry/actions/runs/39067048337)，attempt 1，8/8 成功；检出合并快照 `61e54dd`（父提交为 `1dc1774` 与 `221df88`），树 `7505d5c`。
  - 保留的失败记录：[run 39066356194](https://github.com/XKHoshizora/specmetry/actions/runs/39066356194)（`2e6e02b`），Go lint 失败（staticcheck QF1001、QF1002），Windows 的 Go 测试被取消。

**B1**：`factgen.Random` 生成的"合法历史"违反冻结的写入规则，因此性质测试和双向对照的输入域不成立。

- 审查方用原参数找到三个反例：
  - #21 性质测试的 seed 2：同一提案先 `reaffirm`、后 `update` 同一 AC；
  - #22 对照的 seed 5：同一提案两次更新同一需求，又两次更新同一 AC；
  - #22 对照的 seed 3：先应用 Goal add、再 Requirement update、再 Goal add，违反 §5.5。
- 这说明违反了 V-P03（同一提案的同一 target 至多一次）和 §5.5（固定的应用阶段，阶段内按 `item_index`）。
- 审查方的说明：
  - 没有复现出生产推导器错误产生 `ready` 的合法反例；
  - 不能删掉这些种子，也不能用推导器或复核器的结果来筛选样本；
  - 修复主要在 #21，#22 同步依赖。

**N1**：两套转换器没有保留 F5 列出的 `seen_decision_id`。

**N2**：R10 的联合断言只覆盖 ready 输出的证据等级。

**P1**：§6.10 的一句"此时阻塞的只有项目级的 `NO_SCOPE`"与项目级差距表在字面上冲突；实现采用"只限定条目级状态差距"的读法。审查方建议负责人作一句限定范围的澄清。

**产品决定与风险**：审查方没有发现需要新增产品功能的事项，也没有新增接受的风险。B1、N1、N2、P1 都不归为接受风险。

**验证边界**（审查方）：
- 在 Windows 本机运行了新增包的测试，全部通过；
- 用 13 个独立的原始事实用例检查了推导器的代表性正反例，全部通过；
- 用隔离的临时检查在三个种子上复现了 B1；
- 没有独立运行作者的桩阶段测试；没有在本机运行 Linux Race Detector。

**处理**（PR #21 分支，再同步到 PR #22）：

| 意见 | 处理 | 状态 |
|---|---|---|
| B1：先复现 | 新增 `factgen/legality`：只读转储行，按冻结规则独立检查，不使用 History、推导器或复核器（见下文的覆盖范围）。回归用例在旧生成器上失败：seed 2 与 seed 5 违反 V-P03，seed 3 违反 §5.5；全量扫描中 1204 个随机历史全部不合法，主要是审核中没有提案条目的显式变更、同一写入中同一实体变更两次、为未变更的实体写引用（提交 `e1498f8`） | 已复现 |
| B1：修复共同根因（审查 047：原三个反例的修复有效，另有三个缺口，见 §59） | 根因在 History：审核按调用顺序写入变更、不经提案条目，同一提案可以重复指定 target。`Review` / `Apply` 改为先计划、再按 `baseline.Apply` 的方式应用：显式变更都来自被接受的条目；一次审核只审一个提案；同一提案的 target 至多一次（跨多次审核）；按 §5.5 的阶段与 `item_index` 应用，级联紧跟其原因，第 8 步的后续处理（带理由）在最后；只为本次写入变更的实体写引用；问题只关联到被接受的新增条目。违反规则的构造会 panic。`Random` 按阶段计划每次写入：条目只指向提案提交时 active 的编号，或同一写入中本提案新增的实体；被问题指向的条目只作为新增；对实质修订的需求生成后续处理（提交 `23b3e9b`） | 已修订，待复审 |
| B1：校验器新发现的执行违规 | 环境未变却标为 `context_changed`、`input_missing` 覆盖了已发生的上下文变化、不合格的"隔离级"执行。执行上下文改为按 §7.4 生成 | 已修订，待复审 |
| B1：手写样例 | derive 与复核器测试中不审核待审条目的写入改为人工编辑；需要条目的审核逐个绑定。5 处不合法的执行记录改为合法记录。"声明了 inputs 的执行没有输入记录"是 §6.13 列出的未知状态，保留并显式标注为合法域之外。所有手写历史在推导或复核之前都经合法性校验 | 已修订，待复审 |
| B1：重新取得覆盖证据 | 性质测试与双向对照只使用经校验的合法历史，失败时给出种子与操作轨迹；按 MVP 域与通用域分别统计；畸形输入由合法历史有意破坏得到，不计入合法域 | 已修订，待复审 |
| N1 | `derive.Decision` 增加 `SeenDecisionID`，推导不读取；`toderive` 与复核器的 `fromdump` 映射它（复核器另保留执行的 worktree 与需求修订、决定的需求修订）；转换器测试检查逐表行数、F5 列出的决定与执行字段，并确认 NULL 与具体前驱 ID 都出现且被区分（提交 `8d92fc8`；#22 的 `0c329f6`） | 已关闭（审查 047，§59） |
| N2 | 复核器新增 `GradeConsistent`，对任何输出比较交付证据等级；双向对照对每个样本检查；新增"非 ready、但仍有 normal 接受贡献等级"的单元用例（#22 的 `0c329f6`） | 已关闭（审查 047，§59） |
| P1 | data-model §6.10 的该句改为只限定条目级状态差距，其他项目级差距仍按项目级表处理。这只是澄清原意，行为不变；新增用例：没有范围确认、来源变化时，`NO_SCOPE` 与 `SOURCE_CHANGED` 都阻塞。如果负责人要的是字面上的"只有 `NO_SCOPE`"，那就改变了项目级差距的阻塞语义，需要负责人另行决定，本轮不改 | 措辞已澄清；负责人已于 2026-10-18 确认（依据审查 047 的技术裁定，见 §59） |

**合法性校验的覆盖范围**（`factgen/legality`）：
- seq：
  - 表内唯一的序号列不重复；
  - 共享 seq 的行属于一次允许的逻辑写入（§6.13），共享本身不判为违规。
- 审核与人工编辑：
  - 一次审核只审一个提案；显式变更来自本次写入中被接受的条目，每个条目至多一次变更；
  - V-P03：target 是提交时 active 的编号，同一提案至多一次；关联的需求或目标是提交时 active 的编号，或同一写入中本提案新增的实体；
  - §5.5 的阶段与 `item_index` 顺序；级联紧跟其原因；后续处理在第 8 步，带理由，只针对本次实质修订的需求、且未被条目改动的 AC；
  - 修订规则（§5.5、§7.2 的强制 material）；同一写入中每个实体至多一次变更（审查 047 B1-R1：这不是 V-P03 的含义，误拒绝合法的级联；§59 改为至多一次条目变更、一次级联、一次后续处理）；引用只针对本次写入变更的实体、绑定其新修订；
  - 问题只关联到被接受的新增条目，且只在当时仍 open 时回填；
  - 附带范围绑定新版本。
- 范围确认、Claim（V-C03、V-C05）、执行（§7.4、§4.11.2，以及每个域允许的证据等级）、决定（§7.1、§7.3）的前置条件。
- 不检查（转储不含这些字段）：范围确认的 `acknowledged_waivers` 与 `seen_scope_confirmation_id`、决定的 `acknowledged_risks`、人类专属写入的操作者与通道。

**重新统计的合法域分布**（固定种子，本轮重新取得，不沿用审查 046 之前的数字）。审查 047 判定：这些只是"通过当时校验器的样本分布"，seed 4、7 是反证；已由 §59 的分布取代，保留为历史记录，不再作为当前合法域的验收证据。

| 测试 | MVP 域 | 通用域 |
|---|---|---|
| WP2.1 `TestGapsAndDelivery`（400 个） | 266 个：非 ready 185；ready 81（observed 72、none 9） | 134 个：非 ready 98；ready 36（snapshot_isolated 22、observed 10、none 4） |
| WP2.2 `TestEngineAndCheckerAgree`（600 个） | 300 个：非 ready 202；ready 98（observed 87、none 11） | 300 个：非 ready 192；ready 108（snapshot_isolated 60、observed 36、none 12） |

- 双向对照中，两个域的非 ready 样本都出现了 R1 ~ R9 每一条的失败；600 个样本的 ready 结论与证据等级（包括非 ready 输出的等级）两边全部一致。
- 合成的隔离级执行仍只是通用域的单元测试证据，不表示 MVP 具备隔离执行或自动沿用的能力。

**是否发现算法缺陷**：在修正后的合法历史上，没有发现推导器或复核器的实际缺陷；两边在全部合法样本上一致，各条件的反例与正例都成立。

本节**不宣布审查关闭，也不宣布工作包验收**，等待复审。

**保留的事项**：
- M1 的验收结论（`7e77fd6`）、审查 038、039、043 ~ 045 的结论不变。
- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13 与 M0 N1）保持未关闭。
- 没有获批的草案保持原状态；M2 只授权了 WP2.1、WP2.2。

## 59. Review 046 修复的定向复审（codex-review-047）与修复

**审查结论**：PR #21（WP2.1）**NO-GO**，PR #22（WP2.2）**NO-GO**。

- B1：Review 046 的三个原反例（seed 2、5、3）已修复，但仍有三个新缺口（B1-R1 ~ B1-R3）阻塞。
- N1、N2：可以关闭。
- P1：技术澄清成立。
- 没有发现新的、确定的生产推导算法反例。

审查原文按收到的原样入库（[docs/reviews/codex-review-047.md](reviews/codex-review-047.md)），只在 PR #21 的分支中保存一份，PR #22 经合并得到。

- **审查对象**：
  - PR #21：HEAD `b3af21b`，base `666efe2`；
  - PR #22：HEAD `bd88372`，base `b3af21b`（叠加在 PR #21 之上）。
  - 审查 047 的结论只适用于这两个 HEAD，本节的修复提交不改变审查对象。
- **CI**（审查 047 核对）：
  - PR #21：[run 39072862769](https://github.com/XKHoshizora/specmetry/actions/runs/39072862769)，attempt 1，8/8 成功；合并快照 `ec7e66a`，树 `5c07bd7`。
  - PR #22：[run 39072884994](https://github.com/XKHoshizora/specmetry/actions/runs/39072884994)，attempt 1，8/8 成功；合并快照 `ed8203d`，树 `aa027eb`。
  - 保留的失败记录：
    - [run 39071770560](https://github.com/XKHoshizora/specmetry/actions/runs/39071770560)（`bc27868`）：Go lint 失败（unparam：`review_test.go` 中 `base` 的返回值 `goal` 无人使用），其余 7 个作业成功；
    - [run 39071848979](https://github.com/XKHoshizora/specmetry/actions/runs/39071848979)（`b7d4072`）：同一处失败；
    - §58 记录的 run 39066356194。

**B1-R1**：生成器与校验器把"V-P03：同一提案的 target 至多一次"扩大成了"同一写入中每个实体至多一条变更"，误拒绝合法的级联历史。

- 例子：一次人工编辑退役 Goal G，并实质更新其需求 R 的标题。按 §5.5：
  - Goal 步骤先执行，其级联让 R 失去 G（editorial，修订仍为 1）；
  - 之后的需求步骤把 R 更新为修订 2。
- 同一版本中 R 有两条变更，其中一条是级联。`baseline.Apply` 按 `change_index` 逐条记录它们；`baseline_changes` 的唯一约束是 `(version_id, change_index)`。

**B1-R2**：同一不可变提交的同一路径，先被记为缺失、后又存在，却仍被计为合法样本。

- 例子：#22 对照参数下的 seed 4（MVP 域），截止点 41：
  - seq 27，run `…0038`：在 `head_start = 686780ca` 上，对唯一声明的输入 `a` 记为 `error` / `input_missing`；
  - seq 38，run `…003V`：在同一提交上记录了 `a` 的条目。
- 依据：§4.11.2、§7.4 规定在 `head_start` 的固定提交树中查找输入。提交树不变，所以同一提交的同一路径不能一次存在、一次缺失。

**B1-R3**：问题以局部引用关联到被接受为新 Goal 的条目，却通过了合法性校验。

- 例子：#22 对照参数下的 seed 7。Question `…001X` 在 seq 16 创建，所关联的条目 `…001V` 在 seq 20 被接受为 Goal。
- 依据：`proposal.CheckStatic` 规定，问题的局部 `about` 只能指向需求或 AC 条目。
- 受影响的手写样例：`derive/history_test.go` 中关联到 Goal 条目的问题。

**N1、N2**：审查 047 判定两项都可以关闭，本节登记为**已关闭**。

**P1**：负责人于 2026-10-18 确认了以下最小澄清：

> 没有范围确认时，条目级状态差距照常列出但不阻塞；其他项目级差距仍按项目级表处理，包括 NO_SCOPE、SOURCE_CHANGED 等各自既有的阻塞规则。

- 确认只限这一范围：不修改 ready 谓词，不批准其他待决草案，不新增接受的风险。
- 技术依据是审查 047 §三的 P1 裁定：澄清与项目级表、§6.8、F5 R9 及原有实现相容，行为不变，测试与规范一致。
- data-model §6.10 中的该句（§58 写入）与上述确认一致，本轮只补登确认，不改文字。

**产品决定与风险**：

- 审查方没有为这三个测试基座的缺陷提出产品审批要求，也没有新增接受的风险。
- 校验器没有携带的字段是证据的明确边界，不归为接受的风险。

**处理**（PR #21 分支，再同步到 PR #22）：

| 意见 | 处理 | 状态 |
|---|---|---|
| 先复现 | 提交 `42fe239` 只加测试，在受审代码上失败（见下文"修复前后"）。<br>B1-R2、B1-R3 的回归各检查两次：一次用 `legality.Check`，一次用测试中按规则直接扫描转储行，以免校验器与测试共用同一个错误假设 | 已复现 |
| B1-R1：校验器 | 提交 `63c0e4d`。取消笼统的"每个实体一次变更"。<br>• 每条变更都在本次写入前面各条留下的状态上检查。<br>• 每个实体在一次写入中至多一次条目变更（V-P03 加上"一次审核一个提案"）、一次级联、一次后续处理。<br>• 退役所欠的级联必须紧跟其原因、全部写出（§5.2）。<br>• 级联与退役不改变取值。<br>• 关联在条目应用时必须是 active 的实体（`baseline.Apply` 的 `link`），所以不能关联本次写入前面步骤退役的 Goal | 已关闭（审查 048，§60） |
| B1-R1：生成器 | 提交 `1c6705f`。<br>• `History` 允许"级联后再由条目更新或退役同一需求"。<br>• 仍拒绝：重复 target（V-P03）、target 是本次写入新增的实体、修改被级联退役的实体、关联不 active 的 Goal、第二次后续处理。<br>• `Random` 不再排除"Goal 被本次写入退役的需求" | 已关闭（审查 048，§60） |
| B1-R1：推导与复核 | 把这种合法历史送入两边，按规则核对结论，而不只是核对两边一致：<br>• 编辑前 ready、observed；<br>• 编辑时 R 为修订 2、无 Goal，AC 的 `requirement_revision` 仍为 1，原因 `parent_revised`，交付 `needs_reverification`、等级 none；复核器只有 R6 失败；<br>• reaffirm 之后，原验收继续适用，ready、observed。<br>测试：`TestGoalRetiredAndRequirementRevisedInOneWrite`（#21），`TestGoalCascadeThenARequirementRevision`（#22） | 已关闭（审查 048，§60） |
| B1-R2：校验器 | 新增规则"commit tree"：<br>• 一个 `head_start` 上，同一个规范化查找路径（`\` 换成 `/` 后 `path.Clean`）只有一种条目；<br>• `input_missing` 的执行必须有一个声明的输入，在该提交上没有任何记录显示存在。<br>只读 `head_start` 的树，所以脏工作区或上下文变化都不受影响 | 提交树矛盾的修复有效（审查 048）；零声明输入的残余见 §60 |
| B1-R2：生成器 | `Random` 为每个生成的提交固定一棵树（`Case.Trees`）：每个输入路径以 0.85 的概率存在，每个（提交、路径）有固定的条目。<br>• 执行在 HEAD 的树缺少声明的输入时、且仅在此时为 `input_missing`；否则记录树中的条目。<br>• 交付阶段：如果 AC 的检查所声明的输入在 S 上缺失，就在人工编辑中把它更新为 S 上可执行的检查。<br>• 生成器见证 `legality.CheckTrees`：逐条核对输入记录与 `Case.Trees`；性质测试与双向对照都使用它 | 修复有效（审查 048，§60） |
| B1-R3：校验器 | • 关于提案条目的问题：所指条目属于同一提案；该条目如果被接受，只能是需求或 AC 的新增。<br>• 关于现有实体的问题：只能是 active 的需求或 AC（`proposal.v1` 的 `about`、提案存储的 `find`） | 已关闭（审查 048，§60） |
| B1-R3：生成器与手写样例 | • `Random` 中有问题指向的条目只进入需求或 AC 阶段；`History` 拒绝关联到 Goal 条目的问题。<br>• `TestStateAtCutoff` 改为关联需求条目。<br>• `TestBlockingQuestions` 的"about a goal"（问题关联现有 Goal）不能由生产写入产生：`proposal.v1` 的 `about` 只接受需求编号、AC 编号或局部引用。但数据库列允许 `goal`，推导仍要处理，所以该用例保留，并显式标注为合法域之外。<br>• 新增约束：标注为合法域之外的样例，必须确实不合法 | 已关闭（审查 048，§60） |
| 正反例 | 每条规则都配了：<br>• 应接受的合法正例；<br>• 只破坏该规则的反例（结构上必然连带的规则逐条列出）；<br>• 原审查反例及相关边界：路径在另一提交上存在、另一个声明路径确实缺失、上下文变化与脏工作区、`./a` 与 `a` 是同一查找路径、问题关联到被拒绝或待审的条目 | 已补 |
| 覆盖不消失 | 全量扫描断言：这几类合法组合确实被生成（数字见下文） | 已补 |
| 生成器参数 | 有了提交树模型后，对照中"MVP：ready、none"只有 8 个，低于测试要求的 10 个。<br>提交 `4e82438` 把交付阶段"S 上没有合格执行时走例外验收"的概率从 0.2 调为 0.3（§7.3 允许的合法操作），得到 12 个。<br>没有丢弃或按结论挑选样本，门槛不变 | 已调整 |
| N1、N2 | 审查 047 判定可以关闭 | 已关闭 |
| P1 | 负责人确认（见上文） | 已确认 |

**修复前后**：

1. **受审代码**（`b3af21b` 的生成器与校验器，加上 `42fe239` 的测试）：
   - `TestReview047GoalCascadeThenAnExplicitUpdate` 失败（`one change per entity: REQ-001 changes twice`）；
   - `TestGoalCascadeThenAnExplicitChange` 的两个子用例失败（`REQ-001 changes twice in one write` 的 panic）；
   - seed 4、seed 7 的回归失败，信息与审查 047 相同（run `…0038` 在 seq 27、提交 `686780ca`、路径 `a`；Q-003 关联的条目 `…001V` 在 seq 20 被接受为 Goal）；
   - 全量扫描中 178/1204 个历史不合法，而旧校验器全部接受：
     - 按测试组：derivation 60/400，comparison 93/600，shuffle 1/4，malformed base 24/200；
     - 按缺口：提交树矛盾 76 个，问题关联 Goal 条目 103 个。
2. **`63c0e4d`**（新校验器，旧生成器）：
   - 校验器独立发现同样的 178 个历史，在全部 1204 个历史上与测试内扫描的判断一致；
   - 新增的级联、关联、取值与计数规则在旧生成器上没有误报；
   - 手写样例中被拒的只有上述两处问题。
3. **`1c6705f` 起**：全部通过。

**合法性校验的覆盖范围**（在 §58 的基础上）：

- 每个实体在一次写入中可以有多条变更：至多一次条目变更、一次级联、一次后续处理。
- 级联：
  - 每次退役所欠的级联都写出，紧跟其原因；
  - 级联与退役保持取值不变；
  - 级联的需求更新为 editorial、修订不变、只清除 Goal。
- 关联：
  - 在条目应用时必须是 active 的实体；
  - 需求的更新改变 Goal 时也受 V-P03 的提交时规则约束；
  - 更新不能把 AC 移到另一个需求。
- 提交树：一个提交上一个查找路径只有一种条目；`input_missing` 需要一个未被任何记录显示存在的声明路径。
- 问题：
  - 关于提案条目的问题：条目属于同一提案；被接受时只能是需求或 AC 的新增；
  - 关于实体的问题：active 的需求或 AC。
- 生成器见证 `CheckTrees`：
  - 树由生成器在每个历史开始时按固定种子声明。
  - 它证明这些记录可以由"每个提交一棵树"解释，不验证 Git Reader 如何计算条目，也不扩展生产 Git Reader。
  - 没有见证的手写样例只检查记录之间的矛盾。
- 不检查（转储不含这些字段）：
  - §58 列出的各项；
  - 提案条目在接受之前的 op、类型与 target，所以待审或被拒条目的类型无法检查，由生成器按构造保证；
  - 提交树中有哪些路径，所以只检查记录之间的矛盾，再加上随机历史的见证。

**本轮覆盖的合法组合**：

| 组合 | derivation（400 个） | comparison（600 个） | malformed base（200 个） |
|---|---|---|---|
| 级联之后再有条目变更 | 12 | 21 | 6 |
| `input_missing` | 35 | 58 | 18 |
| 问题回填到被接受的条目 | 81 | 109 | 37 |
| 问题关于被拒绝的条目 | 31 | 56 | 18 |

**重新统计的合法域分布**：固定种子，本轮重新取得。§58 的分布按审查 047 的判定只是"通过当时校验器的样本分布"（seed 4、7 是反证），作为历史记录保留，不再作为当前合法域的验收证据。

| 测试 | MVP 域 | 通用域 |
|---|---|---|
| WP2.1 `TestGapsAndDelivery`（400 个） | 266 个：非 ready 198；ready 68（observed 61、none 7） | 134 个：非 ready 87；ready 47（snapshot_isolated 29、observed 14、none 4） |
| WP2.2 `TestEngineAndCheckerAgree`（600 个） | 300 个：非 ready 209；ready 91（observed 79、none 12） | 300 个：非 ready 194；ready 106（snapshot_isolated 65、observed 29、none 12） |

- 双向对照中，非 ready 样本在两个域都覆盖了 R1 ~ R9 每一条的失败：

  | 域 | R1 | R2 | R3 | R4 | R5 | R6 | R7 | R8 | R9 |
  |---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
  | MVP | 5 | 18 | 121 | 35 | 97 | 105 | 7 | 64 | 29 |
  | 通用 | 5 | 16 | 118 | 31 | 84 | 99 | 12 | 62 | 25 |

- 600 个样本的 ready 结论与证据等级（包括非 ready 输出的等级）两边全部一致。
- 在新的合法历史上，以下测试都通过：乱序（4 个历史 × 1000 次）、重放、`--diff-since`、设置不改变交付、畸形输入（200 个，不计入合法域），以及手写正反例与截止点测试。
- MVP 样本没有隔离级；合成的隔离级执行仍只是通用域的单元测试证据。

**是否发现算法缺陷**：没有。在修正后的合法历史上，没有发现推导器或复核器的实际缺陷。B1-R1 的历史按规则逐项核对了截止点状态、父子修订、ready 与等级，不只以"两边一致"为依据。

**验证边界**：

- 本地在 Windows 上运行；本机不运行 Race Detector，也没有安装 golangci-lint，两者由 CI 验证。
- 没有用生产 CLI 或 SQLite 写库做动态对照。审查 047 的这一尝试也因隔离环境的权限问题没有完成。
- 真实数据库的复核器前端与 E2E 属于 WP2.9。

本节**不宣布 B1 关闭，也不宣布工作包验收**，等待复审。审查 048 的结论与处理见 §60。

**保留的事项**：

- M1 的验收结论（`7e77fd6`）、审查 038、039、043 ~ 046 的结论不变。
- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13 与 M0 N1）保持未关闭。
- 没有获批的草案保持原状态；M2 只授权了 WP2.1、WP2.2。

## 60. 合法域修复的定向复审（codex-review-048）与修复

**审查结论**：PR #21（WP2.1）**NO-GO**，PR #22（WP2.2）**NO-GO**。

- B1-R1、B1-R3：可以关闭，本节登记为**已关闭**。
- B1-R2：原提交树矛盾（seed 4）与随机生成路径的修复有效；仅剩一处残余：检查没有声明 inputs，却记录了 `error` / `input_missing`。
- 没有发现确定的生产推导算法反例，也没有发现复核器的新算法缺陷。
- N1、N2 保持已关闭（§59），P1 保持已确认（§59）。

审查原文按收到的原样入库（[docs/reviews/codex-review-048.md](reviews/codex-review-048.md)），只在 PR #21 的分支中保存一份，PR #22 经合并得到。

- **审查对象**：
  - PR #21：HEAD `ca2aec2`，base `666efe2`；
  - PR #22：HEAD `53a186c`，base `ca2aec2`（叠加在 PR #21 之上）。
  - 审查 048 的结论只适用于这两个 HEAD，本节的修复提交不改变审查对象。
- **CI**（审查 048 核对）：
  - PR #21：[run 39079396186](https://github.com/XKHoshizora/specmetry/actions/runs/39079396186)，attempt 1，8/8 成功；合并快照 `1a0e775`，树 `34b3378`。
  - PR #22：[run 39079421743](https://github.com/XKHoshizora/specmetry/actions/runs/39079421743)，attempt 1，8/8 成功；合并快照 `41eb507`，树 `aed530d`。
  - 失败与取消的记录见 §58、§59，保持不变。

**B1-R2 的残余**：

- 位置：
  - `legality.Check` 只在声明了 inputs 时才检查 `input_missing`，又允许这种执行没有输入记录；
  - `legality.CheckTrees` 对没有声明 inputs 的执行直接跳过；
  - 手写决策表样例 `criterion_test.go`"4: input_missing counts as a failure"用的是默认检查，而默认检查没有声明任何 inputs。
- 依据：data-model §7.4 第 2 步，在声明了 inputs 时才读取这些路径，路径不存在才记录 `input_missing`。零个声明输入不可能有"至少一个声明输入缺失"。补写操作者、确认信息或树见证，都无法把这种历史补全为合法写入。
- 审查方的复现：
  - 两个门禁对这种转储都返回零问题；
  - 只给 `Check` 加上这一条规则后，上述决策表样例就在合法性门禁处失败。

**产品决定与风险**：审查方没有提出新的产品决定，也没有新增接受的风险。

**处理**（PR #21 分支，再同步到 PR #22）：

| 意见 | 处理 | 状态 |
|---|---|---|
| 先复现 | 提交 `03b599f` 只加测试 `TestInputMissingNeedsADeclaredInput`。两个门禁分别断言，不只看组合调用是否最终失败。<br>在受审代码上：<br>• 零声明输入的 `input_missing`，两个门禁都返回零问题（两个用例，其中一个 HEAD 的树有路径）；<br>• `CheckTrees` 不检查"声明了 inputs，却没有输入记录"；<br>• 其余正反例在旧门禁上已经成立 | 已复现 |
| `Check` | 提交 `bb762be`。对 `input_missing` 无条件要求至少一个声明输入，与 `head_start` 无关；在比较声明输入与记录之前检查。<br>明确区分三种情况：<br>• 没有声明 inputs：不能是 `input_missing`，也不记录输入；<br>• 声明了 inputs、路径缺失：`input_missing`，没有记录；<br>• 其他：记录全部声明输入 | 已修订，待复审 |
| `CheckTrees` | 同一提交。在"没有 inputs 直接跳过"之前先处理零声明输入的 `input_missing`，单独就能拒绝，不依赖调用方先运行 `Check`。<br>另外拒绝两种记录：声明了 inputs 却没有输入记录；记录了输入，树中却缺少某个声明路径。<br>配置不在任何变更中的执行仍由 `Check` 拒绝 | 已修订，待复审 |
| 手写决策表样例 | 修复后的门禁拒绝了原样例（`input_missing of a check that declares no input`），与审查方的校准一致。<br>改为：先由一次人工编辑让检查声明一个提交 A 的树中没有的路径（material，修订 2），再记录 `input_missing`。<br>样例断言该执行绑定修订 2 与新配置、没有输入记录；同一提交上没有其他记录显示该路径。<br>它仍是合法域的决策表用例，保留 `failing` / `check_failed` 的断言 | 已修订，待复审 |
| 同类构造 | • derive 的测试中只有这一处。<br>• 复核器的测试中没有直接写 `input_missing` 的构造。<br>• `Random` 只在 HEAD 的树缺少声明路径时写 `input_missing`，空声明列表不会产生 | 无其他 |

**两个门禁的正反例**（分别断言）：

| 用例 | `Check` | `CheckTrees` |
|---|---|---|
| 没有声明 inputs：普通执行，没有输入记录 | 接受 | 接受 |
| 没有声明 inputs：`input_missing`（审查 048） | 拒绝 | 拒绝 |
| 没有声明 inputs：`input_missing`，HEAD 的树有路径 | 拒绝 | 拒绝 |
| 声明了 `a`，树中没有：`input_missing` | 接受 | 接受 |
| 声明了 `a`，另一条记录显示存在：`input_missing` | 拒绝 | 拒绝 |
| 声明了 `a`，只有树显示存在：`input_missing` | 接受（记录之间没有矛盾，存在合法补全） | 拒绝 |
| 声明了 `a`，执行没有输入记录 | 拒绝 | 拒绝 |
| 声明了 `a`、`b`，树中只有 `a`：`input_missing` | 接受 | 接受 |
| 声明了 `a`、`b`，树中两者都有：`input_missing` | 接受 | 拒绝 |
| 声明了 `./a`，树中有 `a`：`input_missing` | 接受 | 拒绝 |

- 审查 047 的提交树、多输入、路径规范化、上下文变化与脏工作区等正反例继续成立，见 `review047_rules_test.go`。

**覆盖与分布**：

- 性质测试、双向对照、合法域门禁与原有正反例都没有回归。
- 生成的历史不变，两组分布与生成组合的计数都与 §59 相同，本轮核实，没有调整任何概率：
  - WP2.1：MVP 266 个，非 ready 198、ready 68（observed 61、none 7）；通用 134 个，非 ready 87、ready 47（snapshot_isolated 29、observed 14、none 4）。
  - WP2.2：MVP 300 个，非 ready 209、ready 91（observed 79、none 12）；通用 300 个，非 ready 194、ready 106（snapshot_isolated 65、observed 29、none 12）。
  - R1 ~ R9 的失败计数与 §59 相同。

**新的直接关联问题**：

- 修复时发现，`CheckTrees` 原先也不检查"声明了 inputs 却没有输入记录"和"记录了输入、树中却缺少声明路径"。
- 这两项属于本轮要求区分的同一组情况，已在同一提交中补上。
- 没有发现其他直接关联问题。

**验证边界**：与 §59 相同：

- 本机不运行 Race Detector，也没有安装 golangci-lint，两者由 CI 验证；
- 树见证只是生成器的声明，不验证 Git Reader；
- 没有生产 CLI 或 SQLite 的动态对照。

本节**不宣布 B1 关闭，也不宣布工作包验收**，等待复审。

**保留的事项**：

- M1 的验收结论（`7e77fd6`）、审查 038、039、043 ~ 047 的结论不变。
- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13 与 M0 N1）保持未关闭。
- 没有获批的草案保持原状态；M2 只授权了 WP2.1、WP2.2。

## 61. 最后一处残余的定向复审（codex-review-049）、首批合并与 WP2.1／WP2.2 的验收

**审查结论**：PR #21（WP2.1）**GO**，PR #22（WP2.2）**GO**。

- Review 048 剩下的 B1-R2 残余已关闭。连同已关闭的 B1-R1、B1-R3，**审查 046 的 B1 整体关闭**。
- 没有 BLOCKER，没有新增的 NON-BLOCKER、产品决定或接受的风险。
- 没有发现直接回归或确定的算法反例。

审查原文按收到的原样入库（[docs/reviews/codex-review-049.md](reviews/codex-review-049.md)）。按负责人的安排，为保持受审 SHA 不变，原文在两个实现 PR 合并之后，才随本节的文档收尾 PR 入库。

- **审查对象**：
  - PR #21：HEAD `6178f9d`，base `666efe2`；
  - PR #22：HEAD `d0b3c01`，base `6178f9d`（叠加在 PR #21 之上）。
- **CI**（审查 049 核对）：
  - PR #21：[run 39104694681](https://github.com/XKHoshizora/specmetry/actions/runs/39104694681)，attempt 1，8/8 成功；合并快照 `e0fd1b0`，树 `ea13608`。
  - PR #22：[run 39104720866](https://github.com/XKHoshizora/specmetry/actions/runs/39104720866)，attempt 1，8/8 成功；合并快照 `77d6b14`，树 `d799ff2`。
- **审查方的验证边界**：
  - 在 Windows 本机离线运行；独立复验了 `03b599f`（失败）与 `bb762be`（通过）、最终包测试、11 个场景的两门禁矩阵，以及决策表样例的合法补全。
  - 跨平台与 Race Detector 的结论来自核对过身份与日志的 GitHub CI；本机没有运行 Linux Race Detector，也没有复验 macOS。
  - 原始桩阶段，以及真实的 Git、SQLite、CLI，不在本轮范围。

**负责人授权**（2026-10-18）：

- 按顺序以 merge commit 合并 #21、#22。
- 两个合并结果与受审内容一致、各自的 master CI 通过后，正式验收 WP2.1、WP2.2 的纯函数层交付。
- 建立本纯文档收尾 PR；限定内容核对正确、CI 通过后合并并核验。
- 不授权 WP2.3 或其他工作包的业务实现，不宣布 M2 整体验收。

**合并与核验**：

| 步骤 | 结果 |
|---|---|
| 合并 PR #21 | merge commit `1364af1`，父提交 `666efe2`、`6178f9d`。<br>树 `ea13608` 与受审 HEAD `6178f9d` 相同，`git diff` 无差异。<br>master CI：[run 39108257137](https://github.com/XKHoshizora/specmetry/actions/runs/39108257137)（push），attempt 1，8/8 成功 |
| #22 改 base | base 从 `feat/m2-derive` 改为 master（`1364af1`），HEAD 保持 `d0b3c01`，没有改写提交历史。<br>相对新 master 的差异：仍只有受审的 WP2.2 增量（7 个文件），与 `6178f9d..d0b3c01` 的 diff 逐字节相同；共享的 derive、factgen 源码没有差异。<br>预期合并树 `d799ff2`，与受审 HEAD 的树相同 |
| #22 新 base 的 PR CI | 改 base 不触发 `pull_request`，所以两次关闭并重新打开 PR，以 `reopened` 事件触发，没有修改代码，也没有空提交：<br>• [run 39108890146](https://github.com/XKHoshizora/specmetry/actions/runs/39108890146)：8/8 成功，但 8 个作业检出的都是旧 base 的合并快照 `77d6b14`（父提交 `6178f9d`、`d0b3c01`）。GitHub 当时尚未重算合并引用，所以**这次运行不计为新 base 的 CI**。<br>• [run 39109376799](https://github.com/XKHoshizora/specmetry/actions/runs/39109376799)：8 个作业都检出新快照 `5570bfd`（父提交 `1364af1`、`d0b3c01`，树 `d799ff2`），attempt 1，8/8 成功；Ubuntu 作业带 `-race` |
| 合并 PR #22 | merge commit `8cf54e0`，父提交 `1364af1`、`d0b3c01`。<br>树 `d799ff2` 与受审 HEAD `d0b3c01` 相同，`git diff` 无差异；master 同时包含 `6178f9d` 与 `d0b3c01`。<br>master CI：[run 39109898115](https://github.com/XKHoshizora/specmetry/actions/runs/39109898115)（push），attempt 1，8/8 成功 |

**正式验收**（产品负责人，2026-10-18，按上述授权）：**WP2.1、WP2.2 的纯函数层交付已验收**。

- 范围：
  - WP2.1：推导引擎（`internal/domain/derive`）；
  - WP2.2：独立的 `ready` 复核器（`internal/testutil/readycheck`）；
  - 已审查的纯函数、合法域测试（`factgen`、`factgen/legality`、`factgen/toderive`、`readycheck/fromdump`）及相关证据（§58 ~ §60 与本节）。
- 受审 HEAD：PR #21 `6178f9d`，PR #22 `d0b3c01`。
- 合并：`1364af1`、`8cf54e0`；master CI 为 run 39108257137、run 39109898115，均为 8/8。
- 最终的合法域分布（与 §59、§60 相同）：
  - WP2.1：MVP 266 个，非 ready 198、ready 68（observed 61、none 7）；通用 134 个，非 ready 87、ready 47（snapshot_isolated 29、observed 14、none 4）。
  - WP2.2：MVP 300 个，非 ready 209、ready 91（observed 79、none 12）；通用 300 个，非 ready 194、ready 106（snapshot_isolated 65、observed 29、none 12）。
  - 两个域都覆盖了 R1 ~ R9 每一条的失败。
- **不在这次验收之内**，由对应的工作包验证：CLI、真实数据库的读快照与高水位、Git 与磁盘读取、生产写入的前置条件与并发、输出 Schema、SQLite 复核器前端、WP2.9 的 E2E。
- **M2 尚未整体验收**；WP2.3 与其余 M2 工作包尚未授权。

**文档收尾 PR #23 的 CI 与新的维护事项 M-14**：

- [run 39110669282](https://github.com/XKHoshizora/specmetry/actions/runs/39110669282) attempt 1：Go tests (macos-latest) 失败，其余 7 个作业成功。
  - 失败测试：`internal/store/sqlite` 的 `TestSeqIsUniqueAcrossProcesses`。
  - 日志：`cannot read the database version: cannot read the database: attempt to write a readonly database (264)`。
- attempt 2 由"重跑失败的作业"触发。API 列出 8 个作业，作业 ID 都是新的，但实际重新执行的只有 Go tests (macos-latest)：08:27:53 ~ 08:29:37，检出 `4f2393d`，该测试通过。其余 7 个作业的日志与起止时间都与 attempt 1 相同，沿用的是 attempt 1 的结果。
- 这个 PR 只改文档，没有修改运行代码，所以不能把这次失败归因于本 PR 的实现改动。
- 相同代码此前通过、这次重跑也通过，都不能说明它只是 CI 环境的问题或已经无害。"另一个进程正在做 WAL 恢复，造成瞬时竞争"只是假设，不是已确认的根因。
- 登记为维护事项 **M-14**（[m1-exit-evidence.md §9](m1-exit-evidence.md)）：未解决，根因待调查，与 M-1 分开登记，不归为接受的风险；重跑通过不关闭它。
- 这次 WP2.1、WP2.2 纯函数层的验收，不表示存储层的这一问题已经解决。

**保留的记录与事项**：

- 失败、取消与不计入的运行都保留：
  - §58 的 run 39066356194（Go lint 失败，Windows 取消）；
  - §59 的 run 39071770560、39071848979（Go lint 的 unparam 失败）；
  - 本节的 run 39108890146（检出旧快照）。
- 审查 044 ~ 048 的审查对象与结论保持原样。
- M1 的验收结论（`7e77fd6`）、审查 038、039、043 ~ 048 的结论不变。
- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13、M-14 与 M0 N1）保持未关闭。
- 没有获批的草案、风险边界保持原状。
- 审查 044 的 N1（提交关联的生命周期）仍在 WP2.3 开工前与 M2-Q7 一起冻结。

## 62. WP2.3 的负责人决定与规范草案

**负责人决定**（2026-10-18）：

1. **M2-Q7 / TD-17**：
   - MVP 固定采用 `Specmetry-Ref` trailer 加正文编号匹配，不提供关闭正文匹配的开关。
   - trailer 键不区分大小写。
   - 编号遵守既有编号规范，不扩大类型，也不放宽格式。
   - 只关联本项目已有的需求与 AC，可以包括已退役的。
   - 同一提交对同一实体只保留一个关联，trailer 优先。
   - 关联只提供参考进度，不代表完成、检查通过或验收。
2. **关联的生命周期**（审查 044 N1）：
   - 已入库 SHA 的关联在首次入库时固定，MVP 不重算、不补写。
   - 没有入库的提交，以后重扫时如果首次满足入库条件，按这次的发现序号入库，不回填旧序号。
   - 常规增量扫描不主动回头重算没有入库的旧提交。
   - 代价：后来新增的编号，不保证能与此前扫描过的提交建立关联；重扫后的首次入库，可能产生新的参考进度。
   - 不新增"所有已见 SHA"的账本，也不为此给 `commit_links` 增加 `seq`。
3. **没有新入库提交的扫描**：
   - 不分配业务 `seq`，可以更新游标。
   - `scanned_seq` 的含义、首次扫描没有匹配的提交、不同 ref 共用 SHA 等情况，写成可审查的规则。
   - 有新增提交时，沿用 F4 的逻辑写入与共享 `seq`。

这些决定不修改已冻结的进度或 ready 规则，不批准其他 M2 草案。WP2.3 的实现尚未授权。

**规范草案**（本节对应的 PR）：

- 新文档 [wp2.3-git-reader.md](wp2.3-git-reader.md)，内容包括：
  - TD-23 的读取机制与实验；
  - 工作区状态的读取；
  - 提交关联的匹配、扫描范围、生命周期、写入事务、失败与重试；
  - `internal/gitx` 的新接口与扫描层的职责；
  - 迁移设计（`commits`、`commit_links`、`git_scan_cursors`）；
  - 冲突与待决事项、正反例、验收映射、实现测试计划，以及 M-7 的处理顺序。
- 同步的规范：
  - data-model：§4.1 不提供 `commit_ref_mode`；§4.11.2 的读取机制改为指向 WP2.3 规范；§4.12 写入上述决定；§6.13 的提交行写明"不补写"这一前提已经成立。
  - architecture：§6.7；§16 的 TD-17（已决定）与 TD-23（读取机制的方案待审查）。
  - acceptance-criteria：新增 MVP-WT-06、MVP-EV-14 ~ 17，并更新需求追溯。implementation-plan 的 M2 退出条件随之包含它们。
  - implementation-plan：§3 的两行、M2 的状态、M-7 的顺序。
- 本 PR 不提交生产代码、迁移 SQL 或实现测试，不改写历史审查。

**实验**：

- 环境：Git for Windows 2.56.0，Windows 11；隔离的临时仓库；与 `gitx` 相同的受控环境与全局参数。
- 结果：见 WP2.3 规范 §4.2（T1 ~ T23）、§4.3（S1 ~ S8）、§5.2（L1 ~ L10）。
- 未验证：Linux、macOS、2.32 ~ 2.43 版本、真实的部分克隆、大型仓库的耗时。

**冲突与待决事项**（WP2.3 规范 §8）：

- **D1（冲突，需负责人决定）**：
  - data-model §1 冻结的 `clean` 命令中没有 `--ignore-submodules=none`。仓库配置 `diff.ignoreSubmodules=all` 时，子模块不一致会被判为 `clean`（实验 S7），与 §4.11.2 矛盾。
  - 推荐：在定义中加入该参数。
- **D2（冲突，需负责人决定）**：
  - "各 worktree 的分离 HEAD"中，没有登记的 worktree 没有 `worktree_id`，游标没有身份。
  - 推荐：分离 HEAD 的游标按规范化的 worktree 路径识别，写入前做密钥扫描。
- **D3 ~ D8（审查）**：
  - D3：仓库根的记录；
  - D4：新 ref 的扫描范围；
  - D5：还没有编号时不扫描；
  - D6：并发时整次不写；
  - D7：ref 名称命中密钥规则时跳过该 ref；
  - D8：`ls-tree` 的意外输出按读取失败处理。

**状态**：规范草案，待独立审查；不宣布冻结。

**保留的事项**：

- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13、M-14 与 M0 N1）保持未关闭。
  - M-14 另行诊断，结果不进入本 PR。
  - M-7 须在 WP2.4、WP2.5、WP2.7 的新输出接口之前修复。
- WP2.1、WP2.2 的验收（§61）、M1 的验收结论与其他审查结论不变。
- 没有获批的草案保持原状态。

## 63. WP2.3 规范的独立审查（codex-review-050）、负责人的第二轮决定与修订

**审查**：[codex-review-050](reviews/codex-review-050.md)（原样归档，未修改）。

- 审查对象：PR #24，HEAD `9c0973535a18019b3d02db4c35749937a0c5f50a`，base `a236e5350ff92dcffd9bc05d09b1f3aac3996704`。
- 审查方核对的 CI：run 39114809047，8/8 成功；检出合并快照 `abe8efaea7d6d7ca1281b3fd4597f95bef13f972`，内容树 `85678772b064fd70456f96026f81d040d8884a45` 与固定 HEAD 相同。审查方说明，这次 CI 只对已提交的文档与现有实现有效，不是 T1 ~ T23、S1 ~ S8 或扫描器的证据。
- **结论：NO-GO**，不建议合并，不授权 WP2.3。
- **BLOCKER**：
  - B1：`--ignore-missing` 不只作用于排除项，正向 tip 缺失时被当作成功的空扫描，游标随之前进；
  - B2：旧 tip 缺失的退出码判断不成立（`merge-base --is-ancestor` 与带 `^{commit}` 的 `cat-file -e` 都返回 128），无法执行承诺的改写后重扫；
  - B3：`Clean` 的 `status` 会执行仓库配置的 clean / process 过滤器，受控环境与 Lazy Fetch 门禁不足以保证只读。
- **NON-BLOCKER**：
  - N1：git 2.32 的命令支持不能从 Windows 2.56 推广（2.32 的 `worktree list` 没有 `-z`，`--stdin` 不接受 `--not` 行），需要兼容矩阵与受控的拒绝路径；
  - N2：没有业务候选时仍需游标事务；EV-15 的验收需要前提；
  - N3：`committed_at` 的单位与通用约定（UTC Unix 毫秒）不一致。
- **D1 ~ D8 的裁定**：
  - D1、D2、D7 是产品决定；
  - D3、D8 是合理的工程细化，D4、D5、D6 方向可以保留（D4 须改正 B1 的手段，D6 不得声称 git 的 ref 也被锁定）。
- **审查认可、本轮不重开的部分**：已批准的关联规则、生命周期与"没有新入库提交不分配 `seq`"；编号解析与 trailer 优先；逐路径读取输入元数据（TD-23）的方向。
- **其他建议**（审查 050 §7、§8）：
  - `commit_links` 的 `BEFORE INSERT` 检查；只追加保护覆盖 UPDATE / DELETE / REPLACE；
  - "关联只在提交首次入库的事务中写入"是必需的不变量，由唯一的写入入口、跳过旧 SHA 与回归测试保证，不使用"父提交的 `seq` 等于计数器"的证明；
  - B1 ~ B3 的反例要进入技术验收；单看"git 非 0 就拒绝"发现不了 B1 的假成功。
- **M-14 的边界**（审查 050 §9）：
  - 同一段"cannot read the database version"的包装在 `store.go` 中出现三处，不能只凭这段文字定位到只读预检；
  - 辅助探测重现相似的错误，不等于验证了真实 `Open` 的完整路径，也不等于证明 macOS 上同根因；
  - "本次业务调用没有提交"与"数据库、WAL、SHM 的字节完全不变"是两种结论，后者没有验证；
  - 这次 CI 成功不关闭 M-14。

**负责人决定**（2026-10-18）：

- **D1**：采用在 `clean` 定义中加入 `--ignore-submodules=none` 的方向，同步 `clean` 的定义与验收。外部过滤器的问题必须另行解决，不能因此宣称现有的 `status` 方案安全。
- **D2**：保留所有 worktree 的分离 HEAD，以规范化路径作为游标身份，不缩小到已登记的 worktree。明确路径移动、删除、重用、大小写与规范化的行为。
- **D7**：不静默跳过敏感的 ref 或路径。命中密钥规则时拒绝本次扫描，不写提交、关联、`seq` 或游标，不回显敏感值。错误分类遵守既有契约，不新增隐私豁免。
- 其余没有获批的草案保持原状。

**修订**（PR #24 的后续提交；规范见 [wp2.3-git-reader.md](wp2.3-git-reader.md)）：

| 问题 | 修订 | 位置 |
|---|---|---|
| B1 | 不再使用 `--ignore-missing`。每个 ref 一次严格的 `git log -z --stdin`：stdin 中先写正向 tip，再为每个排除项写 `^<完整 SHA>`。排除项只用**已确认存在**的旧 tip；可靠地不存在的旧 tip 不作排除项，只会让范围变大。任何非 0 退出都使整次扫描失败（退出码 6），游标不前进；不因命令成功或输出为空就推进游标 | 规范 §5.2 第一、四步，§5.6；architecture §6.7；data-model §4.12 |
| B2 | 三态分类，只看协议输出与退出码，不看错误文字：一次 `cat-file --batch-check` 读入所有旧 tip；`<sha> commit <n>` 为存在；`missing` 时再用裸 ID 的 `cat-file -e`：1 为可靠地不存在（按改写重扫），0 为存在但不可读（退出码 6）；其他一律退出码 6。不使用 `^{commit}`。`is-ancestor` 只对已确认存在的旧 tip 运行：0 为增量，1 为改写，其他为退出码 6 | 规范 §5.2 第二、三步 |
| B3 | 不在用户的仓库上直接运行 `status`。Specmetry 建立私有的 git 目录：`HEAD`、指向仓库对象目录的 `alternates`、索引的副本、`info/exclude` 与 `info/attributes` 的副本，以及只含白名单键的 `config`（没有任何 `filter.*`、`core.fsmonitor`、`core.hooksPath`、`include*`）。超级项目用 `--ignore-submodules=all` 读取，不启动子进程；已初始化的子模块逐个读取（HEAD 与 gitlink 比较，再递归）。调用之前就保证不执行程序，不依赖"仓库可信"或事后检测。保守的差异：属性指定了过滤器、并且 stat 变化的文件按未经过滤器的内容比较，只会把 `clean` 变成 `dirty` | 规范 §4.3；data-model §1 |
| N1 | 新增兼容矩阵（规范 §13）：最低 2.32 只是配置隔离的下限；PD-26 另由探测决定（真实的 2.32.0 在读取对象之前就被拒绝）；不支持 `worktree list -z` 的 git 上，扫描返回退出码 6，不按行解析；stdin 中不写 `--not` 行。不提高最低版本，不推翻 PD-26。低于 2.36、却通过 PD-26 的 git 上的影响列为待决（规范 §8） | 规范 §5.1、§13；architecture §6.7 |
| N2 | 没有候选时，游标仍经过整次扫描的写事务；删去"不必进入写入事务"。EV-15 的验收加上前提：项目已有其他需求或 AC | 规范 §5.3、§9；acceptance MVP-EV-15 |
| N3 | `committed_at` 改为 UTC Unix 毫秒（git 的秒 × 1000） | 规范 §7；data-model §4.12 |
| D1 | `clean` 的定义加入 `--ignore-submodules=none`，并写明这条命令只定义语义、不是读取方式；读取方式见 B3 | data-model §1；规范 §4.3；新增 MVP-EV-18 |
| D2 | 分离 HEAD 的游标身份为 `gitx.Canonical`（`Clean` 后 `EvalSymlinks`；路径不存在时解析父目录；再不行只做字面的 `Clean`）。移动后为新游标、按新 ref 读取，原游标保留用于排除；`prune` 之前照常扫描、之后游标保留；同一路径上的新 worktree 沿用原游标、按改写重扫；Windows 上统一大小写，macOS、Linux 按字节；Unicode 组合方式未验证；路径保存之前做密钥扫描 | 规范 §5.1；data-model §4.12；architecture §6.7；MVP-WT-06 |
| D7 | 要写进游标的 ref 名称或路径在读取任何提交之前扫描；任何命中都拒绝整次扫描，沿用 architecture §10.8 与 §6.1.2（`init` 对分支名的先例）的分类：`invalid_input`（退出码 2），只给出名称的类别与规则，不回显；不静默跳过。调用扫描的命令如何呈现，由 WP2.7 确定 | 规范 §5.1、§5.6；architecture §6.7；data-model §4.12；MVP-WT-06 |
| D6 | 措辞改为"只保证本次不发布过期的计划，下次重扫"，不声称 git 的 ref 被锁定 | 规范 §5.5、§8 |
| 固定读取结果 | 写明扫描开始时取得的 tip 是固定的读取结果，不是全局原子快照 | 规范 §5.1 |
| 审查 §7 的建议 | `commit_links` 的 `BEFORE INSERT` 触发器（同项目、对应类型）；只追加保护覆盖 UPDATE / DELETE / REPLACE 并沿用 `recursive_triggers`；"关联只在首次入库的事务中写入"的不变量由唯一入口、跳过旧 SHA 与回归测试保证；不使用计数器的证明 | 规范 §6、§7 |
| 审查 §8 | B1 ~ B3 的反例进入验收：EV-16 断言游标没有前进；EV-18 用写标记的过滤器断言没有程序执行；WT-06 标明 D2、D7 已决定，D4、D5 仍是待复审的工程细化 | acceptance |

**实验**（作者在仓库之外、隔离夹具中运行，没有提交；文档 CI 不是这些实验的验证）：

- 环境：Git for Windows 2.56.0（`git version 2.56.0.windows.2`），Windows 11（MINGW64_NT-10.0-26300）；与 `gitx` 相同的受控环境与全局参数。
- **严格读取与分类**（规范 §5.2 的 R1 ~ R13）：
  - 正向 tip 不存在：严格读取 128，带 `--ignore-missing` 时 0 且没有输出（复现 B1 的假成功）；正向 tip 损坏：128；排除项不存在：128；
  - 不存在的 SHA：`is-ancestor` 128，`cat-file -e <sha>^{commit}` 128，裸 ID 的 `cat-file -e` 1；
  - 损坏或为空的松散对象：`--batch-check` 报告 `missing`，裸 ID 的 `cat-file -e` 返回 0，因此能与真正的不存在（1）区分；
  - 被回收的旧 tip：`missing` 且 `cat-file -e` 为 1；
  - 打包对象的数据被改写：没有被误判为不存在；
  - 正常扫描：0，结果正确；`c3` 与 `^c3`：0 且没有输出。
- **工作区状态**（规范 §4.3 的 C1 ~ C11）：
  - 对照：原草案的 `status` 执行了 clean 与 process 过滤器；在已初始化的子模块中启动了 2 个子进程，执行了子模块配置的过滤器。
  - 本方案：所有情形都没有标记、子进程为 0；修改、未跟踪、已暂存、删除为 `dirty`，被忽略的文件与只改修改时间的普通文件为 `clean`；会改写内容的过滤器下只改修改时间为 `dirty`（保守的差异）；子模块的提交与 gitlink 不同为 `dirty`，`diff.ignoreSubmodules=all` 时也一样。
- **未验证**：Linux、macOS 与其他 git 版本（2.32、2.43 只有审查方的静态核查）；真实的部分克隆、长驻 process 过滤器协议、LFS、拆分索引与稀疏索引、`extensions.worktreeConfig`、以 gitfile 指向的子模块与多层嵌套；对象目录没有读取权限或 I/O 错误时 `cat-file -e` 的结果；真实的并发 `gc`；NFC / NFD；大仓库的耗时。这些都列入实现测试（规范 §11）。

**验收的变化**：

- MVP-WT-06：D2 的身份规则、D7 的拒绝（退出码 2、什么都不写）、`worktree list -z` 不支持时退出码 6；去掉"该 ref 跳过"的草案期望。
- MVP-EV-15：加上前提。
- MVP-EV-16：加上严格读取与三态分类的正反例，以及 `committed_at` 的单位。
- **新增 MVP-EV-18**：工作区状态的读取不执行仓库配置的程序，子模块变化与保守的差异（M2，集成，FR-EV-04）。
- 需求追溯 FR-EV 改为 EV-01 ~ 18；implementation-plan 的 M2 退出条件改为 EV-12 ~ EV-18。

**维护事项 M-15**（新增，登记在 [m1-exit-evidence.md §9](m1-exit-evidence.md)）：

- 来源：PR #25（M-14 的诊断工具）的被测提交 `1d8b73c9e637e339f2e1ebac4c81b5bd029480de`，[run 39118832445](https://github.com/XKHoshizora/specmetry/actions/runs/39118832445)。
- 作者报告：Windows CI 的真实 `Open` 场景中，写入阶段在 200 轮中的 2 轮出现 3 次 `SQLITE_BUSY`（`database is locked (5)`），分类为 internal（退出码 1）。
- 这是**待独立审查的诊断观测**，与 M-14（打开时的版本探测）分开登记，没有证据表明二者同根因。
- 未解决，不是已接受的风险；它本身不批准任何错误映射或重试策略。
- PR #25 的 HEAD 保持 `db8d7bc29269727448f5dba893e3dce2a165a460` 不变。

**状态**：规范按审查 050 修订，待复审；不宣布冻结；**WP2.3 的实现尚未授权**；PR #24 保持未合并。

**保留的事项**：

- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13、M-14、M-15 与 M0 N1）保持未关闭。
- 审查 050 的原文与审查对象的身份不变；此前各节的记录不改写。
- WP2.1、WP2.2 的验收（§61）、M1 的验收结论与其他审查结论不变。
- 没有获批的草案保持原状态。

## 64. WP2.3 规范的定向复审（codex-review-052）与修订

**审查**：[codex-review-052](reviews/codex-review-052.md)（原样归档，未修改）。

- 审查对象：PR #24，HEAD `f82e166a07b28cff8db0d69987caac41231be832`，base `a236e5350ff92dcffd9bc05d09b1f3aac3996704`；审查 050 的原对象 `9c0973535a18019b3d02db4c35749937a0c5f50a`。
- **规范技术结论：NO-GO**，尚不具备整体冻结条件。
- **合并检查：未通过**。审查方核对的 CI run 39121984050 attempt 1 为 7/8：检出合并快照 `2a0bf2281380041563c4ce693dd924b87e41b0c8`，内容树 `ea92fad2972df1fe3c1c9ef86b14fa7bafdba2b1` 与 HEAD 相同；Ubuntu 的 Go tests 失败（`TestBuildFixtureIgnoresInheritedGitEnvironment/GIT_CONFIG_COUNT_hooks_path`，`gitenv_test.go:70`，`maintenance.lock` 不存在）。审查方没有重跑，也没有据此认定根因。
- **BLOCKER**：
  - B1-R：commit-graph 使已经消失的排除对象仍被严格 log 接受（Git for Windows 2.56：退出 0；关闭 commit-graph：128）。审查 050 B1 的原始普通反例已修正，完整的拒绝保证未关闭。
  - B2-R：旧提交所在包的 `.idx` 被破坏后，`--batch-check` 报告 `missing`、裸 ID 的 `cat-file -e` 返回 1，被判为可靠地不存在。
  - B3-R1：取消过滤器之后，"差异只会把 `clean` 变成 `dirty`"不成立，存在定义下 `dirty`、私有目录方案 `clean` 的反方向。
  - B3-R2：alternates 不提供只读保护；稀疏索引的展开会改变原对象文件的修改时间。
- **NON-BLOCKER**：N1（gitfile 子模块与 unborn HEAD 的构造步骤）、N2（临时副本的生命周期与错误标签）、N3（D2 重用路径的验收措辞须保留祖先判断）。
- **产品决定**：D1、D2、D7 的批准保留，不重开。审查方指出两项**不能算作已批准**：取消过滤器之后的安全退化策略及其用户可见的限制；对通过 PD-26 却不支持 `worktree list -z` 的低版本构建，是否长期不提供扫描。
- **已接受的风险**：只有 §62 的关联生命周期限制；不新增。
- **审查方认可、本轮不重开的部分**：关联规则、生命周期、`seq` 与事务原则；输入元数据的逐路径方向；D3 ~ D6、D8 的合理部分；审查 050 N2、N3 的落实。

**修订**（PR #24 的后续提交；规范见 [wp2.3-git-reader.md](wp2.3-git-reader.md)）：

| 问题 | 修订 | 位置 |
|---|---|---|
| B1-R | 重新论证遍历与对象读取的关系：git 可借 commit-graph、grafts、替换引用与浅边界在不读取对象的情况下完成遍历。扫描的读取固定 `-c core.commitGraph=false -c core.multiPackIndex=false`，设置 `GIT_GRAFT_FILE` 使 grafts 失效，替换引用沿用既有的 `--no-replace-objects`；浅克隆待 P3。读取中途失败时已读到的部分结果丢弃。不再仅凭 `log` 退出 0 证明对象都已读取：被访问到的提交都要读取对象，没有被访问到的不影响范围 | 规范 §5.2"遍历必须读取对象"、第四步；data-model §4.12；architecture §6.7 |
| B2-R | 重新定义分类：可靠地不存在需要 `missing`、裸 ID 的 `cat-file -e` 为 1、两条命令都**没有任何 stderr 输出**（只看有没有，不看文字），并且**存储检查**通过（对象目录都能列出；每个 `.idx` 的校验和正确；对应的 `.pack` 存在、可读、尾部校验和与 `.idx` 一致；alternates 可读且目录存在）。其他一律无法确认，退出码 6。保留了真正被回收的旧 tip 的重扫要求，没有把所有 `missing` 改为拒绝。写明判据不能证明的情形 | 规范 §5.2 第二步；data-model §4.12；architecture §6.7 |
| B3-R1 | 撤回"只会 clean → dirty"。不可判定的情形：仓库 local（及 worktree）配置定义了过滤器驱动，并且受过滤的文件需要比较内容（stat 与索引不同，或 racy）。racy 单独判断，因为 `diff-files` 对 racy 条目只做原样比较。不可判定的取值待 P1 | 规范 §4.3 第 4 步；data-model §1 |
| B3-R2 | 复制 `info/sparse-checkout`；列举只用 `ls-files --sparse`；带稀疏目录条目的索引为不可判定（工作区中出现与稀疏目录同名的目录时，即使复制了模式文件，私有 `status` 仍改变原对象的修改时间，这是本轮新发现的反例）；范围待 P2。普通索引、拆分索引、untracked cache、没有稀疏索引的稀疏检出，实验中原仓库文件的内容与修改时间都不变 | 规范 §4.3；data-model §1 |
| N1 | 子模块按 `rev-parse --absolute-git-dir` 解析出的实际 git 目录读取；unborn 时私有 `HEAD` 写为符号引用 | 规范 §4.3 第 1、3 步 |
| N2 | 私有目录的位置、权限、固定前缀、成功／失败／取消都清理、清理失败返回环境错误、遗留目录的识别与清理；不进入事实、日志或备份；错误中的子命令名称是固定标签，不从带 `--git-dir=<路径>` 的参数中取 | 规范 §4.3、§5.6 |
| N3 | 重用路径时仍按旧 tip 的存在性与祖先关系决定增量或重扫 | 规范 §5.1、§9 L20；MVP-WT-06 |
| 审查的其他要求 | `Classify.Present` 只表示对象头可读；EV-16 覆盖 commit-graph、grafts、部分结果与损坏的包索引；EV-18 覆盖过滤器的两个方向、原对象的内容与修改时间；兼容矩阵补上新命令，标明哪些在 2.43 上实际运行 | 规范 §6、§10、§13；acceptance-criteria |

**实验**（作者在仓库之外、隔离夹具中运行，没有提交；文档 CI 不是这些实验的验证）：

- 环境：Git for Windows 2.56.0（Windows 11）；WSL Ubuntu 的 git 2.43.0（非 root 用户）。都使用受控环境与全局参数。
- **G1 ~ G7**（遍历）：commit-graph 的绕过在 2.56 上出现、在 2.43 上不出现，关闭后两者都为 128；grafts 隐藏缺失的父提交，`GIT_GRAFT_FILE` 指向不存在的路径后为 128；浅边界之外的缺失不报错（P3）；`log` 可能先输出部分结果再失败。
- **K1 ~ K17**（分类）：只有真正被回收的旧 tip 被判为可靠地不存在。`.idx` 的 SHA 表被静默改坏、`.pack` 被删除或不可读、`.idx` 不可读、包目录不可进入时，都是 `missing`、1、**没有** stderr，只有存储检查能发现；审查 052 的反例（截断的 `.idx`）有 stderr，存储检查也能发现。
- **F1 ~ F7**（过滤器）：审查 052 的两个方向都复现；racy 的反方向（F7）只有 racy 规则能发现；私有读取中过滤器都没有执行。
- **S1 ~ S14**（原对象）：复现了审查 052 的稀疏索引反例；新发现 S8（稀疏目录同名的目录存在时，复制模式文件也不够）；其余布局没有变化。
- **N1**：本文环境中，直接传 gitfile 的 `--git-dir` 返回 0（审查方的环境中为 128），所以不依赖它；省略私有 `HEAD` 时 `status` 返回 128。
- **未验证**：macOS，2.43、2.56 以外的版本；真实的部分克隆、并发 `gc`、网络与虚拟文件系统；提交时间倒置时关闭 commit-graph 的遍历范围；稀疏索引的其他触发条件；长驻 process 过滤器协议、LFS 的本地驱动、多层子模块；大仓库的耗时。

**待负责人决定**（D1 不包括这些）：

- **P1**：不可判定记为什么。推荐 `unknown`；其他选项为 `dirty`、环境错误、不支持定义了本地过滤器驱动的仓库。
- **P2**：稀疏索引的范围。推荐带稀疏目录条目的索引一律不可判定；其他选项为预先检查后照常读取（没有证明其他触发条件不存在）、不支持 sparse index。
- **P3**：浅克隆。推荐按 git 声明的边界停止，并写明扫描只覆盖本地存在的历史；其他选项为使浅边界失效、浅克隆一律失败，或浅克隆不扫描。
- **低于 2.36、却通过 PD-26 的 git**：推荐保持 fail-closed；"长期不提供扫描"这一兼容性取舍待决定。

**CI 失败的处理**：

- run 39121984050 的失败与本规范的修订分开处理，没有重跑 #24 的那次 CI。
- 原因与修复见独立的 PR #26（测试夹具关闭自动维护与自动 gc；未合并，待独立审查）。它不混入 #24。

**状态**：规范按审查 052 修订，待复审；不宣布冻结；**WP2.3 的实现尚未授权**；PR #24 保持未合并。

**保留的事项**：

- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13、M-14、M-15 与 M0 N1）保持未关闭。
- 审查 052 的原文与审查对象的身份不变；此前各节的记录（包括 §63 中当时的 EV-18 表述）不改写。
- WP2.1、WP2.2 的验收（§61）、M1 的验收结论与其他审查结论不变。
- 没有获批的草案保持原状态。

## 65. 测试夹具的自动维护隔离（PR #26）的独立审查（codex-review-053）

> 编号说明：§62 ~ §64 由尚未合并的 PR #24 使用，所以本节编号为 §65。

**审查**：[codex-review-053](reviews/codex-review-053.md)（原样归档，未修改）。

- 审查对象：PR #26，HEAD `ba75ff310e82c116c0eaa31a7a2ba8719a828677`，base `a236e5350ff92dcffd9bc05d09b1f3aac3996704`。补丁只修改 `internal/testutil/git.go`、新增 `internal/testutil/maintenance_test.go`（141 行增加、4 行删除）。
- 核对的 CI：run 39127098623，attempt 1，8/8 成功；检出合并快照 `f4b7b6fa31a5cfd9aa987368a28e422443ad307c`，两个父提交依次为 base、HEAD，内容树 `54726dd495061280247187f69460bba6c402989e` 与 HEAD 相同。
- **结论：GO**，没有 BLOCKER。这不表示原 Ubuntu 偶发失败已有排他性的根因证明，不关闭 M-14、M-15，也不授权 WP2.3。
- **审查方认可的部分**：
  - 两个开关有效，作用域只在夹具私有的全局配置中：不写用户的真实配置，不改生产的 git 行为。
  - trace2 正对照与"移除配置"的变异结果相互支持：变异中普通夹具提交、两种继承变量、配置内容四项失败，正对照通过。测试实际观测到了有无修复时子进程的差异。
  - 继承的 `GIT_*`、HOME、XDG_CONFIG_HOME 仍被过滤；Hook、模板、对象目录、索引与外部仓库的既有隔离检查仍然有效。
  - 快照（`snapshot.go`，两端的 blob 都是 `9cb674c55bc412538bd0bf59b835a53985c70baa`）、`AssertTreeUnchanged` 与 `plainGit` 所在的文件（两端的 blob 都是 `ab9f32404bd63cc99380b08af27448935192e11b`）都没有改变。
  - CI 中三个平台的相关测试都实际运行；Ubuntu 的测试步骤启用了 Race Detector；Linux 专属的 root 跨用户步骤成功，macOS 与 Windows 按设计跳过。
- **验证边界**（审查方披露）：
  - 修复的边界是夹具默认的受控环境。测试有意使用 `-c maintenance.auto=true`、仓库本地的显式覆盖或手动运行 maintenance / gc 时，仍可改变行为。
  - trace2 的读取器是可信隔离测试中的窄范围观测器，不是 PD-26 的能力分类器，没有验证完整的 SID 与根进程的生命周期。
  - 审查方的独立运行：Windows，Git 2.56.0.windows.2，Go 1.27.1。定向命令退出 0（约 80 秒）；移除配置的变异只运行两个新增测试，退出 1（约 35 秒），等价于恢复补丁之前的空配置，不是重放原 Ubuntu 的竞争。
  - **另一次较宽的本机运行**包含 `TestProjectFixtures`，达到审查方自设的 3 分钟上限，堆栈停在 `applications` 子测试等待其应用命令，整个运行退出 1。**这一批不记为通过**；审查方没有据此确定耗时的根因，也没有宣称存在死锁。`TestProjectFixtures` 的成功证据来自 CI，而不是这次运行。
  - 审查方没有在本机重跑 Linux、macOS 或旧版 git，也没有自行触发 Actions。作者的锁轮询、人工构造的快照竞争与 WSL 中的旧版 git 记录属于作者证据，审查方没有重做。
- **NON-BLOCKER N1**：原 CI 根因与后台分离的表述应保留证据限定。证据支持修复所针对的机制，但不能单独证明原失败时具体进程的生命周期、唯一根因，以及所有 git 版本的后台分离行为；正对照显式关闭了分离，它证明的是观测器能看见维护子进程。

**处理**（负责人授权按 GO 收尾并合并）：

- N1：只修正措辞，不改变配置开关、测试行为、快照、`AssertTreeUnchanged` 或 `plainGit`：
  - `git.go` 的注释与 `maintenance_test.go` 的说明注释：原失败"与自动维护锁的竞争一致"；本修复消除已经观察到的自动维护启动路径；原 CI 的具体竞争没有直接复现；后台分离限定为有 `maintenance.autoDetach` 的 git、在平台允许分离时。
  - PR 描述的 Summary 同样修正。
- 合并方式：merge commit，锁定最终 HEAD。合并后的 merge SHA 与 master 的 CI 记录在 PR #26 中。

**保留的事项**：

- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13、M-14 与 M0 N1）保持未关闭；M-15 的登记随 PR #24，尚未合并。
- PR #24、#25 不在本轮处理；#26 合并会推进 master，它们与新 master 的整合以后另行核对。
- 没有获批的草案保持原状态。

## 66. WP2.3 规范的定向复审（codex-review-055）、负责人的第三轮决定与修订

**审查**：[codex-review-055](reviews/codex-review-055.md)（原样归档，未修改）。

- 审查对象：PR #24，固定 HEAD `5d707eb28e95631cf97808a3287d81375fd0e60f`，固定 base `a236e5350ff92dcffd9bc05d09b1f3aac3996704`；上一轮的 HEAD `f82e166`。审查方另记录了当时的 master 已前进到 `46a4b26`，差异仍按固定 base 计算。
- 核对的 CI：run 39132130490，attempt 1，8/8；检出合并快照 `b57ba45eaf75c7c8b334cc6ebe8680063d21cb81`，内容树 `b0152f2642a892f2140c04c630a8ff7e5c865f8b` 与 HEAD 相同。审查方说明它只是文档提交对应的既有测试通过，不是 G / K / F / S 实验或 Git Reader 的验证；原 run 39121984050 的 Ubuntu 失败记录保留。
- **规范技术结论：NO-GO**，不具备整体冻结条件；与 master 有文档冲突，整合未核验；WP2.3 未授权。
- **BLOCKER**：
  - **B2-R2**：存储检查只查 idx → pack。一个可读的 `.pack` 失去 `.idx` 时，"每个 `.idx` 都有 `.pack`"在空集合上成立，旧提交被当作可靠缺失；恢复 `.idx` 后同一个对象又能读出。
  - **B3-R2**：稀疏索引的判断在第 4 步，晚于第 2 步的私有 `status`。审查方用复制了模式文件的私有目录复现：`status` 展开索引，3 个原树对象的修改时间改变、内容不变。把结果标为不可判定只是结果门禁，不能撤销已发生的修改。
- **产品决定**（审查方提出）：P3 的静态描述不足以决定浅克隆加深之后的扫描。审查方用无网络的状态模拟说明：在浅边界上建立的游标，加深之后 `last_tip = T`，边界之外的提交 A 永远不会被读到。
- **审查方认可的部分**：commit-graph 的原反例已修正；grafts、替换对象、multi-pack-index 的处理方向成立；部分输出的丢弃；坏 `.idx` 等的修正（受 B2-R2 限制）；过滤器的判据方向成立（取值待 P1）；白名单配置；N1 ~ N3 的落实；D1、D2、D7 与关联、`seq`、事务的既有结论。
- **已接受的风险**：只有 §62 的关联生命周期限制；它不涵盖可识别的存储故障、稀疏读取改写修改时间，或浅边界带来的新增发现限制。
- **审查方的边界**：只在 Windows、Git 2.56.0.windows.2 上运行；没有重跑 Linux、macOS、真实的 2.32 / 2.43、部分克隆、并发 `gc`、权限与 I/O 的完整矩阵、LFS、长驻 process、崩溃清理或真实的网络加深；作者的 G / K / F / S 实验仍是作者证据。

**负责人的第三轮决定**（2026-10-19，即收到授权的日期）：

- **P1**：不可判定使用 `unknown`，不能当作 `clean`。
- **P2**：含稀疏目录条目的索引一律不可判定，安全门禁前置；不扩大到所有稀疏检出。
- **P3**：MVP 暂不支持浅克隆的提交扫描。实际需要扫描时，以只读方式明确检测；浅克隆或检测无法确认，都返回环境错误（退出码 6），不写提交、关联、`seq` 或游标，不伪装成成功的空扫描，不自动拉取或修改浅边界。触发扫描的命令应明确呈现这一限制。将来支持时另行设计 deepen / unshallow 与游标的生命周期。
- **`worktree list -z`**：缺少这一能力时，MVP 的扫描返回环境错误（退出码 6），不增加按行解析的回退；明确为能力要求，其他命令的支持范围不变。
- 这些是范围的限制，不是允许误报 `clean`、修改原对象或放行损坏存储的风险豁免。

**整合**：

- 两轮审查结束之后，已核验的 master `46a4b26855cc582b6387883260433c7badb1f2f9` 以 merge 合入本分支（提交 `7c7da34`），与修订提交分开。
- 唯一的冲突在本文件：两边都在 §61 之后追加。解决方式是完整保留本分支的 §62 ~ §64 与 master 的 §65，按编号排列，原文与原审查身份不改。
- 下一轮审查的基准为这个 master；此前各轮审查仍以它们当时的 HEAD 与 base 为准。

**修订**（规范见 [wp2.3-git-reader.md](wp2.3-git-reader.md)）：

| 问题 | 修订 | 位置 |
|---|---|---|
| B2-R2 | 存储检查改为包与索引**双向配对**：每个 `pack-<哈希>.pack` 都要有同名的 `.idx`，反之亦然；`objects/pack` 中只允许可识别的文件（成对的包与索引、它们的伴随文件、multi-pack-index 的文件），其他文件（例如 `tmp_pack_*`、孤立的 `.rev`）说明存储不完整；备用对象目录递归同样检查。不成立即无法确认，退出码 6，不发布结果、不推进游标。检查不写入任何文件，不重建索引或修复包。正常回收的路径保留 | 规范 §5.2；data-model §4.12；architecture §6.7；MVP-EV-16 |
| B3-R2 | 读取步骤重新排序：建立私有目录之后，**先由 Specmetry 解析索引副本（不运行 git）**，有稀疏目录条目或 `sdir` 扩展即为 `unknown`，这个仓库与它的子模块都不再运行任何 git 命令；不能解析即退出码 6。子模块在运行自己的 `status` 之前先过它自己的门禁。超级项目的 `diff-files` 带 `--ignore-submodules=all` | 规范 §4.3 第 2 ~ 4 步；data-model §1；MVP-EV-18 |
| 浅克隆的加深 | 按 P3：扫描之前由 Specmetry 直接检查公共 git 目录中的 `shallow` 项（存在即拒绝，其他错误为无法确认），不使用 `--is-shallow-repository`；读取另设 `GIT_SHALLOW_FILE`；浅克隆期间不建立、不推进游标，恢复为完整仓库之后按既有规则 | 规范 §5.2"浅克隆"；data-model §4.12；architecture §6.7；MVP-WT-06 |
| P1、P2、`-z` | 写入决定，撤去"待决"的表述 | 规范 §2、§4.3、§5.1、§8、§13；data-model；architecture；验收项 |

**实验**（作者在仓库之外、隔离夹具中运行，没有提交；文档 CI 不是这些实验的验证；Windows 的 Git 2.56.0 与 WSL Ubuntu 的 git 2.43.0，非 root 用户）：

- **K18 ~ K22**：
  - 审查 055 的反例复现：`.pack` 失去 `.idx` 时，`missing`、1、没有 stderr，审查 052 版的检查通过；双向配对不通过，恢复 `.idx` 后为 `commit`。
  - 备用目录中的同样情形、残留的 `tmp_pack_*`、孤立的 `.rev`：都为无法确认。
  - 正对照（带 `.bitmap`、`.rev`、`.keep` 的正常回收）：可靠地不存在。
  - 两个平台结果相同。
- **S15 ~ S22**：
  - 审查 052 版的顺序（先 `status`）复现了修改时间的改变。
  - 门禁先行之后，超级项目、子模块与失败路径中，原 `.git` 都没有变化，门禁之后没有运行任何命令。
  - 正对照照常读取。
  - 索引解析覆盖版本 2、3、4、拆分索引与 untracked cache，损坏与截断为错误。
- **X1 ~ X8**：
  - `ls-files --sparse --stage` 本身在稀疏目录同名的目录存在时也会展开索引，所以门禁不能用它；只有另设 `sparse.expectFilesOutsideOfPatterns=true` 才不展开（X8，只在 Windows 上）。
  - 超级项目的 `diff-files` 不带 `--ignore-submodules` 时会修改子模块原对象的修改时间。
  - 一次较早的运行中 `ls-files --sparse` 没有展开，原因没有查明，本文不依赖它。
- **H1 ~ H7**：
  - 浅克隆的检测不写入。
  - 生命周期的模拟：浅克隆期间拒绝；恢复后，没有游标时读取全部历史，有游标时增量读取，没有遗漏。
  - `GIT_SHALLOW_FILE` 的防护有效。
  - `shallow` 文件不可读或是目录时，`--is-shallow-repository` 报告 `false`（H6、H7，Linux）。
- **未验证**：macOS；2.32 与其他版本的实际运行；真实的部分克隆、并发 `gc` 与网络文件系统；真实的网络加深；稀疏索引与拆分索引同时使用；Go 中的索引解析器（本轮只有 Python 原型）。

**状态**：规范按审查 055 修订，待复审；B2-R2、B3-R2 与浅克隆生命周期的状态都是"已修订，待复审"，没有由实施方自行宣布关闭；不宣布冻结；**WP2.3 的实现尚未授权**；PR #24 保持未合并。

**保留的事项**：

- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13、M-14、M-15 与 M0 N1）保持未关闭。
- 审查 055 的原文与审查对象的身份不变；§62 ~ §65 的原文不改写。
- 没有获批的草案保持原状态。

## 67. WP2.3 规范的定向复审（codex-review-057）、冻结与条件授权

**审查**：[codex-review-057](reviews/codex-review-057.md)（原样归档，未修改）。

- 审查对象：PR #24，HEAD `61c69736613323bc7c8e7ec17a56d2b48e1e0ff7`，base `46a4b26855cc582b6387883260433c7badb1f2f9`；审查 055 的原对象 `5d707eb`，当时的 base 为 `a236e53`。审查方复用审查 055 已成立的结论（关联、`seq`、事务、严格遍历、过滤器判据），只检查新的差异与必要的调用顺序。
- 核对的 CI：run 39144293701，attempt 1，8/8；检出合并快照 `f5510c5ab7321943d1cd082cc63d3a2ffb2ffc55`，内容树 `07b29b46601580ce73a70e8cb25546fa93dc5fcf` 与 HEAD 相同。它只证明这个文档树对应的既有检查通过，不证明仓库外的 K / S / X / H 实验已在三个平台运行，也不证明 Git Reader 已经存在。
- **规范技术结论：GO**，没有剩余的实质性 BLOCKER；规范具备整体冻结条件。这不表示 Git Reader 已实现、实现测试已通过。
  - **B2-R2 关闭**：包与索引双向配对，不再在空的 idx 集合上错误通过。审查方用独立的窄范围规则见证复验了移走 idx、alternates 中的同样情形、恢复 idx、残留的 `tmp_pack` 与孤立的 `.rev`、正常回收。`.bitmap`、`.rev`、`.keep` 的组合正例没有被审查方全量重放。
  - **B3-R2 关闭**：直接解析的门禁前置，超级项目不再先进入子模块的索引。审查方复现了旧顺序下 3 个原树对象的修改时间改变；新顺序下后续 git 调用数为 0，原 `.git` 的内容与修改时间都不变；普通索引的正例照常读取。子模块的动态组合没有在审查方本机重放。
  - **浅克隆的生命周期**：准备报告层面的缺口关闭；实际的游标集成仍待实现验证。
- **NON-BLOCKER N1**：把"主索引不存在"的正常分支写进实现测试。真正不存在的主索引（新建的空仓库）按空索引处理；零字节或截断的索引、缺失的被引用共享索引、权限或 I/O 错误仍然 fail-closed。建议覆盖空仓库、已有 HEAD 但没有索引、`link` 的哈希为零、缺失的共享索引。
- **产品决定**：本轮没有新增必须由负责人裁决的产品语义；P1、P2、P3 与 `worktree list -z` 已获批。
- **已接受的风险**：只有 §62 的关联生命周期限制；没有新增。M-14、M-15 与其余维护事项未关闭。
- **审查方的边界**：只在 Windows、Git 2.56.0.windows.2、Python 3.14.1 上运行；没有重跑 WSL / Linux 2.43、macOS、2.32、真实的部分克隆、网络加深、并发 `gc`、权限与 I/O 的完整矩阵、长驻过滤器或 LFS。临时夹具第一次准备时遇到 Windows 只读对象文件与 alternates 文件的 CRLF 格式问题，改正之后才取得有效结果，没有当作规范的反例。审查方的格式见证没有实现完整的拆分索引 EWAH 重建，作者的 Python 原型也不能代替 Go 实现与 CI 回归。

**负责人的批准**（2026-10-19）：

- **冻结**审查 057 所审查的 WP2.3 规范（[wp2.3-git-reader.md](wp2.3-git-reader.md)），以及此前明确确认的产品范围（§62 的关联规则、生命周期与空扫描；D1、D2、D7；P1、P2、P3 与 `worktree list -z` 的能力要求）。
- **条件授权 WP2.3 的实现**：在 PR #24 合并、合并核验与对应的 master push CI 全部通过之后生效。
- 其他 M2 工作包仍未授权；M2 尚未整体验收。

**实现阶段的约束**（审查 057 §三、§六与 N1，纳入可追踪的记录，见规范 §11"审查 057 的实现约束"；它们是"完整解析、不能解析就失败"等已有要求的工程落实，不是新的产品决定）：

1. 索引解析：条目计数、整数、前缀回退、路径终止、对齐与扩展长度都不能越界或溢出；区分可以跳过的可选扩展与不支持的必需扩展（小写开头的必需扩展不能当作可忽略）。
2. `link` 与共享索引：引用、边界与必要内容有效；缺失、损坏或无法完整理解时失败；不能只扫描主索引而遗漏共享索引的条目；`link` 的哈希全零表示没有共享索引。
3. 主索引或任一共享索引中的稀疏目录条目或 `sdir`，都在运行可能展开索引的 git 命令之前停止；更复杂的布局暂不支持时安全拒绝，不能猜测为普通索引。
4. 直接读取、复制失败与中途变化都保守处理：输出安全的错误、清理私有内容，不修复原索引。
5. 主索引不存在（ENOENT）按空索引处理；零字节或截断的索引、缺失的共享索引、权限与 I/O 错误 fail-closed（N1）。
6. 必须兑现的测试：索引格式与失败分支；超级项目与子模块门禁顺序的调用记录；真实文件内容与修改时间的快照；pack / idx / alternates 的正反例；浅克隆的检测与恢复生命周期；严格读取失败之后，事务、`seq` 与游标都不变。
7. 命令的呈现仍属 WP2.7，E2E 属 WP2.9，不被本次文档 CI 提前验收。

**三类增量的区分**：

| 类别 | 内容 |
|---|---|
| 审查 057 的原对象 | PR #24 的 HEAD `61c6973`（base `46a4b26`） |
| 新 master 中经审查 056 通过的增量 | PR #25 合并（merge `ca2808dee8b2bd8f423fe600b433df9acca2e3ad`）：M-14 诊断工具、有效性回归、诊断说明与审查 051 / 054 / 056 的归档；不改变 WP2.3 规范 |
| 本次纯收尾增量 | 审查 057 原文的归档；合入新 master 的整合提交；状态同步（规范头部与 §8 各阻塞项的状态列、TD-17 / TD-23、实施计划、AGENTS.md、data-model、acceptance-criteria 的 MVP-WT-06 / EV-17 / EV-18 中的"待复审""待审查"，以及 m14-diagnostics 中"M-15 随 PR #24 登记、尚未合并"的说明）、本节与规范 §11 的实现约束清单。没有改变规范的语义 |

**状态**：规范**已冻结**；WP2.3 的实现获得本次条件授权，待合并与 master CI 核验之后开工。PR #24 由负责人授权以 merge commit 合并。

**保留的事项**：

- 所有未关闭的维护事项（M-1、M-2、M-4 ~ M-10、M-13、M-14、M-15 与 M0 N1）保持未关闭。
- 审查 057 的原文与审查对象的身份不变；§62 ~ §66 的原文不改写。

## 68. WP2.3 实现的首次独立代码审查（codex-review-058）与修复

**审查**：[codex-review-058](reviews/codex-review-058.md)（原样归档，未修改）。

- **审查对象**：PR #27，HEAD `0c9c51d37a39bb53a6d8da0430fa3c8668f464af`，base `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`。审查方完整阅读了 base 中冻结的规范、对应的 data-model / architecture / 验收项、审查 057 与 PR 描述，并把冻结规范与实际实现分开判断，没有把作者的新发现当作审查 057 已审的内容。
- **结论：NO-GO**，PR #27 暂不合并，不支持 WP2.3 本层验收。
- **B1：私有目录没有保护祖先路径与目录身份**。
  - 私有目录建在系统临时目录下，没有检查那里的路径。另一个账户可以在锁住之前把目录移走、换成替代物；按路径设置的 DACL、空目录检查、`O_EXCL` 与锁都不能证明目录身份或属主可信。清理还会删除只是前缀相同、时间较早的无关目录。Clean 的私有配置、索引副本，以及扫描的 grafts / shallow 防护路径都受影响。
  - 审查方的独立验证：
    - 在 Windows 的隔离根上加了"允许其他用户替换"的 ACL，私有目录照常建立；
    - 用同一账户的同步钩子在建立后替换目录，`Clean` 返回 `clean`；
    - 一个只是前缀相同、时间较早的目录中的见证文件被删除。
  - 审查方说明，替换是同账户的故障调度，没有冒称执行了真实的双用户攻击。
  - 审查方明确：这是本库自身的安全契约，不能延期到 WP2.5，也没有获准的豁免。
- **B2：截断的 `link` 位图使公开接口 panic**。外层校验和为零（skipHash）、`link` 内只有 8～11 字节位图头的 68～71 字节索引，使 `IndexIsSparse` 与 `Clean` panic（`slice bounds out of range [12:8]` 至 `[12:11]`）。
- **B3：`.specmetry/` 仍能影响 Clean**。`.specmetry/` 中一个受过滤文件的变化使 `Clean` 返回 `undecidable`，而受控环境中的定义命令输出为空；gitlink 的遍历同样不应计入这类路径。
- **NON-BLOCKER N1**：进程内并发跳过测试的"对方提交后"快照取晚了（外层 Scan 返回之后才取），检测不到外层在返回前的错误写入。审查方认为这是测试断言的改进，不是事务算法缺陷。
- **审查方认为成立的部分**：读取层与错误分类、Tips / Classify / 严格扫描、子模块四种变化的补足、Clean 的其余判据与门禁顺序、扫描事务、关联与迁移 0007、隐私（详见原文第三节）。这些不抵消 B1～B3。
- **规范同步的要求**（原文第五节）：子模块四种变化、`--no-show-signature`、私有空间、祖先读取的版本差异、worktree 元数据路径、B2 与 B3 的测试；不能称作审查 057 已验证。
- **产品决定与已接受的风险**：没有新增需要负责人裁决的事项。已接受的风险只有 §62 的关联生命周期限制。M-14、M-15 未关闭。
- **审查方的验证边界**：
  - 本机为 Windows/amd64、Git 2.56.0.windows.2、Go 1.27.1。
  - 没有运行：CLI 整包、Linux / macOS、Race Detector、真实旧版本的 Git、真实的双用户切换、网络加深、部分克隆、并发 `gc`、LFS、长驻过滤器、大仓库。
  - 核对了 CI run 39165490748（attempt 1，8/8，合并快照 `2dd41f6`，树 `c98223c`），以及前两次 lint 失败（39163839528 为 7 项，39164780269 为 2 项）。审查方说明，绿色 CI 不覆盖其新增的反例。

**修复**（实施方；每项先在修复前的代码上复现失败，再修复）：

- **B1**（`3901c55`）：
  - 私有目录改建在数据目录的 `tmp/` 中（`gitx.NewScratch`）：先调用数据目录的写入前验证 `datadir.Ensure`，原样复用 TD-22 与 architecture §11.1 的路径、属主、权限与链接规则；准备过程不打开数据库。
  - 每个私有目录：随机名称；在目录之前以 `O_EXCL` 建立锁文件，并在使用期间一直持有；记录目录身份，在写入、git 使用与删除之前复核。复核失败时读取失败，不删除该路径上的任何内容，并删除其锁，使以后的清理也不动它。
  - 遗留清理只针对名称符合格式、能取得锁的锁文件及其同名目录；不再使用、也不清理系统临时目录；去掉了 `d0b2dad` 中显式设置子目录 DACL 的做法。
  - 接口：`Clean`、`Tips`、`Classify`、`IsAncestor`、`Log` 增加 `*Scratch` 参数；`gitscan.Options.Scratch` 为必填。
  - 修复前的反例（本地临时钩子，未提交）：替换不安全的根被接受、替换后返回 `clean`、无关目录被删除，三项都复现。
  - 修复后的回归：不安全的祖先或内容被拒绝、系统临时目录不被使用、私有目录被换成目录或链接（Clean 与扫描读取）、`tmp/` 被替换、遗留清理的正反例、成功 / 失败 / 取消 / 删除失败的生命周期、对已填充的私有目录做完整的 `datadir.Check`。
  - 双用户测试复用 `internal/store/sqlite` 的跨用户基座（以 root 运行）：可写父目录下攻击者持续替换数据目录、安全父目录、属于他人的祖先目录、sticky 父目录。
- **B2**（`1fe57c5`）：EWAH 位图先要求完整的 12 字节框架，字数必须容得下剩余长度，`readLink` 不越过载荷。回归在修复前都 panic，修复后为环境错误：
  - 原反例经 `IndexIsSparse` 与 `Clean`；`Clean` 在任何读取索引的 git 命令之前失败，`.git` 的内容与修改时间不变；
  - 两段位图的逐长度截断（外层有效，校验和正确或为零）；
  - 异常的字数与 running-length 位置，包括真实的共享索引；
  - 合法拆分索引的对照；
  - 有界的随机损坏，以及模糊测试的种子。
  - 另外以 `go test -fuzz` 做了 60 秒的有界探索（约 22.6 万次执行，没有失败）。
  - 生产代码没有用 recover 吞掉异常。
- **B3**（`9746459`）：`ls-files`、`diff-files` 与 `status`、`diff-index` 使用相同的 git pathspec（超级项目中排除 `.specmetry`，子模块中不排除）；稀疏门禁仍作用于整个索引。回归以受控环境中真实运行的定义命令为对照：
  - 修复前失败的 4 项：`.specmetry/` 中受过滤文件的三种变化（内容与修改时间、只改修改时间、racy），以及 `.specmetry/` 中子模块目录被删除；
  - 修复前后都通过的对照：`.specmetry/` 中已暂存与已删除的文件、`.specmetry-not/` 中的同类变化仍然计入、`.specmetry-sub` 处的子模块、子模块内部的 `.specmetry/` 仍然计入、`.specmetry/` 的稀疏目录条目仍被门禁拦下。
- **N1**（随 `3901c55`）：快照在钩子中、内层扫描提交之后、外层继续之前取得。
- **规范同步**：
  - [WP2.3 规范 §14](wp2.3-git-reader.md)（新增"实现中的细化，待复审"），并在 §4.3、§5.2、§6 加了指向；
  - architecture §6.7 与 §8（数据目录布局中的 `tmp/`）、§11.1；
  - acceptance-criteria 的 MVP-EV-16、MVP-EV-18；
  - development.md；
  - 实施计划中的 WP2.3 状态。

**状态**：B1～B3、N1 **已修复，待独立复审**（审查 059）。PR #27 不合并；不宣布 WP2.3 验收；不启动后续工作包。

**保留的事项**：

- M-14、M-15 未解决；M-9 照旧（git 2.32.0 下 gitx 的 3 个既有正向测试）。
- Windows 上没有双账户的动态验证。
- 审查 058 的原文与审查对象的身份不变；§62 ~ §67 的原文不改写。

## 69. WP2.3 实现修复的定向复审（codex-review-059）与 B1 残余的修复

**审查**：[codex-review-059](reviews/codex-review-059.md)（原样归档于 `ddffd8a`，未修改；SHA-256 `c2693b67e9ea121c91a0d0f43e82d84f3c0575329fae2569a74cae43752ce6fa`）。

- **审查对象**：PR #27，HEAD `4519e7a37a744e2b2d0dc2da719314964267c0d0`，base `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`，上轮对象 `0c9c51d`。审查方核对了开始与结束时的远程身份，复用审查 058 对未改动部分的结论，没有重做整个 WP2.3 的审查。
- **结论：NO-GO**。PR #27 暂不合并；WP2.3 本层验收的证据尚不充分。
- **关闭**：B2（截断的 `link` 位图）、B3（`.specmetry/` 的排除）、N1（并发跳过的快照）。审查方的依据见原文第五至第七节，其中包括在旧提交上独立复现、在目标提交的两个平台上独立对照，以及一个额外的游标字段写入变异被测试检出。
- **B1 未关闭（B1-R，私有目录的替换后清理与根的身份）**。三个反例都在本仓库之外的隔离副本上，用既有的测试钩子构造：
  1. **删除重试不复核身份**：第一次删除时把私有目录移走、在原路径放入含 `replacement-content` 的目录，并返回临时错误；第二次删除执行真实的递归删除。`Clean` 与 `Log` 都得到 `nil`（Clean 为 `clean`），替代内容被删除。Windows 与 WSL Linux 上都复现。
  2. **复核失败时按路径名删除锁**：在私有目录建立之后，把整个 `tmp/` 移走，在原路径重建 `tmp/`、同名的私有目录与见证文件，以及内容为 `keep-replacement-lock` 的同名 `.lock`。两条入口都正确返回退出码 6，替代目录中的见证文件保留，但替代根中的同名 `.lock` 被删除。只在 WSL Linux 上复现；Windows 上，移动 `tmp/` 这一步被操作系统拒绝（`Access is denied`）。审查方说明，这一拒绝不能算作该反例已在 Windows 复现，也不能算作实现中的身份保护已经验证。
  3. **只复核最终子目录**：把 `tmp/` 移走，并让原路径成为指向移走位置的符号链接。`Clean` 返回 `clean`，`Log` 返回 `nil`；写入、git 使用与删除都经过这个链接（Linux）。
- **审查方的要求**：
  - 把数据目录 / Scratch 根、私有目录与锁句柄的身份关系保留到使用与清理阶段；
  - 每次删除重试之前都复核；
  - 锁文件的删除要确认父名字空间与锁文件仍是本次持有的对象，根或锁已被替换时不删除替代物；
  - 同一规则用于 Clean、扫描的防护目录、失败清理与遗留清理；
  - 补上三组回归与正对照；
  - 收紧 §14.3 中"复核失败后删除它的锁"的表述。
  - 审查方明确：不需要新的产品取舍或风险豁免，不应延期到后续工作包。
- **审查方认为成立的部分**（原文第四节）：`NewScratch` 先调用 `datadir.Ensure`，没有放宽既有规则；不安全的祖先在创建之前被拒绝；不再使用、也不清理系统临时目录；正常的遗留清理以受保护的专用名字空间、精确命名与锁为依据；Linux CI 的四项跨用户测试确实使用不同 UID 的进程。
- **审查方的验证边界**：
  - 本机 Windows/amd64、Git 2.56.0.windows.2、Go 1.27.1：gitx、gitscan、datadir 整包；WSL Linux（UID 1000、Git 2.43.0，交叉编译的测试二进制）：B1-R 调度与相关定向测试。
  - 没有运行：CLI 整包与全仓测试、独立的 Linux Race Detector、macOS 本机、真实的 Git 2.32、Windows 双账户动态测试、大规模模糊或压力测试。
  - 核对了 CI run 39203362680（attempt 1，8/8，合并快照 `dab32b4`，树 `fff2648`，与 HEAD 的树一致），并保留 run 39202572684（`8dfb31e`，7/8，lint 的两项 unparam 失败）的历史。
- **NON-BLOCKER**：没有新增。**PRODUCT DECISION**：没有新增。**ACCEPTED RISK**：仍只有 §62 的关联生命周期限制；Windows 没有双账户动态验证属于验证边界，不是已接受的风险。

**修复**（实施方，B1 残余，`ddb908a`；先在修复前的代码上复现，再修复）：

- **修复前的复现**（在 `4519e7a` 的代码上，用仓库外保存的一次性测试，只操作测试自建的隔离夹具，经 `Clean` 与 `Log`；修复后该测试文件已移出仓库，因为它用到的钩子随修复改变）：
  - 反例 1：Windows 与 WSL Linux 上都失败。两次删除尝试后读取返回 `nil`，替代内容被删除。
  - 反例 2、3：WSL Linux 上失败：替代根中的同名锁被删除；`tmp/` 变成链接之后读取仍然成功。在 Windows 上按设计跳过，因为移动 `tmp/` 被拒绝。
- **机制**（WP2.3 规范 §14.3 重写）：
  - Scratch 记录从卷根到 `tmp/`（解析为不含链接的真实路径）每一级目录的身份，并在使用期间持有 `tmp/` 的句柄；新增 `Close`。
  - 身份是 Unix 的设备号与 inode，或 Windows 的卷序列号与文件 ID（`FILE_ID_INFO`），从句柄或不跟随链接的读取中一次取得。Windows 的 `os.SameFile` 会按路径延迟读取文件 ID，因此不用。
  - 私有目录经 `tmp/` 的句柄建立并打开。只有在原路径上、不经链接、身份与句柄相同、并且为空时，才当作刚建立的目录。
  - 所有写入、稀疏门禁对索引副本的读取，以及删除，都经私有目录的句柄（`os.Root`）进行。
  - 锁改为私有目录中的所有者文件 `specmetry-owner`：使用期间锁定，并记录目录与它自身的身份；`tmp/` 中不再有同级锁文件。
  - 路径上每一级目录的身份（指向原处的链接也算变化），在以下时机复核：写入之前、git 使用之前、**每一次**删除尝试之前。
  - 删除所有者文件与目录条目之前，都核对名称处仍是原来的对象：
    - Windows 在核对身份的同一个句柄上设置删除，只会删除该对象；
    - 其他平台相对父目录的句柄执行 rmdir，只能删除空目录。
  - 无法确认身份或归属时：停止、只释放自己持有的句柄与锁、保留路径上的一切与本次读取自己的目录、返回退出码 6。
  - 遗留清理与正常删除走同一流程。它另外要求：所有者文件的锁无人持有，且记录写明的身份正是该目录与该文件；复制品、替代物与没有记录的目录都保留。
- **检查与操作之间的保护依据**（规范 §14.3"威胁模型"与"仍然存在的窗口"）：
  - 其他用户不能进入或替换 `tmp/` 的名字空间（`datadir.Ensure`；Linux 跨用户测试，本轮补了"攻击者不能在 `tmp/` 中建立条目"）。
  - 对同一用户的进程：
    - 句柄使写入与删除不随路径转向；
    - Windows 上，`tmp/` 及其上各级目录在 Scratch 打开期间不能被重命名；私有目录在其所有者文件删除之前也不能（本机 NTFS 观察）；
    - 复核发现持续存在的变化。
  - 如实保留的窗口：
    - 复核之间往返恢复的变化；
    - 非 Windows 平台上，核对与 rmdir 之间放入的空目录，以及私有目录内所有者文件的同名替换；
    - 所有平台上，建立与打开之间放入的空目录。
  - 不声称"再检查一次"消除了窗口。
- **修复后的回归**（都经 `Clean` 与扫描读取 `Log` 的公开调用链）：
  - `TestRemovalAttemptsCheckTheDirectory`（反例 1，失败发生在内容删除之前与之后）；
  - `TestScratchRootReplacedWhileInUse`（反例 2，替代物中放入复制的所有者文件、见证文件与旧布局的 `.lock`）；
  - `TestScratchRootRedirected`（反例 3：`tmp/` 指向原处、`tmp/` 指向含同名目录的别处、数据目录指向原处）；
  - `TestRemovalChecksWhatItRemoves`（核对之后、删除之前的替换：同名的配置与所有者文件只经句柄删除自己的，空的替代目录不被 rmdir）；
  - `TestPrivateDirectoryReplacedAsItIsMade`（建立与打开之间换成非空目录或链接）；
  - 每个失败用例都断言替代目录、见证文件与替代锁不变，以及读取自己的目录被保留，并在之后再建立 Scratch，确认遗留清理也不删除它们。
  - 正对照：
    - `TestRemovalRetriesTheSameDirectory`（同一目录的临时失败照常重试成功）；
    - 生命周期测试（未被替换时正常释放并清空 `tmp/`；内容删不掉时目录连同所有者文件留给下一个 Scratch 清理；目录条目删不掉时保留空目录；Scratch 已关闭时为内部错误）；
    - 遗留清理测试（崩溃遗留被删除；使用中、复制品、无所有者文件、所有者文件为目录、记录不符、链接、其他名称、旧布局的 `.lock`、系统临时目录都保留）。
  - 审查 058 的回归保留，并按新机制调整：在 Windows 上，持有期间的移动被拒绝，测试记录这一拒绝并验证读取照常完成。
- **平台**：
  - Windows 单账户：所有者文件删除之后的替换（重试之前、核对与删除之间），以及建立与打开之间的替换，都实际发生并被拒绝；其余的移动被操作系统拒绝，测试中有记录。
  - Linux 普通用户：所有移动都实际发生。
  - Linux 跨用户：既有四项，并在 SafeParent 中加了"不能在 `tmp/` 中建立条目"。
  - 静态依据：Windows 的重命名规则只在本机 NTFS 上观察。
- **辅助的变异检查**（只作支持，不代替回归；在 Windows 与 WSL Linux 上，按变异可观察的平台运行）。以下 12 项都被检出：
  - 只在第一次尝试前复核；
  - 目录条目不核对身份（Unix、Windows 各一项）；
  - 内容按路径删除；
  - 所有者文件按路径删除；
  - 遗留清理不核对记录；
  - 只复核 `tmp/` 而不复核上级目录；
  - 私有目录的复核不包括根；
  - 路径复核跟随链接；
  - 建立时不检查是否为空；
  - 不调用 `Ensure`；
  - 完全不复核。
  - 其中一项第一次运行时是编译失败，修正变异之后重跑，才确认为检出。
- **未改变**：B2、B3、N1 的修复与回归（公共的 `IndexIsSparse` 改为经同一读取函数实现，其测试照常运行）；CLI、Schema、迁移、推导规则、生产 SQLite 的重试与错误映射；其他工作包。
- **规范同步**：
  - [WP2.3 规范 §14.3](wp2.3-git-reader.md) 重写，§14 的说明与 §14.8 的测试表已更新；
  - architecture §8、§11.1；
  - acceptance-criteria 的 MVP-EV-18；
  - development.md；
  - 实施计划。

**状态**：B1 **已修复，待独立复审**（审查 060）；B2、B3、N1 已由审查 059 关闭。PR #27 不合并；不宣布 WP2.3 验收；不启动后续工作包。

**保留的事项**：

- M-14、M-15 未解决；M-9 照旧。
- Windows 上没有双账户的动态验证；Windows 的重命名规则只在 NTFS 上观察。
- 规范 §14.3 所列的同用户窗口。
- 审查 059 的原文与审查对象的身份不变；§62 ~ §68 的原文不改写。

## 70. B1 残余修复的定向复审（codex-review-060）与待决的设计报告

**审查**：[codex-review-060](reviews/codex-review-060.md)，原样归档于 `659b914`，未修改。SHA-256 为 `25c6f4e2ea6e0ebbf2e24fe261fd0f929ae3c138c30534ea91c9641854b4fab2`，源文件、归档文件与提交内容三者一致。

**审查对象**：

- PR #27，HEAD `ee144af61e00cb4d24bbae27feab10d8be60ecfc`，base `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`，上轮对象 `4519e7a`。
- 审查方核对了开始与结束时的远程身份，检查了 `ddffd8a`、`ddb908a`、`ee144af`。
- 复用了此前对未改动的读取、迁移与事务的结论，没有重做整包审查。

**结论：NO-GO**。PR #27 暂不合并；WP2.3 本层验收的依据尚未闭合。

**原三个持续替换反例：已修复**。审查方核对的结果是：

- 每次删除重试都重新复核；
- 整体替换 `tmp/` 时，复制来的所有者记录与新目录、新文件的身份不符，以后的清理不采用它；
- 根链复核能发现持续存在的链接变化。

Windows 上持有期间的移动由操作系统阻止。审查方把"操作系统阻止"与"替换已发生、代码随后拒绝"分开记录。

**B1-R：仍未关闭**。审查方在四个精确窗口上独立复现，以下按原文第三节摘要：

- **A（mkdir 与打开之间）**：放入空的替代目录。Windows 与 WSL Linux 上，`Clean` 与 `Log` 都把它当作刚建立的目录使用并删除，返回 `nil`；原目录留在移走处。已提交的回归只覆盖非空目录与链接。
- **B（Unix 最终身份核对与 rmdir 之间）**：同名的空替代目录被删除，`Clean` 与 `Log` 都返回 `nil`（WSL Linux）。
- **C（Unix 所有者文件的最终核对与 unlink 之间）**：替代文件被删除，两个入口都返回 `nil`（WSL Linux）。
- **D（Git 使用路径期间替换、之后恢复）**：WSL Linux，真实 Git 2.43.0。
  - `Log` 返回 `nil`，但回调**只收到 2 个提交中的 1 个**：**漏读提交却返回成功**。
  - `Clean` 在该调度下返回 `dirty`，使用了替代目录。审查方没有声称复现了错误的 `clean`。
  - 审查方没有把这些结果延伸为已复现数据库游标的错误发布。

审查方的判断：

- 这些不是已证明的跨用户权限突破；
- 但违反了沿用的"不使用、不删除不能证明属于本次读取的对象"的保证；
- §14.3 如实披露了这些窗口，但披露不能代替尚未取得的保证调整批准。

**保持关闭**：B2、B3，以及审查 058 / 059 的 N1，不重新开启。

**新增 NON-BLOCKER N1（审查 060）**：

- `TestRemovalChecksWhatItRemoves` 的 `testHookRemove("entry")` 位于外层复核之后、`removeEntry` 内部最终核对之前。
- §69 中"核对之后、删除之前""空的替代目录不被 rmdir"的说法，应改写为这一层次；该用例不能作为最终窗口已经安全的证据。
- 这只是证据表述问题，本轮只登记，随下一轮修正。

**PRODUCT DECISION**：

- 若选择保留这些窗口，需要负责人对审查 060 第四节列出的局部保证调整作出决定。
- **没有批准任何保证调整**；保持原保证的技术路线不需要新的产品语义。

**ACCEPTED RISK**：

- 没有新增；仍只有 §62 的关联生命周期限制。
- 同用户窗口、Windows 验证的不足、M-14 / M-15 都不归入已接受的风险。

**审查方的验证边界**：

- **Windows/amd64**（Git 2.56.0.windows.2、Go 1.27.1）：12 项私有目录测试与相关回归；
- **WSL Linux**（UID 1000、Git 2.43.0、交叉编译的测试二进制）：已提交测试，以及四类同步调度；
- **没有运行**：macOS、Linux root 跨用户测试、Race Detector、全仓或 CLI 整包、旧版本 Git、安全压力与模糊测试；
- **CI**：核对了 run 39218204960（attempt 1，8/8，合并快照 `c7dcd75`，树 `c758d79`，与 HEAD 的树一致），并保留 39163839528、39164780269、39202572684 的失败历史。

**设计报告（待负责人决定）**：

- **文件**：`docs/wp2.3-b1-design.md`，由 Codex 撰写，日期 2026-10-19，固定对象为 `ee144af` / `e0a21b4`。共 314 行，SHA-256 为 `266d5fbaa130da323a0558806e47eb62531343d760cd6ff336c703601b92c2a4`。
- **性质**：这是实现前的设计分析与最小隔离验证，**不是审查，不是已批准或已冻结的设计，也不是修复**。生产代码、冻结规范与 §14.3 都没有改变。
- **要点**：
  - 共同方向：扫描读取不再依赖临时防护路径，改为给两个覆盖变量传显式空值。目前只是候选：在 Git 2.56.0.windows.2 与 2.43.0 上做了隔离原型，macOS、其他 Git 构建与生产的能力探测都还没有验证。
  - 方案 A：保持原保证，由独立权限主体管理只读临时区。端到端尚未验证；需要负责人决定新增本地服务、安装方式与临时区位置。
  - 方案 B：无特权的协作域。这是保证调整的备选，**没有获批**，报告也不推荐作为默认选择。
  - 待决事项为 B1-D1 至 B1-D4。
- **仓库外材料**：实验材料在 Codex 的本机目录中，固定 HEAD 导出包的 SHA-256 为 `181186633d01bf095301b54f2c628453cf094e94f4df84fdba11b02ce4b3b7c4`。本仓库没有复制这些材料，也没有重跑这些实验。
- **为什么没有提交**：
  - 报告第 96 行有 3 个指向本机绝对路径的 Markdown 链接，CI 的"Markdown links are repository-relative"步骤会拒绝它们；
  - 报告自身写明"未提交"。
  - 因此它原样留在工作区，未修改、未提交。是否原样提交、把这些链接改为纯文本后提交、或保留在仓库之外，由负责人决定。

**状态**：

- B1 **未关闭**（审查 060），等待负责人对 B1-D1 至 B1-D4 及保证问题作出决定。
- 本轮没有提交生产修复，也没有修改保证文字。
- PR #27 不合并；不宣布 WP2.3 验收；不启动后续工作包；Auto-fix / Auto-merge 保持关闭。

**保留的事项**：

- M-14、M-15 未解决；M-9 照旧。
- Windows 上没有双账户的动态验证。
- 审查 058 至 060 的原文、审查对象的身份与 §62 至 §69 的历史结论不改写。§69 中"B1 已修复，待独立复审"是当时的状态，由本节接续。

## 71．B1 方案 B 的批准与实施范围

**负责人明确批准（2026-10-19，本轮实施指令）**：实施方为 Codex；允许实现、测试、提交并推送现有 PR #27，不允许合并、宣布审查 GO、B1 关闭或 WP2.3 验收。保留 `659b914`、`d6f9153` 及审查 058～060 原文。原生产对象 `ee144af61e00cb4d24bbae27feab10d8be60ecfc`，原 base `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`。

- **B1-D1**：扫描采用显式空值 `GIT_GRAFT_FILE=`、`GIT_SHALLOW_FILE=`，按实际 Git 二进制身份探测行为；不支持时环境错误／exit 6，不回退 missing 路径。保留 P3、PD-26、严格失败、禁外部程序及部分读取丢弃。Log 漏读却成功未获豁免。
- **B1-D2**：采用设计报告 §8 的方案 B，保留普通用户单二进制，不引入 broker／系统服务。支持遵守临时区协议的正常 Specmetry 并发；排除不合作同用户进程对 Git 私有临时命名空间的干预，**包括普通误操作**。明确接受：Unix mkdir/open 可接纳空替代目录；最终比较与 unlink/rmdir 之间可删除替代 owner 文件或空目录；Clean 可受替代输入影响，不能排除 false-clean。此局部保证调整不是旧绝对保证已实现；不放宽其他账户保护、合作并发、数据库、来源权限、隐私或交付证据保证。
- **B1-D3**：发现身份不符、未知或额外条目即停止相关操作、保留对象并环境错误。正常结束清理自身内容，清理失败仍 exit 6。可保留协议必需的最小锁和标识，不保存项目内容。不凭前缀、年龄或 PID 接管遗留。
- **B1-D4**：保留 Scratch 参数和缺失／关闭错误契约，仅调整内部机制。探测也遵守安全临时区协议；新旧命名空间隔离，不凭“未发现旧进程”假定安全。

**工程协议**：新命名空间为 `tmp/git-reader-v2/`，旧代码只认顶层 `tmp/read-*`，新代码不接管这些旧目录；新版本不把 v1 owner 当 v2。持久合作锁串行化每个私有目录从建立、全部 Git 调用到清理的租约（包括能力探测）；扫描实际读取不使用私有路径。锁竞争安全失败，不自动重试业务写入。创建清单记录相对名、类型和身份，未知对象保留。

设计报告原稿 SHA-256 与格式整理说明见 [设计报告 §12](wp2.3-b1-design.md#12-归档格式说明与后续授权)。§11.3 的旧建议不作为当前授权。更早 hook 的检查层与最后 unlink 窗口不同；不改写原审查以声称域外行为已被代码阻止。

**状态**：批准范围已记录，实施与证据待完成、待独立会话复核。B2、B3、N1 保持关闭；M-14／M-15 未解决，不改生产 SQLite 重试或分类。Auto-fix／Auto-merge 保持关闭，不启动后续工作包。

**§71 实施进展**：批准记录／设计报告格式归档为 `9462a56168c9a27a0413992da88cdc7a00c0ec95`；实现与回归为 `979f705d38d4f42412de13223096a02ee3ddd428`，已推送现有 PR #27。保留并推送了 `659b914`、`d6f9153`。平台、CI、文件清单、失败与重跑、获批残余边界见 [实施交付记录](wp2.3-b1-implementation.md)。这不是审查关闭或工作包验收；PR 保持 Open，交独立会话定向复核，后续工作包未授权。

## 72．Review 061 与两项定向修复授权

2026-10-19，独立 [Review 061](reviews/codex-review-061.md) 审查 PR #27 的 `8e089ab7ed5a12f250c30723d9fc1d1df75af6ea`，base `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`，结论 **NO-GO，B1 未关闭**。完整编号、证据及意见以原文为准：

- **B1-R1**：owner 在完整预检前已缺失，检查仍通过，随后清理删除其余内容；必须在任何删除之前校验 owner 的存在、类型、原身份及记录，区分本次可信移除 owner 后的重试阶段。
- **B1-R2**：调用方异常退出后两把锁被释放，但真实 Git 子进程仍可能存活；下一 Scratch 会回收活动目录。必须保证消费者未被确认结束时保留目录并安全失败，不能用父进程死亡、PID、年龄或锁释放代替结束证明。
- 空覆盖修复、Windows 原子创建及同句柄删除子机制成立；正常取消的等待／清理正对照成立；B2、B3、已关闭 N1 保持关闭。
- **N-doc**：PR 当前描述和开发指南主表需要接续现行对象；历史审查不改写。

原文已原样归档为 `d4a42e6`。源文件、仓库归档及该提交 blob 的 SHA-256 均为 `622940c44df6a80f6918676d5e6254c342dcb60e4291e1b2238186bb60d0388c`，逐字节一致；没有改写审查对象或结论。

**负责人本轮明确授权**：按完整 Review 061 定向修改、先建立失败回归、测试、提交并推送现有 PR #27；不合并，不由实施方宣布 GO／B1 关闭／WP2.3 验收，不启动后续包，也不生成下一轮复审提示词。方案 B 的批准范围不变，不能把这两项归入不合作同用户干预。M-14／M-15 等维护事项未解决，生产 SQLite 重试／错误映射、迁移、推导、CLI 与 Schema 不在修改范围。

**实施不变量与选择**：Git 只要仍可能消费私有目录，正常合作进程就不能回收它。采用跨平台持久消费屏障：在每次消费目录的 Git 启动前，将未确认结束标记写入 owner 并同步；只有命令返回、确认启动失败或已等待子进程退出，且完整输入验证通过后，才清除并同步。标记未清除或记录无法确认的崩溃遗留保留、环境错误；即使 Git 后来退出，也不凭 PID／固定等待自行清除。该保守失败分支按本轮授权落实；不以未经验证的 fd／handle 继承充当证明。正常完成和取消仍可清理；无在途消费者的完整可信遗留仍可回收。协议升级为独立 `tmp/git-reader-v3/`，不接管缺少结束证明的旧 v2 或 v1 遗留。清单预检明确验证路径上的 owner 及其完整记录；只有本次可信删除 owner 后才允许目录删除重试。代码、修复前后测试、三平台与 CI 身份接续 [实施记录 §8](wp2.3-b1-implementation.md#8review-061-后续两项阻塞的定向修复)，实施方不作关闭或验收结论。

## 73．Review 062 与完成记录 Sync 失败定向修复

2026-10-20 接收完整 [Review 062](reviews/codex-review-062.md)，受审对象仍为 `d121bfb42fdbdb922fc136265a3792a86875709b`，base `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`。结论 **NO-GO**；Review 061 的 **B1-R1（owner 预检）可以关闭**，原 **B1-R2 调用方崩溃但真实 Git 仍存活的反例已修复**。唯一残余 BLOCKER 为 **B1-R2-S［P2］**：Git 已结束、完整 false 记录写入后 Sync 失败，错误返回成立，但本次 cleanup 仍删除全部临时内容。B1 整体仍未关闭，完整编号、证据及边界以原文为准；不改写历史报告。

原样归档提交 `ac8f54c31df2ec5e6c877d5fa63e78648b0a308f`。源文件、仓库归档、暂存及该提交 blob 的 SHA-256 均为 `bba92f73c87cb96596e4cd75c910af1e4be00d74f9f313f1b5995d7a105d1fbf`，逐字节相同。没有把原 CI 通过改写为已覆盖 Sync 故障。

负责人授权按完整报告定向修改、先建立失败回归、测试、提交并推送现有 PR #27；不合并、不宣布 GO／B1 关闭／WP2.3 验收，不启动后续包，不生成下一轮复审提示词。最小修复为独立的本次完成未确认状态：Git 结束后，只有完成记录写入及 Sync 成功才允许本次清理／复用；失败时原错误、清理资格和释放路径一致，保留全部现存内容，仅释放自身资源，不能普通重试覆盖。现有 v3 后续 Scratch 根据实际完整记录及归属检查回收或保留，区别于前次调用观察到的 Sync 结果，不升级协议。规范 §14.3 和验收项同步，实际修复前后证据接续 [实施记录 §9](wp2.3-b1-implementation.md#9review-062-后续完成记录-sync-失败的定向修复)。

owner 预检、原消费者崩溃反例、空覆盖及 Windows 子机制按报告已有结论保留，仅运行直接回归。方案 B 边界不变，B2、B3、已关闭 N1 保持关闭；M-14／M-15 等维护事项未解决，不修改生产 SQLite 重试／错误映射、迁移、推导、CLI 或 Schema。Auto-fix／Auto-merge 保持关闭；实施方不代替独立复审或负责人验收。

## 74．Review 063：技术 GO、B1 在方案 B 下关闭与条件合并授权

**审查**：2026-10-20 接收完整的 [Review 063](reviews/codex-review-063.md)，原样归档为 `adc80a79062f2adda397525d5fa2e0c3beb05e17`。源文件、暂存与该提交 blob 的 SHA-256 均为 `a99ee7b113cda89a63351e41222a72308b190d632bb71432d334b89ff16cf00a`，逐字节一致。

**审查对象**（不随之后的文档提交而改变）：

- PR #27，HEAD `c4dbfb10ccf59899eb1223889fcb3913a7248bfb`，base `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`；
- Review 062 的对象（增量起点）为 `d121bfb42fdbdb922fc136265a3792a86875709b`；
- 代码修复为 `a2f7bd643fefa39155b883951a786f6cf63353e5`。

**结论：技术 GO**。完整编号、证据与边界以原文为准；历史报告不改写。

- **B1-R2-S 关闭**。依据（原文第二至第四节）：
  - `completionUnconfirmed` 把三种状态分开：Git 仍可能消费；Git 已结束但完成记录写入或 Sync 失败；完成记录已写入并 Sync。
  - 后两种中，只有最后一种允许本次继续读取或按清单清理；完成失败后，begin、finish、remove 都拒绝，保留全部现存内容，只释放本次自己的资源，不能用普通重试覆盖。
  - 原精确反例在 Windows/NTFS 与 WSL Linux 上：旧版 `d121bfb` FAIL，目标版本 PASS。
  - 审查方另做了两项独立验证：同一次调用不能覆盖失败；后续的独立进程按实际记录判断（完整的 false 可以回收，true 或截断的记录保留）。完整的 false 只能证明那次消费者已结束，**不能证明前次 Sync 成功**。
- **B1 整体关闭，但只适用于已批准的方案 B（§71 的 B1-D1～D4）**。既有裁定照旧有效：
  - owner 预检（B1-R1，审查 062 关闭）；
  - 消费者生存期（B1-R2 的原反例，审查 062 确认已修复）；
  - 扫描的显式空覆盖（B1-D1）；
  - Windows 原子创建与同句柄删除的子机制。
  - 这不是旧的绝对保证已经实现，也不扩大 §71 的批准边界。
- **继续保留的边界**（§71 B1-D2）：
  - 对不合作的同用户进程（**包括普通误操作**）：Unix 的 mkdir/open 可能接纳空的替代目录；最终比较与 unlink/rmdir 之间可能删除替代的 owner 文件或空目录。
  - Clean 可能受替代输入影响，**不能排除 false-clean**。
- **Log 的完整性没有豁免**（B1-D1）：不支持显式空覆盖时为环境错误 / exit 6，不回退到 missing 路径。
- **其他**：B2、B3 与已关闭的 N1 不重开。没有新增 BLOCKER 或代码 NON-BLOCKER；没有新增已接受的风险，仍只有 §62。

**审查方的验证边界**：

- 本机运行：Windows/amd64（Go 1.27.1、Git 2.56.0.windows.2、NTFS）；WSL Linux（UID 1000、Git 2.43.0，交叉编译后实际执行）。
- `-race`、root 跨账户测试与 macOS 的证据只来自 CI 日志。
- 故障注入只是在 I/O 边界返回错误，不是物理磁盘损坏或掉电的实测。
- 没有验证：真实的 unsafe 旧版 Git、Windows 双账户与 ReFS、穷尽式压力测试、完整的 CLI/E2E。

**CI（如实记录接口状态）**：

- **代码 CI**：[run 39276404802](https://github.com/XKHoshizora/specmetry/actions/runs/39276404802)，`a2f7bd6`，attempt 1，completed/success。合并快照 `08a59cc`，树 `8a1d4d9`，与代码 HEAD 相同。
- **受审 HEAD 的 CI**：[run 39277972806](https://github.com/XKHoshizora/specmetry/actions/runs/39277972806)，`c4dbfb1`，`pull_request`，attempt 1。
  - 审查时与本轮读取时（2026-10-20），run 的汇总都是 `in_progress`、conclusion 为 `null`（`updated_at` 为 2026-10-19T15:33:04Z）；
  - jobs 接口显示 8 个 job 都是 completed/success；
  - 审查方读取 jobs 接口时曾遇到 ReadTimeout 与 HTTP 502；
  - 日志中的实际检出为 `1c008da`，其父提交为 base 与 `c4dbfb1`，树 `e9af915` 与 HEAD 相同。
  - 这**不记作整次运行已通过，也不认定为代码失败**；没有取消、重跑，也没有追加空提交刷新状态。

**本轮负责人授权（2026-10-20）**：

- 原样归档 Review 063，完成纯文档收尾并推送；
- 新的最终 HEAD 的 CI 正式完成、且全部必需检查成功之后，用 merge commit、以 `--match-head-commit` 锁定该完整 HEAD，合并 #27；
- 核验合并内容，以及对应的 master push CI。
- 不授权：修改生产代码、测试、Schema、迁移或冻结规则；启动后续工作包；代替负责人宣布 WP2.3 或 M2 验收。

**收尾增量**：相对 `c4dbfb1`，只有本 Review 063 的归档与状态文档。生产代码、测试、Schema、迁移与受审对象完全一致。

**尚待完成的核验**：

- 新最终 HEAD 的 CI 正式结果；
- 合并（merge SHA、父提交、包含关系、内容树与 diff）；
- 对应的 master push CI。
- 本节不预写这些结果；实际证据写入 PR 描述与交付回执，不为写回结果追加提交。

**状态**：技术复审通过，等待合并核验。WP2.3 与 M2 都没有宣布验收。M-14、M-15 未解决，M-9 与其他维护事项照旧；Auto-fix / Auto-merge 保持关闭。

## 75．Review 064、#27 的合并后核验与 WP2.3 库层的验收

**PR #27 的合并后核验**（§74 的待完成项）：

- **合并**：
  - merge commit `5985b319ac4cdb23a8f9cbc9b40653c154257fec`，以 `--match-head-commit a41c991e5210a080c73b5a9c16cb3dd787bb212a` 锁定；
  - 父提交为 `e0a21b46b5ad7e05146f254aa7523d6054ce2e76` 与 `a41c991`；
  - master 包含 `a41c991`；树 `7283db1` 与最终 HEAD 相同，diff 为空。
- **最终 HEAD 的 CI**：run 39282753234，attempt 1，completed/success，8/8。检出合并快照 `ed717e1`，父提交为 `e0a21b4` 与 `a41c991`，树 `7283db1`。
- **master push CI**：run 39287153098：
  - **attempt 1**：completed/failure。8 个作业都检出 `5985b31`，其中 7 个成功。失败的是 Go tests (windows-latest)（118177199541）：`internal/store/sqlite` 的 `TestSeqIsUniqueAcrossProcesses` 中，一个 `seqwriter` 辅助进程 `exit status 1: database is locked (5) (SQLITE_BUSY)`。
  - **attempt 2**：按负责人的一次授权执行 `gh run rerun 39287153098 --failed`，结果 completed/success。只有 Go tests (windows-latest)（118203422167）实际重新执行，检出 `5985b31`，该测试通过；其余 7 个作业沿用 attempt 1 的结果。它不是首轮 8/8，也不是三个平台都重新运行。
  - **首轮失败保留**，作为 M-15 的一次新观测：根因未验证，未解决，不是已接受的风险。失败发生在 `Open` 还是写入阶段、与以往观测是否同一根因，都没有确认；attempt 2 成功不关闭 M-15。维护记录在 WP2.4 规范 PR 中补充。

**WP2.3 库层的正式验收**（产品负责人决定，2026-10-20）：

- **验收对象**：`5985b319ac4cdb23a8f9cbc9b40653c154257fec`。
- **范围**：Git Reader（`internal/gitx`）、`internal/gitscan`、迁移 0007，以及已审查的库层验证。
- **依据**：审查 063 技术 GO；合并内容与最终 HEAD 一致；master CI 39287153098 attempt 2 正式成功。
- **保留**：首轮失败的记录；M-15 未解决，不是已接受的风险；方案 B 的边界（§71）不变。
- **不在本次验收之内**：CLI 接入、输出集成与后续 E2E，由对应工作包完成。**M2 未整体验收。**

**审查 064**：

- **归档**：原样归档为 `85d909d293a03be0a1d3bc9eab46b4c96e1c24fd`。源文件、暂存与该提交 blob 的 SHA-256 均为 `504d009e9e3967e6f5c42401e59b023f151504ee1151ad95379389124ef2beac`。
- **审查对象**：PR #28，HEAD `a482dacb1bd61cd3b583cffcc519b1d205e18dc1`，base `5985b319ac4cdb23a8f9cbc9b40653c154257fec`。之后的收尾提交不改变审查对象。
- **结论：技术 GO，BLOCKER 0**；M-7 的修复要求已满足。
  - 指定的 CI 39299026564 attempt 1 正式 completed/success，8/8。
  - 实际检出 `d454091`，父提交为 base 与 HEAD，树与 HEAD 相同。
- **NON-BLOCKER N1**：
  - `questionsProject` 夹具只有 open 的问题，所以不能区分默认列表与 `--all`。这限制该夹具对过滤语义与错误配对的检出力，不影响 M-7 的保证（两份完整输出都符合 `question-list.v1`）。
  - 过滤语义由既有的 `TestQuestionAnswerAndDismiss` 覆盖（审查方只静态核对）。
  - 不要求本 PR 扩展业务测试。
- **审查方的验证边界**：只在 Windows 隔离副本上运行了两项定向测试与三项变异；三个平台的结果来自 CI；没有在本机重跑整个 CLI 包或 TTY/E2E。
- 没有新的产品待决事项，也没有新的风险接受。

**M-7**：审查通过，**等待合并后核验**。合并内容与 master push CI 都核验通过后，按负责人的决定登记关闭；实际依据在 WP2.4 规范 PR 中登记。

**状态**：

- PR #28 技术审查通过，等待合并核验。
- WP2.3 库层已验收（对象 `5985b31`）；M2 未整体验收。
- WP2.4 及其余 M2 工作包尚未授权实现。
- M-14、M-15、M-9、M-10 及其他维护事项保持原状；Auto-fix / Auto-merge 保持关闭。

## 76．M-7 关闭、M2-Q6／Q9 的批准与 WP2.4 规范草案

**M-7 关闭**（产品负责人 2026-10-20 的决定：PR #28 完成合并内容核验，且 master CI 正式成功之后登记关闭）：

- **合并**：
  - PR #28 合并为 `7a361663ccb53f6a7db594a784c08ad54a24e4c8`，merge commit，以 `--match-head-commit 79bc6931d0ca676eb6b2adf22820b03ba2fc05c5` 锁定；
  - 父提交为 `5985b319ac4cdb23a8f9cbc9b40653c154257fec` 与 `79bc693`；
  - master 包含 `79bc693`；树 `4c7e616` 与最终 HEAD 相同，diff 为空。
- **最终 HEAD 的 CI**：run 39301966402，`pull_request`，attempt 1，completed/success，8/8。检出合并快照 `33e765a`，父提交为 `5985b31` 与 `79bc693`，树 `4c7e616`。
- **master push CI**：run 39303615136，`push`，attempt 1，completed/success，8/8；每个作业都检出 `7a36166`（`git log -1 --format=%H`）。
- **审查**：审查 064 对 `a482dac` 技术 GO，BLOCKER 0。合并内容相对受审对象只多了文档（§75）。
- **结论：M-7 关闭**。审查 064 的 N1（夹具不能区分默认列表与 `--all`）作为该夹具的检出边界保留，不是 M-7 的遗漏。

**M2-Q6 / TD-13：批准建议默认**（产品负责人，2026-10-20）：

- 沿用启发式通道识别、`RequireHuman`、现有的 `confirm` 入口，以及操作者与通道的审计，如实声明能力边界；通道识别不等于身份认证。
- M2 新增的人类专属操作沿用该机制，不增加签名、独立操作系统用户等新的隔离机制，也不提前实现 M3 的 Hook。
- Claim 中只有 `manual` 批次是人类专属，不能把整个 `claim` 命令都限制为人工。
- WP2.4 规范须写明识别批次类型、通道拒绝、输入处理、数据库访问、确认与写入的顺序，保持既有的拒绝与隐私契约。

**M2-Q9：批准独立回执的方向**（产品负责人，2026-10-20）：新增 `claim-receipt.v1`，包含：

- 批次身份、`seq`、`kind`、`base_version`；
- CLI 实际采集的 `worktree_id`、`head`、`tree_state`；
- 只有 `sync` 有 `compare_from`，含起点与来源；
- `claims` 与输入逐项对应、顺序一致，含索引、AC、修订、断言，以及引用的存在性结果；
- 无引用、空 `claims` 与既有的 `unmapped_work` 范围，都有明确的表达；
- 不回显摘要、正文、命令、输出或环境变量的值；
- 存在性结果不代表验证通过或验收通过。

字段细则依据现有模型写成可审查的契约，不借此改变 Claim 或推导的语义；具体的规范与 schema 仍须独立审查后冻结。

**M-15 的新观测**：登记在 m1-exit-evidence §9（见 §75）。根因未验证，未解决，不是已接受的风险。

**WP2.4 规范草案**：

- **文件**：[docs/wp2.4-claim.md](wp2.4-claim.md)，以及 `schemas/claim-receipt.v1.schema.json`。另有 3 个合法示例（`sync`、`audit`、空批次）与 25 个 schema 反例（SCH-166 ~ SCH-190），由既有的契约测试验证。
- **内容**：处理顺序；`manual` 批次的通道、确认与审计；写锁下的前置条件；代码上下文与 `compare_from`；引用检查与 PD-26；写入与 `seq`；回执的字段，以及 schema 与业务测试的分工；回执在提交之前校验；隐私；验收映射；正反例；测试计划。
- **待负责人决定**：D1 ~ D3（规范 §14）。
- **待审的工程细化**：E1 ~ E6。
- **状态**：**WP2.4 规范待独立审查，尚未授权实现**，未冻结。本 PR 只含规范、schema、示例与必要的状态同步，不含 Claim 的生产实现或迁移 SQL；M2-Q2、TD-23 等已冻结的决定不重开。

**状态**：

- M-7 已关闭。
- M2-Q6、M2-Q9 已批准；WP2.4 规范待独立审查，尚未授权实现。
- M2 未整体验收。
- M-14、M-15、M-9、M-10 及其他维护事项保持原状。

## 77．WP2.4 规范 D1 ~ D3 的决定、生命周期表述的修正与维护记录

**负责人决定**（2026-10-20；M2-Q6、M2-Q9 的批准不重新打开）：

- **D1**：`gitx.Clean` 返回实际的环境错误时，退出码 6，不写入本次 Claim 的业务事实，不消耗本次写入的 `seq`。
  - 正常返回的 `undecidable` → `unknown`，以及非 git 项目的 `unknown`，保持既有规则。
  - 不把正常打开与方案 B 的保证扩大成"数据目录或辅助文件绝对不变"。
  - 理由：`tree_state` 是批次必需的代码上下文；WP2.3 规定读取失败不报告 `clean`。
  - 代价：这类环境中 `claim` 不可用。
- **D2**：`audit_scope.requirements` 中有不存在或已退役的需求编号时，退出码 2。
  - 理由：声明的范围必须指向当前的需求，与 V-C03 对 AC 的要求一致。
  - 代价：编号写错的批次被整体拒绝。
- **D3**：`audit` Claim 的目标 AC 不属于所声明的需求时，退出码 2；在写锁下核对当前的实体与所属关系。
  - 不要求一次提交覆盖范围内的全部 AC；空 `claims` 等既有合法输入保持合法，不新增完整盘点的门槛。
  - 理由：声明的范围与写入的 `audit` Claim 一致，局部 audit 不能计入范围外的覆盖率。
  - 代价：跨需求的盘点要么一并声明，要么分批提交。
- **同步**：
  - WP2.4 规范 §5、§6.2、§12 ~ §15；
  - data-model 的 V-C05 与 §4.10；
  - acceptance-criteria 的 MVP-INV-01、MVP-CLM-04（都标为"规范待审查"）。
- **E1 ~ E6** 仍是待独立审查的工程细化，不是本轮已获批的内容。规范 §14 列出了其中可能含有的产品取舍：
  - E4：有歧义的提交前缀是标记还是拒绝，回执是否区分"有歧义"与"不存在"；
  - E6：回执中测试名只计数、`audit_scope` 不回显。

**生命周期表述的修正**：

- WP2.4 规范 §10 原写"私有临时区……读取结束即删除"。这不能覆盖方案 B 已批准的情形：失败保留、消费者状态未知、完成记录持久化失败。
- 现改为准确引用 WP2.3 §14.3：满足可信清理条件时按清单清理，否则按既有要求保留并报告（退出码 6）。
- 方案 B、已接受的边界与生产实现都不变。

**维护记录**：

- **M-15** 补记 PR #29 的 run 39307739451 attempt 1：
  - Go tests (windows-latest)，作业 118249164263，检出 `60f8820`；
  - `TestSeqIsUniqueAcrossConnections` 与 `TestSeqIsUniqueAcrossProcesses` 都出现 `SQLITE_BUSY`；前者是新增的测试场景。
  - 错误阶段、与以往观测是否同根因，都未确认。
- **M-16**（新登记，下一个空闲编号）：Windows CI 上 `internal/cli` 包达到 `go test` 默认的每包 600 秒超时，耗时余量不足。
  - 此前 Windows CI 上为 282 ~ 454 s。
  - 同一作业中各包都变慢，这一点只作为观测；资源竞争是待验证的解释。
  - 与 M-10（本地 Windows 超时）分开。
- 两项都未解决，不是已接受的风险。本轮不修改 SQLite 重试、错误分类、CI 超时、并行度或测试行为。

**CI 记录**：

- PR #29 原 HEAD `91e7e5f` 的 run 39307739451，attempt 1，completed/failure。
  - 8 个作业都检出 `60f8820`（父提交 `7a36166`、`91e7e5f`，树 `6b0a376` 与 HEAD 相同）；7 个作业成功。
  - 失败记录保留；没有重跑，没有用空提交触发检查。
- 新 HEAD 的 CI 按正常触发的运行核对，结果记在 PR 描述中。

**状态**：

- WP2.4 规范待独立审查，未冻结，尚未授权实现；PR #29 不合并。
- M2 未整体验收。
- 其他维护事项与既有验收边界不变；Auto-fix / Auto-merge 保持关闭。

## 78．Review 065、B1 的版本优先修正与 E4／E6 的决定

**审查**：[Review 065](reviews/codex-review-065.md)，原样归档为 `ca3e4950d274f9fa05b3eb1157df8372f19f4a79`。源文件、暂存与该提交 blob 的 SHA-256 均为 `109c823528dc4ecd9c352840f73af143eca2e5167e17ec00113a0e91b29fc0f3`，逐字节一致。

**审查对象**：

- PR #29，HEAD `058b9a07ef3d62acae6ab51e45fb338f7e5820e3`（树 `e479771`），base `7a361663ccb53f6a7db594a784c08ad54a24e4c8`；
- 审查的是整个 base → HEAD（15 个文件），不只是最后一次提交的六个文档。

**结论：技术 NO-GO**，唯一 BLOCKER 为 B1；PR #29 不建议合并，规范尚不具备冻结条件。

- **B1**：规范要求写锁下先检查 `base_version`（不一致即退出码 5），但 §13 的并发退役反例与 §15 的测试计划又要求旧输入返回退出码 2。
  - 合法的退役必然推进基准版本（`baseline.Apply`），所以同一段合法历史不能同时满足两种结果。
  - 审查方明确：这是规范内部的冲突，不是已复现的 Claim 实现缺陷，当时也还没有 Claim 实现。
- **NON-BLOCKER**：
  - N1：失败总述中的"不写入任何数据"应限定为本次的业务事实；
  - N2：提交之后的输出故障可能留下部分 stdout，不能承诺再追加一份完整的 `error.v1`；重新读取状态也不能证明原提交不存在。
- **PRODUCT DECISION**：
  - P1：E4，有歧义的前缀与回执中的原因；
  - P2：E6，回执的最小内容。
  - 审查方的推荐都是批准当时的方案，但不代为批准。
- **E1、E2、E3、E5**：审查方判断为符合既有契约的工程细化，并给出补充说明（E1 提示的内容；E2 不证明身份；E3 的准确选择与例子；E5 只是如实披露）。
- **已成立的部分**：
  - `claim-receipt.v1` 没有结构或条件分支缺陷；24 个 schema 编译、43 个合法示例通过、190 个非法反例拒绝；SCH-166 ~ SCH-190 的拒绝原因与描述相符；
  - Q6 的处理顺序、D1 ~ D3、事务与 `seq`、隐私、方案 B 的引用、验收映射，均与既有契约一致。
- **实现时须兑现的要求**：受控的前缀解析函数、固定 `head` 的引用检查、迁移约束、完整的业务回滚与首次 worktree 的 `seq`、回执与数据库行的对应、输出故障与真实通道的测试。审查方说明这些是工作包的交付要求，不要求规范 PR 提前实现。
- **审查方的验证边界**：
  - 运行环境：Windows/amd64、Go 1.27.1、Git 2.56.0.windows.2；固定 HEAD 经 `git archive` 导出到仓库外。
  - 运行内容：契约测试，以及逐项核对新反例的临时测试。临时测试的一次计数失败是该检查程序自己的问题，审查方没有把它记成产品检出。
  - 没有 Claim 命令可运行，没有伪造任何实现证据。
  - CI：核对了 run 39424865715（`058b9a0`，attempt 1，8/8，检出 `c340ca6`，树 `e479771`）与 run 39307739451（`91e7e5f`，失败）。新的 CI 成功不证明旧失败的根因，M-15、M-16 不解决。
- **ACCEPTED RISK**：没有新增。

**修订**（实施方，PR #29）：

- **B1，按版本优先修正**（不改变 §5 中先检查 `base_version` 的顺序，不撤销 D2、D3，不新增盘点完整性的要求）：
  - 输入基于旧版本：读取之后、取得写锁之前发生合法退役，推进了基准版本 → 先返回退出码 5，`error.current` 按既有契约给出当前的基准版本；不写入本次 Claim 的业务事实，不消耗本次写入的 `seq`。
  - 输入携带当前的基准版本，但目标或 `audit_scope` 引用已经无效 → 版本检查通过之后，按 V-C03、D2、D3 返回退出码 2，保留对应的 JSON Pointer 与无写入保证。
  - 人工确认之后的重新检查适用同一优先级。
  - 修改位置：规范 §4 第 5 步、§5 第 10 步、§13、§14、§15；acceptance-criteria 的 MVP-INV-01、MVP-CLM-02。
  - 测试计划不构造"退役而不推进基准版本"的非法历史，也不恢复"AC 迁移到其他需求"的不可能场景。
- **N1、N2**：规范 §5、§9.3、§9.4、§13 的表述已收口；不修改其他命令的输出助手。

**负责人决定**（2026-10-20）：

- **E4**：
  - 格式合法的提交前缀，只有唯一指向可读提交时 `resolved = true`；
  - 真正不存在、指向非提交对象、真实歧义时 `resolved = false`，照常提交；
  - MVP 回执不新增区分"不存在 / 非提交 / 歧义"的原因字段，沿用既有的 `unknown_commit` 标记语义；
  - 调用方可以延长前缀或使用完整 SHA 消除歧义。
  - 必须保留：格式错误仍按输入契约拒绝；对象损坏、读取失败、无法可靠分类或 PD-26 拒绝时退出码 6；不得把所有非零退出都转换成 `resolved = false`；存在性不代表验证或验收通过。
  - 理由：与"只标记，不拒绝"一致。代价：回执不说明 `false` 的原因。
  - 落点：规范 §7、§13、§14、§15；data-model 的 V-C06；MVP-INV-02。
- **E6**：
  - `refs.tests` 只输出输入测试名的数量，不回显测试名；
  - `claim-receipt.v1` 不输出 `audit_scope`；它仍按既有数据模型保存在批次事实中，输入校验与 D2、D3 不变；
  - 回执用于确认写入结果，不是完整的输入备份，调用方需要保留原输入。
  - 不改变 `status.v1`、`gaps.v1` 的既有重放保证，也不修改其他已批准的回执字段（包括 `claims` 与 `unmapped_work` 的逐项对应）。
  - 理由：调用方已有原输入，减少暴露。代价：回执不能单独还原审计范围或测试名。
  - 落点：规范 §9.1、§14。
- 现有的 schema、示例与反例已经表达这两项决定，所以不改动它们的结构。

**状态**：

- 规范已修订，**待独立复审**，未冻结；B1 由复审判断，实施方不宣布关闭。
- PR #29 不合并；WP2.4 尚未授权实现；M2 未整体验收。
- M-15、M-16 等维护事项保持未解决；方案 B 的边界不变；Auto-fix / Auto-merge 保持关闭。

## 79．Review 066：WP2.4 规范 GO、冻结与条件授权

**审查**：[Review 066](reviews/codex-review-066.md)，原样归档为 `49e86e2b6b0caeae3002532822fea1279b11880e`。源文件、暂存与该提交 blob 的 SHA-256 均为 `0103578333c24c18a1068dd9b4cfb199753a23f4f9272530cc0956d87ebf5335`，逐字节一致。

**审查对象**（不随之后的收尾提交而改变）：PR #29，HEAD `29834d1fb0c70d3abd95b0b3efba777c98bdba9c`（树 `7619127`），base `7a361663ccb53f6a7db594a784c08ad54a24e4c8`，上轮对象 `058b9a0`。

**结论：GO，没有 BLOCKER**。

- **Review 065 B1 关闭**：版本优先的冲突已修正。合法的并发退役先返回退出码 5；当前版本下的无效实体在版本检查通过之后才返回退出码 2；`manual` 确认之后同样如此。
- **N1、N2 在规范层面收口**；实现阶段仍须验证所有失败路径与输出故障。
- **E4、E6 已忠实落实负责人的批准**，不再是待决事项；E1、E2、E3、E5 复审确认。
- **没有剩余的 WP2.4 产品待决事项**；Q6、Q9、D1 ~ D3 不重开。
- **NON-BLOCKER N-doc**：并发测试中的"`seq` 计数器不变"应写明比较快照的基点，不能把另一个连接的合法提交算作被拒 Claim 的写入。
- **ACCEPTED RISK**：没有新增；方案 B 的边界不扩大；M-15、M-16 等维护事项不是接受的风险。
- **审查方的验证边界**：
  - 只在 Windows/amd64（Go 1.27.1）的仓库外隔离副本中运行了契约回归，PASS；
  - 没有 Claim 实现，没有运行虚构的命令、数据库竞争或 git 解析实验；
  - 核对了 CI run 39433229408（`29834d1`，attempt 1，8/8，检出 `0c3e10c`，树 `7619127`），并保留旧 run 39307739451 的失败；没有重跑任何 CI。

**收尾**（实施方）：

- **N-doc 落实**：规范 §15 写明比较基点。
  - 单连接的反例，以被拒的 Claim 开始写事务之前的快照为基点。
  - 并发用例按确定的调度执行：A 暂停在写锁或确认之前，B 完整提交，读取 B 提交之后的快照 S1，再放行 A；A 得到退出码 5 之后与 S1 比较，没有 A 的批次、Claim、未映射工作、首次 worktree 登记或额外的 `seq`，B 的全部事实保持不变。
  - 不要求回退另一个连接已提交的事实或 `seq`。
  - 本条不改变"本次 Claim 不提交事实或 `seq`"的契约。
- **状态同步**：规范的状态行；规范中 E1 ~ E3 的"待审查"标注；acceptance-criteria 的 MVP-INV-01、MVP-INV-02、MVP-CLM-02、MVP-CLM-04；data-model §8、V-C05、V-C06；architecture §9.1；prd §6；AGENTS.md；README；实施计划。
- 历史结论与审查身份不改写：§76 ~ §78 及审查 065 的原文保留。
- 没有修改生产代码、测试行为、Schema、迁移或工作流。

**负责人决定与授权**（2026-10-20）：

1. **冻结**经审查 066 GO 的 WP2.4 规范：`docs/wp2.4-claim.md`，以及 `schemas/claim-receipt.v1.schema.json` 与它的示例、反例。
2. **条件合并 PR #29**：最终 HEAD 的 CI 正式成功之后，以 merge commit、以 `--match-head-commit` 锁定完整的最终 HEAD 合并，并核验合并结果与对应的 master push CI。
   - CI 失败时的处理只适用于本次规范收尾：若唯一的失败是已登记的 Windows M-15 / M-16 特征，且相关实现与测试都没有改，保存原失败证据之后，每个运行最多允许一次 `--failed` 的受控重跑。
   - 沿用的结果与实际重跑的作业分开记录；再次失败或出现其他失败时，停止合并或后续开工。
   - 这一授权不延伸到后面的实现 PR；重跑成功也不关闭维护事项。
3. **条件授权 WP2.4 的实现**：只有合并核验与对应的 master push CI 全部成功之后，才可以实现、测试、提交、推送，并创建独立的实现 PR。
   - 不授权合并实现 PR、验收工作包，也不授权 WP2.5 或其他工作包。

**尚待完成的核验**：最终 HEAD 的 CI、合并（merge SHA、父提交、包含关系、树与 diff）以及对应的 master push CI。本节不预写这些结果；实际证据写入 PR 描述与交付回执。

**状态**：

- WP2.4 规范已冻结，实现须等上述核验全部完成之后开工。
- M2 未整体验收。
- M-14、M-15、M-16 等维护事项保持未解决；方案 B 的边界不变；Auto-fix / Auto-merge 保持关闭。

## 80．Review 067：PR #30 的 WP2.4 实现 NO-GO（实现 B1、B2）与定向修复

**审查**：[Review 067](reviews/codex-review-067.md)，原样归档为 `cb70356f9b843a0830edcd483ee4f634d25fa653`。源文件、归档与该提交 blob 的 SHA-256 均为 `3794cd450746b76d6f5f623d647ec762a0ed39d2e3b57d4889309746302be7b6`，逐字节一致。

**审查对象**（不随之后的修复提交而改变）：PR #30，HEAD `69d2f86bb1c9f2b3f842e127576ef04b7d886d7a`（树 `e828b90`），base `d8f8a50a7f01d9fec9ce7dd699495ff829c9229f`；首次实现审查，覆盖 base→HEAD 的全部 21 个文件。

**结论：NO-GO，两项 BLOCKER**；合并检查（CI run 39464343744，attempt 1，8/8，检出 `1a2caaf`）成功与技术审查 NO-GO 分别成立。

- **B1（实现）**：`gitx.ResolveCommits` 只完整读取并核对将返回 `true` 的那个提交。唯一的非提交对象与歧义前缀的候选只凭对象头分类，头部可读、正文损坏的对象被当作普通的 `resolved = false`，整个批次照常写入。审查方在 Windows git 2.56 与 WSL git 2.43 上经公开 CLI 复现了新增的批次、Claim 与 `seq`。这违反冻结的 E4 / V-C06："对象损坏 → 退出码 6"。
- **B2（实现）**：默认文本模式跳过了提交前的回执 Schema 校验（`claimOutput` 在校验之前返回文本）。同一个注入的非法回执，在 `--json` 下被拒绝并回滚，在文本模式下却被提交，还打印出错误的 `seq`。这违反规范 §5 第 10 步与 §9.3。
- **NON-BLOCKER N1**：PR 描述把最终改为非 git 项目的 PTY 测试仍列为 git 相关用例；应分开陈述"PTY 证明通道""真实 git 用例证明上下文"。
- **PRODUCT DECISION**：没有新增；Q6、Q9、D1 ~ D3、E4、E6 都是冻结输入。PR 描述中"并发 Claim 是否等待合作锁"的设想不是当前实现必需的决定，本轮不批准等待策略，也不授权修改方案 B。
- **ACCEPTED RISK**：没有新增；只沿用 §71 的方案 B 局部排除。M-14、M-15、M-16 不是接受的风险。
- **审查方的验证边界**：
  - 在仓库外的隔离副本中运行（Windows/amd64 与 WSL Ubuntu/amd64、Go 1.27.1）；
  - 没有独立重跑作者的桩阶段红灯、全部变异或真实 git 2.32.0 实验，不把作者的变异计数当作穷尽证明；
  - 没有 macOS 本机（macOS 结果来自 CI）。

**与既往结论的区分**：Review 065 的规范 B1（并发退役的版本优先）已由 Review 066 关闭，不因本轮重开。Review 067 的 B1、B2 是实现层面的另两项发现，**尚未关闭**，须经独立复审确认。

**修复**（实施方，提交 `dd353d9`、`8b8311f`；**已修复，待独立复审**）：

- **B1**（`internal/gitx/resolve.go`）：
  - 前缀列出的每个对象，在给出任何 `true` / `false` 之前，都经同一次 `cat-file --batch` 流式读取：对象头须是该 ID、已知类型；正文须恰好是声明的长度，并按 `"<type> <size>\0"` 与正文重新计算对象 ID。
  - 正文只流过哈希，不保存；按对象 ID 去重；取消沿用现有机制。
  - 任何不一致、stderr 输出或意外状态都是退出码 6，不返回任何前缀的部分结果。
  - 健康的 blob、tree、标签对象（不因可解引用到提交就当作提交）与真实歧义仍为 `false`（E4 不变）；存储检查与 PD-26 不变。
  - 根因的补充事实（实施方实验，git 2.56）：头部可读的损坏 loose 对象，blob 在 `cat-file --batch` 下只输出残存的内容（短于声明长度）；tree（`--batch`）与 commit（`cat-file commit`）则以进程内存中的内容补足到声明长度；一个对象的文件装着另一个合法对象的字节时，git 按那个对象（及其类型）输出。以上情况退出码都是 0，所以只能以声明长度与重新计算的对象 ID 判断。
- **B2**（`internal/cli/claim.go`）：`claimOutput` 先按 `claim-receipt.v1` 校验回执，再按所选模式编码；`--json` 只决定呈现方式。不符合时为内部错误（退出码 1），整个事务回滚，包括首次的 worktree 登记。`manual` 的预演与最终写事务共用这一路径。提交之后的输出故障行为不变。
- **回归**（均先在旧实现上确认失败，证据见 PR #30 描述）：
  - gitx：`TestResolveCommitsVerifiesEveryCandidate`（8 个子场景）；Unix 上歧义前缀中不可读的候选。
  - CLI：`TestClaimRefusesADamagedCommitCandidate`（文本与 JSON、新 worktree、正常与异常混合的引用、修复后的对照；对象内容不进入任何输出或数据目录）；`TestClaimReceiptCheckedInEveryMode`（三种批次在两种模式下的正常对照；注入失败在两种模式下回滚，`manual` 分预演失败与确认之后最终事务失败）；`TestClaimRegistersTheWorktreeOrNothing` 扩展到文本模式。
  - 变异检查 4 处全部捕获：不比较重算的 ID；接受短于声明长度的正文；只验证唯一候选；只在 `--json` 时校验。另有一处首次写错而偶然通过的变异，更正后捕获，记录保留。
- **N1**：PR 描述已更正测试口径。

**发现的其他直接关联问题（本轮未修复，待决定）**：Claim 的文件引用检查，与 `propose` 共用 `proposal` 的代码引用检查，同样信任 git 的成功读取：

- HEAD 提交只按对象头解析（`CommitOf`）；
- 文件列表来自 `ls-tree -r`，一棵装着另一棵树字节的树会被照常列出；
- 行数来自 `cat-file blob`，头部可读的损坏 blob 按残存内容计数（实施方实验，退出码 0）。

结果可能是错误的 `exists` 标记，而不是退出码 6。修复须沿每个引用路径逐个验证 commit、tree、blob 对象，并且会改变 M1 已验收的 `propose` 的行为（共用路径），超出 Review 067 B1（提交前缀候选）的范围。建议由负责人决定：是否在 WP2.4 内为 Claim 的文件引用补上沿路径的对象验证，以及 `propose` 是否同步（M1 的变更）。决定之前，行为不变。

**状态**：

- PR #30 已修复，待独立复审；不合并，不宣布 B1、B2 关闭。
- WP2.4 与 M2 都未验收；WP2.5 等其他工作包未启动。
- M-14、M-15、M-16 等维护事项保持未解决；方案 B 不变；Auto-fix / Auto-merge 保持关闭。

## 81．Review 068：PR #30 定向复审，实现 B1、B2 关闭，B3（文件引用完整性）阻塞

**审查**：[Review 068](reviews/codex-review-068.md)，原样归档为 `1c0a0f7930c5f3eb102297af6e9593642561f5a3`。源文件、归档与该提交 blob 的 SHA-256 均为 `09c6b472d610162f9a47aab709e3fd2f3aacc6446b48b66d476ff299a95b31a7`，逐字节一致。

**审查对象**（不随之后的修复提交而改变）：PR #30，HEAD `e70c09bb404db8371ac268c2bf64c4379141f26f`（树 `245c4da`），base `d8f8a50a7f01d9fec9ce7dd699495ff829c9229f`；增量是 §80 之后的 5 个提交、9 个文件。

**结论：仍为 NO-GO**；CI run 39526415079（attempt 1，8/8，检出 `b0ffc03`）成功与技术审查 NO-GO 分别成立，CI 成功与验证边界不能替代缺陷的关闭。

- **Review 067 B1 关闭**：提交前缀的全部候选，包括唯一的非提交对象与歧义前缀的每个候选，都先完整读取并核对长度与对象 ID，再给出结果；失败不返回部分结果。
- **Review 067 B2 关闭**：文本与 JSON 都在事务内、COMMIT 之前校验规范回执并准备输出；`manual` 的预演与最终事务共用这一路径。
- **B3（新的阻塞，沿用报告原编号）**：文件引用检查没有可靠核对对象的完整性。
  - 位置：`proposal` 的 `CheckCodeRefs` / `CheckCodeRefsAt` 共用的检查，经 `gitx.TreeFiles`（信任 `ls-tree -r`）与 `gitx.CountLines`（信任 `cat-file blob` 的字节）。
  - 审查方在 Windows git 2.56 与 WSL git 2.43 上独立复现了三种假成功：头部可读、正文截断的 blob 被标为存在；blob 装着另一个合法 blob 的字节时按替代内容计数；子树装着另一棵树的字节时，把固定提交中没有的文件标为存在。
  - 经公开命令入口，Claim 在 JSON 与文本模式下都成功提交了批次、Claim、首次 worktree 与 `seq`。
  - 这违反 WP2.4 规范 §6.1、§7 与 V-C06。缺口在本 PR 之前的共用路径中，但直接影响 WP2.4 的交付行为。
- **NON-BLOCKER N1**：batch 解析器的头长限制依赖写入的分块（同一块中先 `begin()`，之后的长度检查看不到已处理的头）。
- **PRODUCT DECISION**：没有新增；E4、E6、D1 ~ D3、版本优先与存在性的含义都不变。负责人把文件引用的修复（包括共用的 `propose` 路径）纳入 WP2.4，是实施范围的决定，不是放宽保证。
- **ACCEPTED RISK**：没有新增；只沿用 §71 的方案 B 局部排除。M-14、M-15、M-16 不是接受的风险。
- **审查方的验证边界**：仓库外隔离副本（Windows/amd64 与 WSL Ubuntu/amd64，Go 1.27.1）；没有独立重跑真实 git 2.32.0、作者的全部变异、正式发布二进制的 E2E 或物理故障；macOS 结果来自 CI。

**修复**（实施方，提交 `87fb135`；**已修复，待独立复审**，不宣布 B3 关闭）：

- **B3**：
  - `gitx.FileLines` 取代 `TreeFiles` 与 `CountLines`。它只读答案所依据的对象：固定提交、通往每个路径的树（逐层）与文件的 blob（按对象 ID 去重），不读其他树、blob 或历史。
  - 每个对象都经与 Review 067 B1 相同的受校验读取（共用的 `verify.go`：一次 `cat-file --batch`，核对 ID、类型、长度与完整内容），并须是树条目给出的类型。
  - 解析树与计算行数都用同一次已核对的读取；只缓存已验证的派生结果（解析后的树、行数）；不保存 blob 内容。
  - 任何不一致、读取失败或 stderr 输出都是退出码 6，不返回任何路径的结果，不记为 `exists = true` 或普通的 `missing`。
  - 健康树中不存在的路径、行号越界、目录、符号链接、子模块、unborn、非 git 项目的既有语义不变；PD-26 与部分克隆的拒绝仍在先。
  - `propose` 与 Claim 共用这一路径；Claim 仍绑定采集的 HEAD。不再使用的 `runObjects`、`runObjectsTo` 一并删除。WP2.3 的 check.inputs（`TreeEntry`）不变。
- **N1**：校验器在换行分支中先检查完整头长再 `begin()`，判定与分块无关。
- **回归**（均先在旧实现上确认失败，证据见 PR #30 描述）：
  - `TestCheckCodeRefsVerifiesTheObjects`（两个接口；Review 068 的反例；修复后的对照；HEAD 移动）；
  - `TestFileLines`（原 `TreeFiles` 的预期值）、`TestFileLinesVerifiesTheObjects`、`TestFileLinesChecksTheTypes`、`TestObjectVerifierHeaderLimit`；
  - CLI：`TestClaimRefusesADamagedFileObject`、`TestProposeRefusesADamagedFileObject`（文本与 JSON；Claim 的首次 worktree；对象内容与 git 的 stderr 不出现在输出或数据目录中）。
  - 既有的 promisor 与 PD-26 测试改为经 `FileLines` 读取，断言不变。
  - 变异检查 4 处全部捕获：不比较重算的 ID、接受短于声明长度的内容（两者在 gitx 与 `propose` 的共用路径上都被捕获）、不检查对象类型（补 `TestFileLinesChecksTheTypes` 之后捕获，首次未捕获的记录保留）、N1 改回旧写法。

**状态**：

- PR #30 已修复，待独立复审；当前仍为 NO-GO，不合并，不宣布 B3 关闭。
- WP2.4 与 M2 都未验收；WP2.5 等其他工作包未启动。
- PR #31（M-16 的 CI 调整）单独审查，本轮不修改或合并。M-14、M-15、M-16 保持未解决；方案 B 不变；Auto-fix / Auto-merge 保持关闭。

## 82．Review 069：PR #31（M-16 的风险缓解）GO、合并与核验；PR #30 整合合并后的 master

**审查**：[Review 069](reviews/codex-review-069.md)，在 PR #31 的收尾中原样归档为 `b6b8a16bdf417a3d33954b9e10cb4f48ea6181a1`。源文件、归档与该提交 blob 的 SHA-256 均为 `eb257e9177ee514d40a288b9fa9325a0b083aeb6e87f3473ace06b0e3b1e41b7`，逐字节一致。

**审查对象**（不随收尾提交而改变）：

- PR #31，HEAD `0de32c5b4a319619b49a27fe1c8aaac1c80c5d8d`（树 `7406acf`），base `d8f8a50a7f01d9fec9ce7dd699495ff829c9229f`。
- 一个提交、三个文件：`.github/workflows/ci.yml` 中 Go tests 的 Test 步骤、`docs/m1-exit-evidence.md` §9 的 M-16 行、实施计划的 M-16 行。
- 审查依据的 CI 是 run 39600735762（attempt 1，8/8，检出 `3c901fb`）。

**结论：GO**。

- 没有 BLOCKER，也没有本 PR 新增的 NON-BLOCKER；报告没有提出新的产品决定。
- **分组**：Windows 的全部包来自实时的 `go list -tags nowebui ./...`，恰好分成非 CLI 与 CLI 两组。前者保留默认的每包 10 分钟，后者单独使用 `-timeout 20m`。
- **失败处理**：包枚举失败使步骤失败；任何一组失败都使步骤失败，第一组失败后第二组仍执行。没有自动重试、失败过滤、测试删减或固定的包数量。
- **不变的部分**：Ubuntu 与 macOS 的执行方式、Ubuntu 的 `-race`、Linux 的 root 跨用户步骤，以及其他 Job。
- **M-16 继续未解决**：一次成功不证明 20 分钟始终足够，也不证明 CI 变慢的原因。CLI 挂起的报告可能从 10 分钟推迟到 20 分钟；这是负责人批准的本次取舍，不延伸为其他保证豁免。

**负责人决定**（2026-10-22）：

- 条件授权合并 PR #31。
- PR #31 合并核验及 master CI 全部成功后，把已核验的 master 整合进 PR #30。
- 不授权合并 PR #30、关闭 B3、宣布验收或启动后续工作包。

**PR #31 的收尾**（只有文档）：

- `b6b8a16` 归档 Review 069；`f13d231` 在 M-16 的活跃记录（`docs/m1-exit-evidence.md` §9 与实施计划）中登记 GO 与条件授权。
- 相对 `0de32c5` 只有上述三个文档文件。工作流、`internal`、`cmd`、`schemas`、`web`、`scripts`、`go.mod`、`go.sum` 都不变。
- 收尾 HEAD `f13d23156932e48e558ac6b9609e3991fd8b73f0` 的 CI 是 run 39720152759：`pull_request`，attempt 1，8/8 正式成功，没有重跑。
  - 8 个 Job 都检出 `949228c`（父提交 `d8f8a50`、`f13d231`），其树 `cd156ae` 等于 `git merge-tree --write-tree` 的结果。
  - Windows 枚举出 31 个包，分成 30 + 1 两组，每个包执行一次；CLI 一组最后执行，用时 399 s。
  - 三个平台的包集合相同，没有 FAIL。

**合并与核验**：

- 2026-10-22 以 merge commit 合并，用 `--match-head-commit f13d231…` 锁定完整的最终 HEAD。合并提交为 `8b45ef598da1599ea672072338fef5ed9c6f77ee`，父提交 `d8f8a50`、`f13d231`。
- master 包含 `f13d231`；合并树 `cd156ae` 与最终 HEAD 相同，`git diff` 为空。
- master push CI 是 run 39721731294：`push`，attempt 1，8/8 正式成功，没有重跑。
  - 8 个 Job 都检出 `8b45ef5`。
  - Windows 按新分组执行：31 个包，30 + 1，每个包一次；CLI 一组用 20 分钟预算，耗时 363 s。
  - 三个平台的包集合相同，没有 FAIL；Ubuntu 的 root 步骤执行了 8 个 `TestCrossUser*`。
- 两次 CI 都没有用到 M-15 的重跑许可。

**M-16**：缓解已在 master 生效（Windows 分组执行，CLI 的包级超时为 20 分钟）。这是风险缓解，根因未验证，M-16 保持未解决；代价是 Windows CLI 的挂起最晚 20 分钟才被报告。

**PR #30 的整合**：

- 在 `feat/m2-wp24-claim` 上以 merge commit 合并 `8b45ef5`（不改基，不强推）。合并提交为 `cc2d006251d9d272914fdc9e0f27902caa9119ad`，父提交 `8c0db79`、`8b45ef5`。
- 只有 `docs/implementation-plan.md` 的相邻两行冲突。PR #30 的 M2-Q6/Q9 行（含 §80、§81）与 master 的 M-16 行都保留，各一次。§80、§81 原样保留，本节接在其后。
- 整合后：
  - 源码、测试、Schema、迁移与依赖（`internal`、`cmd`、`schemas`、`web`、`scripts`、`go.mod`、`go.sum`）与 `8c0db79` 相同；
  - `.github` 与 master `8b45ef5` 相同；
  - 相对 `8c0db79` 的其余变化，只有 PR #31 带来的 M-16 文档与 Review 069 原文，以及本节与状态行的同步。
- 整合后的 CI 结果记录在 PR #30 的描述中，不为记录 CI 追加提交。

**状态**：

- Review 067 的 B1、B2 已由 Review 068 关闭；Review 068 的 B3 与 N1 **已修复，待独立复审**。本节不改变这一点，也不改写此前任何审查的结论。PR #30 在复审前仍为 NO-GO，不合并。
- 最终复审对象是整合后的 PR #30 HEAD，即本节所在的提交（SHA 见 PR #30 的描述）。相对 Review 068 的审查对象 `e70c09b`，源码与测试的增量仍只有 `87fb135`；工作流的变化全部来自已合并的 PR #31。
- WP2.4 与 M2 都未验收；WP2.5 等其他工作包未启动。M-14、M-15、M-16 保持未解决；方案 B 不变；Auto-fix / Auto-merge 保持关闭。

## 83．Review 070：PR #30 定向复审 GO，B3、N1 关闭

**审查**：[Review 070](reviews/codex-review-070.md)，原样归档为 `6b78cddd9e3c9bf56401da4bc9b5b7584e695c8a`。源文件、归档与该提交 blob 的 SHA-256 均为 `13396f52b21153658ff6c5ad390231a0ed9be35484cf8f0ddea571696dc24b80`，逐字节一致。

**审查对象**（不随收尾提交而改变）：

- PR #30，HEAD `b8606b5e51ee2ca7de5cf9bfe0b2f45f6d431827`（树 `abaf20d`），base `8b45ef598da1599ea672072338fef5ed9c6f77ee`。
- 核心修复是 `87fb135`，整合提交是 `cc2d006`；Review 068 的对象 `e70c09b` 及其当时的 base `d8f8a50` 保持原样。
- 审查依据的 CI 是 run 39723458143（attempt 1，8/8，检出 `1041a85`）。

**结论：GO**。

- **B3 关闭**：
  - 固定提交、必要的树与目标普通文件的 blob 都完整验证；解析路径与计数使用同一次已验证的读取，失败不返回部分结果。
  - 保证只覆盖本次引用判定实际需要的对象，不扩展到未引用的 blob、无关的树或历史，也不要求 `check.inputs` 的 `TreeEntry` 因此读取 blob。
- **N1 关闭**：头长检查在解析和清空头部缓冲区之前进行，判定与分块无关。
- **Review 067 的 B1、B2**：没有发现直接回归，保持关闭。
- **BLOCKER**：无。**NON-BLOCKER**：没有新增确定的实现缺陷。**PRODUCT DECISION**：没有新增待决事项。
- **ACCEPTED RISK**：没有新增。
  - 只沿用 §71 的方案 B 局部边界，PR #31 的 CLI 预算调整按已批准的范围适用。
  - M-14、M-15、M-16 不是接受的风险，都未关闭。
- **验收**：审查认为，结合 Review 067、068 仍适用的证据，足以支持负责人对 WP2.4 本工作包范围作出验收决定。报告本身不宣布验收，不覆盖后续的检查执行、`status`/`gaps` 或 E2E 工作包，也不宣布 M2 整体验收。

**审查方的独立验证**（仓库外的隔离副本；Windows/amd64，Go 1.27.1，git 2.56.0.windows.2；WSL Ubuntu/amd64，普通用户，git 2.43.0）：

- **B3 反例**：Review 068 的反例原样重放，两个接口都是 exit 6、没有结果；每个场景都有健康与恢复后的对照。
- **同次读取调度**：新增四个场景（提交、根树、中间树、blob），证明判定来自已验证的读取流。
- **`propose`**：三类损坏 × 文本与 JSON。完整业务快照不变；输出与数据目录中都没有 Canary。
- **直接回归**：B1、B2 以及 Claim 与 `propose` 的直接回归。

**必须保留的本机失败与边界**（不计为通过）：

1. **首次完整的 `gitx` 包没有通过**：
   - 在默认 10 分钟的累计超时处中止，包记录 601.103 s；当时 `TestClassifyChecksTheStorage` 已运行约 10 s。超时的这次运行不计为通过。
   - 之后以相同的默认超时定向补跑必要子集，包括该存储分类测试、中断的子例、`Clean` 与 `TreeEntry`，结果通过（431.437 s）。
   - 根因没有独立证明，不定性为死锁、M-16 或资源竞争，也没有修改超时或测试门槛。
2. **首次 WSL 启动没有执行测试**：内联 shell 的参数传递使目录变量未按预期传入。这不是产品检出，也不计为通过。
3. **没有独立验证的部分**：
   - 本机没有 macOS，macOS 结果来自 CI。
   - 作者的真实 git 2.32.0 实验、全部变异与桩阶段的红灯、整套 M1、整个 WP2.3 的安全审查、正式发布包的 E2E，都没有独立重跑。
   - CI 中按条件跳过的测试不是通过：真实旧 git、Windows 的权限用例、Ubuntu 普通步骤中的 root 用例，以及 M-14 诊断。

**负责人决定**（2026-10-22）：

- 条件授权合并 PR #30。
- 合并核验与对应的 master push CI 全部成功之后，正式验收 WP2.4 本工作包范围。
- 随后授权 WP2.5 的规范准备、必要的隔离实验、文档提交与独立 PR；不授权 WP2.5 的生产实现、规范冻结或准备 PR 的合并。

**收尾**：只有 Review 070 的归档、本节与活跃状态的同步（AGENTS.md、README、实施计划）。代码、测试、Schema、迁移、依赖与工作流都保持受审内容。

**状态**：

- PR #30 技术审查通过，待合并核验。
- WP2.4 的验收在合并核验与 master push CI 全部成功之后生效，实际的合并证据与验收记录随下一阶段（WP2.5 规范准备）的 PR 入库。
- M2 未整体验收。M-14、M-15、M-16 保持未解决；方案 B 不变；Auto-fix / Auto-merge 保持关闭。

## 84．PR #30 的合并与核验、WP2.4 的验收，与 WP2.5 规范准备的授权

**收尾**（§83）：收尾 HEAD `ac851f231c943712e7059beb7c48136e888786c6`，在审查对象 `b8606b5` 之上有两个文档提交：

- `6b78cdd` 原样归档 Review 070；
- `ac851f2` 是 §83 与活跃状态的同步。

相对 `b8606b5` 只有 5 个文档文件（+317 / −5）。`internal`、`cmd`、`schemas`、`web`、`scripts`、`go.mod`、`go.sum`、`.golangci.yml` 与 `.github` 都与受审内容相同。

**收尾 HEAD 的 CI**：run 39737379188，`pull_request`，attempt 1，8/8 正式成功，没有重跑。

- 8 个 Job 都检出 `e2d98fb`（父提交 `8b45ef5`、`ac851f2`），其树 `fdd1a05` 等于 `git merge-tree --write-tree` 的结果。
- Windows：33 个包分成 32 + 1，每个包执行一次；`internal/cli` 用 20 分钟的预算，耗时 508 s。
- 三个平台：
  - 包集合相同，没有 FAIL；
  - PASS 与 SKIP 行与 Review 070 所审的 run 39723458143 逐行相同；
  - Ubuntu 的 root 步骤执行了 8 个跨用户测试。

**合并与核验**：

- 2026-10-22 以 merge commit 合并，用 `--match-head-commit ac851f2…` 锁定完整的最终 HEAD。合并提交为 `bcbb21c169020e4140aa12fc8adf53477fd6245a`，父提交 `8b45ef5`、`ac851f2`。
- master 包含 `ac851f2`；base 没有变化，合并树 `fdd1a05` 与最终 HEAD 相同，`git diff` 为空。
- master push CI 是 run 39739272853：`push`，attempt 1，8/8 正式成功，没有重跑。
  - 8 个 Job 都检出 `bcbb21c`；
  - Windows 33 个包，32 + 1，`internal/cli` 用 20 分钟的预算，耗时 317 s；
  - 三个平台的 PASS 与 SKIP 行与收尾 CI 逐行相同，没有 FAIL；跨用户测试 8 个通过；条件跳过与之前相同。
- 两次 CI 都没有用到 M-15 的重跑许可。
- **观测**（不改变任何维护事项的状态）：Ubuntu 上 `internal/cli` 在两次 CI 中分别为 530 s、509 s，在默认的每包 600 s 之内。M-16 的缓解只调整了 Windows。

**WP2.4 的验收**（产品负责人 2026-10-22 授权；上述合并核验与 master push CI 全部成功，验收于 2026-10-22 生效）：

- **对象**：PR #30 的合并提交 `bcbb21c169020e4140aa12fc8adf53477fd6245a`。
- **范围**（WP2.4 本工作包）：
  - Claim 的库与 CLI，迁移 0008；
  - 代码上下文、范围（`audit_scope`）与前置条件；
  - 引用检查、`claim-receipt.v1` 回执与事务；
  - 真实通道的验证；
  - 本 PR 中共用的 `propose` 文件引用路径的定向修复（Review 068 B3、N1）与回归。
- **依据**：
  - Review 066（规范 GO，冻结见 §79）；
  - Review 067、068、070 的实现审查（§80、§81、§83）；
  - 上述的合并与 CI 核验。
- **不在本次验收之内**，留在原工作包：`status` / `gaps` 中的展示（WP2.7）、检查执行（WP2.5）、M3 的会话关联、WP2.9 的完整 E2E。
- 不重新宣布 M1 的验收，也不宣布 M2 整体验收。
- **维护事项**：M-14、M-15、M-16 保持原状态，M-16 只是缓解已经生效；方案 B 的局部边界不变（§71）。

**负责人对 WP2.5 的授权**（2026-10-22）：

- 规范准备、必要的隔离实验、文档提交与独立的 PR。
- **不授权** WP2.5 的生产实现、规范冻结，也不授权合并准备 PR。
- M-15 的受控重跑许可不延伸到这个 PR。

**WP2.5 规范草案**（本节所在的 PR）：

- 文件是 [docs/wp2.5-check-runner.md](wp2.5-check-runner.md)，状态为草案，未经独立审查，未冻结。
- 草案区分已冻结的规则（data-model §4.11、§4.11.2、§7.4，TD-23 的读取机制，WP2.3 的 `Clean` 与方案 B）与没有获批的历史草案（TD-06 的建议默认，准备报告 §5.1 的 M2-Q1）。
- 草案列出 18 项待决事项（D1 ~ D18），每项附推荐、代价与最晚决定点。其中 D1 ~ D5 是 TD-06 / M2-Q1 与 EV-04，按 M2-Q10 必须在 WP2.5 实现开工前决定。
- 草案的实验只是作者在本机 Windows 与 WSL Ubuntu 上的观察：没有 macOS，不是 CI 证据，也不是独立复验。其中有几项改变了准备报告的推荐，见草案 §0：
  - Git for Windows 的 `sh` 入口与 PATH；
  - MSYS 的命令行规则；
  - `sh` 命令中的 CR。
- 本 PR 不改变冻结的规范、正式 Schema 或验收语义，不新增延期或接受的风险。

**状态**：

- WP2.4 已验收（本工作包范围）。
- WP2.5 的规范草案待独立审查与负责人决定；WP2.5 的实现、WP2.6 及其他工作包都未开始。
- M2 未整体验收。M-14、M-15、M-16 保持未解决；Auto-fix / Auto-merge 保持关闭。

## 85．Review 071：PR #32 的 WP2.5 准备规范 NO-GO（B1 ~ B4）与定向修订

**审查**：[Review 071](reviews/codex-review-071.md)，原样归档为 `85deecece60c057edf9ed64f04661371547ecc11`。源文件、归档与该提交 blob 的 SHA-256 均为 `03a075e65d3c5c04523a702424725ddd8181827632a05abb61371639c74b3ef0`，逐字节一致。

**审查对象**（不随修订而改变）：

- PR #32，HEAD `bbef1753e67383e0a971628830c9da36a43e603e`，base `bcbb21c169020e4140aa12fc8adf53477fd6245a`。
- 两个提交（`0d474d5`、`bbef175`），6 个文件。
- 审查依据的 CI 是 run 39741619564（attempt 1，8/8，检出 `a8d23f7`）。

**结论：作为准备文档，技术 NO-GO，暂不建议合并**：

- 有四项 BLOCKER（B1 ~ B4）。它们要求修正文档，不要求本 PR 提前实现 Check Runner，也不是因为负责人尚未选择 D1 ~ D18 而否定草案。CI 成功不替代文档的正确性。
- **§84 的 WP2.4 验收记录与状态同步成立**，没有扩大为 M2 整体验收。
- TD-06 / M2-Q1 与全部 WP2.5 规范尚不具备冻结条件；审查不授权实现、合并或冻结。
- 已验收的 WP2.4、WP2.3 与纯函数层的结论保持。**新的接受风险：无。**
- 另有 NON-BLOCKER N1 ~ N4（措辞与验证边界），以及对 D1 ~ D18、E1 ~ E9 的逐项裁定（区分真正的产品取舍与工程细化）。
- 审查方的独立实验，以及它自己首次运行时的失败，都已如实记入审查原文，不计为技术结论。首次运行的失败包括：沙箱中的启动失败；一个无效的 `cmd` 夹具，它在原仓库产生过一个空的 `one/` 目录，已核实为空后删除。

**逐项响应**（本节所在的提交修订了 [wp2.5-check-runner.md](wp2.5-check-runner.md)，第 2 版）。以下各项都是**已修订，待独立复审**，不宣布关闭：

- **B1**（`Clean` 的保证被概括得过强）：
  - 依据表把 WP2.3 §4.3（P1）与 §14.3 / §71（方案 B 的局部边界）分为两行，后者引用原文：不合作的同用户进程（包括普通误操作）干预私有临时命名空间时，`Clean` 可读到替代输入，**不能排除 false-clean**，这一边界不扩展到交付证据。
  - 新增 §7.1“`Clean` 的保证范围”，§7.4 补充说明；§15 不新增“绝不 false-clean”的断言；§0 的摘要同步。
  - 方案 B 没有修改。
- **B2**（批次标记被用于限制逐条执行）：
  - §11.2 重写：`is_target_snapshot` 只是给调用方的汇总，不进入 `runs(S)`、`V` 或推导；加入混合 `head_start` 的例子（`S = A`，第一条在 A 上合格通过、第二条在 B 上，汇总为 false，但第一条照常参与 `V`）。
  - D9 改为汇总含义的选项 (a)、(b)、(c)，第 1 版的表述标为已被否定。
  - §12、§13 新增第 7 项、§15 加入混合批次的用例。推导规则不变。
- **B3**（输出管道与进程结束没有形成失败闭环）：
  - **收尾协议**（新增 §8.5）：区分主进程退出、进程树结束的确认、输出排空与读取结束上下文四个事件；S0 ~ S7 的顺序在所有分支（正常、超时、取消、启动失败）中都相同，任何一步都不等排空完成之后才终止进程树；`T_kill`、`T_drain` 两个有限的时限；失败出口 F1（没有确认结束：不读取结束上下文，不走合格记录的路径，停止批次）与 F2（没有排空），以及 Windows 上建立 Job 失败时的回收。
  - **输出对象**（§9.1）：指出 `cli/root.go` 的 `termtext` 包装；检查转发使用 `cli.Execute` 的原始输出对象，分为继承路径与 Specmetry 自建 `os.Pipe` 的管道路径；禁止把非 `*os.File` 的 Writer 交给子进程。
  - **平台机制**（§8.2）：
    - Linux subreaper 写出归属、PID 身份（pidfd 与二次核对）、重复收养与回收、与 Go `Wait` 及 `gitx` 子进程协作的要求，并说明 X2、X9 只是正例，不是完整的保证；
    - macOS 没有任何实验；
    - Windows 的终止请求与结束确认分开。
  - **衔接**：§4.2、§7.1 改为只在确认结束之后读取结束上下文。
  - **新增的待决与门槛**：技术门槛 T1 ~ T6（§17.2）；新增待决 D19（进程树没有确认结束）、D20（排空没有完成）。D7 的保证调整仍待负责人选择，不写成已接受的风险。
- **B4**（EV-04 改写承诺了不能保证的行为）：
  - §6.1 改为按解释器的最终状态分层；§6.2 写明优先规则（取消与 F1、上下文变化、输入缺失、进程无法创建、超时先于最终状态），退出码 0 不一定是 `pass`。
  - §6.3 的方案 A 改为“按最终状态分类，不发现命令串内部的命令查找失败，也不发现被命令串掩盖的失败”，第 1 版的断言标为已被否定。
  - §14 的改写草案加入正例与对照：正例是“缺失命令之后成功、管道最后一项覆盖”最终以 0 结束，对照是“同一最终非零码不区分原因”。
  - D5 仍待批准；正式的 EV-04、Schema 与迁移都没有修改。
- **N1**：
  - §5.3：`/bin/sh` 是本产品的支持策略，不是 POSIX 的规定；自定义查找器显式忽略相对路径项，不以 `ErrDot` 代替。
  - §5.5：启动文件（`BASH_ENV`、`ENV`）如实说明，不承诺不读。
  - §8.1 与 E8：终端访问不作一般性承诺。
- **N2**：
  - §9.2：读取环境与记录环境分开。
  - §9.3：转发中的 canary 是有意的，存储要分别查验；基准中保存配置与运行时不捕获命令是两回事。
- **N3**：
  - §11.4：`error.v1` 的 `details` 限制；`internal` / 1 与 environment 区分；整份文档的校验失败不能回滚之前的执行；各阶段的故障分别注入。
  - §4.3：没有记录不能证明没有执行；不暗示安全的自动重试。
  - §4.1 第 4 步：worktree 路径的密钥扫描在任何检查启动之前。
  - §4 开头：“不写记录”不扩大为数据目录不变。
- **N4**：§5.5、§15、§16 的表述边界（字符串集不是穷尽的证明；测试计划不是已通过的测试；作者与审查方的证据分开）。
- **待决事项的重整**：按 Review 071 第五节分为三类，保留原 ID：
  - §17.1：既有契约的实现细化，15 项：D10、D12、D14 ~ D17、E1 ~ E9。它们不再列为产品决定。
  - §17.2：必须先解决的技术问题 T1 ~ T6。
  - §17.3：需要负责人选择的，14 项：D1 ~ D9、D11、D13、D18、D19、D20。D9、D13 是契约澄清，仍需负责人确认。

  被否定的表述另列追踪表。**本轮没有批准 D1 ~ D20 中的任何一项，也没有新的接受风险。**

**本轮的作者实验**（仓库外，有硬时限，结束后没有遗留进程；草案 §16 第 2 轮）：

- X8：自建管道下的收尾顺序，Windows 与 WSL；
- X9：WSL 上的 subreaper 加 pidfd；
- X10：解释器的最终状态，两个平台。

没有 macOS；不是 CI 证据。

**M-16 的补记**（本节所在的提交，`docs/m1-exit-evidence.md` §9 与实施计划）：在现有的 M-16 下补记 Ubuntu 上 `internal/cli` 的已知耗时，不新增重复的事项。

- 涉及的运行：
  - run 39704552634 为 526.706 s；
  - run 39720152759 为 307.266 s；
  - run 39721731294 为 324.686 s；
  - run 39723458143 为 367.256 s；
  - run 39737379188 为 530.246 s；
  - run 39739272853 为 508.720 s；
  - run 39741619564 为 445.355 s。
- 这些运行都没有超时，都在默认的每包 600 s 之内；Ubuntu 的测试步骤带 `-race`。
- M-16 的 Windows 缓解不覆盖 Ubuntu。根因未验证，M-16 保持未解决；不调整 Ubuntu 的超时或并行度。

**状态**：

- PR #32 已修订，待独立复审；没有合并，规范没有冻结，WP2.5 没有实现。
- §84 的 WP2.4 验收记录不变。
- M-14、M-15、M-16 保持未解决；Auto-fix / Auto-merge 保持关闭。

## 86．Review 072：PR #32 第 2 版的定向复审（B1、B2、B4 关闭，B3 残余 R1 ~ R3）与修订

**审查**：[Review 072](reviews/codex-review-072.md)，原样归档为 `cecf0cce49bd6fc02e03fca58d9584262a9baa91`。源文件、归档与该提交 blob 的 SHA-256 均为 `b858bf8632a56b01fa848e46e16fb26eef7e4b259a7780c6343743fda43cbbd2`，逐字节一致。

**审查对象**（不随修订而改变）：

- PR #32，HEAD `be3f4f6f908c8b789b0ac70dafc20ececb565813`，base `bcbb21c169020e4140aa12fc8adf53477fd6245a`。
- 相对上轮对象 `bbef175` 共 7 个文件。
- 审查依据的 CI 是 run 39761200142（attempt 1，8/8，检出 `c85add2`）。

**结论：准备文档技术 NO-GO，暂不建议合并**：

- **关闭**（只限准备文档的修订）：Review 071 的 B1（`Clean` 的方案 B 边界）、B2（批次汇总与逐条证据，D9 的汇总含义仍待确认）、B4（EV-04 的最终状态语义；正式 EV-04 没有修改，D5 没有批准）。
- **B3 部分修复，未关闭**，残余三项：
  - **B3-R1**：放弃复制 goroutine 之后，没有保证当前的错误呈现及后续调用的输出安全。审查的反例中，后续的 JSON 之后又出现了旧的检查输出。
  - **B3-R2**：Linux 的收养归属推理不成立；审查的反例中，既有子进程的新后代被收养。Unix 数字 PGID 的保护没有覆盖首次终止时组已消失的窗口。
  - **B3-R3**：推荐的取消与 F2 组合没有唯一的事实与退出结果。
- **NON-BLOCKER**：
  - N1：T1 ~ T6 应区分开工前的隔离原型与授权之后的生产验收；
  - N2：§85 的提交定位有一个小误差。
- **其他**：D1 ~ D20 的分类总体忠实于 Review 071，都没有批准。新的接受风险：无。§84、§85 与 M-16 的补记成立。规范冻结与实现开工都尚不具备条件，也没有授权。

**本轮响应**（下一个提交修订 [wp2.5-check-runner.md](wp2.5-check-runner.md) 为第 3 版；各项都是**已修订，待独立复审**，不宣布 B3 关闭）：

- **B1、B2、B4**：不重开。本轮的修改没有触及它们的论证，只核对没有回退。
- **B3-R1**（§8.5 的事件与 S6、§9.1 的目的端契约、D20）：
  - **撤回**：“关闭读端、放弃复制 goroutine、流程继续”与“没有无界的等待”的论证。
  - **区分四件事**：停止读取子进程的输出、已进入目的端 Write 的操作、复制任务真正结束、`Execute` 返回之后是否还可能写出。另新增 E-copy-end；E-drain 只指以 EOF 结束且写完的复制。
  - **目的端的候选契约**：
    - 继承文件句柄：没有复制任务；
    - Specmetry 控制的、Write 不阻塞的内存目的端：停止读取之后，复制在时限内返回并被等到，之后才写最后的文档、才返回，没有迟到写出，重复调用不累积；
    - 其他 Writer：在启动之前拒绝（`internal` / 1）。

    任意不可中断 Writer 的生命周期尚无可依赖的方案，列为未解决的门槛（T6）。这一限制交 D20 确认。
  - **继承路径的 F1**：残留的进程仍可能写出迟到的输出（JSON 模式只在 stderr），如实写明。
- **B3-R2**（§8.2、§8.5 的 E-main / S3 / S5、T1 ~ T3）：
  - **归属与身份分开**。
    - 删除“新收养的子进程都来自检查”的推理。归属改为**排他检查区间**的候选规则：基线必须为空（僵尸也算），区间内 Specmetry 进程没有其他子进程来源。强制这种互斥属于 T2。不满足时不启动。
    - pidfd 只证明身份，不证明归属。
  - **组身份**：组长在收尾完成之前不被回收（有时限地观察退出，以 `WNOWAIT` 取得状态）。从第一次 `kill(-pgid)` 到确认结束都在这一前提下进行，回收之后不再按数字发信号；不能保持时进入 F1。
  - **其他平台**：Windows 只补了“句柄持有，没有数字复用”的直接说明；macOS 写明没有任何证据。
- **B3-R3**（新增 §8.6 的决策表）：
  - **覆盖的组合**：取消与 F1 / F2、树没有确认结束与输出失败、结束上下文失败与输出失败、启动与 Job 管理失败与输出收尾、拟记录之后的数据库失败、校验失败与输出失败。
  - **每行写明**：用户代码是否运行、是否可能仍有进程、是否读取结束上下文、当前项是否尝试记录（结果与 qualifying）、后续项、退出码与输出通道。
  - **两条候选的优先关系**：
    - 是否写当前项：S1 > F1 > 取消 > D6 > F2 > 正常；
    - 退出码：1 > 6 > 3 > 10 > 0，`internal` 不被 environment 覆盖。

    依赖 D6、D8、D19、D20 的内容标为“若采用”。补表不把推荐变成正式规则。
- **N1**：§17.2 分为“冻结与实现授权之前（规格与隔离原型）”与“授权之后、本层验收之前（生产实现与测试）”两段，不要求先完成生产实现才能授权实现。
- **N2（§85 的勘误）**：§85 中“本节所在的提交修订了 wp2.5-check-runner.md，第 2 版”有误。实际上草案的第 2 版在 `afcfd3e` 中修订，§85 在其后的 `be3f4f6` 中引入。按“保留 §85”的要求，原文不改，在此更正。§85 记录的审查身份、NO-GO 与“待独立复审”的状态都不受影响。
- **D19 的补充**（Review 072 第五节）：主进程仍未结束时，`finished_at`、`duration_ms` 不是已知的实际结束时刻；“记录”是尝试持久化，数据库失败时同样没有记录；S1 中没有运行用户代码的管理失败不在 D19 之内。这些写入 D19，仍待选择。

**本轮的作者实验**（仓库外，有硬时限，草案 §16 第 3 轮）：

- **X11**（Windows 与 WSL）：旧设计复现了迟到输出的反例；候选方案中，复制在关闭读端之后立即返回，迟到输出没有进入目的端，JSON 是最后一项。
- **X12**（WSL）：组长不回收时组 ID 保持有效，回收之后为 ESRCH。
- **X13**（WSL）：复现收养反例，并验证新规则的拒绝。
  - **首次运行的夹具无效**：后代继承了管道，扫描等不到 EOF，被 120 s 的硬时限中止。
  - 它留下了一个自己的 `sleep 300`，核对命令行之后终止；修正夹具后重跑，之后没有遗留进程。
- **X14**（WSL）：对 pidfd 做有时限的 `poll`，不遗留等待任务。

没有 macOS；没有做强制复用 PID 或信号误投的实验；这些都不是 CI 证据。

**仍未解决的技术门槛**（草案 §17.2）：

- T1：收尾协议在三平台上的定稿与原型，macOS 尚缺；
- T2：排他区间的进程级强制方式（包括进程内并发调用）尚无原型；
- T3：macOS 没有任何实验；
- T4：Windows 失败分支的原型；
- T5：解释器适配器与安装布局；
- T6：任意 Writer 的生命周期尚无方案，按契约拒绝。

**状态**：

- PR #32 已修订，待独立复审；没有合并，规范没有冻结，WP2.5 没有实现。
- D1 ~ D20 都没有批准，没有新的接受风险。
- §84、§85 与历史审查原文保持不变。
- M-14、M-15、M-16 保持未解决；Auto-fix / Auto-merge 保持关闭。

## 87．Review 073：PR #32 第 3 版的定向复审（B3-R2 关闭，B3-R1、B3-R3 残余）与修订

**审查**：[Review 073](reviews/codex-review-073.md)，原样归档为 `fbe3d0efc652bae4aab8e1ca19d01687ff985c22`。源文件、归档与该提交 blob 的 SHA-256 均为 `9c36c05010667f92c6e2cb11350ef7966d8942e59e17161a961dd5a1b6d30106`，逐字节一致。

**审查对象**（不随修订而改变）：

- PR #32，HEAD `78a96dd7365f7c377c58f917a38b3f0397927225`，base `bcbb21c169020e4140aa12fc8adf53477fd6245a`。
- 相对上轮对象 `be3f4f6` 共 5 个文件。
- 审查依据的 CI 是 run 39779339263（attempt 1，8/8，检出 `8ed9c51`）。

**结论：准备文档技术 NO-GO，暂不建议合并**：

- **关闭**：Review 072 的 **B3-R2**（进程归属与组身份），只在准备文档层面。原来的错误归属推理和“先回收组长、再按数字 PGID 发信号”的顺序都已撤回；新机制仍是有前提的候选，不认可生产清扫器已经正确，也不批准 D7。Review 071 的 B1、B2、B4 保持关闭，没有直接回退。
- **B3-R1-S**（未关闭）：用尽排空预算之后，又在“剩余预算”内等复制结束，这不闭合；“没有确认复制结束即为 `internal` / 1”与“只有确认之后才返回”冲突；第 3 版对受支持目的端的完整保证不成立。
- **B3-R3-S**（未关闭）：持久化失败的行把 F1 之后的状态重置为“没有残留进程、已读取结束上下文”；笼统的数据库失败被写成“没有记录”，混淆了确定的回滚与提交结果不确定；`input_missing` 进入结束上下文的衔接不明确。
- **NON-BLOCKER**：
  - N1：真实 F1 之后宿主资源的归属要在 T1、T2 中定稿；
  - N2：§86 中“下一个提交修订……第 3 版”与实际顺序相反；
  - 受控目的端的同步、别名、复用规则属于 T6。
- **其他**：没有新的接受风险。规范冻结与实现开工都尚不具备条件，也没有授权。审查方本轮没有成功进行新的动态实验：本机 Go 的 shim 失败，WSL 返回 `E_ACCESSDENIED`；它如实报告了这一点，没有算作产品缺陷。

**本轮响应**（本节之前的提交修订了 [wp2.5-check-runner.md](wp2.5-check-runner.md)，第 4 版；各项都是**已修订，待独立复审**，不宣布 B3 关闭）：

- **B3-R1-S**（§8.5 S6 与时限、§8.3、§9.1、T1、T6、D20）：
  - **新的时间线**：D 阶段（`T_drain`，正常排空）→ 停止读取 → J 阶段（独立的 `T_join`，等复制返回）→ 封闭这一次的复制入口。F1 与 S1 的终止未确认时跳过 D 阶段。最长时间公式改为 `timeout_sec + T_kill + T_drain + T_join`。
  - **三种结果**：O1 完整（唯一的 E-drain）、O2 已结束但不完整、O3 未确认结束。
  - **O3 时能保证的与不能保证的**：
    - 入口已封闭，复制任务之后的写入都被拒绝。所以最后的文档、之后的错误呈现与同一目的端的后续调用都收不到迟到输出；
    - 不保证复制任务已经结束。它是不能写出的残留任务，处置方式在冻结前确定（T1、T6），不写成已接受的风险。
  - **撤回**：“剩余预算”内的等待、`internal` / 1 的出口，以及“复制任务在时限内确认结束、重复调用不累积”的保证。
  - **受控目的端的规则**（§9.1）：一个锁串行化全部写入；Write 不阻塞，有容量上限；复制入口与所有者入口分开；最后的文档在封闭之后整体写入（stdout 与 stderr 别名时同样）；别名、并发调用与复用的规则。JSON 模式中检查输出的目的端（stderr）与文档的目的端（stdout）分开说明。
- **B3-R3-S**（§8.6、§4.2、§7.5、§11.4、D19）：
  - **按维度叠加**：§8.6 改为六个维度（用户代码、进程树、结束上下文、输出、取消、持久化），后来的失败只更新它影响的维度，最后的报告带着全部维度。决策表分为三部分：A 收尾的组合、B 持久化的叠加、C 全部提交之后。
  - **持久化的三种结果**：
    - 确认未提交：BEGIN、写锁下的检查、写入或执行项校验失败；
    - 已提交；
    - 结果不确定：提交阶段返回错误，或结果没有被观察到。

    Runner 用“写事务的函数体是否已成功返回”区分它们（`withTx` 先运行函数体再提交），不改变存储层、重试或错误映射。
  - **Review 073 点名的组合**：
    - F1 加 D19(a) 之后写入失败、校验失败或提交不确定；
    - `input_missing` 或 `spawn_error` 之后写入失败（报告检查没有运行）；
    - F2 之后写入失败；
    - 已提交之后输出失败。

    这些都写成状态叠加，结果唯一。
  - **不启动进程时的衔接**：`input_missing`、`spawn_error` 没有本次的消费者，不经过 §8.5 的进程与输出步骤，直接读取结束上下文（§4.2 第 4 步的例外）。
  - **同步**：§11.4 与 D19 的说法相应修改。退出码仍只决定最终的错误类别，残留进程、未读的结束上下文、不完整的转发与记录状态都写进说明。
- **N1**：§8.5 F1 新增“宿主资源的归属”的候选：关闭本次自己打开的句柄与 pidfd，清除 subreaper 属性；已收养的进程仍是 Specmetry 的子进程，使同一进程中下一次检查的基线不为空而拒绝启动。同时写明不能宣称的部分；列为 T1、T2 冻结前的规格。
- **N2（§86 的勘误）**：
  - §86 中“下一个提交修订 wp2.5-check-runner.md 为第 3 版”有误：第 3 版已在其前的 `f2ca74a` 完成，§86 在 `78a96dd` 中引入。
  - 同一节 D19 补充中的“数据库失败时同样没有记录”按 B3-R3-S 更正为：提交之前失败时确认没有记录，提交阶段出错时记录不确定。

  按“保留历史记录”的要求，§86 原文不改，在此更正；§86 的审查身份、NO-GO 与“待独立复审”的状态都不受影响。
- **保持关闭**：B1、B2、B4、B3-R2 的章节（草案 §2、§7.1、§7.4、§8.2、§11.2、§14）与 `78a96dd` 逐字节相同，本轮没有触及。

**本轮的作者实验**（仓库外，有硬时限，草案 §16 第 4 轮）：X15，Windows 与 WSL。

- **A**：一个忽略关闭、永不结束的复制任务在 D、J 两个阶段都没有返回。封闭立即完成，之后目的端只多出所有者的文档；复制任务在下一次写入时被拒绝而返回。
- **B**：F1 情形，stdout 与 stderr 别名到同一个目的端。封闭之后写入的文档是最后一项，迟到输出没有进入。

阻塞在 Read 中、永不返回的任务没有模拟；没有 macOS；没有失败或无效的运行；实验结束后没有遗留进程；这些都不是 CI 证据。

**仍未解决的技术门槛**（草案 §17.2）：

- T1：收尾协议与三个时限的定稿；O3 残留任务与 F1 宿主资源的处置；macOS 的原型；
- T2：排他区间的进程级强制、PID 保留的前提；
- T3：macOS 没有实验；
- T4：Windows 失败分支的原型；
- T5：解释器适配器；
- T6：受控目的端的规则定稿，任意 Writer 尚无方案。

**状态**：

- PR #32 已修订，待独立复审；没有合并，规范没有冻结，WP2.5 没有实现。
- D1 ~ D20 都没有批准，没有新的接受风险。
- §84 ~ §86 与历史审查原文保持不变。
- M-14、M-15、M-16 保持未解决；Auto-fix / Auto-merge 保持关闭。

## 88．Review 074：PR #32 准备文档技术审查通过；收尾、条件合并与原型授权

**审查**：[Review 074](reviews/codex-review-074.md)，原样归档为 `a8ad5386f7be269e38e6d342bd5d72b7f959000f`。源文件、暂存、归档与该提交 blob 的 SHA-256 均为 `400267575d042d399f12f1105a2860329e894db34c4070463febe72e30ba7cd4`，逐字节一致。

**审查对象**（不随收尾而改变）：

- PR #32，HEAD `a53f2d38b133e567fc8f2ddc88831e56767c764f`，base `bcbb21c169020e4140aa12fc8adf53477fd6245a`。
- 相对上轮对象 `78a96dd` 共 5 个文件。
- 审查依据的 CI 是 run 39797220419（attempt 1，8/8，检出 `237df12`）。

**结论：准备文档技术审查通过（GO），可以作为准备文档由负责人决定合并**：

- **关闭（准备文档层面）**：
  - **B3-R1-S**：独立的 `T_join` 预算与封闭入口，使“不能再写出”与“任务确已结束”分开；
  - **B3-R3-S**：持久化结果不再重置收尾状态，提交阶段的错误按不确定处理；现有的 `WriteTx` 接口足以支持保守的阶段分类。

  B3 在准备文档层面整体关闭。
- **保持关闭**：B1、B2、B4、B3-R2，对应论证没有变化，没有直接回退。
- **BLOCKER**：无。
- **NON-BLOCKER**：
  - N1：§11.3、§11.4 中，通用的执行项校验失败仍写作“已执行”，`input_missing`、`spawn_error` 时并不准确；
  - N2：§9.1 中“最后的文档完整”的概括，应限定为“所有者写入成功时，文档字节不被复制输出插入”，并且不扩大为“别名流只有一份 JSON”。
- **不由本报告给出**：
  - 生产 Runner 的正确性证明；
  - O3、F1 或产品选项的批准；
  - 规范冻结、实现授权、新的接受风险、工作包或 M2 的验收。
- **冻结之前仍缺**：14 项产品选择、T1 ~ T6 冻结之前的规格与必要原型、获批之后的正式同步，以及负责人的明确授权。
- **审查方的验证边界**：
  - 一个仓库外的 PowerShell 7.6.7 / .NET 最小门禁模型：同一临界区内的封闭检查与追加、所有者入口分开、闸门模拟、20 s 硬时限。它覆盖了封闭之后的拒绝、临界区内与排队中的写入、别名、容量耗尽与所有者写入失败，以及复用时的新入口；首次运行与复核都成功，没有遗留进程。夹具的 SHA-256 为 `0c00bd8082de1d3425e1209ed07f95de9be5366d12a63ee564436cf4ecea8f1b`，保留在仓库外。
  - 这个模型不证明 Go / Windows 管道的中断、macOS、任意 Writer、生产类型的接线、真实进程树或数据库故障的恢复。
  - 审查方没有重跑作者的 X15、CI 或既有实验。

**负责人决定**（2026-10-23）：

- 准备文档收尾之后，条件授权以 merge commit 合并 PR #32。条件：
  - 收尾增量只有归档、非语义措辞修订与状态记录；
  - 最终 HEAD 的 CI 8/8 正式成功，实际检出适用；
  - 合并之前再核对 HEAD 与 base；
  - Auto-fix / Auto-merge 关闭。
- 合并核验与 master push CI 全部成功之后，授权从已核验的 master 另开 PR，开展冻结之前的 test-only 机制原型（T1 ~ T6），不合并，等待独立审查。
- **不授权**：产品选项的批准、整体规范冻结、生产 Check Runner、迁移 0009、CLI 接线或 WP2.5 验收。

**收尾**（相对审查对象 `a53f2d3` 只有归档、非语义的措辞与状态记录）：

- **N1 的收口**：§11.3 与 §11.4 的执行项校验失败改为“这一次确认未提交”。用户代码是否运行过沿用 U 的状态（`input_missing`、`spawn_error` 时没有运行）；F1 的可能残留、未读的结束上下文与 O 的状态也沿用（与 §8.6 的 A、B 表一致）。
- **N2 的收口**：§9.1 的概括与“最后的文档”一条限定为所有者写入成功时，说明加锁不保证写入成功（失败按 §8.6 C2 与 §11.4 处理），也不表示别名流只有一份 JSON。
- **状态记录**：草案头部、AGENTS.md、实施计划写为“准备文档技术审查通过”，同时写明这不是规范冻结、机制没有全部验证、生产实现没有授权、工作包没有验收。

**保持不变**：

- 14 项产品选择的待批准状态；
- T1 ~ T6 冻结之前的缺项；
- 既有的观测级与方案 B 边界；
- M-14、M-15、M-16 的未解决状态；
- 既有的决定、验收结论与历史审查原文（§84 ~ §87 没有改动）。

合并结果与 CI 回执写入 PR #32 的描述，不为补记结果直接向 master 追加提交；它们之后随原型 PR 一并记入。

**状态**：

- PR #32：准备文档技术审查通过，收尾之后待按条件合并。
- 规范没有冻结，WP2.5 没有实现或验收，M2 没有整体验收。
- Auto-fix / Auto-merge 保持关闭。

## 89．PR #32 的合并核验；WP2.5 冻结之前的 test-only 机制原型（PR #33，待独立审查）

**PR #32 的合并**（按 §88 的条件；§88 约定合并结果随原型 PR 一并记入，没有为此直接向 master 追加提交）：

- **最终 HEAD** `f54850ed0c53af4ab103dcb9c5d3d5bfeb4cf707`，base `bcbb21c169020e4140aa12fc8adf53477fd6245a`。收尾增量相对审查对象 `a53f2d3` 为 5 个文件，只有归档、非语义措辞与状态记录；代码、测试、Schema、迁移、依赖与 workflow 与 `a53f2d3` 相同。
- **PR CI**：run 39808598954，`pull_request`，attempt 1，completed/success，8/8，没有重跑。每个 Job 检出 `11abea51d9bb16d63110707fd7408fdbf38606d5`（`f54850e` 合并进 `bcbb21c`），它的树 `3d77f4f` 等于 `git merge-tree --write-tree origin/master f54850e` 与 `f54850e` 自身的树。PASS / SKIP 行与上一轮逐行相同。
- **合并之前**再次核对：远端为 `XKHoshizora/specmetry`，HEAD 与 base 未变，可合并，Auto-fix / Auto-merge 关闭。
- **合并**：`gh pr merge 32 --merge --match-head-commit f54850ed0c53af4ab103dcb9c5d3d5bfeb4cf707`，得到 **`ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`**（2026-10-22T16:07:59Z），父提交 `bcbb21c` 与 `f54850e`；没有强推，没有空提交。master 包含 `f54850e`；base 没有移动，合并树等于 `f54850e` 的树，`git diff f54850e ecb83ba` 为空。
- **master push CI**：run 39811328918，`push`，attempt 1，completed/success，8/8，每个 Job 检出 `ecb83ba` 本身；PASS / SKIP 行与 PR CI 逐行相同；Ubuntu 的 root 步骤运行了 8 个 `TestCrossUser*`；Windows 的 `internal/cli` 以 `-timeout 20m` 用时 456 s，Ubuntu 的 `internal/cli` 用时 521 s（低于默认的 600 s；M-16 的观察）。
- 合并结果与 CI 回执也已写入 PR #32 的描述。合并准备文档**不是**规范冻结、实现授权或验收。

**原型**（按 §88 的授权：合并核验与 master push CI 全部成功之后，从已核验的 master 另开 PR）：

- **PR #33**，分支 `proto/wp2.5-prototype`，起点 `ecb83ba`。不合并，待独立审查。
- **内容**：
  - `prototype/wp25/`：独立的 Go 模块，只有测试文件，依赖只有 `golang.org/x/sys` v0.49.0。Specmetry 的模块不包含它、无法导入它，根目录的 `./...` 不进入它。
  - `.github/workflows/wp25-prototype.yml`：只在原型分支、并且原型代码或该 workflow 改动时运行的三平台诊断；没有手动触发，不取消运行，文档改动不触发。
  - [wp2.5-prototype.md](wp2.5-prototype.md)：方法、有效性规则、逐平台结果、候选规格 C-1 ~ C-18、产品决定单与仍缺的部分。
  - 本节，以及 AGENTS.md、实施计划与草案头部的状态记录。
- **没有改动**：生产代码、`schemas/`、迁移、SQLite 的保护、重试与错误映射、`ci.yml`（Job、名称、预算与保护步骤）、已冻结的文档与历史审查原文；没有 `check run` 命令。
- **证据**（原型代码固定为 `475414228b995c8dbabeb35d78ce6c1dd9bd9a95`）：诊断 workflow run 39816938304 在 Ubuntu（带 `-race`）38/38、macOS 26.6.2（arm64）27/27、Windows Server 2025 29/29 都是 `confirmed`；上一轮 run 39815658463（对 `bcf080f`）同样全部 `confirmed`。作者本机 Windows 11 29/29，WSL2 37 个 `confirmed` 与 1 个 `unsupported`（没有 pwsh）。作者本地草稿运行中发现并修正的 5 个原型或测试基座问题如实记在原型文档 §2.5。
- **按门槛**（细节与边界见原型文档）：
  - **T1 / T6**：管道路径的时间线与封闭入口在三平台成立；`os.Pipe` 读端上的停止读取在 0.4 ms 以内返回，没有出现 O3。O3 只在阻塞模式的读端上出现（反例），这时“不再写出”从封闭起成立，“资源已结束”要等 Read 返回。容量耗尽时应继续读并丢弃（反例：否则检查被迫超时）。O3 之后的后续调用推荐“拒绝”（候选，C-4）。
  - **T2**（Linux）：进程级闸门把排他区间变成可强制的规则，覆盖 git 与进程内并发的检查；重复收养、pidfd 身份、拒绝路径与 F1 之后的资源都已执行。旁路启动与其他回收者是反例，需要静态规则（C-9）。
  - **T3**（真实 macOS）：kqueue 的有时限、不回收的退出观察，`waitid(WNOWAIT)` 可用，组身份规则成立；`setsid` 的后代无法发现（边界保留）；只剩僵尸组长时 `kill(-pgid, 0)` 返回 EPERM（新发现，C-12）。
  - **T4**（Windows）：启动链每一步的失败都不恢复、确认结束、句柄有主、用户代码没有运行；终止或查询失败时不确认结束。测试负载下没有观察到“请求之后仍有活动进程”的窗口，这一区别只由注入证明。
  - **T5**：所测解释器上命令文本 23/23 原样到达，MSVCRT 引号交给 MSYS 的反例复现，`cmd` 的多行只执行第一行。
- **候选规格 C-1 ~ C-18** 都待审查，不改变产品行为；**14 项产品选择**（D1 ~ D9、D11、D13、D18 ~ D20）仍待负责人决定，原型文档 §10 给出按行为分组的推荐、代价与本轮证据，都**没有批准**。

**保持不变**：

- 规范草案仍是第 4 版，未冻结；T1 ~ T6 的规格待审查之后定稿；
- 观测级与方案 B 的边界；M-14、M-15、M-16 的未解决状态；
- 既有的决定、验收结论与历史审查原文（§84 ~ §88 没有改动）。

**不授权**：原型 PR 的合并、产品选项的批准、规范冻结、生产 Check Runner、迁移 0009、CLI 接线、WP2.5 或 M2 的验收。

**状态**：

- PR #32：已合并为 `ecb83ba`，合并核验与 master push CI 正式成功。
- PR #33：原型已提交，三平台诊断全部 `confirmed`，**待独立审查**，不合并。
- Auto-fix / Auto-merge 保持关闭。

## 90．Review 075：PR #33 原型的独立审查（NO-GO，B1）与定向修复

**审查**：[Review 075](reviews/codex-review-075.md)，原样归档为 `1ee51059fb0f5b7d6f1e0e711af39fa9fec5f1e2`。源文件、暂存、该提交 blob 的 SHA-256 均为 `92a7b398352a41b01b277c83aad32eca7c703cd878e863297cf076f86ece584a`，逐字节一致。

**审查对象**：

- PR #33，HEAD `932b4a80852210a9fcc04cb0baf79aa4de04a7cf`，base `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`；原型代码与 workflow 固定为 `475414228b995c8dbabeb35d78ce6c1dd9bd9a95`。
- 28 个文件，+4588/−4。

**结论**：

| 层次 | 裁定 |
|---|---|
| A．PR #33 | **技术 NO-GO**：B1，实验的有效性门禁没有兑现它声明的拒绝规则 |
| B．冻结之前的技术门槛 | T1 ~ T6 得到程度不同的**局部支持**，尚不能整体宣布完成 |
| C．产品决定 | 14 项（D1 ~ D9、D11、D13、D18 ~ D20）仍未批准；C-4 与 D20 的关系应明确 |
| 新增接受风险 | 无；M-14、M-15、M-16 保持未解决 |

- **B1**（审查第三节）：
  - 报告的读取把畸形、无法解析、未知事件、超长行、读取失败都静默跳过或当作空，最终门禁只看成功解析出来的 `limit`；
  - 场景专属的负载（例如 Windows fanout 的 30 个后代、31 个活动进程）没有成为判定条件，通用的 `work > 0` 可以由主进程单独满足；
  - Linux 的 `/proc` 读取错误被丢弃，`pidProc.ended` 把一切读取失败当作“已结束”。
- **审查的两个反例**（仓库外隔离副本、真实测试二进制）：
  - 辅助进程的报告多写一行 `limit not-a-pid`，原有的 silent-holder 场景仍然 confirmed、exit 0；
  - fanout 改为不建后代，原有的五轮 Job 场景仍然 confirmed、exit 0，work=5。
- **NON-BLOCKER**：
  - N1：C-4（O3 之后拒绝同进程的后续检查）不能脱离 D20 称为“不是产品选择”；
  - N2：Windows 的观察句柄没有显式关闭，在已关闭句柄上调用 `GetHandleInformation` 的检查在审查方环境中异常退出（`0xc0000008`）；
  - N3：辅助进程内部的 60 s 时限在挂起创建的区间无效，应与父端的有界等待分开说明；§12 的 Ubuntu 内核版本与 §3.2 不一致。
- **审查方的验证边界**：本机 Windows 的定向运行与仓库外校准，Linux 与 macOS 只做日志与静态核查；不把 CI 的正式成功等同于门禁有效。

**负责人授权**（本轮）：原样归档 Review 075；修复原型测试基座及直接受影响的场景；增加有效性门禁的正反例；更新结果文档、处理记录与 PR 描述；正常推送修复，运行三平台原型诊断与常规 CI。**不授权**生产实现、产品决定、规范冻结、合并或验收。

**回归先行**（`5140588e4cbb54eb224dc8131b113856ed43c781`，与修复分开）：

- `gate_test.go` 的 `TestGate` 在新的进程中重新运行一个真实场景，经 `WP25_INJECT` 注入一种故障，检查真实诊断给出的结论与退出码：损坏的运行必须 invalid 并失败，正对照必须 confirmed 并通过。每一项打印一行 `WP25GATE`，与场景分开计数，**不计为机制的 confirmed**。
- 注入：审查的两个反例；记录缺失、未知种类、重复、截断、超长、读取失败；辅助进程启动子进程后失败；fanout 只建三分之一；进程链少一层；结束无法确认；进程状态读不到；`/proc` 读不到。
- **旧门禁上的结果**：
  - 作者本机：Windows 16 项中 10 项失败，WSL 17 项中 10 项失败；
  - 三平台诊断 run 39830995366（push，attempt 1，completed/failure，每个 Job 检出 `5140588`）：macOS 13 项中 9 项失败，Windows 16 项中 10 项失败，Ubuntu 17 项中 10 项失败；
  - 失败的每一项，内层的真实场景都是 confirmed、exit 0（假通过）。同一运行中普通场景仍全部 confirmed（27、29、38），正对照与“结束无法确认”通过；`chain-short` 在旧代码上已是 invalid，原因是等待超时，不是负载门禁。
- 同一提交的常规 CI（run 39831007212）completed/success；它不运行原型。

**修复**（`a793cedfcef5596bad05612df7f4165490319f28`）：

- **报告协议**：每个事件一整行 `<kind> <pid>`，四种 kind，规范的正整数 PID，行长与文件大小有界。读取失败、畸形、未知、重复、超长、截断或 `limit` 都使场景 invalid，不跳过。辅助进程仍可能写入时，未结束的最后一行留到下一次读；写者结束之后，每个报告必须恰好符合场景声明的规格。
- **负载门禁**：场景声明它需要的角色、数量与阶段（辅助进程的成功退出码、报告的每个后代、停止时阻塞的复制任务、Windows fanout 的 30 个后代与 31 个活动进程）。负载没有建立 → invalid；完整负载下结果不同 → refuted。
- **辅助进程**：写不了记录、等不到子进程的记录、写不出输出，各用自己的退出码，由场景核对。
- **归属**：报告的 PID 只有核验为本次检查的进程（Unix：主进程的后代或在其进程组中；Windows：在其 Job 中）之后才可操作。父辅助进程等待 stdin 的释放，所以核验时它的子进程仍是它的子进程。
- **观察与清理**：进程状态、进程列表、组成员读不到时是“未知”，不是“不存在”或“空”：清理不算核验，观察为 invalid，基线或清扫停止。提前停止的场景在组长仍被持有时结束自己的组。Windows 的观察句柄在清理之后关闭；Runner 自己的句柄以关闭调用的结果核对，不再在已关闭的句柄上调用 API（N2）。
- **新增两项回归**：规格之外的合法记录（只有终判能拒绝）；报告给出不属于检查进程树的 PID（只有归属核验能拒绝；用的是场景自己启动的兄弟进程）。
- **作者本机的修复后结果**：Windows 29 个场景全部 confirmed，门禁回归 18/18 通过；WSL 37 个 confirmed、1 个 unsupported（没有 pwsh），门禁回归 19/19 通过。
- **隔离变异**（仓库外的副本，每次只撤回一道门禁）：宽松解析、撤负载门禁并撤终判、撤终判、未知当作结束（Unix）、读不到当作空组（Linux）、撤归属核验、撤辅助进程退出码门禁，各自的回归都失败，正对照仍通过；只撤 fanout 负载门禁时终判仍拒绝（两道门禁互为后备）。M7 的第一次变异写错、编译失败，这次运行不算，如实保留。

**修复后的三平台诊断**：run 39835302375（push，attempt 1，completed/success，没有重跑；每个 Job 检出 `a793ced` 本身）：场景 Ubuntu（内核 6.17，带 `-race`）38/38、macOS 26.6.2 arm64 27/27、Windows Server 2025 29/29 confirmed，没有 refuted、invalid 或 unsupported；门禁回归 19/19、15/15、18/18 通过；唯一的 SKIP 是只供内层运行的 `TestInnerForeignPID`。各平台的场景 ID 集合与 run 39816938304 相同（并集 53）。常规 CI 不运行原型，另行记录在 PR 描述中。

**受影响的结论**（细节见原型文档 §2.6）：

- 旧的两轮诊断（run 39815658463、39816938304）与 PR CI（run 39818439605）**保留为历史观测**；它们的 confirmed 汇总不再作为证据，也不是通过独立审查的证据。
- 依赖报告、负载或 Unix 清理核验的场景，旧结论的有效性由修复后的运行重新给出：T1 的 F1、silent-holder；组身份的两个场景；T2 中经 `runCheck` 的场景与 subreaper；Windows 的 fanout。
- 不依赖报告的场景（受控目的端、E-main、waitid、Windows 失败链、适配器、SIGCHLD 等）的原始观察仍可引用，但同样以修复后的运行为当前证据。

**C-4 与 D20**（N1）：O3 之后登记未结束的复制任务是工程机制；**拒绝同进程中的后续检查并返回 6** 则依赖调用方的支持契约与 D20 的选择。D20(a) 只决定当前批次停止，不决定宿主中新的调用；D20(b) 若允许继续，C-4 的拒绝须明确优先关系；拒绝的代价可能是长驻宿主之后都不能执行检查，不能由“样本中没有出现 O3”推为几乎没有代价。原型文档撤回“候选规格都不改变产品行为”的表述，把 C-4 归入 D20 的待决范围；本轮**不批准**任何选项。

**T1 ~ T6**：按 Review 075 的裁定保持**局部支持**，不因门禁修好而宣布完成。冻结之前仍缺的规格与原型、授权之后的生产接入与测试，分开列在原型文档 §11、§12。

**保持不变**：14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；M-14、M-15、M-16 未解决；§84 ~ §89 与历史审查原文没有改动。

**状态**：B1 **已修复，待独立复审**；不宣布 BLOCKER 关闭或 GO。PR #33 保持未合并，Auto-fix / Auto-merge 关闭。

## 91．Review 076：PR #33 有效性门禁的定向复审（NO-GO，B1 残余两项）与修复

**审查**：[Review 076](reviews/codex-review-076.md)，原样归档为 `50156ece8e86dfef23fd6d7a9562ab6bc8b28db0`。源文件、暂存、该提交 blob 的 SHA-256 均为 `5d5e58b5609e365de78038dd0a20411826fbe3bb1942c641ef243a57b95f11a6`，逐字节一致。

**审查对象**：PR #33，HEAD `3ddd14fb1d478e333ad922310f9bcfc2b70033d9`，base `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`；修复后的原型代码 `a793cedfcef5596bad05612df7f4165490319f28`。19 个文件，+1692/−441。

**结论**：**技术 NO-GO，Review 075 B1 尚不能整体关闭**。

- **已修复**（审查确认）：Review 075 的两个反例（畸形的 `limit not-a-pid`、零后代的 fanout）；部分 fanout、缺失与损坏的报告、规格之外的记录、父辅助进程的异常退出、归属拒绝；持续失败的观察。健康对照与合法的反例、注入、拒绝场景仍为 confirmed。
- **B1-R1**：一次进程状态观察失败可以被后来的成功观察覆盖，最终仍 confirmed、exit 0。审查方在 Windows 隔离副本中以一次性故障注入复现，外层 `TestGate/control.silent-holder` 同样通过。
- **B1-R2**：Linux 不完整进程链的提前拒绝，没有收尾已经启动、脱离原组的部分链（`runCheck` 在全部记录到齐之后才登记；拒绝时三个链节点没有登记、没有被清扫，离开区间时 subreaper 已清除，只剩辅助进程自己的 60 s 定时器）。审查方的证据是源码级的，没有在 Linux 上动态运行。
- **Review 075 N1 ~ N3**：处理成立。
- **N4**（NON-BLOCKER）：持有者写 `epipe` 记录失败时，F1 场景记为 refuted 而不是 invalid（工具故障被记为机制被推翻）；没有形成假通过。
- 不批准产品选择、不冻结规范、不授权生产实现；没有新的接受风险；M-14、M-15、M-16 保持未解决。

**本轮授权**：原样归档 Review 076；修复共享的观察判定、拒绝路径的清理及直接关联的调用者；增加真实调用链的回归、健康对照与必要的隔离变异；更新证据文档、处理记录、状态与 PR 描述；推送修复，运行原型诊断与常规 CI。**不授权**产品决定、规范冻结、生产实现、合并或验收。

**回归先行**（`1df33dff77ea252c4cae2535f82dd197ffd58697`，只有观察工具与回归，门禁未改）：

- **外层的独立核验**：外层门禁检查给内层运行一个见证文件；每个辅助进程启动时写下自己的 PID 与自己读到的启动时间，注入的故障到达注入点时也留下一行（内层测试进程中为 `WP25INJECT` 行）。内层运行结束之后，外层按见证的身份核对每个已启动的辅助进程都已结束（仍存活的报告出来，再由外层按见证的身份结束：这是本轮自己的夹具）；并要求每个已启动的辅助进程或在内层的已登记列表中，或被内层申报为未入账。
- **新增回归**：`observe.fail-once`（进程结束的观察第一次失败，之后都成功：审查原文的反例，三个平台都在 `pidProc.ended` 注入，每个被观察的进程只注入一次）；`load.chain-corrupt`（短链的报告读不懂）；`load.chain-short` 要求看到注入。
- **旧门禁上的结果**：
  - 作者本机 Windows：`observe.fail-once` 注入命中 3 次，内层 confirmed、exit 0；7 项报告损坏的拒绝与部分 fanout 有已启动而未入账的进程；
  - 作者本机 WSL：同上，另外 `load.chain-short` 在内层结束后留下 3 个存活的链节点（PID 986、992、998），`load.chain-corrupt` 留下 4 个；
  - 三平台诊断 run 39886679862（push，attempt 1，completed/failure，每个 Job 检出 `1df33df`）：Ubuntu 21 项中 10 项失败（短链留下 3 个、损坏的链留下 2 个存活节点；`observe.fail-once` 为 confirmed、exit 0；7 项报告损坏未入账），macOS 16 项中 8 项失败，Windows 19 项中 9 项失败；普通场景仍全部 confirmed（38、27、29）。
  - 留下的节点都由外层按见证的身份结束；没有操作任何未知进程。

**修复**（`8fea6c607b8e760ddd3428d7d674739e2d6b8197`）：

- **B1-R1**：
  - 进程的第一次观察失败被保存（`observeErr`），之后的成功观察不清除它；清理仍然结束并回收这个进程，但这次运行为 invalid。清理的完成与运行的有效性分开记录。
  - 同一问题的其他观察点：等待主进程退出、组长的状态、场景结束时组或 Job 不能确认为空、清扫中读不到的列表或状态，都以 `flag` 记入统一终判，之后的任何结果都不能恢复。
- **B1-R2**：
  - 报告的进程在它的记录到达时即核验并登记（`awaitOwned`），不再等全部到齐；拒绝时无法登记的进程申报为未入账，清理因此不称“已核验”。报告中的 PID 仍然只有经归属核验之后才可操作。
  - 在排他区间内被拒绝的检查，在离开区间之前先结束它的进程组，再清扫区间内被收养的全部进程（`abort`）：按区间的规则它们都属于这次检查，不论是否报告过；所以报告读不懂的链也被结束。
  - Linux 上，场景结束时本进程留下新的子进程（存活或未回收的僵尸）即为 invalid。
  - Windows 的提前结束在关闭 Job 句柄之前先终止并确认 Job 为空。
- **作者本机的修复后结果**：Windows 29 个场景全部 confirmed，门禁回归 19/19 通过；WSL 37 个 confirmed、1 个 unsupported（没有 pwsh），门禁回归 21/21 通过；短链的 3 个节点全部登记、另申报 1 个未入账，损坏的链 4 个节点在区间内被清扫、申报 4 个未入账，外层核验都没有存活的辅助进程。
- **隔离变异**（作者本机）：撤回“保存失败的观察”（M8）在 Windows 与 WSL 上都使 `observe.fail-once` 回到 confirmed、exit 0；撤回逐条登记（M9）使短链的 3 个节点未入账；撤回区间内的收尾（M10）使损坏的链留下 4 个存活节点（短链仍由逐条登记与清理兜住：两层互为后备）；撤回未入账的申报（M12）使 `report.missing` 与 `load.chain-corrupt` 失败。健康对照在每个变异中都通过。

**修复后的三平台诊断**：run 39887528435（push，attempt 1，completed/success，没有重跑；每个 Job 检出 `8fea6c6` 本身）：场景 Ubuntu（内核 6.17，带 `-race`，没有数据竞争的报告）38/38、macOS 26.6.2 arm64 27/27、Windows Server 2025 29/29 confirmed，没有 refuted、invalid 或 unsupported；门禁回归 21/21、16/16、19/19 通过（`observe.fail-once` 三平台注入都命中 3 次，内层 invalid、exit 1；Ubuntu 的 `load.chain-short` 已启动的 3 个节点全部登记、申报 1 个未入账，`load.chain-corrupt` 申报 4 个未入账，两者内层结束后都没有存活的辅助进程）；唯一的 SKIP 是只供内层运行的 `TestInnerForeignPID`。各平台的场景 ID 集合与 run 39835302375 相同（并集 53）。回归先行提交的常规 CI（run 39886688329）被 ci.yml 自己的并发设置取消（completed/cancelled）；常规 CI 不运行原型，另行记录在 PR 描述中。

**N4**：本轮没有修复（不在本轮的授权范围内）；登记在原型文档 §13，建议在之后的轮次中把持有者的报告失败作为独立的工具故障状态传给场景，并增加分类回归。

**受影响的结论**（细节见原型文档 §2.8、§3.4）：

- run 39835302375（`a793ced`）的诊断与此前的常规 CI 保留为历史观测；它的门禁回归在当时的范围内仍然成立，但不能证明“一次观察失败必使运行无效”或“拒绝路径已收尾全部已启动的进程”。
- 依赖进程结束观察的场景（含经 `pidProc` 清理的全部场景）与经 `runCheck` 的 T2 场景，以修复后的运行为当前证据。
- T1 ~ T6 仍只是局部支持；N1 ~ N3 保持审查结论；C-4 与 D20 的待决关系不变。

**保持不变**：14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；§84 ~ §90 与历史审查原文没有改动。

**状态**：B1 的两项残余**已修复，待独立复审**；不宣布 B1 关闭或 GO。PR #33 保持未合并，Auto-fix / Auto-merge 关闭。

## 92．Review 077：PR #33 的观察错误与拒绝收尾定向复审（NO-GO，B1-R1-S、B1-R2-W）与修复

**审查**：[Review 077](reviews/codex-review-077.md)，原样归档为 `d3c9cb5`（`docs: archive Codex review 077 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `c3d4b31072f44d39e06f06e07305b77ad7fcf12b805b59432582efb8abb3395b`，逐字节一致。

**审查对象**：PR #33，HEAD `acf699b1df18343bf97e734fdb2a3af379637fbf`，base `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`；修复代码 `8fea6c607b8e760ddd3428d7d674739e2d6b8197`。22 个文件，+1104/−100。

**结论**：**技术 NO-GO，Review 075 B1 尚不能整体关闭**。

- **已确认的修复**：Review 076 的两个精确反例。`pidProc.ended` 的首次观察错误不再被清理成功覆盖；Linux 短链与损坏链在排他区间内逐条登记、终止原组并清扫收养的节点，不再必然绕过未报告的节点。
- **B1-R1-S**：Linux 清扫在 pidfd 绑定之后的那次状态读取，把“未知”记为身份不符，下一轮成功即可 confirmed；场景起点的子进程枚举失败返回空列表。审查给出的是具体调用方的源码级反例，审查方没有在 Linux 上动态运行。
- **B1-R2-W**：外层见证不是 fail-closed。写入失败留下空见证，外层仍 PASS；`unknown` 身份被当作原进程已结束；未知 kind、重复、无换行的尾行被接受；创建之后、写见证之前被结束的进程没有记录。前两点审查方在 Windows 隔离副本中独立复现。
- **N5**（NON-BLOCKER）：`missing` 是声明缺额，不是已启动对象的账户；`Unowned <= Missing` 只是数值容纳。
- **Review 076 N4**：继续为 NON-BLOCKER。Review 075 N1 ~ N3 不重开。
- 不批准产品选择、不冻结规范、不授权生产实现；没有新的接受风险；M-14、M-15、M-16 保持未解决。

**本轮授权**：原型层的修复、测试、证据更新、归档、提交与推送。范围限于两条共享链（观察错误的传播，见证的有效性）及直接关联的调用者，不扩展到生产 Runner 或其他已验收的模块。**不授权**产品决定、规范冻结、生产实现、合并或验收。

**两条共享链的对应表**：见原型文档 §2.9（A：观察入口 → 返回 → 调用者 → 失败的累积 → 终判；B：辅助进程的启动 → 见证写入 → 外层读取 → 身份验证 → 结束核验 → 外层判定）。表中逐项列出共享入口、失败由谁传播、允许的等待或重试，以及最终由谁拒绝。

**回归先行**（`8f42e58f43a31b433c540c8146fc3873dec3aa2c`，只有注入点与回归，门禁的判定未改）：

- **B1-R1-S**：`observe.sweep-recheck-once`、`observe.sweep-wait-once`、`observe.baseline-once`、`observe.prctl-fails`（Linux），`observe.injob-once`（Windows）。每个注入只作用一次，之后的调用照常；注入点打印 `WP25INJECT`。
- **B1-R2-W**：
  - 见证本身的故障：写入真实失败（审查的反例一）、短写、身份读不到（反例二）；
  - 内层结束时真实地截断、重复、追加未知 kind 的记录、删除文件；
  - 故意的遗留：没有自己的记录，或没有任何记录（`TestInnerLeak`，只在门禁的内层运行）；
  - Windows 上没有任何记录、已经结束的进程；
  - 一项正例：辅助进程在留下自己的记录之前结束。
- **判定的方式**：门禁回归要求外层以指定的理由拒绝那份证据，内层的结论只记录。辅助进程的命中记在独立于见证文件的日志中。
- **旧门禁上的结果**：
  - 作者本机：Windows 31 项中 10 项失败，WSL 35 项中 12 项失败。
  - 三平台诊断 run 39951830770（push，attempt 1，completed/failure，每个 Job 检出 `8f42e58`）：Ubuntu 35 项中 12 项、macOS 26 项中 8 项、Windows 31 项中 10 项失败，都是新增的回归。
  - 具体表现：
    - 五项 `observe.*` 内层 confirmed、exit 0，注入各命中 1 次；
    - 见证写入全部失败时见证为空、外层没有任何问题；`unknown` 身份、截断、重复、未知 kind 都被接受；
    - 两项遗留都没有被报告。Linux 上，没有任何记录的那一个在旧的 10 s 宽限内按自己的 8 s 定时器结束；
    - 早退的进程外层看不到。
  - 普通场景：Ubuntu 38、macOS 27 confirmed，Windows 28 confirmed、1 refuted（见下文“新登记”）。

**修复**（`e77af946291862191759d3af6e94e372221748a2`）：

- **B1-R1-S**：
  - 清扫在绑定之后读不到状态，记为观察失败、不发信号，不再计为身份不符；
  - `poll`（EINTR 之外）与 `waitid` 的失败保存；`waitid` 只在 `si_pid` 指名这个进程时才算回收；
  - 发信号失败为机制停止，不回退；
  - 场景起点的子进程列表读不到即 invalid；
  - 同源的 subreaper、子进程计数、Windows 失败链的 Job 成员读取与句柄等待、Unix 清理中的回收与 `stop` 的读取、macOS 早退出的 `waitid`，都不再丢弃错误。
  - 被测机制预期的故障（身份不符、pidfd 不可用、注入的 Job 查询失败）仍按预测判定，不一律改为 invalid。
- **B1-R2-W**：
  - **启动**：只有 `start` 可以启动进程（`TestStartPoints` 扫描源码）。
  - **记录**：创建者在 `cmd.Start` 之后读子进程的身份并写 `created` 记录。辅助进程在做任何事之前，先读自己的身份，等到自己的创建记录，再写 `launched` 记录。
  - **写入与失败**：每次写入都核对打开、写入长度与关闭。任何失败都带外通报外层（Unix 的信号，Windows 的命名事件），写者结束自己刚启动的子进程并停止。
  - **内层的自述**：内层测试进程写 `root` 记录，并在结束时打印封条。
  - **外层的判定**：
    - 严格解析，任何格式问题都拒绝，不跳过；
    - 核对 `root`、封条、每条回执与创建记录的身份，以及每个创建者；Windows 上另核对外层 Job 的已创建进程数；
    - 内层结束之后、任何补救之前，按本次检查的环境标记（Linux、macOS）或外层 Job（Windows）普查存活进程，再按每条记录的身份读取；
    - 补救只针对普查找到的本轮进程，从不使用无效见证中的 PID；补救之后另行记录，永远不使检查通过。
  - **登记**：按 PID 与身份匹配。
- **N5**：`missing` 改为 `shortfall`（预期登记的缺额），外层另列出已创建而未登记的进程及其结束状态。

**修复后的证据**：

- **三平台诊断** run 39954957954（push，attempt 1，completed/success，没有重跑；每个 Job 检出 `e77af94` 本身）：
  - 场景：Ubuntu（内核 6.17，带 `-race`，没有数据竞争的报告）38/38、macOS 26.6.2 arm64 27/27、Windows Server 2025 29/29 confirmed，没有 refuted、invalid 或 unsupported；
  - 门禁回归：36/36、27/27、32/32 通过；
  - 直接检查：`TestWitnessJudge`（24 例，期望写死在表中）、`TestRecordAlive`、`TestStartPoints` 三平台都通过；
  - SKIP 只有只供内层运行的 `TestInnerForeignPID` 与 `TestInnerLeak`；
  - 各平台的场景 ID 集合与 run 39887528435 相同（并集 53）。
- **作者本机**，对 `e77af94` 的工作树：
  - Windows 11：场景 29/29 confirmed，门禁回归 32/32；
  - WSL2：37 个 confirmed、1 个 unsupported（没有 pwsh），门禁回归 36/36；`TestStartPoints` 因二进制旁没有源码而 SKIP。
- **拒绝之后的清理**（外层在补救之前记下的状态）：
  - 除故意遗留的两项外，所有门禁回归在内层结束时：普查没有存活进程，逐条记录都已结束，没有状态读不到的对象；
  - 已创建而未登记的进程都已结束，数量都不多于缺额：报告损坏的 7 项各 1 个，`load.chain-corrupt` 4 个；
  - 两项故意的遗留在补救之前被报告（其中一项没有任何记录，只有普查看得到），随后被外层结束，补救之后没有存活进程。
- **隔离变异**（作者本机，仓库外的副本，每次只撤回一项修复）：
  - M13 ~ M17：撤回五个共享入口的错误传播，对应的 `observe.*` 都回到 confirmed、exit 0；
  - M18 不带外通报、M19 宽松解析、M21 不核对 Job 计数：对应见证故障的证据被接受；
  - M20 不普查：没有记录的遗留看不到。Linux 上外层没有任何问题，Windows 上只剩 Job 计数；
  - M22 不核对封条：内层写入失败缺少封条的理由，证据仍因带外通报（Windows 另有 Job 计数）被拒绝；
  - M23 身份读不到时写 `unknown` 并接受：内层回到 confirmed，外层只靠创建记录发现身份不符；
  - M24 普查之前等待 9 s：遗留按自己的定时器结束，外层不再报告。
  - 健康对照在每个变异中都通过。变异只在本机运行，macOS 没有变异。
- **计数分开**：门禁自检的通过（`WP25GATE`）、内层机制的结论（`WP25RESULT`）、外层证据是否有效（`WP25GATE` 的 `problems`）分别统计。见证故障类自检的通过，表示外层正确地判定那份证据无效，不是任何机制的 confirmed；直接检查的期望不来自门禁回归，门禁回归的命中也不来自见证文件。
- **常规 CI**：常规 CI 不运行原型，另行记录在 PR 描述中。
  - 回归先行的 run 39951833625 与修复的 run 39954960926 都是 pull_request、attempt 1、completed/success、8/8。
  - 最终 HEAD 的常规 CI 单独核验。

**新登记（本轮发现，不在授权范围内，没有修复）**：Windows 上 `T1.pipe.escaped-holder` 的“阶段 D 用满 `T_drain`”预测，截止时刻由随时钟中断更新的 `time.Now()` 算出，计时用的却是性能计数器。

- **观察**：回归先行的诊断 run 39951830770（999.82 ms）与作者本机同一提交的一次运行（999.92 ms）都因此判为 refuted。修复后的 run 39954957954 计得 1.00045 s。
- **性质**：测量工具的不一致，不是机制的发现；不形成假通过，但会使诊断失败。
- **建议**：截止与计时用同一时钟，或按时钟粒度放宽预测，并增加回归；登记在原型文档 §13。没有为取得绿色而重跑。

**N4**：本轮仍没有修复（不在授权范围内）。

**受影响的结论**（细节见原型文档 §2.9、§3.4）：

- run 39887528435（`8fea6c6`）的诊断与常规 CI 保留为历史观测。它的门禁回归在当时的范围内成立，但没有证明两点：
  - 其余共享入口的一次失败必使运行无效；
  - 外层核验在见证本身失败时仍拒绝通过。
- 经 Linux 清扫的 T2 场景、Linux 全部场景的留下子进程检查、核对 subreaper 的场景、Windows 的 `T4.start.fail-resume`，以修复后的运行为当前证据。
- C-19 的措辞已按审查收窄：“一次失败、之后成功”只在 6 个观察点上动态验证。C-20 的拒绝收尾证据现由外层普查独立核验。
- T1 ~ T6 仍只是局部支持；N1 ~ N3 保持审查结论；C-4 与 D20 的待决关系、C-19 与 C-20 的待审查状态不变。

**保持不变**：14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；§84 ~ §91 与历史审查原文没有改动。

**状态**：B1-R1-S、B1-R2-W **已修复，待独立复审**；不宣布 B1 关闭或 GO。PR #33 保持未合并，Auto-fix / Auto-merge 关闭。

## 93．Review 078：PR #33 的共享观察链、见证与外层盘点定向复审（NO-GO，B1-R2-W-R）与修复

**审查**：[Review 078](reviews/codex-review-078.md)，原样归档为 `bebc1a3`（`docs: archive Codex review 078 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `d2195de1d72e5a98479eb2a3449181aff9759393b703169da3ca7bc7afcab4ed`，逐字节一致。

**审查对象**：PR #33，HEAD `f82dfb8fad7f6078af663d6cb68a8073b9f97169`，base `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`；修复代码 `e77af946291862191759d3af6e94e372221748a2`。24 个文件，+2433/−278。

**结论**：**技术 NO-GO，暂不建议合并；Review 075 B1 尚不能整体关闭。**

- **B1-R1-S 关闭**，限本轮原型门禁与已追踪的调用链：两项精确反例与同源的观察入口，已把工具的观察错误保留到内层终判，之后的成功不恢复实验的有效性。审查方在 Windows 上独立执行了 `pidProc.ended` 与 Job 成员读取两点；其余 Linux 点由三平台诊断的完整日志与源码传播支持。
- **B1-R2-W**：原精确反例已修复，包括创建记录、回执、严格解析、根记录、封条、独立普查与 Windows Job 计数；但不能整体关闭，残余为 **B1-R2-W-R**（BLOCKER）。
  - **问题**：负向自检的判定 `judged` 只要找到预期的拒绝类别就返回成功，同时忽略外层自身的观察、命中或补救失败。
  - **复现**：审查方在 Windows 隔离副本中，只让外层 `remediate` 在负载非空时返回错误。两项故意遗留的自检（`witness.leak-unreceipted`、`witness.leak-unrecorded`）在 `still-alive` 的同时 PASS，`gate_passed=3`，整个测试进程 exit 0；没有修改最终判定。
- **Review 077 N5**（术语）关闭。**Review 076 N4** 继续为 NON-BLOCKER。
- **N6**（NON-BLOCKER）：Windows 阶段 D 的截止用 `time.Now` / Go 定时器，计时用性能计数器，严格比较会导致随机拒绝。审查核对了 Go 的源码，支持口径差异的结论；没有证明输出安全失效或假通过，也没有证明唯一根因。最小建议：统一时钟或定义，或按分辨率制定容差，并保留边界回归。
- **N7**（NON-BLOCKER）：原型文档残留旧运行来源的措辞——§4 的数值仍引旧 run，§3.4 的“当前证据”不够清楚。
- Review 075 N1 ~ N3 保持既有处理结论；没有新增机制反例或生产算法反例；不批准产品选择、不冻结规范、不授权生产实现；没有新的接受风险；M-14、M-15、M-16 保持未解决。

**本轮授权**：

- 原样归档 Review 078；
- 修复外层自检的组合判定与直接共用的调用者；
- 增加真实路径的回归、正对照与少量隔离变异；
- 更新证据、处理记录与 PR 描述；
- 提交、推送，运行诊断与常规 CI。

保持范围集中：不重写见证协议，不启动生产实现。**不授权**产品决定、规范冻结、合并或验收。

**回归先行**（`4efc9fcfb3cda5e10a765c7c35c864c7f8d6673e`，判定未改）：

- **注入点**：外层被测路径的普查、一次记录读取、补救与补救的结束确认，各有一个注入点，只在组合回归中使用：
  - `remedy-error` 是审查的原注入：补救返回存活对象与一个错误，什么也不做；
  - 其余三个让真实的一步执行，再换掉它的结果。
- **注入不碰的地方**：内层结论、见证、最终的 `pass` 字段。
- **`TestGateCombos`**：在真实的门禁回归上组合出 10 种情形，要求被测自检在应当拒绝时不通过；组合计在 `WP25COMBO` 行，与门禁回归分开。
  - 预期拒绝命中，同时：补救成功、补救失败（有记录或无记录的遗留）、结束无法确认、普查失败、记录读取失败、出现用例不允许的其他问题；
  - 预期拒绝没有命中；
  - 健康：无故障，或有一次记录读取失败。
- **保险清理**：在判定之后、独立于被测路径，单独记录，保证这些实验不留下进程。
- **旧判定上的结果**：
  - 作者本机 Windows 与 WSL 各 10 个组合，6 个失败；门禁回归 32/32、36/36 仍通过。
  - 三平台诊断 run 39964760402（push，attempt 1，completed/failure，预期；每个 Job 检出 `4efc9fc`）：三平台都是同样的 6 个组合失败，其中被测自检在外层失败的同时 PASS；门禁回归 36/36、27/27、32/32 通过。
  - 场景：Ubuntu 38、macOS 27 confirmed，Windows 28 confirmed、1 refuted（`T1.pipe.escaped-holder`，N6，999.9851 ms）。
  - 审查原反例（`combo.reject-remedy-error`）：内层 invalid、exit 1，补救之前 1 个存活，补救返回错误，`still_alive` 为同一进程，被测自检 PASS；随后保险清理结束了它。
- **首次失败**：同一 run 的 macOS Job 中，`combo.reject-unrecorded-remedy-error` 的保险清理报告 FAILED。
  - 它重新普查时读不到遗留进程的环境（`sysctl kern.procargs2` EINVAL），而约 0.2 s 之前被测路径的普查读到了它；那个进程最迟在自己的 8 s 期限结束，这不计为清理，也没有在 runner 上核验。
  - 原因没有查明。
  - 修复中，保险清理改为先按已认出的身份结束对象；被测路径的普查没有放宽。
- **常规 CI**：回归先行的常规 CI（run 39964764139，pull_request）被 ci.yml 自己的并发设置取消（completed/cancelled）。

**修复**（`31c0b290aed741f2506503aa0bb73790779a1a16`）：

- **判定**（`judge`）按分开保存的维度：
  - 预期拒绝是否找到；
  - 证据的其他问题，即用例允许的连带类别之外的问题；
  - 外层自己的失败：`inner`、`census`、`state-unknown`、`hit`、`still-alive`、`insurance`，任何预期拒绝都不能豁免；
  - 补救：不需要、成功、失败、没有执行；
  - 本轮进程的最终状态，取被测路径自己的事实：已确认结束、仍存活、未知（未知从不当作结束）。

  全部满足才 PASS；找到预期拒绝只证明第一项。允许的连带类别逐用例写明（两项故意遗留与内层写入失败），其余用例不允许任何连带类别。
- **保险清理**：在判定之后，先结束被测路径已认出的对象（再核对启动时间），再结束新的普查找到的进程。它只会使结论变差：被测路径说已结束、它却还找到进程或失败，即记为 `insurance` 而不通过。
- **允许的重试**：没有新增，仍只有原有协议中的那些。
- **组合回归与判定表**：组合回归检查每一维；`TestGateJudge` 以 13 行固定的期望直接检查判定。
- **原型文档**：§2.10 写出判定表；N7 的措辞已改——§3.4 区分“当时的证据”与“当前证据”，§4 ~ §8 的数值改为当前重跑 run 39966556540 的数值。

**修复后的证据**：

- **三平台诊断** run 39966556540（push，attempt 1，completed/success，没有重跑；每个 Job 检出 `31c0b29` 本身）：
  - 组合：三平台各 10/10，每一维符合要求；
  - 门禁回归：Ubuntu 36/36、macOS 27/27、Windows 32/32；
  - 场景：38/38、27/27、29/29 confirmed，没有 refuted、invalid 或 unsupported；Ubuntu 带 `-race`、没有数据竞争的报告；
  - 直接检查：`TestGateJudge` 13 行、`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台都通过；
  - SKIP 只有 `TestInnerForeignPID` 与 `TestInnerLeak`；场景 ID 集合与此前各轮相同（并集 53）；
  - macOS 上那一项组合的保险清理为 `ended 1`。
- **作者本机**，对 `31c0b29` 的工作树：
  - Windows 11：场景 29/29、门禁回归 32/32、组合 10/10；
  - WSL2：37 个 confirmed、1 个 unsupported（没有 pwsh）、门禁回归 36/36、组合 10/10。
- **被测补救与保险清理分别记录**（原型文档 §2.10 的表）：
  - 补救失败的两项：被测路径为“失败、仍存活”，被测自检 FAIL；保险清理随后结束了那 1 个进程。
  - 其余：保险清理“什么也没剩”。
  - 保险清理的结果没有使任何一项通过。
- **隔离变异**（作者本机，Windows 与 WSL 结果相同）：
  - M25 恢复旧规则：6 个组合与 8 行判定表失败；
  - M26 不看其他问题：1 个组合与 2 行失败；
  - M27 不看外层自己的失败、M28 不看补救与最终状态：组合仍全部通过（两维互为后备），由判定表各 2 行检出；
  - M29 二者同时：重现 6 个假通过；
  - M30 未知当作结束：4 个组合的维度不符；
  - 健康对照每次都通过。
  - M25 第一次准备时变异不能编译，记为无效，改正之后重做。
- **计数分开**：门禁回归（`WP25GATE`）、组合（`WP25COMBO`）、内层机制的结论（`WP25RESULT`）、直接检查。被测自检在组合中的“正确拒绝”不计为门禁回归的通过。
- **常规 CI**：修复提交的常规 CI 与最终 HEAD 的常规 CI 另行核验，记录在 PR 描述中；常规 CI 不运行原型。

**受影响的结论**（细节见原型文档 §2.10、§3.4）：

- run 39954957954（`e77af94`）的诊断与常规 CI 保留为历史观测：它没有证明外层负向自检在自己的补救、结束确认、普查或读取失败时不会通过。健康的正对照与已修复的原反例不受影响。
- 当前证据为 run 39966556540。
- T1 ~ T6 仍只是局部支持；N1 ~ N3 保持审查结论；C-4 与 D20 的待决关系、C-19 与 C-20 的待审查状态不变。

**N4、N6**：本轮没有修复（不在授权范围内），登记在原型文档 §13；每一次 N6 的 refuted 都如实保留。

**保持不变**：14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；§84 ~ §92 与历史审查原文没有改动。

**状态**：B1-R1-S 已经 Review 078 关闭；B1-R2-W-R **已修复，待独立复审**；不宣布 B1-R2-W 或 B1 整体关闭，不宣布 GO。PR #33 保持未合并，Auto-fix / Auto-merge 关闭。

## 94．Review 079：PR #33 的组合终判与保险清理定向复审（NO-GO，B2）与修复

**审查**：[Review 079](reviews/codex-review-079.md)，原样归档为 `325c646`（`docs: archive Codex review 079 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `6e86a9ad1dd40107984a6dccde917b8c7d3d9dd8a212a700b59eb5d47c5bb405`，逐字节一致。

**审查对象**：PR #33，HEAD `8910825c8145f7eea0cec6451ff8a84b885e48ab`，base `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`；修复代码 `31c0b290aed741f2506503aa0bb73790779a1a16`。12 个文件，+1081/−99。

**结论**（按审查原文分层）：

- **A．门禁实现与直接回归：技术 NO-GO。**
  - **B1-R2-W-R 可以关闭**：预期拒绝不再豁免工具错误、未知结束状态或补救失败；旧版的 6 个错误 PASS 在新版本都被拒绝。
  - **B1-R1-S 保持关闭**。
  - **原始 B1 的已知判定缺陷**可按既有的原型门禁范围收口；这不等于本轮的直接回归 B2 也已通过。
  - **新增 BLOCKER B2**：Unix 的保险清理把未确认归属的见证身份升级为信号授权。调用方把只由见证记录提供、没有被本轮普查确认归属的身份加入 `known`；保险清理只凭 PID 与启动时间相符就发 SIGKILL；见证即使已被判为无效，也没有阻止这一动作。
  - **审查方的验证**：在 Windows 上安全验证了共用的数据路径（一条 `created` 记录被换成外层自身的身份后，进入了保险清理的入口，但不在内层 Job 中，没有动作）；Unix 的发信号分支是源码级证明，没有在 Unix 上实际执行。
  - **最小修正**：区分“记录声称存在的身份”与“已经独立证明属于本轮的身份”；保留对可信的旧普查身份在动作前的复核（不要求重读环境）；无法证明归属时保守报告，只处置已有归属证明的对象。并补一个“格式合法但归属无效的 `created` 指向外层或独立自有对照进程”的负例。
- **B．当前合并条件：未满足。**
  - 最终常规 CI run 39968755313 为 attempt 1、completed/failure、7/8：Windows 的 seq 跨进程测试报告 SQLITE_BUSY。原型诊断的成功与旧 CI 都不能替代这一失败。
  - 那次错误的特征与 M-15 已登记的 SQLITE_BUSY 观察相似；M-14 的定义是版本探测路径的 `SQLITE_READONLY_RECOVERY`（264），不能因为测试同名就把当前错误并入 M-14。没有批准重试、映射变更或风险豁免。
- **C．规范与授权：没有改变。**
- **非阻塞**：N7 可关闭；N4、N6 保持未修复、非阻塞；C-4 与 D20、C-19 与 C-20 的前提与待决状态保持；源码扫描仍只是当前原型启动入口的检查，不等于生产 C-9。
- 没有新增产品决定，没有新的接受风险；M-14、M-15、M-16 保持未解决。

**本轮授权**：

- 原样归档 Review 079；
- 修复保险清理的候选来源、归属验证及直接共用的入口；
- 增加安全的真实链路回归、正对照与必要的变异；
- 更新证据、处理记录、状态与 PR 描述；
- 提交、推送，运行三平台原型诊断与新 HEAD 的常规 CI。

**不授权**：生产 Runner、全仓库重构、保证调整、合并或验收。

**候选来源、身份与归属**：

- **原则**：PID 与启动时间只能证明“还是那个进程”，不能证明“它属于本轮检查”。
- **原型中可以核对的归属依据**有两种：
  - Unix：进程环境中本次检查唯一的标记，只有内层运行的进程继承它；
  - Windows：外层 Job 的成员资格，内层运行的进程不能脱离。
- **见证记录**是内层运行自己的声明，见证本身可能无效。
- 对应表（候选来源 → 身份 → 归属依据 → 允许动作的入口 → 失败如何上报）见原型文档 §2.11。
- **外层的信号路径**：
  - Unix：补救与保险清理，都经同一个接缝；
  - Windows：两个入口都只终止外层 Job；
  - 带外通报（向外层自身发 SIGUSR1）不是清理路径。
- 保险清理的直接调用者只有 `runGate`。

**回归先行**（`f1b8566bde691d738b804c4defd129605367c14e`，谁可以被动作没有改）：

- **信号接缝**：外层的信号都经一个接缝，组合回归可以让它对夹具只记录、不发送。
- **外层注入 `forge-owner`**：在外层读取见证之前，追加一条格式合法、以内层名义写的 `created` 记录，指向组合自己启动、持有、负责结束的夹具进程；夹具不属于内层运行，也不在外层 Job 中。
- **三个组合**：健康运行、同时有本轮自己遗留的进程、身份已经变化（夹具 PID 配另一个启动时间）。
- **每个组合要求**：夹具不被交给按 PID 动作的入口，没有发向它的信号，之后仍存活；随后由组合自己结束。
- **旧代码上的结果**：
  - 作者本机 Windows：两项 `forge-owner` 组合失败（夹具被交给动作入口；入口只终止 Job，没有信号）。
  - 作者本机 WSL：同样两项失败，保险清理向夹具发出 477 / 471 次信号意图，全部只记录、没有发送。
  - 三平台诊断 run 39976843072（push，attempt 1，completed/failure，预期；每个 Job 检出 `f1b8566`）：三平台组合都是 13 个中这 2 个失败；Ubuntu 401 / 400、macOS 423 / 435 次信号意图（只记录），Windows 只有数据路径。门禁回归 36/36、27/27、32/32，场景 38、27、29 confirmed。
  - 夹具在所有平台都仍存活，随后由组合自己结束。没有向任何未知进程或夹具实际发送过信号。

**修复**（`8a2b97ccd521c6af587baf4e3f45421d68dafcd9`）：

- **分拣候选**：普查找到的直接可动作；只凭记录存活的，只有在那一刻证明归属（Unix：同一进程且带本次检查的标记；Windows：同一进程且在外层 Job 中）才可动作。
- **拒绝**：证明不成立或读不到的，不进入遗留清单，不交给补救或保险清理；记为 `ownership` 工具错误，任何预期拒绝都不能豁免；最终状态为未知。
- **复用身份**：保险清理只复用已证明的身份，动作前再核对启动时间；对可信的旧普查身份不要求重读环境，保留 macOS EINVAL 修正的方向。
- **其余入口**：补救在 Unix 上仍在动作前核对标记；Windows 的两个入口仍只终止外层 Job。
- **没有为此禁用保险清理**。
- **直接检查**：`TestCandidates` 以 5 行固定的期望检查分拣。

**修复后的证据**：

- **三平台诊断** run 39977737240（push，attempt 1，completed/success，没有重跑；每个 Job 检出 `8a2b97c` 本身）：
  - 组合：三平台各 13/13；
  - 门禁回归：Ubuntu 36/36、macOS 27/27、Windows 32/32；
  - 场景：38/38、27/27、29/29 confirmed，Ubuntu 带 `-race`；
  - 直接检查：`TestCandidates`、`TestGateJudge`、`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台都通过；
  - 两项 `forge-owner` 组合中，夹具在三平台都被拒绝，没有被交给动作入口，信号意图为 0，之后仍存活，再由组合自己结束；`forge-stale` 中夹具不是候选，没有动作。
- **作者本机**，对 `8a2b97c` 的工作树：
  - Windows 11：门禁回归 32/32、组合 13/13、直接检查都通过；场景 28 个 confirmed、1 个 refuted——`T1.pipe.escaped-holder`，N6，阶段 D 999.5519 ms，没有重跑；
  - WSL2：37 个 confirmed、1 个 unsupported（没有 pwsh），门禁回归 36/36，组合 13/13。
- **被测补救与保险清理分别记录**（原型文档 §2.11 的表）：
  - 本轮自己遗留的进程，照常由被测补救结束（`forge-owner-mixed` 中补救成功）；
  - 被拒绝的夹具，两个入口都没有动作；
  - 保险清理的结果没有使任何一项通过。
- **Review 078 的组合不回退**：10 个组合、全部门禁回归与健康对照，在修复后仍然通过；补救失败、结束未知与保险清理矛盾的组合仍使被测自检不通过。
- **隔离变异**（作者本机，Windows 与 WSL）：
  - M31，只凭记录存活的进程不经证明就动作：两项组合失败、分拣表 3 行失败，WSL 上另有信号意图。
  - M32，归属只凭身份：两项组合失败（分拣表通过，只有真实路径检出）。
  - M33，保险清理也拿到只凭记录的进程：WSL 上两项组合失败（信号意图）；Windows 没有检出，因为它的保险清理只终止 Job、不用这份清单。
  - M34，拒绝之后最终状态不再记为未知：两项组合的维度不符。
  - 健康对照每次都通过；所有信号意图都只被记录，没有发送。
- **常规 CI**：常规 CI 不运行原型。本轮各提交与最终 HEAD 的常规 CI 另行核验，记录在 PR 描述中。旧 HEAD 的 run 39968755313 不重跑，也不据此确认 M-15 的根因。

**受影响的结论**（细节见原型文档 §2.11、§3.4）：

- run 39966556540（`31c0b29`）的诊断与常规 CI 保留为历史观测：它没有证明“保险清理只对归属已证明的进程动作”。当时的健康运行中没有伪造的记录，没有证据表明当时发生过这样的动作。
- 当前证据为 run 39977737240。
- T1 ~ T6 仍只是局部支持；N1 ~ N3 保持审查结论；C-4 与 D20、C-19 与 C-20 的待决状态不变。

**仍未验证的边界**：

- 身份核对与信号之间的 PID 复用没有构造，也没有解决。
- 归属依据是受控原型下的标记继承与 Job 成员，不是对任意宿主的保证。
- “归属读不到”只由分拣表覆盖。
- Unix 上发向夹具的信号只由接缝记录。

**N4、N6**：本轮没有修复（不在授权范围内），登记在原型文档 §13。

**保持不变**：14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；§84 ~ §93 与历史审查原文没有改动。

**状态**：B1-R2-W-R 已经 Review 079 关闭，B1-R1-S 保持关闭；**B2 已修复，待独立复审**；不宣布 B2 关闭或 GO。PR #33 保持未合并，Auto-fix / Auto-merge 关闭。

## 95．Review 080：PR #33 原型技术 GO（B2 关闭）；收尾与条件合并

**审查**：[Review 080](reviews/codex-review-080.md)，原样归档为 `3f2377843da1ace5aef4802ba37010a42826c167`（`docs: archive Codex review 080 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `8fff4fb5cf31f1f986c5a93152ec57e8ba31e8c351506d04dc00363aef86ded8`，逐字节一致。

**审查对象**（不随收尾而改变）：

- PR #33，HEAD `06213169f4f5cbe9758943a64eeebd2447f748e4`，base `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`；回归先行 `f1b8566bde691d738b804c4defd129605367c14e`，修复代码 `8a2b97ccd521c6af587baf4e3f45421d68dafcd9`。
- 相对 Review 079 的对象 `8910825` 共 13 个文件，+905/−74。`8a2b97c → 0621316` 只有文档变化，两者的 `prototype/wp25` 子树均为 `e70f83a0be9492805fbee8e692382ae7ee1c5cb7`。

**结论**（按审查原文分层）：

- **A．Review 079 B2 可以关闭；原型 PR #33 技术 GO。**
  - 只凭见证记录存活的身份，必须经 `ownedNow` 证明归属，才进入补救与保险清理共用的动作清单；归属不成立或读不到时拒绝动作，记为不可豁免的 `ownership` 工具错误，最终状态为未知。
  - 此前可信普查取得的身份仍可交给保险清理，动作前复核启动身份，不要求重读环境；保留 macOS EINVAL 修正的方向。
  - B1-R1-S、B1-R2-W-R 等既有的关闭结论保持。
  - 没有发现新的直接 BLOCKER。结论限定在本原型声明的对象来源与受控继承 / Job 前提内。
- **B．当前固定对象的 CI 合并条件已经满足**：
  - 最终 HEAD 的 run 39979587006 为 attempt 1、completed/success、8/8。它检出合并快照 `d3841c7920e076f1af83e136f5456377066a3b2f`，父提交为固定的 base 与最终 HEAD，树 `e55e4f41d76a35fb59f755fb514a1421f0932e11` 与最终 HEAD 的树相同。
  - 修复代码的常规 CI run 39977743051（8/8）与三平台诊断 run 39977737240（3/3）也正式成功。
  - 可以建议负责人合并本原型 PR。
  - 上轮 run 39968755313 的 SQLITE_BUSY 失败继续保留，没有重跑；本次成功不证明它的根因已经消失，也不关闭 M-15。
- **C．冻结与实施的条件没有因此满足**：T1 ~ T6 仍只得到局部支持；14 项产品选择没有批准；规范没有冻结；生产 WP2.5 没有授权；WP2.5 与 M2 没有验收。
- **NON-BLOCKER**：
  - **N4** 保持未修复、非阻塞：`epipe` 报告写入失败仍可能被分为 refuted 而不是 invalid；没有新增的假通过证据。建议独立传递报告工具的错误，并增加分类回归。
  - **N6** 保持未修复、非阻塞：Windows 上截止与计时的口径及严格的边界比较导致随机拒绝，既有裁定不变。建议统一时钟的定义或明确分辨率规则，并保留边界回归；不能只为取得绿色而加任意容差。
  - **N5、N7** 保持已关闭的既有范围。
- 没有新增产品待决项，没有新的接受风险；M-14、M-15、M-16 保持未解决。

**审查方的验证与边界**：

- **本机**：Windows 11（26300），现有的 Go 1.27.1，仓库外的隔离副本，`GOTOOLCHAIN=local`、`GOPROXY=off`。
  - 回归先行 `f1b8566`：13 个组合中恰好两个 `forge-owner` 组合按预期失败，exit 1。
  - 固定对象：32 项 Windows 门禁回归、13 个组合与直接检查都通过，exit 0。
  - 在真实的外层调用链上，于真实的 `IsProcessInJob` 查询之后注入一次归属读取错误：两个 owner 组合都命中，被测自检失败并保留未知的最终状态，没有误动作，exit 0。这支持“错误沿真实调用链不可豁免地传播”，**不是**原生 Windows 查询失败或 Unix 环境读取失败的动态复现。
- **Unix**：结论来自三平台诊断的完整日志（旧代码 401 / 400、423 / 435 次信号意图，都只由接缝记录；修复后 0 / 0），不是审查方本机的复现。作者 WSL 的信号意图与 M31 ~ M34 只作为作者的辅助证据。
- **没有覆盖的**：
  - 审查方本机没有重跑全部 29 项 Windows 机制场景，也没有运行 N6 的计时场景；作者本机对 `8a2b97c` 的 28 confirmed、1 refuted（999.5519 ms）照原样保留。
  - Unix 上身份复核与按 PID 发信号之间的原子性窗口、PID 复用，归属读取失败的原生动态验证，以及任意宿主的适用性，都仍未验证或未解决。标记继承与 Job 成员是受控原型的前提。
- 审查方没有触发、重跑或取消任何 workflow。

**负责人决定**（2026-10-24）：

- 收尾之后，条件授权以 merge commit 合并 PR #33，并用 `--match-head-commit` 锁定完整的最终收尾 HEAD。条件：
  - 相对审查对象，收尾增量只有 Review 080 的归档与状态记录；原型代码与已诊断的版本一致；生产代码与 workflow 没有变化；
  - 最终收尾 HEAD 的常规 CI 正式 completed/success、8/8，并核对实际检出、合并快照的父提交与内容树；
  - 合并之前再核对 HEAD、base、master 与 Auto-fix / Auto-merge；base 移动时，先核对新增内容与预期的合并树；
  - 唯一的失败是 Windows 已登记的 seq 并发测试出现 SQLITE_BUSY (5) 时，合计允许一次只重跑失败的作业；不据此关闭 M-15 或确认其根因。
- 合并核验与 master push CI 全部成功之后，授权从已核验的新 master 另开独立的小 PR，修复原型的 N4、N6（test-only），不合并，等待独立审查。
- **不授权**：14 项产品选择的批准、规范冻结、生产 Check Runner、迁移 0009、CLI 接线、WP2.5 或 M2 的验收。

**收尾**（相对审查对象 `0621316`，只有归档与状态记录）：

- **原型文档**：状态改为“技术审查通过，待合并核验”；头部、§0、§2.11 的边界、§11 第 1、2 项与 §13 的 N4、N6 登记 Review 080 的结论；T1 ~ T6 的局部支持与冻结之前的缺项保留。
- **状态记录**：AGENTS.md 与实施计划同步。
- **没有改动**：
  - 代码、测试、Schema、迁移、依赖与 workflow；
  - 规范草案 `docs/wp2.5-check-runner.md`，包括它头部的状态记录（仍写到审查 079）。

**保持不变**：

- 14 项产品选择待批准；草案第 4 版没有冻结；
- T1 ~ T6 冻结之前的缺项（原型文档 §11、§12；Review 080 第九节）；
- M-14、M-15、M-16 未解决；
- §84 ~ §94 与历史审查原文没有改动。

合并结果与 CI 回执写入 PR #33 的描述，不为补记结果直接向 master 追加提交。

**状态**：

- PR #33：技术审查通过，待合并核验。B2 经 Review 080 关闭，B1-R1-S、B1-R2-W-R 保持关闭；N4、N6 没有修复。
- 规范没有冻结，WP2.5 没有实现或验收，M2 没有整体验收。
- Auto-fix / Auto-merge 保持关闭。

## 96．PR #33 的合并核验；原型 N4、N6 的修复（独立小 PR）

### 96.1 PR #33 的合并与核验（§95 的条件授权）

**收尾**：

- `3f2377843da1ace5aef4802ba37010a42826c167`：Review 080 原样归档，SHA-256 `8fff4fb5cf31f1f986c5a93152ec57e8ba31e8c351506d04dc00363aef86ded8`（源文件、暂存与提交 blob 相同）。
- `38d15ef950fc1b7ba79ea08dfb47ab042a14cafa`：§95 与状态记录。
- 相对审查对象 `0621316` 共 5 个文件，都是文档。`prototype/wp25` 的子树仍为 `e70f83a0be9492805fbee8e692382ae7ee1c5cb7`，与已诊断的 `8a2b97c` 相同；`git diff 8a2b97c 38d15ef -- prototype .github` 为空，没有重新触发诊断。

**收尾 HEAD 的常规 CI**：run 39986318364，pull_request，attempt 1，completed/success，8/8，没有重跑。

- 8 个 Job 都检出合并快照 `2ecee7478cb7d2fcb031d4dbba9ae44d870152b9`；它的父提交为 `ecb83ba` 与 `38d15ef`，树 `daa0e2b02b1345d092766d04b4721953814b1e70` 与收尾 HEAD 相同。
- 三个 Go 测试 Job 的 PASS / SKIP 行与 run 39979587006 相同；没有 `FAIL`，没有原型包。

**合并**：

- 合并之前再核对：PR HEAD `38d15ef`，base 与 master 都是 `ecb83ba`（没有移动），CLEAN，8/8；Auto-fix / Auto-merge 关闭。
- 以 merge commit 合并，`--match-head-commit 38d15ef950fc1b7ba79ea08dfb47ab042a14cafa`，时间 2026-10-23T16:40:24Z（UTC）。
- 合并提交 `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a`：
  - 父提交 `ecb83ba`（master）与 `38d15ef`（收尾 HEAD）；master 包含收尾 HEAD；
  - 树 `daa0e2b` 与收尾 HEAD 相同，`git diff 38d15ef 5e9a42c` 为空；
  - 相对 `ecb83ba` 共 43 个文件，+11139/−4，即 PR 的内容。

**master push CI**：run 39988732786，push，attempt 1，completed/success，8/8，没有重跑。

- 8 个 Job 都检出 `5e9a42c` 本身。
- 三个 Go 测试 Job 的 PASS / SKIP 行与 run 39986318364 相同。
- 原型 workflow 没有在 master 上运行（它的 push 只针对原型的分支）。

**其余**：

- 本轮没有失败、取消或跳过的 Job，没有用到受控重跑；M-15 不因此关闭或确认根因。
- 结果写入了 PR #33 的描述；没有为此向 master 追加提交。本节随 N4、N6 的 PR 记入。
- PR #33 的合并**不是**规范冻结、生产授权或 WP2.5 / M2 的验收。

### 96.2 原型 N4、N6 的修复（§95 授权的独立小 PR）

**分支** `test/wp2.5-prototype-n4-n6`，从已核验的 master `5e9a42c` 开出；细节见原型文档 §2.12。

| 提交 | 内容 |
|---|---|
| `e7dbcdcf5690e6cbbca50515f37cefe1c49c2845` | 诊断 workflow：push 的分支列表加入本分支一项精确匹配（单独列出；其余不变，`ci.yml` 不变） |
| `91229b5beb7b6b06fe3a2d97b3e99f25d1a9e882` | **回归先行**，判定没有改：三项 F1 门禁回归、`phaseClock` 接缝（默认仍是 Go 的时钟）、`TestPhaseClock` |
| `3fe5701f0c602ffdcf42200ce93b793341603da3` | N4 的修复 |
| `5054e54a14d3e9084aa675880175232e61361c17` | N6 的修复 |
| `ac3b3bed4489404124d51181c1e8ed902d55c82e` | 直接检查 `TestPhaseClock` 改用内存管道（**修复代码固定于此**） |
| 之后 | 只有文档 |

**N4**：持有者的 `epipe` 记录写不出时，F1 场景曾把“没有记录”记为 refuted。

- 修复之后，没有记录时先读持有者的状态：仍在运行为 refuted（写入没有失败），已结束为 invalid（观察工具没能报告）。
- 持有者自己在记录写不出时以 `exitReport` 退出。
- 回归：`report.epipe-unwritable`（要求 invalid）、`control.f1-read-kept`（另持读端，工具正常而预测不成立，要求 refuted）、`control.f1-skip-drain`（要求 confirmed）。

**N6**：阶段 D 的截止在 Go 的时钟上，时长却在 stopwatch（Windows 上为性能计数器）上计量，两者在 Windows 上可以相差一个中断周期。

- 修复之后，阶段 D 与阶段 J 都在同一个 `phaseClock` 上限定与计量；等待只在这个时钟读到截止之后才以截止结束；stopwatch 的读数另记，不作判定。
- 没有改动 `T_kill`、`T_drain`、`T_join` 或任何超时，没有容差；不宣称硬实时保证。
- 确定性的边界回归 `TestPhaseClock`：恰好到截止、差 1 ns、早到 450 µs 的唤醒。实际管道的对照 `T1.pipe.escaped-holder` 保留严格比较。

**证据**：

- **回归先行**：三平台诊断 run 39991866363（push，attempt 1，completed/failure，预期；检出 `91229b5`）：`report.epipe-unwritable` 三平台都得到 refuted；`TestPhaseClock` 三行三平台都失败；场景 38、27、29 confirmed；组合 13/13。作者本机 Windows 与 WSL 相同。
- **`5054e54` 的诊断** run 39992975299（completed/success）保留为历史观测：其后作者本机发现 `TestPhaseClock` 在 Windows 上的超时（60 次重复中 180 个子检查失败 12 个），原因是检查自己用了操作系统的管道（Windows 上关闭读端只取消已发出的读取）。`ac3b3be` 改用内存管道，200 次重复中 600 个子检查 0 失败。
- **修复后**：三平台诊断 run 39994591073（push，attempt 1，completed/success，没有重跑；检出 `ac3b3be`）：
  - 场景 38/38、27/27、29/29 confirmed（Ubuntu 带 `-race`，没有数据竞争的报告），并集 53；
  - 门禁回归 39/39、30/30、35/35；组合 13/13；
  - B2 的两项 `forge-owner` 组合中夹具零动作、零信号；
  - 直接检查都通过；SKIP 只有两个只供内层运行的测试。
- **作者本机**（`ac3b3be`）：Windows 29/29、门禁回归 35/35、组合 13/13；WSL 37 + 1 unsupported、39/39、13/13。
- **隔离变异** M35 ~ M38：每个都只被对应的检查检出（原型文档 §2.12 的表）。
- **常规 CI**：PR 打开之后对最终 HEAD 运行，结果记录在 PR 描述中；常规 CI 不运行原型。

**新的观察**（登记，待审查，不由本轮决定）：Windows 上关闭读端只取消已经发出的读取；“停止读取会使复制器返回”依赖读取先到达系统。是否需要另外的保证，属于冻结之前要明确的技术前提（原型文档 §2.12、§13）。

**不改动**：生产代码、根模块的依赖、Schema、迁移、`ci.yml`；规范草案 `docs/wp2.5-check-runner.md`（包括它头部的状态记录，仍写到审查 079）；§84 ~ §95 与历史审查原文。

**状态**：

- PR #33：已合并为 `5e9a42c`，合并核验与 master push CI 正式成功。
- N4、N6：**已修复，待独立审查**；不宣布关闭。修复的 PR 保持未合并。
- 14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；Auto-fix / Auto-merge 关闭。

## 97．Review 081：PR #34 的 N4 / N6 修复复审（NO-GO，B1、N4-R）与定向修复

**审查**：[Review 081](reviews/codex-review-081.md)，原样归档为 `9c612d696d215773d3fc59ada2651b485f39d8b7`（`docs: archive Codex review 081 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `d1c1227c2b16ccd097db2c29cd128f40ad6811c3079cad1c25643866f0ca9280`，逐字节一致。

**审查对象**：PR #34，HEAD `4e009a6b7fa7c81ca614b6d20789735175a8caaf`，base `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a`；修复代码 `ac3b3bed4489404124d51181c1e8ed902d55c82e`。12 个文件，+630/−58。

**结论**（按审查原文分层）：

- **PR #34：技术 NO-GO，当前不建议合并。** 正式 CI 已成功，技术修正条件尚未满足。
- **N6 可以关闭**（原时钟口径的范围内）：D、J 的截止、剩余预算与阶段时长统一在 `phaseClock` 上，提前唤醒会重新核对截止，stopwatch 只作观察；确定性边界检查与真实管道正对照都有有效证据。这不包括 Close 的生存期与真实 I/O 的有界保证。
- **N4 不能整体关闭**：“报告写不出且持有者已经退出”的旧路径已修，但“没有记录、持有者仍活着，所以写入没有失败”不成立。审查方在真实报告写失败之后只延后退出，就再次得到 refuted（N4-R）。它是工具故障的错误分类，没有形成 confirmed 或绿色的假通过，沿用原 N4 的非阻塞严重度。
- **新增 BLOCKER B1**：Windows 上，S6 驱动可能在阶段 J 之前同步阻塞于 `Close`，无法兑现有界进入封闭与 O3 的候选保证。
  - `runS6On` 在调用者中逐个关闭读端，全部关闭返回之后才开始阶段 J，之后才封闭。
  - 换用 `io.Pipe` 正确隔离了时钟检查，却没有修复真实的操作系统管道路径；原型文档仍把它主要归为检查本身的缺陷，并以此前的等待推定读取已经发出，现有调用链没有建立这一前提。
  - 审查方的复验：在仓库外以真实的 `runS6` / `runS6On`、自建的 `os.Pipe`、真实的 `os.File.Close` 与 `ReadFile`，经 `SyscallConn.Read` 在读锁之后、系统读取之前设置可释放的闸门。默认 `T_join` 下，2 s 之后仍阻塞在 `(*FD).Close`、尚未开始阶段 J；测试向自己的写端写入 1 字节之后全部返回。
  - 最小修正：给停止读取与 Close 自身明确的完成、预算与失败出口，并有可信的封闭及任务、句柄归属协议；或者明确撤回现机制不能支持的保证、登记反例并收口文档。单纯延长等待、用 `io.Pipe` 取代、依赖全局测试超时，都不是修复。
- **其余**：PR #33 的 B1、B2 的关闭范围保持；正式 CI 条件已满足（最终 HEAD 的 run 39996294870 为 attempt 1、8/8）。check-runner 草案头部与 PR #33 的标题仍是过时的措辞，本轮不改。
- 没有新增产品决定，没有新的接受风险；M-14、M-15、M-16 保持未解决。

**本轮授权**：

- 原样归档 Review 081；
- 在 PR #34 上定向修复 B1 与 N4-R，回归先行，定向撤回验证；
- 更新原型文档、处理记录与状态；
- 提交、推送，运行三平台原型诊断与最终 HEAD 的常规 CI。

**不授权**：合并、验收、生产代码、Schema、迁移、根 `ci.yml`、诊断 workflow 的扩大、14 项产品选择、规范冻结、生产 WP2.5。

**回归先行**（`bde00fc30cae12c0827a6d3125d04273d10af164`，判定没有改）：

- `TestStopRead`（直接检查）：真实的 `os.Pipe` 与 `File.Close`，假时钟上的 `runS6On`；读取经 RawConn 在读引用之后、系统读取之前停在闸门前。对照（读取已在系统调用中）、窗口在阶段 J 内放开、窗口持续到阶段 J 结束。
- `report.epipe-unwritable-alive`（门禁回归）：真实的 `epipe` 记录写入失败，持有者之后再存活 5 s。
- 三平台诊断 run 40066182430（push，attempt 1，completed/failure，预期；检出 `bde00fc`）：两个窗口行三平台都失败（停止挡住了阶段 J），对照通过；`report.epipe-unwritable-alive` 三平台都得到 refuted。作者本机 Windows 相同。
- 对照行的第一版要求“阶段 J 请求了唤醒”，在旧代码上也失败（复制器先返回时不请求唤醒）；这是检查自己的错误，提交之前已改正。

**修复**：

- **B1**（`1a6fec5a988f29332c871ef04ff7ac92d5aedc17`）：阶段 J 以停止开始。
  - 每个被停止的流有一个运行持有的停止任务；`T_join` 同时覆盖停止与复制器，阶段 J 结束即封闭。
  - 结果逐流给出：停止已请求；停止已完成或挂起；复制器已返回或 O3；入口已封闭；仍在运行的交给调用方（`s6Result.left`）。
  - 场景以 `adopt` 接手，在进程结束之后等待它们，期限内没有结束即 invalid；O3 登记同时持有挂起的停止，两种后续调用策略都看得到（C-4、D20，未批准）。
  - Windows 上，停止任务在 Close 等待期间每毫秒经自有的复制句柄调用一次 `CancelIoEx`，直到 Close 返回；作者在仓库外的探针中核实了经复制句柄的取消会取消经原句柄发出的同步读取。
  - 没有改动任何预算，没有容差。
- **N4-R**（`079f9c72d57a0b182d970b0f8455a80d580e04c1`）：没有 `epipe` 记录时，只有正面证据，即持有者的迟到字节确实到达了场景另持的读端（FIONREAD、PeekNamedPipe 计数，不读取），才是 refuted；报告尚未到达、报告失败与状态不可读都是 invalid。“持有者仍在运行”只写进理由。
- **补充的回归**（`125c4f70114c312828f7f8fc74734761e14c343a`）：`cleanup.leftover-running`，在修复之后补上，因为定向撤回时发现场景的等待没有检查覆盖；在 `bde00fc` 的树上加入同样的注入与用例为 refuted（门禁回归失败），在当前树上为 invalid。**修复代码固定为 `125c4f7`。**

**修复后的证据**：

- 三平台诊断 run 40066910980（push，attempt 1，completed/success，没有重跑；检出 `125c4f7`）：
  - 场景 38/38、27/27、29/29 confirmed（Ubuntu 带 `-race`，没有数据竞争的报告），并集 53；
  - 门禁回归 41/41、32/32、37/37；组合 13/13；
  - B2 的归属核验零动作、零信号，本轮自有遗留照常清理；
  - 直接检查（含 `TestStopRead`、`TestPhaseClock`）三平台都通过；SKIP 只有两个只供内层运行的测试。
- 中间提交 `079f9c7` 的诊断 run 40066611501 为 success（历史观测）。
- 作者本机与定向撤回 M39 ~ M44：见原型文档 §2.13。

**对 T1 / T6 证据的影响**：在 `1a6fec5` 之前，管道路径的有界收尾只在读取已进入系统调用时成立。现在阶段 J 的结束、封闭与 O3 在 `T_join` 之内到达；停止在 Unix 上于读取离开系统调用时、在 Windows 上于读取进入系统调用时完成。读取始终不进入系统调用时停止挂起，资源没有上界：这是**受限的正例**，资源边界与 C-4 / D20 属于冻结之前的门槛。C-2、C-3 与 D20 的证据概括已改写；Windows 的重复取消是新增的候选，待审查。

**常规 CI**：PR 打开期间每次推送都触发 `ci.yml`，被下一次推送按它自己的并发设置取消的运行如实记录在 PR 描述中；最终 HEAD 的常规 CI 同样记录在那里。常规 CI 不运行原型。

**保持不变**：14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；§84 ~ §96 与历史审查原文没有改动。

**状态**：B1、N4-R **已修复，待独立复审**；不宣布关闭或 GO。N6 经 Review 081 关闭（原时钟口径的范围内）。PR #34 保持未合并，Auto-fix / Auto-merge 关闭。

## 98．Review 082：PR #34 停止结果与 N4-R 的复审（NO-GO，B1-Ra、B1-Rb、N4-R）与定向修复

**审查**：[Review 082](reviews/codex-review-082.md)，原样归档为 `986ae6212924093f3730d0c5a322695abce02232`（`docs: archive Codex review 082 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `fe853b8906cf1458d2838da4431996672d9ac5dea75022a6a907f40fee99ab2e`，逐字节一致。

**审查对象**：PR #34，HEAD `0f1f20b8227bcec097a34c42cca393aaf22254cb`，base `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a`；修复代码 `125c4f70114c312828f7f8fc74734761e14c343a`。相对上轮 19 个文件，+1217/−84。

**结论**（按审查原文分层）：

- **PR #34：技术 NO-GO，当前不建议合并。** 最终常规 CI 已正式成功，技术修正条件仍未满足。
- **B1 的原反例已修复**：停止任务在阶段 J 开始之后异步启动，所有流共用一个截止，之后封闭、移交；三种受控读取调度在本机与三平台 CI 上都有相应支持。
- **B1 的完整契约不能关闭**：
  - **B1-Ra**：挂起停止的晚到错误不进入终判。`stopTask.until` 只返回是否结束；只有运行返回之前已结束的停止才放入 `stopErr`；挂起项交成只有 what / until 的 `leftover`；`adopt` 只看当时的 `stopErr`，`done` 只看是否结束，`lingering.pending` 直接丢掉已结束的项。审查方以受控 Closer 复现：同一错误早到为 invalid，晚到为 confirmed / exit 0。
  - **B1-Rb**：Windows 的重复取消与释放自有复制句柄的错误被静默丢弃。`ERROR_NOT_FOUND` 可以是正常竞争，但不能把其他失败一概忽略；审查方以 API 接缝注入 `ERROR_ACCESS_DENIED`，两处 `closeRead` 都返回 nil。
  - 最小修正：保留遗留任务的终态与错误，接手者（`adopt` / `done` 与 `lingering`）都要消费并保留晚到错误；区分正常的无请求竞争与实际的 API 故障，检查自有句柄的释放；同步现有证据概括。
- **N4-R 不能整体关闭，保持 NON-BLOCKER**：精确的“报告失败之后延迟退出”已修复，但迟到字节与同次的报告失败可以共存。审查方只保留 stdout 的读端，持有者写 stdout 成功、写 stderr 失败、报告也失败，结论仍为 refuted（非零，不是绿色的假通过）。
- **N6**：没有直接回退，保持 Review 081 的关闭范围。
- **Windows 重复取消**：只在受控的同步管道布局、成功的 API 与可释放的调度窗口下得到支持，不是任意取消失败、驱动或调度下的完成保证。
- **其余**：check-runner 草案头部与 PR #33 的标题仍是过时的措辞（非阻塞，沿用）；没有新增产品决定或接受风险；M-14、M-15、M-16 保持未解决。

**本轮授权**：原样归档 Review 082；在 PR #34 上定向修复 B1-Ra、B1-Rb 与 N4-R，回归先行，定向撤回；更新原型文档、处理记录与状态；提交、推送，运行三平台原型诊断与最终 HEAD 的常规 CI。

**不授权**：合并、验收、生产代码、Schema、迁移、根 `ci.yml`、诊断 workflow 的扩大、14 项产品选择、规范冻结、生产 WP2.5。

**回归先行**（`e4eadfa9ae3e6e4d21a7532bdd59ee33494f87cd`，判定没有改）：

- 门禁回归 `stop.*` 7 项（内层 `TestInnerStopError`）、直接检查 `TestLingeringFailedStop`、Windows 直接检查 `TestStopAPIErrors`（经接缝注入的两项 API 故障）、门禁回归 `report.epipe-mixed-fault`（审查方的组合故障）；`TestStopRead` 的正常行另要求停止没有错误。
- 三平台诊断 run 40072151282（push，attempt 1，completed/failure，预期；检出 `e4eadfa`）：运行之后、复制器之后、两路之一这 3 项停止错误得到 confirmed；登记放行；Windows 的两项 API 故障结果为 nil；组合故障得到 refuted。作者本机 Windows 相同。
- macOS 另有一项非设计内的失败：`report.epipe-unwritable-alive` 的内层结论正确，但外层普查读同一用户的另一进程（2088，不在内层的创建记录中）的环境得到 EINVAL。这是原型文档 §13 已登记的普查前提，没有重跑。

**修复**：

- **B1-Ra、B1-Rb**（`920220c2b3c7fee89b9a30371ac62734d0307c34`）：
  - 停止任务保留它遇到的全部错误，随到随记，之后的成功不清除；done 只表示关闭已返回。
  - `leftover.until` 返回“是否结束”与“迄今的错误”；移交的是任务本身，接手者在结果到达时读取。
  - `done` 读取每一项的结果，运行返回之后才到的错误、仍在运行的停止已遇到的错误都使观察失败。
  - O3 登记保留带错误结束的项；拒绝策略下据此拒绝后续运行，允许策略下放行而保留记录。
  - 阶段 D 中结束的流的关闭错误不再丢弃。
  - Windows：`CancelIoEx` 的 `ERROR_NOT_FOUND` 是预期竞态，其他失败立即记入停止的错误；关闭自有句柄失败使停止带错误，句柄不再关闭第二次；错误不写句柄的值。
- **N4-R**（`63b74324ee1e61efdcba9d98f3cd7ed810b291b6`）：没有 `epipe` 记录时，场景询问持有者；只有持有者回答 `ok`（只有迄今全部写入都成功的持有者才写），并且停止之后每一路都经保留的读端有迟到字节，才是 refuted；否则 invalid。等待回答期间到达的失败记录仍算数。两个存活注入的存活期由 5 s 改为 10 s。
- **修复之后补的检查**：
  - `ba32dbe2c378cad96a18acaadfd6f7095f9726a0`：`report.ok-unwritable` 与 `report.one-stream-seen`，使 N4-R 判定的两部分各有只由它拦住的用例；旧代码上没有这两个注入，不声称旧代码上的对照。
  - `2413634ba21838f7275ee4ddd523779f1409c9b6`：`TestStopAPIErrors/nothing-to-cancel`（确定地遇到 `ERROR_NOT_FOUND`）与 `report.epipe-late`（记录在等待回答期间到达）。对 `ba32dbe` 的定向撤回中，M50 只被偶然检出、M53 没有检出，因此补上。
  - **修复代码固定为 `2413634`。**
- 错误传播对应表与状态的区分见原型文档 §2.14。

**修复后的证据**：

- 三平台诊断 run 40074633075（push，attempt 1，**completed/failure**，没有重跑；检出 `2413634`）：
  - Ubuntu success：场景 38/38、门禁回归 52/52、组合 13/13（带 `-race`，没有数据竞争的报告）；
  - Windows success：29/29、48/48、13/13；
  - macOS **failure**：27/27、**42/43**、13/13。失败的是本轮没有改动的旧门禁回归 `report.truncated`：内层结论正确，判定之后的保险清理读同一用户的另一进程（1989，不在内层的创建记录中）的环境得到 EINVAL，记为保险清理的矛盾。与回归先行时的那次同类，普查与保险清理的代码本轮没有改动，方向保守。
  - 本轮新增的回归在三平台都通过；B2 的归属核验零动作、零信号，自有遗留照常清理；直接检查都通过；SKIP 只有三个只供内层运行的测试；并集 53。
- 中间提交的诊断（历史观测）：`63b7432` 的 run 40072621588、`ba32dbe` 的 run 40072845216，都是 success。
- 作者本机与定向撤回 M45 ~ M53：见原型文档 §2.14；每个变异只由对应的检查检出。

**待负责人决定的新事项**：macOS 普查读不到同一用户其他进程的环境（EINVAL）时判为普查错误，本轮因此出现两次门禁回归不通过（保守，不是假通过）。是否单独处理这一前提（例如在 macOS 上区分正在退出的进程），不由本轮决定；本轮没有为此重跑。

**常规 CI**：PR 打开期间每次推送都触发 `ci.yml`，被下一次推送按它自己的并发设置取消的运行如实记录在 PR 描述中；最终 HEAD 的常规 CI 同样记录在那里。常规 CI 不运行原型。

**保持不变**：14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；§84 ~ §97 与历史审查原文没有改动。

**状态**：B1（B1-Ra、B1-Rb）与 N4-R **已修复，待独立复审**；不宣布关闭或 GO。N6 保持关闭范围。PR #34 保持未合并，Auto-fix / Auto-merge 关闭。

## 99．Review 083：PR #34 停止错误链、N4-R 与 macOS 诊断失败的复审（B1-Ra、B1-Rb 关闭；N4-R-S、N-MAC）与定向修复

**审查**：[Review 083](reviews/codex-review-083.md)，原样归档为 `ab399e58c41beaa1917f01a8c11eb9cafb6bc8bd`（`docs: archive Codex review 083 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `1989204ff4f8d539ee04660e557e6390aaf35e67a1b95e81d31de7737efef8d4`，逐字节一致。

**审查对象**：PR #34，HEAD `b51cc5e571dc7f1cad00f49055d862e08a5833f0`，base `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a`；最终代码 `2413634ba21838f7275ee4ddd523779f1409c9b6`。相对上轮 16 个文件，+1130/−76。

**结论**（按审查原文分层）：

- **修复技术 GO**：B1-Ra、B1-Rb 可以关闭，没有新的 BLOCKER。
  - B1-Ra：移交的是持续保存结果的任务，挂起时已有的错误与完成之后的错误都保留；审查方的独立 Closer 用例中，晚到的错误在旧机制上为 confirmed / 0，在目标树上为 invalid / 1。
  - B1-Rb：`ERROR_NOT_FOUND` 与实际错误分开；失败持续保存，关闭副本失败进入结果；不把请求取消当作完成。释放失败不等于已释放。
  - Review 081 B1 在当前原型范围内可以整体关闭；不推广到任意驱动、任意句柄或调度停顿。挂起资源的上界与后续宿主策略仍是冻结之前的门槛。
- **N4-R 不能整体关闭，保持 NON-BLOCKER（N4-R-S）**：
  - 真实的 `ok` 只证明回答以前的写入。
  - 审查方的反例：持有者先写出真实的 `ok`，随后它自己的 stderr 被关闭，下一次写入与 `epipe` 报告都真实失败，并且都发生在场景最终判定之前。两路的迟到字节为 [1800, 1787]，结论仍为 refuted / 1。这是非零退出，没有绿色的假通过。
  - 最小方向：
    - 把正面回答变成有界、完整结束的观察段；
    - 回答失败，以及回答之后已经发生的工具或报告失败，都须进入 invalid；
    - 保留合法的 refuted。
- **N6**：没有直接回退，保持 Review 081 的关闭范围。
- **macOS 最终诊断的失败**（run 40074633075 的 `report.truncated`）是真实的外层工具错误，不是预期的截断拒绝本身失败。这是有效的保守拒绝，没有证据显示据此向 PID 1989 发过信号。相关 PID 的归属与完整清理的结果**未知**。
  - **N-MAC（NON-BLOCKER）**：原型文档 §2.14 与本记录 §98 中“其他进程”“不在内层的创建记录中”“与本轮的改动无关”等措辞过强，应收敛为“既有、未改动的调用链出现读取失败，归属未确认”，并分别写明阶段、未知项与证据缺口。
  - 回归先行那次（run 40072151282，PID 2088）是被测普查的失败（`final=unknown`），与最终诊断中保险清理的失败（此前 `final=ended`）阶段不同。同为 EINVAL 不证明同一根因。
- **当前合并条件未满足**：
  - 最终代码点的原型诊断为 completed/failure；
  - 不以中间提交的绿色、常规 CI 的 8/8 或“源码未改”豁免；
  - 修复技术 GO 与合并条件满足是两项判断。
- 没有新增产品决定、保证调整或接受风险；M-14、M-15、M-16 保持未解决。

**对 §98 的更正**（N-MAC；§98 原文不改）：

- 以下措辞过强：
  - §98“回归先行”一段中“外层普查读同一用户的另一进程（2088，不在内层的创建记录中）”；
  - “修复后的证据”一段中“读同一用户的另一进程（1989，不在内层的创建记录中）”与“与回归先行时的那次同类，普查与保险清理的代码本轮没有改动”；
  - “待负责人决定的新事项”中的“同一用户其他进程”。
- 更正如下：
  - **归属未知**：2088（被测普查）与 1989（保险清理）是否属于本轮都**未知**。
    - 同一用户、启动时间在范围内只是普查的筛选条件，不是归属证明。
    - “不在创建记录中”是从 `created` 计数与 `unowned` 列表推出的。日志没有打印完整的创建记录，不能据此独立确认；即使确认，也不能排除未登记的后代。
  - **原因未知**：EINVAL 的原因**未知**，不能排他地归为“正在退出”或某种系统机制。
  - **清理结果未知**：1989 那一次的 `final=ended` 是保险清理之前被测路径的结论；保险清理之后，普查范围内是否清理完整**未知**。
  - **因果未证明**：与本轮改动的因果**未证明**。源码没有改动是事实，但不构成因果排除。两次的阶段不同，同为 EINVAL 不证明同一根因。
  - 两次都是保守的方向，没有据此向任何 PID 发信号；都没有重跑。
- 能恢复的事实与已无法追溯的事实，分项列在原型文档 §2.15。

**本轮授权**：

- 原样归档 Review 083；
- 登记 B1-Ra、B1-Rb 经独立复审可以关闭，没有新的 BLOCKER；
- 登记 N4-R 不整体关闭，保持非阻塞；
- 在 PR #34 上定向修复 N4-R（`ok` 之后的工具失败），回归先行；
- macOS 先诊断，允许最小的原型诊断与确定性回归，不放宽归属门禁；
- 更新原型文档、处理记录与状态；
- 提交、推送；本轮代码改动正常触发的诊断可以运行。

**不授权**：

- 合并、验收；
- 生产代码、Schema、迁移、根 `ci.yml`；
- 扩大 workflow 的权限或预算；
- 改变 macOS 的归属规则、忽略 EINVAL、自动重试、接受未知的清理风险；
- 重跑旧的失败、为取得绿色提交空提交；
- 14 项产品选择、规范冻结、生产 WP2.5。

**回归先行**（`61a43db7f86ff6ceb4e8b1259cba07d7e63746cd`，判定没有改）：

- **门禁回归 `report.fail-after-ok`**：审查方的精确反例。
  - 两路读端都保留，持有者给出真实的 `ok`；
  - 随后它的 stderr 断开，下一次写入与 `epipe` 报告都真实失败，它仍存活；
  - 场景取得 `ok` 之后，限时等待测试自己的标记，再按原代码判定。
- **门禁回归 `report.ok-not-final`**：回答之后，持有者的输出仍开着。
- 这两项门禁回归在旧代码上都得到 refuted。
- **直接检查 `TestCensusUnreadable`**（darwin）：
  - 普查只覆盖测试自己的辅助进程，它的环境读取经接缝失败为 EINVAL；
  - 要求普查失败，说出该进程的身份与状态，不出现环境内容；状态也读不到时要写明；
  - 旧代码上失败：信息只有 PID 与错误。
- **三平台诊断** run 40084348430：push，attempt 1，completed/failure（预期）；检出 `61a43db`。
  - 三平台都只有上述预期的失败：门禁回归 Ubuntu 52/54、macOS 43/45、Windows 48/50；macOS 另有 `TestCensusUnreadable` 的两行。
  - 场景 38、27、29 confirmed；组合 13/13。
  - 这次 macOS 没有出现计划外的 EINVAL。
- **作者本机 Windows**：两项门禁回归同样得到 refuted。

**修复**：

- **N4-R-S**（`419e4f388ecc7cbdee2c853a337f4c13303a055e`）：询问就是观察的终点。
  - **观察区间**：从停止到持有者的最终回答。持有者看到询问之后不再写，先关闭自己的输出，再给出唯一的最终回答 `ok`。输出关不上或回答写不出，就没有回答。
  - **完成边界**：
    - 收到回答之后，每个保留读端须在 2 s 内读到 EOF（不等待地读），即每一路的写端都已关闭；
    - 迟到字节是每一路从停止到 EOF 实际携带的字节；
    - EOF 之后最后读一次报告；回答旁边若有失败记录，就不算回答。
  - **判定**：
    - 预期的现象发生（包括等待期间到达的 `epipe` 记录）：预测成立；
    - 工具正常而预测不成立（有最终回答、每一路都到 EOF、每一路都保留并有迟到字节）：refuted；
    - 其余情形是 invalid：回答之前的失败；回答之后观察没有结束；完成交接附近的故障；回答与失败记录并存；读取失败。
  - **随修复增加的回归** `report.close-fails`：持有者关不上输出，要求 invalid。旧代码没有这一步，所以不声称旧代码上的对照。
  - 只调整原型协议，不决定生产的产品规则。
- **N-MAC 的诊断**（`56234c364eaffc109f1ddf1cd0bd837420afcbf7`）：
  - **何时生效**：macOS 普查读不到某个进程的环境，复核又不能证明它已结束。
  - **普查照样失败**：不跳过、不重试、不改变归属规则。
  - **错误信息**：错误包装原错误，并附上以下事实：
    - 普查读到的列表项中的启动时间、状态、是否正在退出（`P_WEXIT`）、父进程、进程组；
    - 短命令名是否与本测试的程序相同（是 / 否 / 未知）；
    - 失败之后再读一次状态的结果，读不到时写明。
  - **不记录**参数、环境或进程名。
  - 这只为以后的同类失败留下可追溯的事实，不是归属的证明。
- **修复代码固定为 `56234c3`。** 判定表，以及哪些事实能恢复，见原型文档 §2.15。

**修复后的证据**：

- **三平台诊断** run 40084692941：push，attempt 1，**completed/success**，没有重跑；检出 `56234c3`。

  | 平台 | 场景 | 门禁回归 | 组合 |
  |---|---|---|---|
  | Ubuntu（带 `-race`，没有数据竞争的报告） | 38/38 | 55/55 | 13/13 |
  | Windows | 29/29 | 51/51 | 13/13 |
  | macOS | 27/27 | 46/46 | 13/13 |

  - 新增的回归与 `TestCensusUnreadable` 都通过；B1、N6 与归属保护的回归没有回退。
  - SKIP 只有三个只供内层运行的测试；场景 ID 的并集为 53。
  - 这次没有出现普查的 EINVAL。
- **作者本机与定向撤回**（M51 ~ M54、M56、M59）：详见原型文档 §2.15。
  - 每个变异只由对应的检查检出。
  - M59 第一次的写法编译失败，没有计为检出。
  - N-MAC 的诊断只能在 macOS 上运行，没有本机的撤回验证。

**macOS：旧的未知项与新证据分开**：

- **旧的未知项不变**：run 40072151282（2088）与 run 40074633075（1989）中，进程的归属、EINVAL 的原因、保险清理之后的完整性，仍然未知。当时的身份与状态没有记录，现在重查不能恢复。
- **新证据只说明新的代码点**：
  - run 40084692941 的成功与 `TestCensusUnreadable` 的通过，只说明 `56234c3` 在这次运行中没有遇到同类失败，诊断信息按设计给出；
  - 它们不替代、也不解释旧运行的未知结果；
  - 新的诊断信息还没有在真实的同类失败中出现过。
- **需要另行论证与批准、本轮没有实施的变化**：
  - 改变 macOS 的归属规则；
  - 忽略 EINVAL，或区分“正在退出”的进程之后放行；
  - 自动重试；
  - 接受未知的清理风险。

  本轮没有提出其中任何一项的实施方案。若负责人需要，先交回有证据的方案与取舍。

**合并条件**：

- 最终代码点的三平台诊断为 success。
- 但旧 macOS 失败本身没有得到有效诊断：当时的事实不可恢复，新的诊断信息也还没有遇到真实的同类失败。
- 按本轮指令，就此交付为 PR #34 **不具备合并条件**，不以最终诊断的成功或常规 CI 自行豁免。是否足够，由独立复审与负责人判断。

**常规 CI**：与 §98 相同，PR 打开期间各次推送的常规 CI，以及最终 HEAD 的常规 CI，都如实记录在 PR 描述中。常规 CI 不运行原型。

**保持不变**：

- 14 项产品选择未批准；
- 规范草案第 4 版未冻结；
- 生产 WP2.5 未授权；
- WP2.5 与 M2 未验收；
- M-14、M-15、M-16 未解决；
- §84 ~ §98 与历史审查原文没有改动。

**状态**：

- B1-Ra、B1-Rb 经 Review 083 可以关闭；Review 081 B1 在当前原型范围内整体关闭。
- N4-R-S 与 N-MAC **已修复，待独立复审**；不宣布关闭或 GO。
- N6 保持关闭范围。
- PR #34 保持未合并，Auto-fix / Auto-merge 关闭。

## 100．Review 084：PR #34 最终交接与 macOS 诊断补充的复审（NO-GO，B-N4；N1）与定向修复

**审查**：[Review 084](reviews/codex-review-084.md)，原样归档为 `df819911d96a75b5e40c104094b406ec01d4cb83`（`docs: archive Codex review 084 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `a97d7515c4833a86cb99dac7a48d288b1404c534ed3dffa92afc6ae814a6f41a`，逐字节一致。

**审查对象**：PR #34，HEAD `2414ae939b9a47c73fc7b440bf73d7afa520463a`，base `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a`；最终代码 `56234c364eaffc109f1ddf1cd0bd837420afcbf7`。相对上轮 13 个文件，+986/−30。

**结论**（按审查原文分层）：

- **技术 NO-GO，当前不建议合并**：新的 **BLOCKER B-N4**。
  - 有效的 `ok` 与真实的 `epipe` 记录同时到达时，冲突检查被跳过，内层场景错误地 **confirmed / exit 0**。
  - 原因：EOF、迟到字节与“回答并报告失败”的检查都在“`epipe` 仍为空”的条件之内；重读得到 `epipe` 之后整段被跳过，恰好一条 `epipe` 被判为预测成立。报告的最终校验只核对数量、重复与格式，不拒绝语义矛盾。
  - 审查方在仓库外以真实 helper 复现：真实的 `ok`，随后 stderr 关闭、下一次写入失败、`epipe` 报告写成功；内层 confirmed / 0，外层原有门禁正确检出。最小修正方向的隔离对照为 invalid / 1，健康、可靠反证与正常排空对照保持正确。
  - 这是有效性门禁直接形成的绿色结论，所以是 BLOCKER；没有重开 B1。
- **N4-R-S 不能整体关闭**：Review 083 的旧反例、输出未最终关闭与关闭失败都已拒绝。
- **N-MAC**：
  - 有限诊断关联到原列表实例与复核状态，保留原错误与保守失败，更正了历史归属推论；
  - 旧两次失败的归属、根因与完整清理仍未知，真实的 EINVAL 还没有由新诊断重现；
  - **旧 macOS 失败本身不构成永远不得合并的门槛**；当前不建议合并的依据是可复现的绿色假通过，不是要求反复运行直到旧 EINVAL 再现。
- **NON-BLOCKER N1**：`same program as this test` 实际只是短命令名前缀的启发式，标签比证据强；建议只改标签与文档。
- **B1、N6**：没有直接回退，保持原裁定与适用范围。
- **CI**：最终代码点的原型诊断 run 40084692941 与最终 HEAD 的常规 CI run 40087043315 都正式成功，但都没有覆盖 B-N4 的组合，不能替代技术修正。
- 没有新增产品决定、保证调整或接受风险；14 项产品选择未批准，规范未冻结，生产 WP2.5 未授权，WP2.5 / M2 未验收；M-14、M-15、M-16 保持未解决。

**对 §99 的更正**（§99 原文不改）：

- §99“合并条件”一段把“旧 macOS 失败本身没有得到有效诊断”作为“不具备合并条件”的依据。按 Review 084 与负责人本轮的指示，不可追溯本身不再作为永久的合并门槛。
- 旧的未知项继续保留，不称为已解决。
- §99 中 N4-R-S 修复后的证据成立，但没有覆盖“回答与报告成功的失败记录并存”的组合。

**本轮授权**：

- 原样归档 Review 084；
- 登记技术 NO-GO（`ok` 与真实 `epipe` 并存时冲突检查被跳过，内层 confirmed / 0）；
- 在 PR #34 上定向修复：先以真实链路的回归证明旧实现错误 confirmed / 0，再把最终证据收集与最终分类分开；
- 扫描同一最终分类链的其他提前返回，修复直接同源的冲突遗漏；
- 做定向撤回验证；更新原型文档、处理记录与状态；提交、推送；本轮代码改动正常触发的诊断可以运行。

**不授权**：

- 合并、验收；
- 生产代码、Schema、迁移、根 `ci.yml`；扩大 workflow 的权限或预算；
- 仅增加一次重读、延迟或调整分支顺序作为修复；
- 忽略 EINVAL、放宽归属、操作未知 PID；
- 14 项产品选择、规范冻结、生产 WP2.5。

**回归先行**（`7307d5d20835d0947caf21e3216f1130e75b359f`，判定没有改）：

- 10 项门禁回归，都经过真实的 helper、写入与报告路径，都要求 invalid：
  - `report.ok-epipe-reported`（审查方的精确反例）、`report.epipe-then-ok`（相反的顺序）、`report.ok-unasked`（第一次读取就找到 `epipe` 的路径）、`report.epipe-foreign`、`report.ok-foreign`（他人 PID 的记录）；
  - `report.ok-duplicate`、`report.ok-truncated`、`report.ok-malformed`（违反协议的回答）、`report.epipe-limit`、`report.ok-limit`（任一记录旁的工具失败）。
- 旧代码上，前 4 项为 **confirmed / exit 0**，`report.ok-foreign` 为 **refuted / 1**；其余 5 项旧代码已拒绝。
- 三平台诊断 [run 40095937741](https://github.com/XKHoshizora/specmetry/actions/runs/40095937741)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `7307d5d`。
  - 三平台都只有上表中预期的 5 项失败：门禁回归 Ubuntu 60/65、macOS 51/56、Windows 56/61。
  - 场景 38、27、29 confirmed；组合 13/13。
  - 没有出现普查的 EINVAL。
- 作者本机 Windows 相同。

**修复**：

- **B-N4**（`173d2eb`）：
  - **协议**：`epipe` 表示一次迟到写入失败；`ok` 只在被询问时、输出关闭之后给出，表示写入都没有失败；两者互斥，不论先后，并存即 invalid。
  - **收集**（`observeF1`）：等待失败记录；没有时询问，等待回答，在共用的 2 s 预算内把每个保留读端读到 EOF；途中的失败都记下，收集照常继续。
  - **最终读取**（`finalRead`）：持有者结束之后最后读一次报告，严格读取，截断是错误；结束没有确认时不算最终读取。
  - **分类**（`classifyF1`）：在全部证据之后，每条规则都检查之后才决定；成功不提前返回；任何收集中的失败使任一记录旁的结论为 invalid；记录必须是已核验持有者的 PID，同类至多一条。
  - **同一分类链的扫描**：第一次读取找到 `epipe` 的路径、回答路径中只在 `epipe` 为空时执行的检查、未核对的记录 PID、分类时持有者仍可能写入，都已由上述结构覆盖；其余的提前结束只能得到 invalid；没有扩大到其他场景。
  - **随修复增加**：直接检查 `TestF1Classify`（固定行，每行另以记录逆序再判一次）；门禁回归 `report.epipe-read-fails`、`report.ok-drain-fails`（场景自身的读取失败，旧代码没有最终读取，不声称旧代码上的对照）。
- **N1**（`1e82fbd`）：标签改为“短命令名是本测试名称的前缀（启发式，不是身份）”；普查照样失败，不据此动作。
- **修复代码固定为 `1e82fbd`。** 判定表见原型文档 §2.16。

**修复后的证据**：
- 三平台诊断 run 40096356396：push，attempt 1，**completed/success**，没有重跑；检出 `1e82fbd`。
  - Ubuntu 38/38、67/67、13/13（带 `-race`，没有数据竞争的报告）；Windows 29/29、63/63、13/13；macOS 27/27、58/58、13/13。
  - F1 的健康 confirmed、合法 refuted 与 `report.epipe-late` 保持；新增的 12 项都是 invalid；`TestF1Classify` 在三平台通过。
  - B1、N6 与归属保护没有回退；SKIP 只有三个只供内层运行的测试；场景 ID 的并集为 53；没有出现普查的 EINVAL。
- 作者本机与定向撤回 M60 ~ M62：见原型文档 §2.16。每个变异只由对应的检查检出，健康与合法反证的对照保持通过。

**macOS**：

- 旧的未知项不变：run 40072151282（2088）与 run 40074633075（1989）中，进程的归属、EINVAL 的原因、当时的完整清理，仍然未知。
- 本轮的两次诊断（run 40095937741、40096356396）都没有出现 EINVAL；新的诊断信息仍没有遇到真实的同类失败。N1 只改标签。
- 不可追溯本身不再作为永久的合并门槛（Review 084、负责人本轮的指示）；也不称为已解决。

**常规 CI**：PR 打开期间各次推送的常规 CI 与最终 HEAD 的常规 CI，如实记录在 PR 描述中；常规 CI 不运行原型。

**保持不变**：

- 14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；
- WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；
- §84 ~ §99 与历史审查原文没有改动。

**状态**：B-N4 与 N1 **已修复，待独立复审**；不宣布关闭或 GO。PR #34 保持未合并，Auto-fix / Auto-merge 关闭。

## 101．Review 085：PR #34 证据收集、最终分类与交接完整性的复审（NO-GO，B-N4-R；B-N4、N1 关闭）与定向修复

**审查**：[Review 085](reviews/codex-review-085.md)，原样归档为 `69afa87628d920ff4fc1f8300c4a9eac94932afd`（`docs: archive Codex review 085 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `77693601fde7c66c72f545b6933463e59f9599e29201b14ee6fd4649ee61e0c4`，逐字节一致。

**审查对象**：PR #34，HEAD `4d1e5754175a9b93d7c73faf9999f13ee6f2b3f0`，base `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a`；最终代码 `1e82fbdc21edf12255c7e72a2e981e5c0a6a648d`。相对上轮 12 个文件，+899/−109。

**结论**（按审查原文分层）：

- **原 B-N4 的冲突修复成立，可以关闭**：完整的 `ok` 与 `epipe` 不再因到达轮次或记录顺序绕过检查。这不等于整个 N4 收集链已无阻塞。
- **N1 可以关闭**。
- **技术 NO-GO，当前不建议合并：新的 BLOCKER B-N4-R**。
  - 场景把持有者被结束之后报告的稳定，等同于工具已经完成交接。
  - 完整的 `epipe` 记录存在时，报告操作随后失败、或已到达的工具失败的报告被结束打断，内层仍可 **confirmed / exit 0**。
  - 统一分类正确处理了已经收集到的错误，但没有证明错误都已收集。
  - 审查方的三个反例：报告 Close 失败；期限故障点的报告被结束截断（放开闸门的反向对照为 invalid / 1）；期限报告真实写入失败。
  - 最小修正：为接受证据定义可验证的工具完成 / 失败状态，在 `epipe` 与 `ok` 两条路径都适用；主动结束只能作为清理。
- **观察终点的表述需要修正**：观察期间已发生的失败、报告尚未完成，与观察之后的无关行为，不能合并排除。
- **B1、N6**：没有直接回退。
- **CI**：最终代码点的原型诊断 run 40096356396 与最终 HEAD 的常规 CI run 40098220152 都正式成功，但都不覆盖本轮的组合反例。
- 没有新增产品决定、保证调整或接受风险。14 项产品选择未批准，规范未冻结，生产 WP2.5 未授权，WP2.5 / M2 未验收；M-14、M-15、M-16 保持未解决。

**对 §100 的更正**（§100 原文不改）：

- §100 与原型文档 §2.16 称“持有者在被结束之前没有写出的记录……此后的行为不属于这次观察”。这一说法不成立：它排除了观察期间已发生、但尚未报告或报告失败的错误。
- 观察的终点改为持有者的完成提交，见原型文档 §2.17。
- §100 中 B-N4 修复后的证据成立，但只覆盖已经写出的记录之间的裁定。

**本轮授权**：

- 原样归档 Review 085；
- 登记原 B-N4 冲突修复成立、N1 可关闭；B-N4-R 为当前阻塞，N4 系列未整体关闭；
- 先整理完成协议的状态与证据表；
- 在 PR #34 上定向修复：先以真实调用链的回归证明旧实现错误 confirmed / 0，再修改实现；
- 做定向撤回验证；更新原型文档、处理记录与状态；提交、推送；本轮代码改动正常触发的诊断可以运行。

**不授权**：

- 合并、验收；
- 生产代码、Schema、迁移、根 `ci.yml`；扩大 workflow 的权限或预算；
- 只增加一次重读、延迟，或针对注入名称的特殊判定；
- 把观察期间已经发生、但尚未报告的错误，通过重新定义观察终点排除在外；
- 14 项产品选择、规范冻结、生产 WP2.5。

**完成协议与错误传播**：见原型文档 §2.17 的两张表。要点：

- **预测现象**：最终报告中只有一条属于已核验持有者的完整 `epipe` 记录。
- **工具完成**：持有者的完成提交（`<report>.done`，内容为它的 PID）。它是持有者的最后一步，只在全部报告操作成功、没有工具失败之后写入，以改名原子地出现。
- **报告失败的保存与传递**：持有者不提交完成；场景的完成等待记下“没有提交”，分类判 invalid。写出的失败记录与格式错误由最终读取记下。
- **强制结束**：只在完成提交之后、或 3 s 期限之后进行。之前的结束没有提交，判 invalid。结束只是清理。
- **缺少完成证据**：confirmed 与 refuted 都判 invalid。

**回归先行**（`acfee1fa9d2a4665b2f14858aa526b833dc63130`，判定没有改）：

- 7 项门禁回归，都经过真实的 helper、报告、结束确认与场景分类路径，都要求 invalid：
  - 完整记录之后报告的 Close 失败：`report.epipe-close-fails`、`report.ok-close-fails`；
  - 已到达期限故障点、它的报告被结束截断：`report.epipe-limit-cut`、`report.ok-limit-cut`；
  - 期限报告真实写入失败：`report.epipe-limit-unwritable`、`report.ok-limit-unwritable`；
  - 记录本身被结束截断：`report.epipe-cut`。
- 旧代码上，`epipe` 的 3 项为 **confirmed / exit 0**，`ok` 的 3 项为 **refuted / 1**；`report.epipe-cut` 旧代码已拒绝。
- 三平台诊断 [run 40102730930](https://github.com/XKHoshizora/specmetry/actions/runs/40102730930)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `acfee1f`。
  - 三平台都只有上表中预期的 6 项失败：门禁回归 Ubuntu 68/74、macOS 59/65、Windows 64/70。
  - 场景 38、27、29 confirmed；组合 13/13。
  - 没有出现普查的 EINVAL。
- 作者本机 Windows 相同。

**修复**（`fb3f56d`）：

- **持有者**：`complete()` 为最后一步，写 `done <pid>`、Close、改名为 `<report>.done`。记录、其后的工具操作或提交本身的任一失败，都没有完成提交。
- **场景**：
  - 两条路径上，结束持有者之前都在 3 s 内等待完成提交，并严格读取；
  - confirmed 与 refuted 都需要已核验持有者的完成提交。
- **随修复增加**：门禁回归 `report.done-close-fails`、`report.done-foreign`；`TestF1Classify` 增加 4 行。
- **修复代码固定为 `fb3f56d`。**

**修复后的证据**：
- 三平台诊断 run 40103135762：push，attempt 1，**completed/success**，没有重跑；检出 `fb3f56d`。
  - Ubuntu 38/38、76/76、13/13（带 `-race`，没有数据竞争的报告）；Windows 29/29、72/72、13/13；macOS 27/27、67/67、13/13。
  - F1 的健康 confirmed、合法 refuted 与 `report.epipe-late` 保持；新增的 9 项都是 invalid；`TestF1Classify`（23 行）在三平台通过。
  - B1、N6、已关闭的冲突检查与归属保护都没有回退；SKIP 只有三个只供内层运行的测试；场景 ID 的并集为 53；没有出现普查的 EINVAL。
- 作者本机与定向撤回 M63 ~ M65：见原型文档 §2.17。每个变异只由对应的检查检出，健康与合法反证的对照保持通过。

**macOS**：

- 旧的未知项不变：2088 与 1989 的归属、EINVAL 的原因、当时的完整清理，仍然未知。
- 本轮的诊断没有出现 EINVAL。
- 继续登记，不回填为已解决，也不作为永久合并门槛。

**常规 CI**：PR 打开期间各次推送的常规 CI 与最终 HEAD 的常规 CI，如实记录在 PR 描述中；常规 CI 不运行原型。

**保持不变**：

- 14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；
- WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；
- §84 ~ §100 与历史审查原文没有改动。

**状态**：

- 原 B-N4 与 N1 经 Review 085 可以关闭。
- B-N4-R **已修复，待独立复审**；N4 系列不宣布整体关闭，不宣布 GO。
- PR #34 保持未合并，Auto-fix / Auto-merge 关闭。

## 102．Review 086：PR #34 完成协议与期限回调的复审（NO-GO，B-N4-R/T、B-N4-R/I；N-RENAME）与定向修复

**审查**：[Review 086](reviews/codex-review-086.md)，原样归档为 `1a1caf279e3f8eb878354d42ab44bac7d94dfbbb`（`docs: archive Codex review 086 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `26c7c1c0fd585627c961ba57e6f27945343f02628f9d1f0bd612b96d128bedd2`，逐字节一致。

**审查对象**：PR #34，HEAD `bc5ad7841e31eefe08194019eed3282b72c2b7c0`，base `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a`；最终代码 `fb3f56d8caebc1bc5d464f1db4cbec29a8faf24b`。相对上轮 10 个文件，+781/−41。

**结论**（按审查原文分层）：

- **Review 085 的三类原反例已经修复**；健康与合法反证的对照保持。
- **技术 NO-GO，当前不建议合并；B-N4-R 不能整体关闭**，N4 系列尚未收口：
  - **B-N4-R/T（BLOCKER）**：期限回调已经进入、`limit` 报告仍在进行时，主路径仍能发布 `done`。审查方以缩短到 100 ms 的真实计时器加可释放闸门复现：`epipe` 路径 confirmed / 0，`ok` 路径 refuted / 1；先放行回调的反向对照为 invalid / 1。
  - **B-N4-R/I（BLOCKER）**：故障分支忽略报告返回值；`ok` 报告真实写入失败之后，仍能生成 `epipe` 与有效的 `done`，内层 confirmed / 0。反序分支同样忽略返回值，只指出了源码风险。
- **NON-BLOCKER N-RENAME**：同目录改名的“原子：全有或全无”应限定在已验证的条件之内。
- **B1、N6、已关闭的 B-N4 / N1**：没有直接回退。
- **CI**：原型诊断 run 40103135762 与常规 CI run 40105337157 都正式成功，但不覆盖本轮的组合。
- 没有新增产品决定、保证调整或接受风险；14 项产品选择未批准，规范未冻结，生产 WP2.5 未授权，WP2.5 / M2 未验收；M-14、M-15、M-16 保持未解决。

**对 §101 的更正**（§101 原文不改）：

- §101 与原型文档 §2.17 称“在它之前发生的任何失败都会使它缺失”“提交是持有者最后一个可能失败的操作”。这只对主路径上返回值受检查的串行操作成立，不覆盖并发的期限回调与被忽略的报告失败。
- “以改名原子地出现”收窄为：场景临时目录内的同目录改名，加上严格的内容校验，只在已验证的布局上成立（原型文档 §2.18）。

**本轮授权**：

- 原样归档 Review 086；登记原三类反例已修、B-N4-R 仍未关闭；
- 统一完成协议：先列出持有者的全部相关入口，明确共享状态、所有权与先后关系，再实现；
- 测试先行，以真实链路回归证明旧实现错误发布 `done` 并 confirmed / 0，再修复；
- 做定向撤回验证；更新完成协议与错误传播表；提交、推送；本轮代码改动正常触发的诊断可以运行。

**不授权**：

- 合并、验收；
- 生产代码、Schema、迁移、根 `ci.yml`；扩大 workflow 的权限或预算；
- 只在部分调用点补检查、按注入名称决定结论、增加等待预算或容差；
- 14 项产品选择、规范冻结、生产 WP2.5。

**统一完成协议**（原型文档 §2.18 的三张表）：

- **入口**：主路径、期限回调、模拟期限点、报告操作、完成提交，都经过 helper 进程的 `toolState`，共用一把锁。
- **错误保存**：任何报告操作的失败都在报告层保存，不论调用者是否检查；输出关闭、期限进入、完成发布本身的失败同样保存。只保存第一个，之后的成功不清除。
- **期限与完成的裁定**：
  - 回调的第一步在锁下裁定。完成已发布，则只结束进程；否则工具失败，此后不能发布完成；
  - 完成在整个发布期间持有锁，因此回调只能看到终态；
  - 同一状态不会既是“已发布的完成”又是“期限失败”。
- **完成之后**：不再写入属于本次观察的报告。
- **最终结果的消费者**：场景的完成等待、最终读取与分类，规则不变；超期或未知一律 invalid。
- **同步**：没有死锁、无界等待或无人负责的任务；期限与预算没有改变。

**回归先行**（`750e1390d6cc8cc211c144469d582cad239aca77`，判定没有改）：

- 门禁回归 `report.limit-race-epipe`、`report.limit-race-ok`：真实计时器提前到 100 ms，回调进入后在可释放闸门处等待，完成在它进入之后发布。
- 门禁回归 `report.ok-fails-epipe-reported`、`report.epipe-fails-then-ok`：报告真实失败、调用者忽略返回值，之后另一条记录成功并发布完成。
- 对照 `report.done-rename-fails`。
- 旧代码上：`limit-race-epipe` 与 `ok-fails-epipe-reported` 为 **confirmed / exit 0**；`limit-race-ok` 与 `epipe-fails-then-ok` 为 **refuted / 1**；改名失败对照旧代码已拒绝。
- `report.ok-fails-epipe-reported` 第一次运行的夹具漏了保留读端（命中数 0），不计为复现；修正之后命中 3 次，并得到 confirmed / 0。
- 三平台诊断 [run 40109659805](https://github.com/XKHoshizora/specmetry/actions/runs/40109659805)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `750e139`。
  - 三平台都只有上表中预期的 4 项失败：门禁回归 Ubuntu 77/81、macOS 68/72、Windows 73/77。
  - 场景 38、27、29 confirmed；组合 13/13。
  - 没有出现普查的 EINVAL。
- 作者本机 Windows 相同。

**修复**（`9e3b7ca`）：

- 新增 `tool_test.go`（`toolState`）。
- `report` 成为通往报告的唯一入口；期限回调与模拟期限点经 `limitEntered`；输出关闭经 `failed`；`complete` 经 `tool.complete`。
- 扫描了全部 `report` 调用：失败在报告层保存，调用点是否检查返回值都不再影响完成。
- **随修复增加**：门禁回归 `report.limit-after-done`（完成先发布，回调后进入并裁定，只结束持有者，要求 confirmed）；直接检查 `TestToolState`（5 组）。
- 分类器没有改动。
- **夹具更正**（`f728e11`）：`9e3b7ca` 的诊断 run 40110180232 中，`report.limit-after-done` 在 Ubuntu 与 Windows 上只有 1 次命中，回调尚未裁定持有者就被结束，这一项的通过是空的。现在场景在结束持有者之前等待回调的裁定标记，缺少它判 invalid。
- **修复代码固定为 `f728e11`**（相对 `9e3b7ca` 只有这一处夹具差异）。

**修复后的证据**：
- 三平台诊断 run 40110917799：push，attempt 1，**completed/success**，没有重跑；检出 `f728e11`。
  - Ubuntu 38/38、82/82、13/13（带 `-race`，没有数据竞争的报告）；Windows 29/29、78/78、13/13；macOS 27/27、73/73、13/13。
  - F1 的健康 confirmed、合法 refuted、`report.epipe-late` 与 `report.limit-after-done`（三平台 2 次命中）都是 confirmed 或 refuted；本轮 4 项回归都是 invalid；`TestToolState` 与 `TestF1Classify` 在三平台通过。
  - 已关闭的冲突检查、PID 关联、B1、N6 与归属保护都没有回退；SKIP 只有三个只供内层运行的测试；场景 ID 的并集为 53；没有出现普查的 EINVAL。
- `9e3b7ca` 的诊断 run 40110180232 保留为历史观测（`report.limit-after-done` 在两个平台上空通过）。
- 作者本机与定向撤回 M66 ~ M72：见原型文档 §2.18。每个变异只由对应的检查检出，健康与合法反证的对照保持通过。M68、M71 单独撤回只被 `TestToolState` 检出；两者同时撤回（M72）时真实链路检出。

**macOS**：

- 旧的未知项不变：2088 与 1989 的归属、EINVAL 的原因、当时的完整清理，仍然未知。
- 本轮的诊断没有出现 EINVAL。
- 继续登记，不回填为已解决，也不作为永久合并门槛。

**常规 CI**：PR 打开期间各次推送的常规 CI 与最终 HEAD 的常规 CI，如实记录在 PR 描述中；常规 CI 不运行原型。

**保持不变**：

- 14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；
- WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；
- §84 ~ §101 与历史审查原文没有改动。

**状态**：B-N4-R（B-N4-R/T、B-N4-R/I）**已修复，待独立复审**；不宣布关闭或 GO。PR #34 保持未合并，Auto-fix / Auto-merge 关闭。

## 103．Review 087：PR #34 统一工具状态与完成裁定的复审（技术 GO；B-N4-R/T、B-N4-R/I 关闭）与收尾

**审查**：[Review 087](reviews/codex-review-087.md)，原样归档为 `44cf91bde74e670d1034c7fa24c48e339478f4dd`（`docs: archive Codex review 087 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `d1f3a5e2300b6f44fdc4da205a13c21b82e9610a0b3f56321ce7a7d0b3ac9178`，逐字节一致。

**审查对象**：PR #34，HEAD `245b51bea8bd57e28c2716a86c67ca18a91bafaf`，base `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a`；最终代码 `f728e11498a8b4bc5b71ec3cbff1618c795f6969`。相对上轮 10 个文件，+932/−21。

**结论**（按审查原文分层）：

- **技术 GO**：B-N4-R/T、B-N4-R/I 分别关闭。N4 系列只在当前受控 helper 协议范围内收口：受信任的 helper、固定的入口与受控的文件布局。完成标记不是不可信写者的认证、断电持久性或任意宿主协议的证明。
- **安全的依据**：实际的调用拓扑，不是类型中的 mutex。以后若改变拓扑，须重新核查。
- **“回调进入”分三个事件**：函数开始、在锁下裁定、实际报告。有效性边界由在锁下的裁定线性化。
- **没有新的 BLOCKER。**
- **NON-BLOCKER N-087-1**：`TestToolState` 的“发布进行中回调进入”一组，没有确证回调已经进入。
- **NON-BLOCKER N-087-2**：不能把“没有协议循环”写成无条件的“没有无界等待”；持锁执行的文件 I/O 没有硬截止。
- **N-RENAME 可以关闭。**
- **CI**：最终常规 CI 在审查时满足合并条件，可以建议合并固定对象的原型 PR #34。这不能改写为首轮成功，也不表示存储层的维护问题已经解决。
- 没有新增产品决定或接受风险；方案 B 的局部范围不变；旧 macOS 的未知项继续保留。14 项产品选择未批准，规范未冻结，生产 WP2.5 未授权，WP2.5 / M2 未验收；M-14、M-15、M-16 保持未解决。

**对 §102 的更正**（§102 原文不改）：

- §102“统一完成协议”一段中的“没有死锁、无界等待或无人负责的任务”更正为：
  - 没有引入协议级的循环等待，锁内不调用其他入口；
  - 父端的期限不变；
  - 文件 I/O 与系统调度不提供硬实时保证（N-087-2）。
- §102 与原型文档 §2.18 对 `TestToolState` 第 5 组的描述收窄为：它检查锁与终态的规则，没有确证回调已在发布期间进入（N-087-1）。

**本轮授权**：

- 原样归档 Review 087，登记审查结论；
- 只涉及准确措辞的非阻塞建议在本轮更正，涉及代码或新机制的先登记；
- 登记 CI 的历史与 M-15 的新观测；
- 同步活跃状态，更正草案头部与 PR #33 / #34 标题中的过时状态（只作事实更新）；
- 只增加文档提交；最终 HEAD 的常规 CI 正式成功之后，以 merge commit 条件合并 PR #34，并做合并之后的核验。

**不授权**：

- 修改源码、原型、测试、依赖、Schema、迁移或 workflow；
- 为纯文档收尾额外制造原型诊断运行；把 run 40116851180 的重跑许可延用到新的运行；
- 14 项产品选择、规范冻结、生产 WP2.5、WP2.5 / M2 验收，以及后续业务实现。

**两项非阻塞建议的处理**：

- **N-087-2**（措辞）：原型文档 §2.18 的表格中，在原处加更正说明；§102 的更正记在本节。
- **N-087-1**（涉及测试代码）：只登记为原型文档 §13 的后续事项（用入口握手收紧 `TestToolState` 第 5 组），本轮没有实现；§2.18 的描述已收窄。
- **N-RENAME**：关闭。

**最终常规 CI 的历史**（run 40116851180，对 `245b51b`）：

- **attempt 1**：正式 completed/failure、**7/8**。唯一的失败是 Windows 的 `TestSeqIsUniqueAcrossConnections`（`store_test.go:546: database is locked (5) (SQLITE_BUSY)`）。
- **attempt 2**：负责人授权的一次 `gh run rerun 40116851180 --failed`，正式 completed/success。
  - 只有 Windows 的 Go 测试实际重新执行（作业 120946480814）；其余七项沿用 attempt 1 的结果。
  - 重跑检出同一合并快照 `c0ec2e4`；Windows 全部测试 2999 PASS、10 SKIP、0 FAIL。
- 不能写成首轮 8/8，或三平台全部重跑。逐项见原型文档 §2.19。

**M-15 的新观测**：

- **签名**：`TestSeqIsUniqueAcrossConnections`，`store_test.go:546`，`SQLITE_BUSY (5)`，与 §77 的登记一致。
- **未验证**：根因，以及是否与以往的观测同根因。
- 本 PR 没有改动 `internal/`；重跑成功不关闭这一事项，它也不是已接受的风险。
- 已补入维护登记：`m1-exit-evidence.md` 的 M-15 一行。

**收尾的范围**：相对受审 HEAD `245b51b` 的新增提交只有文档（Review 087 的归档、原型文档、本记录、维护登记、AGENTS.md、实施计划、check-runner 草案头部的事实更新）。源码、原型、测试、依赖、Schema、迁移与 workflow 与受审对象完全相同；原型与 workflow 与已诊断的 `f728e11` 没有差异，所以原型诊断 run 40110917799 仍然适用。

**合并**：只在全部条件成立时，以 merge commit 合并。合并与合并之后的核验写在 PR #34 的描述中，不为登记它们而向 master 追加提交。

**保持不变**：

- 14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；
- WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；
- §84 ~ §102 与历史审查原文没有改动。

**状态**：技术审查通过，待合并核验。

## 104．PR #34 合并之后的 master CI 失败：一次受控重跑与新维护事项 M-17

**PR #34 的合并**：

- 以 merge commit 合并为 `cc4514a012297a4123654616ece02e9d649e0cc7`（2026-10-24 18:43:58 UTC）。
- 父提交依次为原 master `5e9a42c`、最终 HEAD `b392cd1`；merge 的树 `c6c8cee` 与最终 HEAD 的树相同，diff 为空。
- 合并前的最终 HEAD 常规 CI 为 run 40120953702，attempt 1，8/8。
- 合并与核验的完整记录在 PR #34 的描述中。

**master push CI**（[run 40122051541](https://github.com/XKHoshizora/specmetry/actions/runs/40122051541)，对 `cc4514a`）：

- **attempt 1**：正式 completed/failure，7/8。8 个作业都检出 `cc4514a` 本身。唯一的失败是 Windows 的 `internal/cli` `TestSourceRemoveInTerminal`（`source_test.go:542`，`cli.test.exe did not exit within 1m0s`）。
- **attempt 2**：负责人对本次失败的单独授权，`gh run rerun 40122051541 --failed`，执行一次；此前的 M-15 重跑许可没有复用。
  - 结果正式 completed/success。
  - 只有 Go tests (windows-latest) 实际重新执行（作业 120968641840）；其余七项沿用 attempt 1 的结果。
  - 重跑检出 `cc4514a`。Windows 全部测试 2999 PASS、10 SKIP、0 FAIL；PASS 集合等于 attempt 1 的加上原失败项，SKIP 集合与 attempt 1 相同。
- **合并之后的核验**：以 attempt 2 完成，同时保留 attempt 1 的失败与实际重跑的范围。

**新维护事项 M-17**（登记于 `m1-exit-evidence.md` §9；诊断报告 [m17-diagnostics.md](m17-diagnostics.md)）：

- **记录**：精确的 run、作业、merge 与失败位置。
- **发生在**：第一次“空输入取消”调用。
- **已观察到**：提示已完整输出，60 秒内没有观察到子进程退出。
- **不能确认**：提示是否被匹配、是否尝试发送、发送的返回值、子进程是否收到输入。现有日志不足以确认，因为测试工具不检查写入的返回值。
- **与其他事项分开**：与 M-15、M-16 分开登记。
- **状态**：根因未知，未解决，不是已接受的风险。A 阶段重跑的成功不关闭它。
- 同一内容在 PR CI 中通过，不据此定性为环境问题。

**有界诊断**（仓库外，作者本机 Windows 11）：

- **做法**：只在导出副本的 `internal/testutil/pty.go` 中增加观测点与两项受控故障，生产代码、依赖、预算与 60 秒的期限都没有改动。
- **原生运行**：`TestSourceRemoveInTerminal` 40 次、无需提示的退出 20 次，都通过，没有复现。
- **受控注入**：
  - 提示跨读取分块（10 次），都通过；
  - 回答的写入失败、短写（各 1 次），都产生与原始失败相同的报告。
- **结论**：
  - 测试工具忽略写入的返回值，因此“回答没有写入”与原始失败在报告上无法区分；
  - 静态阅读表明，提示大概率被匹配到，写入大概率被尝试了；
  - 原始失败属于哪一种情形，仍然未知。
- **修复**：没有得到明确的修复方案，本轮不实施修复。可考虑的后续方向（只增加观测、在 Windows CI 上有界重复）需另行授权与审查。

**其他**：

- Review 087 的 N-087-1 继续作为原型的非阻塞后续事项，登记在原型文档 §13，不混入本记录对应的 PR，也不改原型代码。
- 本记录所在的 PR 只登记与诊断，不合并、不修复。
- AGENTS.md 与实施计划中 PR #34 的状态（“技术审查通过，待合并核验”）没有在本 PR 中同步，留待经授权的文档收尾。

**保持不变**：14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16 未解决；§84 ~ §103 与历史审查原文没有改动。

**状态**：M-17 已登记，未解决；诊断结论见上，根因未知。

## 105．Review 088：PR #35 的 M-17 维护登记与诊断文档（技术 GO）与收尾

**审查**：[Review 088](reviews/codex-review-088.md)，原样归档为 `ca4913e23ad066a01a7a595f4d34d00b7faafe4b`（`docs: archive Codex review 088 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `fcb78ddb648e737674d490724e8f2bc2a7d425047fd65f49b52fa668fb017347`，逐字节一致。

**审查对象**：PR #35，HEAD `fde01f0ec067aa791d0095adfefae658469b4774`（树 `d525ff7`），base `cc4514a012297a4123654616ece02e9d649e0cc7`；三份文档，+182/−0。

**结论**（按审查原文分层）：

- **技术 GO**，可以合并固定对象的文档 PR #35；没有 BLOCKER。
- M-17 的来源、失败签名、独立编号及“根因未知、未解决、不是已接受的风险”的状态准确。
- **独立核实**：原始失败、受控重跑的范围与结果、PR #34 合并与同树通过的身份；写入返回值被忽略、累积缓冲匹配、确认输入与 go-pty v0.2.3 的调用链。
- **验证边界**：
  - 作者在仓库外的观测版本与 X1 ~ X4 的原始日志，审查方没有定位、读取或复跑；所列的哈希与时延没有独立核验，不能写成独立复现。缺少这些文件不妨碍登记一个尚未查明的问题，也不要求恢复它们才准许合并。
  - 审查方另做了纯缓冲 / 匹配循环的最小验证（85 个两段切分位置、逐字节与 7 字节分块、Expect 内插入控制序列的对照、三种受控 Writer），首次运行即通过。这是源码片段的有界机制验证，不是 ConPTY、真实 CLI 超时或 X1 ~ X4 的复现。
  - “大概率被匹配、写入”是静态推断，没有动态证据，也没有统计依据。
  - 同树通过只排除已提交内容的差异，不能排除运行时数据、环境、调度等差异，也不把其中任何一个确定为根因。
  - PR CI run 40127618277 的成功只证明提交内容的检查，不验证仓库外的观测版本，也不关闭 M-17。
- **三项 NON-BLOCKER**（措辞）：
  - **N-088-1**：限定“分块不会失败”的前提；
  - **N-088-2**：区分“只记录返回值”与“按错误提前失败”；
  - **N-088-3**：收紧摘要中的次数与“相同报告”。
- 没有新增产品决定或接受风险；M-17 与 M-15、M-16 按签名分开。

**三项建议的处理**（都是措辞，本轮更正；没有涉及代码的改动）：

- **N-088-1**：诊断报告第 4 节与第 5 节改为：当前的累积缓冲算法，在相同的输入字节、读取与调度都有效的前提下，不会仅因读取边界的拆分而漏匹配；本机 7 字节分块的 10 次实验没有复现。这不是对所有 ConPTY 渲染、窗口宽度、控制序列或终端布局的保证。
- **N-088-2**：诊断报告第 5 节把后续方向拆成两项：
  - “只记录”匹配、写入的返回值与时间，属于观测；
  - “按错误提前失败”会改变测试工具的失败路径、时间与收尾行为，须另行授权与审查。
  两项都没有实施。
- **N-088-3**：
  - 维护登记的 M-17 行与诊断报告，原生次数写为“原失败测试 40 次 + 无需提示的测试 20 次”，不是原失败路径重复 60 次；
  - “相同的报告”限定为相同的超时错误签名与类似的输出末尾，不是整份输出的逐字节相等；这一签名不具排他性，不能证明原始失败属于注入的原因。
- **其他更正**：
  - 诊断报告的开头写明证据的身份：哪些经审查方独立核实，哪些是作者的仓库外证据；
  - X4b 的“0 字节、nil 错误”是观测包装层的受控故障，不表示原生写入会这样返回；
  - “读取 500 ms”改为“最多再等 500 ms”。
- **§104 的更正**：§104 原文不改，以上措辞的更正记在本节：
  - §104 的“原生运行 `TestSourceRemoveInTerminal` 40 次、无需提示的退出 20 次”保持准确；
  - “都产生与原始失败相同的报告”按 N-088-3 理解为相同的超时签名与类似的输出末尾；
  - “提示跨读取分块……都通过”按 N-088-1 限定前提；
  - “只增加观测”的后续方向按 N-088-2 拆分。
- **M-17 的状态不变**：仍然未解决，根因未知；不因诊断文档入库或 CI 成功而关闭。

**PR #34 状态的同步**（事实更新，不改写历史记录）：AGENTS.md 与实施计划中 PR #34 的“技术审查通过，待合并核验”，更新为“已合并为 `cc4514a`，合并后核验完成”：

- master push CI run 40122051541 的 attempt 1 为 7/8；
- attempt 2 只有 Windows 实际重跑，其余七项沿用 attempt 1 的结果。

原型文档 §2.19 与 check-runner 草案头部仍写着合并之前的状态，没有在本轮改动，留待另行的文档收尾。

**本轮收尾的范围**：

- 相对 PR #35 受审 HEAD `fde01f0` 的新增提交只有文档：Review 088 的归档、诊断报告、维护登记、本记录、AGENTS.md 与实施计划。
- 代码、测试、依赖与 workflow 没有变化；不触发原型诊断。

**保持不变**：

- 14 项产品选择未批准；规范草案第 4 版未冻结；生产 WP2.5 未授权；
- WP2.5 与 M2 未验收；M-14、M-15、M-16、M-17 未解决；
- §84 ~ §104 与历史审查原文没有改动。

**状态**：PR #35 技术审查通过，待合并核验。

## 106．产品负责人 2026-10-25 的决定：WP2.5 的十四项产品选择与 C-4；规范第 5 版草案

**决定**：产品负责人于 2026-10-25 批准 WP2.5 规范草案第 4 版 §17.3 的十四项产品选择（D1 ~ D9、D11、D13、D18、D19、D20）与原型候选 C-4（与 D20 一起），并授权起草规范第 5 版。决定的依据是 §105 之后提供的只读决定单（草案第 4 版 §17.3、[wp2.5-prototype.md](wp2.5-prototype.md) §9、§10）。本节只登记这次决定；§84 ~ §105 与历史审查原文不变，其中“未批准”“候选”的表述按它们当时的状态保留，不改写成当时已经获批。

**PR #35 状态的同步**（事实更新）：PR #35 已于 2026-10-25 03:03:40 UTC 以 merge commit 合并为 `bd45583867b36ef63037690cba383b7a681ac13b`（`gh pr merge 35 --merge --match-head-commit 8af21ccbb7482e4de4597228b9faaf322aa12e35`）。父提交为 `cc4514a` 与 `8af21cc`，合并树与 HEAD 的树 `5378ae6` 相同；master push CI run 40156803607 为 attempt 1、completed/success、8/8，全部作业检出 `bd45583` 本身。§105 末尾的“技术审查通过，待合并核验”是当时的状态，不改写。M-17 仍未解决。

### 一、十四项决定

| ID | 决定 | 理由 | 代价 |
|---|---|---|---|
| D1 | 所有平台默认 `sh`，不回退 `cmd`。任何所需解释器不可用 → 整批不执行、不记录，退出码 6 | 同一个配置哈希不应因机器而执行语法不同的程序；`cmd` 对多行命令只执行第一行且不报错，静默回退会把多行 `sh` 检查变成“只执行第一行、然后通过” | 没有 Git for Windows `sh` 的机器上，省略 `shell` 的检查不可用，不产生检查失败的事实 |
| D2 | Windows 采用 W2：只取 `gitx` 所用 Git 对应的 Git for Windows，使用其 `usr\bin` 下的 `sh` / `bash` 入口。PATH 前置只作用于本次 `sh` / `bash` 检查及其后代；不修改 Specmetry 父进程、Git 读取、`cmd`、`pwsh` 或系统环境。实际路径、安装布局与 PATH 构造列入 T5 待验证的规格 | 直接使用 `usr\bin\sh.exe` 时常用工具找不到，失败与真正的检查失败无法区分；包装程序 `bin\sh.exe` 改动的变量更多且随版本变化 | 偏离 architecture §6.4 “只继承用户环境、只注入 `SPECMETRY_CHECK=1`”的规则（明确的例外）；工具的解析顺序改变；其他 `sh` 不可用 |
| D3 | `cmd` 命令中的 CR / LF 多行，在全部写入路径及执行前拒绝 | 实验：`cmd` 的多行命令只执行第一行、退出码 0 | 多行批处理要改写；不符合的已保存配置在执行前被拒绝 |
| D4 | `sh` / `bash` 命令中的 CR 拒绝；正常的 LF 多行允许。不静默规范化，不改写保存值或哈希 | CR 在 dash 中是参数的一部分，在 Git for Windows `sh` 中被忽略，同一配置在不同平台上结果不同；规范化会改变保存的值与哈希 | CRLF 写成的命令要人工修订 |
| D5 | 采用修订后的方案 A：正常完成时按解释器的最终状态分类，保留超时、取消、上下文错误等优先规则；不承诺识别命令串中被掩盖的失败；区分新的写入规则与历史读取的兼容性 | 经 shell 执行时，“命令不存在”发生在解释器内部，Specmetry 只能看到最终状态（Review 071 B4） | 被掩盖的失败不会被额外识别；“命令不存在”与“测试失败”记录相同 |
| D6 | 树已确认结束、但结束上下文读取失败 → 尝试记录 `error` / `context_changed`，不合格；停止后续检查，退出码 6。含义是“无法证明稳定”，不是“已证明发生变化” | 保留“执行过”的事实，又不让它成为证据 | 环境故障产生一条不合格的执行事实；历史中与 D19 无法区分 |
| D7 | 选择 (a)，保留三平台 CLI 支持。清理保证限定于可确认归属的本次进程组 / Job，以及 Linux 被收养的后代。Linux 排他条件不成立时不启动；不得操作归属未确认的进程 | macOS 没有 subreaper，脱离进程组的后代无法发现；Windows 上不在 Job 中的进程不受控制；误杀无关进程不可接受（Review 072） | 缩小了 FR-EV-02 “整个进程树”的保证 |
| D8 | 普通取消且树已确认结束：当前项不记录、后续不执行、此前已提交的记录保留，退出码 6。取消与 F1 同时发生时先按 D19。写事务已开始时依据实际的提交 / 回滚结果，不承诺没有记录。强制杀死导致来不及处理时，不保证返回码或完整输出。Runner 组合故障的优先级采用 1 > 6 > 10 > 0；不改变其他 CLI 路径既有的 2、3、4、5，人工确认被拒绝等 `human_required` 仍为 3。退出码 6 不表示可以安全地自动重跑 | 取消与残留、转发、持久化的组合需要唯一的结果（Review 072 B3-R3） | 被取消的一次在事实中看不到；不能从没有记录推断没有执行 |
| D9 | 每条执行的 `head_start` 都等于目标快照才汇总为 true；空批次比较命令开始时的 HEAD。汇总不进入 `runs(S)`、`V` 或逐条证据效力的判定 | 原描述与冻结的逐条规则冲突（Review 071 B2） | 一个布尔值不能完整表达混合批次 |
| D11 | 主进程退出之后，终止保证范围内的残留后代，确认之后再读结束上下文；有意留下的后台服务也会被终止 | 仍在运行的后代可能在结束上下文读取之后改动 worktree | 检查不能有意留下后台进程 |
| D13 | 命令开始时一次固定全部所选 AC 的配置；执行记录绑定实际固定的修订与配置；不得因此改变已冻结的逐条推导规则 | 一个批次中的执行使用同一时刻的基准 | 批次中途修订的 AC，后面仍按旧配置执行，只对旧修订有效 |
| D18 | `--all` 没有可执行的 AC 时，必要时单独登记 worktree，分配相应的 `seq`，输出空的 `runs`；明确登记的阶段、失败与输出行为 | `check-run.v1` 需要真实的 `worktree_id`，不能伪造 | 一次没有执行任何检查的命令可能写入一条 worktree 事实与一个 `seq` |
| D19 | `T_kill` 内没有确认结束 → 尝试记录为不合格，退出码 6，停止。结束上下文未知，不能报告已结束；明确 `finished_at`、`duration_ms` 在这种情况下的含义，不冒充已知的实际结束时刻；持久化失败仍区分确认未提交与提交结果不确定 | 保留“执行过”的事实，又不让它成为证据 | 扩展了 `context_changed` 的使用情形；改变了“执行结束之后写入”的适用情形 |
| D20 与 C-4 | 只支持继承句柄与具有可封闭入口的受控内存目的端；其他 Writer 在启动之前拒绝。转发不完整时，根据其他状态决定当前项是否记录及合格，随后停止批次、退出码 6；输出不完整本身不自动使检查结果失败或证据不合格。O3 之后当前宿主拒绝新的检查调用；MVP 不承诺长驻宿主的进程内恢复；将来增加长驻的执行入口必须重新审查 C-4。保留实际拒绝机制、挂起任务与句柄归属、宿主退出的规格；不因正式入口是 CLI 就删除这些要求，也不宣称资源上界已经证明 | 对任意不可中断的 Writer，不能同时承诺有界返回、复制已结束与返回之后不再写出（Review 072、073） | 进程内调用可用的输出对象受限；一次转发故障中止整个批次；O3 之后同一宿主不能再执行检查 |

### 二、保证调整与行为、语义的变化

- **保证调整**：
  - D7 是对既有保证的明确调整，不只是一处措辞修正：FR-EV-02、MVP-EV-03 的“整个进程树”改为可确认归属的范围。进程清理的边界与观测级的代码观测边界分别说明；人工验收中确认 `observed_evidence` 风险，不自动批准进程逃逸的风险。D7 对结束上下文与证据合格性的影响写明，不能同时继续宣称所有检查活动均已结束。
  - D2 是 architecture §6.4 环境规则的明确例外。
- **行为或语义的变化**（不只是 D2、D5、D7）：
  - D3、D4 收紧了 M1 已验收的写入路径（`propose`、`review`、人工编辑），并使不符合的已保存配置在执行前被拒绝；
  - D5 改写了 MVP-EV-04 的验收语义；
  - D13 改变了配置读取的时刻（命令开始时，而不是每次执行开始时）；
  - D6、D19 扩展了 `context_changed` 的使用情形，D19 还改变了“执行结束之后写入”的适用情形与时间字段的含义；
  - D8 定义了取消的记录与退出码，以及 Runner 组合故障的优先级；
  - D9 更正了 `is_target_snapshot` 的描述；
  - D11 使主进程正常退出之后的残留后代也被终止；
  - D18 使一次不执行检查的命令可能写入事实；
  - D20 限制了进程内调用的输出对象，C-4 使 O3 之后同一宿主拒绝新的检查调用。

### 三、兼容性与诊断的边界

- 不以“尚未 dogfooding”假定不存在旧配置。
- 历史事实保持可读；不自动改写历史命令、哈希或修订。
- 旧配置不符合新命令规则时，执行前拒绝，由人通过正常的 `material` 修订流程修改。
- D6、D19 共用 `context_changed` 可以接受，但须写明历史记录可能无法区分具体原因；当次命令的输出准确说明已知的状态。
- 不能把没有记录、读取不到记录或退出码 6 描述为检查没有运行，或可以安全重跑。
- Linux 按实际所需的接口列出能力要求：当前完整方案涉及 `pidfd_open` 与 `waitid(P_PIDFD)`，上游接口版本基线涉及 5.4；最终可用性还需要能力探测，不能只靠版本号或发行版名称。不擅自新增未获批的降级清理路径。

### 四、授权与边界

- **授权**：
  - 起草 WP2.5 规范第 5 版；
  - 同步受影响的 PRD、architecture、data-model、验收项、实施计划与活跃状态；
  - 必要的 Schema 描述 / 约束、示例、反例与契约测试的同步，不自行新增字段或枚举；
  - 从 master `bd45583` 建立独立分支与 PR，保持未合并。
- **不授权**：
  - 规范冻结；生产 WP2.5 的实现；新的机制实验；
  - 修改生产代码、迁移、原型代码或 workflow；
  - 顺带修复 M-17 或 N-087-1；
  - 合并本 PR，宣布 WP2.5 或 M2 验收；
  - 新的产品决定或接受风险。需要超出本次决定的新结构或产品取舍时，明确提出，不默默扩展。
- **技术候选不因此获批**：产品批准不等于技术候选全部获批。D10、D12、D14 ~ D17、E1 ~ E9 与 C-1 ~ C-20 仍待独立审查（C-4 的产品语义与 C-7 即 D20 (i) 已由本次决定，其技术部分仍待审查）。
- **保持不变**：M-14、M-15、M-16、M-17 未解决；Auto-fix、Auto-merge 关闭；不代写独立审查的提示词。

### 五、本轮的交付

- **WP2.5 规范第 5 版草案**（[wp2.5-check-runner.md](wp2.5-check-runner.md)）：
  - 十四项写入正文，标为“产品已决定”；规范仍是“第 5 版草案，待独立审查和冻结”；
  - 组合决策表（§8.6）重写：检查结果、`qualifying` 与 CLI 退出码分别裁定；后来的持久化或输出错误不抹掉之前的状态；第 4 版 A9（结束上下文读取失败、记录之后“继续”）与“环境错误停止”的矛盾按 D6 消除；
  - 新增 §8.7（进程清理的保证范围与观测边界）；
  - 影响矩阵（§17.2）：每项的原规则、新规则、受影响的文件与验收 / 反例；
  - D10、D12、D14 ~ D17、E1 ~ E9（§17.3）与 C-1 ~ C-20（§17.4）逐项的采纳建议、依据与待审查状态；
  - 剩余技术门槛 G1 ~ G15（§17.5）：每项的明确假设、当前证据、缺口、最小验证方法、会改变设计的结果，以及分类（冻结之前必需、证据限制、非阻塞增强、授权之后的生产工作）；第 4 版 T1 ~ T6 中冻结之前的项都映射到这张表，没有被豁免。
- **同步的契约**（不改生产代码）：
  - prd FR-EV-02（D7、D11）、FR-BL-12（D3、D4）；
  - architecture §4 的组件表、§6.4、§9.1（Runner 的退出码优先级）、§16 TD-06（已决定）；
  - data-model §4.11、§4.11.1、§7.3 的说明、§7.4、§8.3 V-P04（标明待实施）、§10；
  - acceptance-criteria：改写 MVP-EV-03、MVP-EV-04，新增 MVP-EV-19 ~ 30、CX-29、CX-30，补充 CX-15（D13），更新追溯表；
  - implementation-plan §3 的 TD-06 行与 WP2.5 的状态；
  - `schemas/`：只改描述（`check-run.v1` 的顶层描述、`is_target_snapshot`、`failure_kind`；`status.v1` 的 `failure_kind`；`common.v1` 的 `checkSpec.shell`），增加两个合法示例（空批次、混合 `head_start`）。没有新增字段、枚举或约束；D3、D4 不写成共用 schema 的约束，因为 `checkSpec` 被 `baseline-show.v1` 等读取模型复用，写成约束会使含旧配置的读取输出不能通过校验；D14 的一致性约束建议在冻结之后与迁移 0009 一起加入；
  - 活跃状态：AGENTS.md、原型文档的头部（PR #34 已合并的事实）。

**状态**：十四项与 C-4 **产品已决定**；WP2.5 技术规范为**第 5 版草案，待独立审查和冻结**；生产实现未授权；WP2.5 与 M2 未验收；M-14、M-15、M-16、M-17 未解决。

## 107．Review 089：PR #36 的 WP2.5 第 5 版草案与契约同步（技术 NO-GO，B1 ~ B3）与定向修订

**审查**：[Review 089](reviews/codex-review-089.md)，原样归档为 `5ba58c772aea0ae5ab01506eedd2686bef04b2a9`（`docs: archive Codex review 089 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `777318b996581dec2fed267ec501eb5af9e8b0174d76dfc1f8ff9b627032e6b8`，逐字节一致。

**审查对象**：PR #36，HEAD `81771f365ddbb0480c6406e2715b91db51cd13d5`（树 `d8347e0`），base `bd45583867b36ef63037690cba383b7a681ac13b`；五个提交，14 个文件，+750/−344。

**CI**（审查方的独立核验；作者在 §106 的回执中只读到运行中）：run 40163013490，pull_request，attempt 1，completed/success，8/8。全部作业检出合并快照 `2ae387b78c6165f870764136be17caa88d58ba3e`，它的父提交依次为 base 与 HEAD，树与 HEAD 的树相同。常规 CI 不运行原型，也没有生产 Runner；8/8 不扩张为新机制的三平台通过。

**结论**（按审查原文分层）：

- **技术 NO-GO**，当前不建议合并。三项 BLOCKER：
  - **B1**（产品决定落实错误）：§8.6 规则 6、H 维度、E3、§4.1 第 2 步与 §7.5 以“登记是否为空”决定准入，E3 写明“登记清空，新的调用照常”。确定的反例：O3 之后任务结束、登记清空，同一宿主再次调用即可执行检查，超出已批准的 C-4 与 MVP-EV-29。
  - **B2**（技术设计与证据同步错误）：§8.5 S6 仍先关闭读端、Read 返回之后才“从停止读取起”开始 J，§8.3 的预算说明相同；与本版列出的 C-2 / C-3 及 Review 082、083 的 Windows Close 反例冲突。按这一顺序，持有读取引用时 Close 等待，J 尚未开始，时限公式不覆盖这次等待，也无法到达封闭与移交。
  - **B3**（时间字段的技术定义不成立）：§10.1 把“没有观察到 E-main”括注为“主进程在 `T_kill` 内仍未退出”，并称开始到第一次终止请求的时间是实际运行时间的下界；data-model §4.11 同。合法的反例：主进程在 t1 自行退出而观察失败，t2 才第一次发出终止请求，t2 − t0 大于实际的 t1 − t0。
- **十四项产品决定与 C-4 已获批准**，不重新列为待负责人选择；本轮没有发现须新增一项产品决定才能修正上述问题的依据。候选技术内容没有因审查而获批。
- **NON-BLOCKER**（不单独构成 NO-GO）：
  - N1：不启动进程而结束上下文读取失败的 D6 状态，应在状态表中显式保留（A8、§7.5 的 U、T）；
  - N2：C-5（容量耗尽后继续读并丢弃）与 §9.1 的容量规则（写错误使复制结束）落点不同；
  - N3：D7 之下确认 E-tree 不必带来 EOF，S6 第 1 步与 §9.1 的无条件推论须限于保证范围；
  - N4：G13 的 L 类不能免掉通常可执行的取消测试；
  - N5：G8 中“支持这些布局须负责人另行决定”应收窄为实际改变支持范围时。
- **作者四项取舍**：Linux 能力探测失败不启动、取消映射 `environment`、O3 拒绝在副作用之前，方向成立；O3 拒绝的生存期（B1）与 D19 的时间解释（B3）不成立。
- **契约检查**：45 个合法示例、190 个非法反例通过；Schema 通过不证明时间或生命周期的语义。审查方的隔离副本另证实：`pass + timeout + exit_code 0` 当前仍被 schema 接受，这是已披露的 D14 未实施的边界，不是合法的业务事实。
- **冻结**：不具备。G1 ~ G8、G10 ~ G12 的 F 类规格仍须写定；G9 的映射获得支持，不等于整体冻结。
- 没有新增接受的风险。

**定向修订**（`6797cc2`；只改三项问题及直接关联的条文）：

- **B1**（[wp2.5-check-runner.md](wp2.5-check-runner.md) §8.6 规则 6、H 维度、E 表，§4.1 第 2 步、§7.5、§11.4、§8.5 F2 与 O3、§9.1 的复用、§0；architecture §6.4；MVP-EV-29；§17.1、§17.2 的 D20 行，§17.4 的 C-3、C-4，§17.5 G1，§17.6）：
  - **禁用状态**：本宿主中一次执行以 O3 结束，宿主即进入禁用状态。它只进不出，持续到宿主进程结束；残留任务之后结束、资源释放、资源清单清空都不撤销它；没有重置或恢复的接口。
  - **资源清单**与禁用状态分开：持有没有返回的复制任务与挂起的停止任务，读取它们晚到的结果；清单可以更新、清空，与准入无关。
  - **新进程不是恢复**：新的进程是新的宿主，从未禁用的状态开始；禁用状态不跨进程保存。
  - 第 5 版初稿的 E3 撤回，改为“读取结果、移除、释放；禁用状态不变，之后的调用仍拒绝”；另加 E4（新的进程）。
  - **冻结之前的规格**（G1，不因正式入口是 CLI 而省略）：置位与准入在同一把锁下；置位之前已准入的并发调用，候选为每一项开始之前再查一次禁用状态（待审查）；清单的生存期、晚到错误的报告与宿主退出时的声明。
- **B2**（§8.5 S6 第 2 ~ 4 步、S1 与 F1 的跳转、F1 的时限要求、§8.5 的时限、§8.3、§9.1 的封闭步骤、§15、MVP-EV-29、§17.4 C-2 与 C-3、§17.5 G2、§17.6）：
  - S6 先在同一时钟上建立 J 的起点与截止，再为每个被停止的流启动停止任务；可能阻塞的句柄准备、取消与关闭都只在停止任务中；运行本身不在 J 之前同步关闭，也不等待停止完成。
  - J 内以同一截止观察复制任务与停止任务；J 结束（到期或两者都已结束）之后，不论停止是否完成，都封闭入口。
  - 分别记录停止请求、停止完成、复制结束、入口封闭与资源释放；挂起的停止与没有返回的复制移交给调用方，最终进入资源清单；停止的错误（包括运行返回之后才到的）由任务保存、由接手者读取，不被之后的成功覆盖，与原型 §2.13、§2.14 的停止结果生命周期一致。
  - D 阶段内全部复制已返回时不进入 J，关闭读端不必等待读取（与原型相同），但不作硬实时保证。
  - 时间公式改称“Specmetry 自己的阶段等待之和”，写明不是硬实时保证，J 之后移交的任务不计入；没有新增预算或容差。
  - Windows 的重复取消仍是有前提的候选。
- **B3**（§10.1、§8.3、§8.5 F1 的典型情形与要求；data-model §4.11；MVP-EV-28；§17.2 的 D19 行，§17.3 E9，§17.5 G12，§17.6）：
  - D19 的时间字段按是否可靠地取得了主进程的退出信息分开定义。
    - **已取得**：`duration_ms` 为开始到观察到退出的时间（E9）。
    - **没有取得**（主进程可能仍在运行，也可能已退出而没有被可靠观察）：`duration_ms` 为开始到第一次终止请求的区间长度，只是这一区间的长度，不是实际运行时间，也不是它的上界或下界；`exit_code` 为 NULL；不用 0 等虚构值代替未知。
  - `finished_at` 仍是放弃等待、尝试记录的时刻。
  - 历史记录不能总是区分这两种定义，也不能区分 D19 与 D6；历史展示中 `context_changed` 且 `tree_state_end = unknown` 的记录，不把 `duration_ms` 称为实际运行时间。不新增字段或枚举。
  - 删除“没有观察到即仍未退出”与“实际运行时间的下界”。D6 的时间说明顺带写明不启动进程时为 0，没有声称观察过不存在的退出。

**文档反例的核对**（按修订后的条文逐步推演，不是新的实验）：

1. **O3 之后任务全部结束**：第一次调用以 O3 结束 → 宿主禁用（E1）；任务稍后结束 → 读取结果、移除、资源释放（E3），禁用状态不变；同一宿主再次调用 → §4.1 第 2 步在打开数据库之前拒绝，退出码 6（E2）。新的进程照常（E4）。
2. **停止操作永不返回**：F1 或 D 到期 → S6 先建立 J 的截止，再启动停止任务；停止阻塞在关闭中 → J 按截止到期（运行本身没有同步关闭或等待停止）→ 封闭入口 → 报告停止挂起、复制 O3 → 两者移交给资源清单；之后的文档在封闭之后写出。
3. **主进程早已退出但退出的观察失败**：没有取得 E-main → F1 → D19 尝试记录；按“没有取得”的定义，`duration_ms` 为开始到第一次终止请求的区间长度，输出不称它为运行时间或它的界限，`exit_code` 为 NULL；不产生虚假的下界。

**本轮没有处理的**：

- N1 ~ N5 只登记，不在本轮修订：N3 的 S6 第 1 步与 §9.1 的 EOF 推论原样保留，N1 的状态表、N2 的容量策略、N4 的取消测试分类、N5 的 G8 措辞也都没有改。建议在下一轮与 F 类规格一起处理，或按负责人的安排。
- M-17、N-087-1 与 G 类门槛的补齐都不在本轮。

**新提出的问题**（须负责人确认，没有自行扩展）：复制已返回而停止仍挂起（O2 + 停止挂起）时，是否也使宿主禁用。C-4 的文字只针对 O3。推荐同样禁用，因为读端仍被持有；这扩大了 C-4 的触发条件。已列入 G1。

**本轮修订的范围**：

- 相对审查对象 `81771f3` 的新增提交：Review 089 的归档、规范草案与 data-model、architecture、acceptance-criteria 的定向修订、本记录与活跃状态。
- Schema、示例、生产代码、迁移、原型代码、测试与 workflow 都没有改动；没有新的机制实验。

**保持不变**：

- 十四项产品决定与 C-4 是产品已决定；
- §106 与历史审查原文没有改动；
- 规范是第 5 版草案，未冻结；生产实现未授权；WP2.5 与 M2 未验收；
- M-14、M-15、M-16、M-17 未解决。

**状态**：已修订，待独立复审；不宣布 B1 ~ B3 关闭。

## 108．Review 090：PR #36 的 Review 089 B1 ~ B3 定向复审（技术 GO）与收尾

**审查**：[Review 090](reviews/codex-review-090.md)，原样归档为 `0adf60a07bb2448d66c77f1cbf44219e6cf5e5ab`（`docs: archive Codex review 090 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `06a727576abdf99b36707f3971e796a44e4ecab400b5a4cad872f17955ad7c25`，逐字节一致。

**审查对象**：PR #36，HEAD `37e785caafeb63bc90b939e57df3e1f064242d9d`（树 `3db3db9`），base `bd45583867b36ef63037690cba383b7a681ac13b`。相对上轮 `81771f3` 三个提交：`5ba58c7`、`6797cc2`、`37e785c`；9 个文件，+492/−62。

**CI**（审查方的独立核验；作者在 §107 之后只读到运行中）：run 40169848371，pull_request，attempt 1，completed/success，8/8。八个作业都检出合并快照 `40d9cc06965dc2f7cfacc47a8477810ea3515421`，它的父提交依次为 base 与 HEAD，树 `3db3db9` 与 HEAD 相同。常规 CI 不运行原型，也没有生产 Runner。

**结论**（按审查原文分层）：

- **技术 GO**，限于规范第 5 版草案的本轮修订；可以建议负责人合并准备文档。没有新的合并级 BLOCKER。
- **Review 089 B1 ~ B3 在准备文档层面关闭**，各有限定：
  - **B1**：限于已批准的 O3 触发的持续宿主拒绝在文档中的落实。原子置位与准入、置位前已准入的调用、每项再查的位置、资源清单的生存期、晚到错误的报告与宿主退出，仍是 G1 的 F 类规格；“每项开始前再查”仍是候选。
  - **B2**：限于协议顺序、共同截止、结果所有权与移交的条文修正；G1、G2 的精确生产接口与必要验证没有全部满足。“资源释放”是后续条件：完成而关闭失败的资源，不能凭“完成”声称已释放，错误、实际句柄归属与最终释放动作仍在 G1 中写定。
  - **B3**：不表示时间字段能在没有新增元数据的情况下证明所有历史分支，也不表示将来的实现已验证。**冻结之前的精确化点**：G2 须把“S3 的第一次终止请求”写成确定的本地事件；身份复核失败、不能发信号时，使用进入或提出 S3 逻辑终止请求的时刻，不能为了取得时间向身份未确认的对象发信号，也不能假定一定发生了 kill / Terminate 调用。这一点不单独阻塞准备文档。
- **规范不具备整体冻结条件**：
  - §107 新提出的“复制已结束、停止仍挂起时是否禁用宿主”扩展了现有 C-4 的触发条件，尚未批准；它的范围还必须包括 **O1 + 停止挂起**，不能只讨论 O2。
  - 审查给出了推荐的最小完整扩展（Review 090 §6.2），不代为批准；须最晚在 G1 冻结之前收口，同时写定 O1 分支的批次与退出行为。
- **NON-BLOCKER**：
  - Review 089 N1 ~ N5 保持非阻塞，没有写成已解决；
  - 新增 **N-090-1**：G1 不能把“复制已结束、停止挂起”只写成 O2，应覆盖 O1。
- **PRODUCT DECISION**：十四项与 C-4 继续按已批准的输入使用；只有 G1 的触发扩展及相应的 O1 批次 / 退出策略待负责人决定，审查不批准。
- **ACCEPTED RISK**：不新增、不扩展。
- **契约检查**：45 个合法示例、190 个非法反例通过；Schema 与契约测试相对上轮没有变化。

**本轮收尾**（只有文档；按负责人的收尾指示）：

- Review 090 的归档；本记录；
- 活跃状态同步为“草案技术审查通过，待合并核验”：AGENTS.md、实施计划、原型文档的头部、规范草案头部的状态行。
- 规范的技术正文、Schema、示例、代码、原型、迁移与 workflow 都没有改动；受审规范的语义不变。
- N-090-1 与 G1 触发条件的扩展**不在本收尾中写入技术正文**，留待合并核验之后的独立 PR。

**保持不变**：

- 规范是第 5 版草案，**未冻结**；生产实现、新的机制实验未授权；WP2.5 与 M2 未验收。
- 冻结之前的门槛 G1 ~ G8、G10 ~ G12 的 F 类规格仍未写定；G13 ~ G15 按原分类。
- Review 089 N1 ~ N5、Review 090 N-090-1 未处理；M-14、M-15、M-16、M-17 未解决；N-087-1 未修。
- §106、§107 与历史审查原文没有改动。

**状态**：PR #36 草案技术审查通过，待合并核验。

## 109．PR #36 的合并核验；负责人决定 D21（C-4 的扩展）；冻结之前的规格草案（第 6 版）

### 一、PR #36 的合并与核验（事实更新）

§108 末尾的“草案技术审查通过，待合并核验”是当时的状态，不改写。负责人 2026-10-25 条件授权以 merge commit 合并 PR #36；条件全部满足之后执行：

- **收尾**：Review 090 的归档 `0adf60a`；§108 与状态同步 `f60e1cc`。相对受审 HEAD `37e785c` 只有文档，6 个文件；技术正文、Schema、示例、代码、原型、迁移与 workflow 没有变化。
- **最终 HEAD 的 CI**：run 40189131061，pull_request，attempt 1，completed/success，8/8。八个作业都检出合并快照 `efc96db77a4e67a461894109a242b2d485fb783d`；它的父提交依次为 `bd45583` 与 `f60e1cc`，树 `58a4fe9` 与 HEAD 相同；八份日志中没有 `--- FAIL`。
- **合并之前的条件**：base 与 master 都是 `bd45583`，没有移动；增量只有文档；Auto-fix、Auto-merge 关闭。
- **合并**：`gh pr merge 36 --merge --match-head-commit f60e1cc75ccf11428a39be33f464ca0979541f71`，2026-10-25 11:45:51 UTC，合并为 `f3fbe837a2abb1143abcbe38d504e5a5d9cb0537`。父提交依次为 `bd45583`、`f60e1cc`；master 包含最终 HEAD；合并树 `58a4fe9` 与 HEAD 相同，diff 为空；相对旧 master 16 个文件，+1479/−364，只在 `docs/`、`AGENTS.md` 与 `schemas/`（只有描述与两个合法示例）。
- **master push CI**：run 40190229214，push，attempt 1，completed/success，8/8，没有重跑；八个作业都检出 `f3fbe83` 本身，日志中没有 `--- FAIL`。
- 合并回执写在 PR #36 的描述中，没有为回写结果向 master 追加提交。

### 二、负责人的决定 D21（2026-10-25）

**决定**（C-4 的扩展；本节登记，§107、§108 中“待负责人决定”的记录按当时的状态保留原文）：

- O3 之后，当前宿主持续禁用至进程结束（C-4，不变）。
- **J 截止时，只要仍有停止任务没有确认结束，同样持续禁用**，不论复制的结果是 O1、O2 还是 O3。
- 之后任务结束或资源清单清空，都不解除禁用。
- 停止任务没有确认结束时，当前批次停止，退出码 6。
- 当前项是否记录、记为什么与 `qualifying` 仍按其他状态裁定；不把 O1 强行改称输出不完整，也不自动使证据不合格。
- 禁用不等于资源已经释放；不新增恢复接口。
- 其他新的禁用条件不得默默扩大；需要时具体提出。

**理由**：停止任务没有确认结束时，读端与停止任务仍被持有，资源的状态未知；继续在同一宿主中接纳新的检查，会在未知的异步资源之上叠加新的执行（Review 090 §6.2 的推荐；N-090-1 指出复制已结束的情形包括 O1，不只是 O2）。

**代价**：一次已经完整转发、甚至合格的执行，也可能因停止没有确认结束而中止批次，并使同一宿主不能再执行检查。生产 CLI 走继承路径、每次都是新的进程，这一代价主要落在进程内的调用（测试基座、将来的长驻入口）上。

**保证的变化**：扩展了 C-4 的触发条件，并新增一个批次的停止条件（行为）；不改变结果分类、`qualifying` 的公式与逐条推导规则。

### 三、Review 089 N1 ~ N5 与 Review 090 的直接关联意见

| 意见 | 性质 | 处理（第 6 版） |
|---|---|---|
| Review 089 N1：不启动进程而结束上下文读取失败的 D6 状态 | 可以确定的工程修正 | §8.6 A8 与 §7.5 的 D6 行允许 U 未运行、T 没有本次的进程；`duration_ms` 为 0；说明不称确认过一棵不存在的进程树；§19.10 的真值表单列这一行 |
| Review 089 N2：C-5 与 §9.1 的容量规则落点不同 | 可以确定的工程修正（依据原型 `T6.pipe.capacity-discard` 与反例 `T6.pipe.capacity-stop`） | §9.1 的容量规则改为 C-5：容量错误之后继续读取并丢弃（只计数），结果为 O2；D、J 与停止的时机不变 |
| Review 089 N3：D7 之下确认 E-tree 不必带来 EOF | 可以确定的措辞修正 | S6 第 1 步与 §9.1 的继承路径限于保证范围：范围之外的进程仍可能持有写端，D 阶段到期即进入 J |
| Review 089 N4：G13 的 L 类不能免掉通常可执行的取消测试 | 可以确定的分类修正 | 通常可执行的取消接线测试（Unix 的 SIGINT、SIGTERM；Windows 独立控制台进程组中的 CTRL_C / CTRL_BREAK；与事务的组合）归 G15（P）；G13 只保留交互式控制台的关闭等特殊情形。另记：生产入口目前只捕获 `os.Interrupt`，捕获 SIGTERM 属于生产实现 |
| Review 089 N5：G8 的批准边界 | 可以确定的措辞修正 | 识别同一来源（Git for Windows）的布局是工程细化，由技术审查确认；把来源扩大到其他 `sh` 才须负责人决定（§19.8） |
| Review 090 N-090-1：复制已结束而停止挂起须覆盖 O1 | 产品问题 | 由 D21 决定（第二节） |
| Review 090 §5：S3 的第一次终止请求须写成确定的本地事件 | 冻结之前的接口细节 | §19.2：`t_S3` 是进入 S3、在任何身份核对或终止调用之前记下的时刻；不为取得时间而发信号 |
| Review 090 §4.1：“资源释放”是后续条件，完成而关闭失败不得凭“完成”声称已释放 | 冻结之前的规格 | §19.1 规则 4、7：只有关闭返回 nil 且复制已结束才记“释放已确认”；关闭出错记“释放未确认”，不是 D21 的触发条件 |
| Review 090 §3.3：B1 关闭的限定（原子准入、已准入调用、清单、晚到错误、宿主退出） | 冻结之前的规格 | §19.1 规则 1 ~ 6 |

### 四、冻结之前的技术规格草案（§19）

- G1 ~ G8、G10 ~ G12 的规格草案写在规范 §19，逐项标明四种状态（已写定规格、已有适用证据、仍需最小原型、授权后的生产测试），汇总在 §19.11。“已写定规格”都**待独立审查**，没有获批。
- **仍需最小原型**的四项只给出验证计划（§19.12），**没有实施**，对应的门槛没有满足：
  - P-1（G3）：主进程仍存活、退出观察失败的 F1；
  - P-2（G6）：seccomp 拒绝相关系统调用时的能力探测；
  - P-3（G5）：Intel macOS（若有可用的 runner）；
  - P-4（G7）：Windows 启动链中未测的原生失败点。
- 常规 CI 只验证仓库中已有的代码，不证明 §19 的任何机制成立。

### 五、待负责人决定的问题与没有提出的契约变化

- **Q-1**：F1 是否也使当前宿主禁用（§19.13）。推荐同样禁用至进程结束：理由与 D21 相同，并使三平台一致；代价是进程内调用在一次 F1 之后不能再执行检查。Linux 上之后的检查已因基线不为空而不启动（C-15），macOS 与 Windows 没有这样的屏障。本版**没有**自行扩展。
- **没有提出的契约变化**：
  - 没有发现必须新增的字段或枚举；
  - 在历史中区分 D19 的两种时间定义与 D6 需要新字段，本版不提出；
  - macOS 的支持范围不缩小；Windows 的 `sh` 来源不扩大；
  - D14 的约束不是新字段或枚举，在冻结之后与迁移 0009 一起加入。

### 六、本轮的范围与边界

- **授权**：登记 D21；更新 G1、C-4 与直接关联的契约；处理 N1 ~ N5 与 Review 090 的直接关联意见；起草冻结之前的技术规格；独立 PR，保持未合并。
- **改动**：规范第 6 版草案；architecture §6.4、data-model §7.4、acceptance-criteria MVP-EV-29 中与 D21 直接关联的句子；本记录与活跃状态。Schema、示例、生产代码、迁移、原型代码、测试与 workflow 都没有改动；没有新的机制实验。
- **不授权**：规范冻结、生产实现、新的机制实验、合并本 PR、WP2.5 或 M2 的验收。
- **保持不变**：十四项、C-4 与 D21 是产品已决定；M-14、M-15、M-16、M-17 未解决；N-087-1 未修；§106 ~ §108 与历史审查原文没有改动。

**状态**：规范第 6 版草案，待独立审查；未冻结。

## 110．Review 091：PR #37 的 WP2.5 第 6 版冻结前规格（技术 NO-GO，B1、B2）与定向修订

**审查**：[Review 091](reviews/codex-review-091.md)，原样归档为 `45c1a5af9079a5bbf48bafe5a551021ebc97fedb`（`docs: archive Codex review 091 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `648d519f70ca08bec183b106f4cb83313e5026f3dd152d3fa99cae2f67c9fce4`，逐字节一致。

**审查对象**：PR #37，HEAD `ba03f45e1c3e765a9c0402e1ae85ec0cad842310`（树 `bc5de32`），base `f3fbe837a2abb1143abcbe38d504e5a5d9cb0537`；两个提交 `b240258`、`ba03f45`；8 个文件，+507/−39。

**CI**（审查方的独立核验）：run 40192159896，pull_request，attempt 1，completed/success，8/8。八个作业都检出合并快照 `307e3527339c64f8cc399a2276323742fcecf4d5`，它的父提交依次为 base 与 HEAD，树 `bc5de32` 与 HEAD 相同。常规 CI 不运行原型，也不证明 §19 的机制。

**结论**（沿用审查原文的编号与裁定）：

- **技术 NO-GO**，当前不建议合并。原因不是 P-1 ~ P-4 尚未实施，而是两处新规格自身的问题：
  - **B1**：G4 从共享 / 排他的互斥推出租约等待有上界，依据不成立。共享子进程没有统一的有界生命周期（`gitx` 使用调用方的上下文，`doctor` 与 macOS ACL 检查没有上下文）；等待者可以取消，与持有者会及时释放是两件事。
  - **B2**：Windows 恢复失败的知识状态与旧的 S1、组合表没有同步：新表承认“可能已运行”，旧表仍把同一路径归为“从未运行 / 仍挂起”，记录规则冲突；`ResumeThread` 的返回 0 没有定义（依据 Microsoft 的 API 契约：返回之前的挂起计数，0 表示没有挂起，1 表示这次恢复，大于 1 表示仍挂起，失败为 `(DWORD) -1`）。
- **D21 的主要落实正确**：O1 / O2 / O3 与停止任务的状态分开；J 截止时仍有停止任务即停止批次、退出码 6、持续禁用；当前行不因此不合格。Review 089 的三项没有重新引入。
- **NON-BLOCKER**：
  - N1：关闭失败本身不触发 D21，但不能覆盖 O3；
  - N2：资源清单须按资源种类定义完成与退役（复制 / 停止任务与 F1 的进程项不同）；
  - N3：macOS 的运行时核对只排除被核对字段的不符，不是全 ABI 的证明；
  - N4：Linux 的 GET 与 SET 不同，探测只证明被调用的那部分能力；
  - N5：D14 的真值表须写明按 §6.2 的优先级命中，并区分没有启动进程时的上下文变化；
  - 原型文档头部仍有“第 5 版”（状态收尾）。
- **各 G 门槛**（审查 §6，照录要点）：G1、G3、G5、G6、G12 部分成立；G2 大体成立；G4、G7 尚不成立；G8 / G11、G10 规格方向成立；G9 继承支持；G13 分类改善；G14 非阻塞；G15 生产阶段。
- **Q-1**：保持待决定，最晚在 G1 / G3 及 S1 失败协议冻结前决定；审查建议把条件写成“已创建本次消费者，经过规定的终止 / 确认窗口之后仍不能确认其保证范围内已结束”，并分别处理正常启动后的 F1、S1 已创建而终止未确认、没有创建进程的情形；不批准。
- **P-1 ~ P-4**：都未实施，不作为证据。P-1 有必要做缩减后的机制原型，当前计划须先修订（命中的构造、内外层分离、计数口径与总期限）；P-2 冻结前有价值，可与 P-1 的 Linux 夹具合并，须补正常组与 GET / SET 的分支；P-3 取决于明确的支持声明，没有 runner 不等于门槛满足；P-4 须在 B2 修正之后，区分内层的移交与外层的清理。
- **验证边界**：契约检查 45 个合法示例、190 个非法反例通过，不证明未实现的机制或业务语义；审查方没有新的进程、shell、seccomp 或平台实验，也没有 macOS / Linux 的动态结果；审查方一次只读查询的脚本语法错误不计为有效的检出。
- **产品决定**：十四项、C-4 与 D21 不重开；Q-1 及相邻的 S1 未知消费者的范围待负责人决定；不新增接受的风险。

**定向修订**（只改两项问题及直接关联的条文）：

- **B1**（规范 §19.4、§19.2、§8.3、§8.4，C-8、C-9，§17.5 G4，MVP-EV-23）：
  - 分别写定共享持有者（从取得到子进程被回收，没有协议给定的上界；`gitx` 随调用方的上下文可取消，`doctor` 与 ACL 检查不可取消）、排他持有者（S0 ~ S5，协议给定 `timeout_sec + T_kill`，同步步骤没有硬期限）与等待者（可取消，但没有固定的总上界）的生命周期；
  - 撤回“等待的上界来自持有者自己的有界阶段……所以没有无限期的持有”；
  - 取消：在闸门的锁下退出自己的排队，不等待他人的子进程，不释放他人的租约；写者优先只计仍在等待的排他请求；
  - 排队时间不计入 `timeout_sec`、`T_kill`、`T_drain`、`T_join` 与 `duration_ms`；§8.3 的阶段等待之和不含排队；
  - 生产 CLI 中排他请求预期不等待共享持有者（理由写明）；不为 Git Reader、`doctor`、ACL 检查增加超时或终止政策，不在超时之后释放仍有活跃子进程的租约；长驻入口需要排队期限时须另行提出；
  - C-9 的规则加入 `Cmd.Wait`，并按 `go/types` 解析的调用目标判断。
- **B2**（规范 §19.7 重写，§8.5 S1 与 F1 的出口，§8.2 的 Windows 顺序，§7.5，§8.6 的 U 维度、规则 1、A4、A5、新增 A12，§17.3 E5，C-16，§17.5 G7，§19.2 的 `t_start`，§19.10）：
  - 五种说法分开：API 调用失败、未确认恢复成功、确定未运行、可能已运行、已恢复；
  - 恢复之前的失败（加入 Job、确认、快照、打开线程）：确定未运行（只依据本进程的操作），A4 / A5，不记录；
  - `ResumeThread` 的四种结果：`-1` 为可能已运行；0 为已运行、时刻未知（在本进程恢复之前就已运行）；1 为已恢复、正常运行；大于 1 为可能已运行（存在其他挂起者）。快照显示多于一个线程同样按可能已运行；
  - 恢复的结果异常：立即 S3，在 `T_kill` 内确认；终止未确认 → F1，按 D19 尝试记录（A6）；终止已确认 → 已批准的规则内没有唯一的记录结果，**单列为 Q-2**，在决定之前不写定 A12 的当前项；都停止，退出码 6；
  - 恢复之后的管理错误：关闭线程句柄出错只是诊断；观察所需的等待或 Job 查询出错是观察失败，按 F1；
  - 所有权写成分支表：同一句柄只属于一个分支；终止未确认时进程句柄不关闭、移交资源清单；关闭出错记“释放未确认”；
  - 父端期限只覆盖对进程句柄的等待与 `ActiveProcesses` 的轮询；同步调用没有期限，不移交；
  - 终止请求不当作结束的确认（C-17）。
- **N1**（S6、§19.1 规则 7、MVP-EV-29）：关闭出错本身不新增禁用条件；同时为 O3 或宿主此前已被禁用时，仍按 C-4 保持禁用。
- **N2**（§19.1 规则 4、§19.3）：复制 / 停止任务按完成信号退役，关闭出错时保留读端对象的强引用；未确认结束的进程项不启动观察任务、不 `Wait`、不因之后退出而移除，直到宿主退出。
- **N3**（§19.5）：运行时核对的结论收窄；登记 ABI 依据（XNU 源码、读取的字段、缓冲区、架构）；Intel 在静态依据被接受之后才作为证据限制，否则门槛没有满足；`EINTR` 只在剩余的期限内重试。
- **N4**（§19.6）：探测的结论收窄；`SET 1` 失败不启动；设置之后读回核对；`SET 0` 失败作为环境错误、批次停止，之后的探测因读取为 1 而拒绝；pidfd 关闭失败保留描述符号、记为未确认关闭。
- **N5**（§19.10）：真值表按 §6.2 的优先级命中；“上下文确实变化”分为启动与没有启动进程两行（后者 `duration_ms` 为 0）；加入 Windows 恢复的结果异常的两行。
- **P-1 ~ P-4**（§19.12）：按 Review 091 §8 重写：公共前提（预期固定、命中有独立记录、先保存被测状态再补救、内外层分别裁定、接缝不等于原生故障、环境不具备为 unsupported）；P-1 的命中构造、按 goroutine 栈而不是总数的断言、并行口径与总期限；P-2 的正常组与 GET / SET 分支、每类 1 次与总期限；P-3 依支持声明；P-4 用真实的返回 0 与大于 1、`-1` 的处理路径、终止未确认的移交与外层的独立观测。**都不授权实施**。
- **Q-1**（§19.13）：按 Review 091 §7 更新推荐的范围；仍待负责人决定。
- **状态收尾**：原型文档头部的“第 5 版”更正。

**文档事件序列的核对**（按修订后的条文逐步推演，不是新的实验）：

1. **共享持有者长期没有释放**：同一进程中，一次 `doctor` 的 `git --version` 持有共享租约而子进程不结束；另一次 `check run` 申请排他租约 → 排他等待者一直等待，没有协议给定的上界（§19.4）；用户发出 SIGINT → 等待者退出排队，当前项不记录，退出码 6；共享持有者不受影响，它的租约仍由它持有；之后的共享请求不再被这个已取消的排他请求挡住。生产 CLI 中这一情形预期不会出现（闸门是进程内的）。
2. **等待者取消与租约释放同时发生**：等待者醒来之后在闸门的锁下先看上下文。已取消 → 退出排队、不取得，即使租约此时已经可以取得；若在取消之前已经取得 → 作为持有者，在 S0 之前处理取消：不启动，释放自己的租约。两种情况下当前项都不记录，退出码 6，每个租约只由它的持有者释放一次。
3. **Windows 的各返回值与恢复之后的管理错误**：
   - `-1` → 可能已运行 → 立即 S3 → 确认则 A12（记录待 Q-2），未确认则 A6（D19）；退出码 6；
   - 0 → 已运行、时刻未知 → 同 `-1`；
   - 1 → 已恢复，正常运行；之后关闭线程句柄出错 → 诊断“释放未确认”，检查照常；之后观察出错 → 观察失败 → S3 → F1（D19），之后的成功不恢复；
   - 大于 1 → 可能已运行 → 同 `-1`；
   - 恢复之前的失败 → 确定未运行 → 确认则 A4、未确认则 A5，不记录，退出码 6。
4. **终止未确认与后续持久化失败的叠加**：
   - 恢复之前的失败、终止未确认（A5）：不尝试记录，持久化不涉及；退出码 6；进程句柄在资源清单中。
   - 恢复的结果异常或恢复之后、终止未确认（A6）：按 D19 尝试记录；写入失败（B2）→ 这一次没有记录，说明仍写明可能已运行、进程树没有确认结束、可能残留、结束上下文没有读取；提交结果不确定（B4）→ 说明“这一次可能已经记录，也可能没有”；两者都为退出码 6，进程句柄都留在资源清单中；P 的结果不改变 U、T、C。

**新提出的问题**（须负责人决定，没有自行扩大 D19 或禁用条件）：

- **Q-2**：Windows 恢复的结果异常而终止已确认时，这一次是否记录、记为什么（§19.13）。推荐 (a)：按 D19 的语义尝试记录不合格的 `error` / `context_changed`；另一选项 (b) 为不记录。两者都停止、退出码 6，都不需要新的字段或枚举。

**本轮修订的范围**：

- 相对审查对象 `ba03f45` 的新增提交：Review 091 的归档、规范草案与 acceptance-criteria 的定向修订、原型文档头部的状态更正、本记录与活跃状态。
- Schema、示例、生产代码、迁移、原型代码、测试与 workflow 都没有改动；没有新的机制实验。

**保持不变**：

- 十四项、C-4 与 D21 是产品已决定，不重开；Q-1 仍待决定；
- P-1 ~ P-4 不授权实施；
- 规范是第 6 版草案，未冻结；生产实现未授权；WP2.5 与 M2 未验收；
- M-14、M-15、M-16、M-17 未解决；N-087-1 未修；
- §106 ~ §109 与历史审查原文没有改动。

**状态**：已修订，待独立复审；不宣布 B1、B2 关闭。

## 111．Review 092：PR #37 的 Review 091 B1、B2 定向复审（B1 关闭；B2 残余，技术 NO-GO）与定向修订

**审查**：[Review 092](reviews/codex-review-092.md)，原样归档为 `f1a515c3944272e1e082d364d5b87b037bb1d8cd`（`docs: archive Codex review 092 unchanged`）。源文件、暂存、该提交 blob 的 SHA-256 均为 `15df11c969df9ad20a2c6a7d3547057bd43f7bdde6e68e0efb18c86c28c32f75`，逐字节一致。

**审查对象**：PR #37，HEAD `25e59b9acd455048ba346346a8c58a73ee121438`（树 `953f6ca`），base `f3fbe837a2abb1143abcbe38d504e5a5d9cb0537`；相对上轮 `ba03f45` 三个提交、7 个文件，+620/−87。

**CI**（审查方的独立核验）：run 40203128396，pull_request，attempt 1，completed/success，8/8；八个作业都检出合并快照 `0713b829fdd462d1312ad930d52d36e6e8666d9a`，父提交依次为 base 与 HEAD，树 `953f6ca` 与 HEAD 相同。

**结论**（沿用审查原文的编号与裁定）：

- **Review 091 B1 关闭**：限于撤回错误的上界、给出自洽的等待与取消规格；不是闸门生产实现的验收。C-9 的按类型识别在实施时仍需具体的解析规则、不能解析时的拒绝策略与导入者测试。
- **Review 091 B2 未关闭**；草案仍为**技术 NO-GO**，不建议合并。残余两项 BLOCKER：
  - **B2-R1**：未挂起 / 已恢复不等于已经执行用户代码。§19.7 把返回 0 写成“已运行……在本进程恢复之前就已运行”，把返回 1 写成“自调用返回起运行”，并进入 U 维度、§110 的作者推演与 P-4 的预期。依据 Microsoft 的 `ResumeThread`、`CREATE_SUSPENDED` 与线程挂起的文档，这些 API 只给出挂起计数与恢复状态，不证明用户指令实际获得 CPU 或从何时开始。
  - **B2-R2**：恢复异常的 A6 与通用 D19 对同一行给出不同的字段。通用规则在取得正常退出信息时保留退出码、按 `t_main − t_start` 计时；新行却无条件取 NULL 与 `t_S3 − t_start`，§19.10 没有说明谁优先。合法的组合：返回 0 之后主进程在终止之前自行退出 7，退出状态可靠取得，而 Job 后代的结束查询失败。
- **B2 中已改善、无需重开的部分**：恢复之前的失败（A4 / A5）、多线程快照、返回 `-1`、返回大于 1、A12 留给 Q-2、恢复之后的观察失败、终止未确认时的句柄保留、关闭失败、父端时间限制。
- **NON-BLOCKER**：
  - **N-092-1**：不应把新的线程干预前提归因于 D7；D7 只限进程清理的可确认归属范围；
  - P-2 的拒绝 / 收尾失败的断言须拆分；P-4 的观察能力须精化。
- **Review 091 的非阻塞项**：N1、N5 的文档问题可以关闭；N2 的主要澄清成立；N3 的过强推论已撤回（固定的 SDK / 源码版本与支持矩阵仍未登记完）；N4 的范围已正确收窄（`SET 0` 失败之后的策略仍属 G3 / G6、P-2 必须覆盖的冻结前失败协议，不能称为“没有副作用”）。
- **Q-1**：当前推荐的范围基本完整；仍待负责人决定。
- **Q-2**：建议 (a)，但须作为**新增的记录语义**明确决定，并写清 `started_at`、`finished_at`、`duration_ms`、`exit_code`、结束上下文、事务与历史展示；还须决定 A12 与写事务之前的取消的先后（建议保留 D8）。(b) 同样可实现，但须明确接受事实缺失。
- **P-1 ~ P-4**：都未执行、未授权，不是证据；建议先修正文与预期，再由负责人另行授权最小的 P-1、P-2、P-4；P-3 依支持声明与可用环境裁定。
- **验证边界**：审查方第一次本地检查因默认 Go 缓存的访问被拒绝而没有进入测试（环境准备失败，不算产品失败，也不算首次通过）；改用隔离的缓存之后通过，45 个合法示例、190 个非法反例。没有新的机制实验、原型执行或 Windows 恢复调度；反例是条文与 API 允许的事件序列。
- 十四项、C-4、D21 不重开；不新增接受的风险。

**定向修订**（只改 B2 残余及直接关联的条文）：

- **统一的知识状态矩阵**（规范 §19.7）：分列线程挂起 / 恢复的 API 结果、用户代码（确定未运行 / 可能已运行 / 按协议视为已启动）、实际执行的证据（任何一行都没有）、实际开始时刻（都未知或不适用）、协议的计时起点（`t_start` 或 `t_anom`）、主进程的退出状态（在 S3 之前 / 之后可靠取得 / 没有取得）与保证范围内的结束（确认 / 未确认）。覆盖 Unix 的 `Start`、恢复之前的失败、多线程快照、`ResumeThread` 的 `-1`、0、1、大于 1，以及返回 1 之后的两种管理错误。
- **B2-R1**：
  - 返回 0 改为“调用之前没有挂起、可被调度，用户代码**可能已运行**”；
  - 返回 1 改为“解除最后一层挂起，**按协议视为已启动**，不是实际执行指令的认证”，`t_start` 是约定，不是实际执行的起点；
  - U 维度的“已运行”定义为按协议视为已启动；A6、A12、§7.5、S1 中的“可能已运行或已运行”改为“可能已运行”；§11.4 的说明改为“检查已启动而没有记录”或“检查可能已执行而没有记录”；
  - P-4 的断言改为三种知识状态，并要求夹具事件的命中，不以挂起计数证明历史。
- **B2-R2**（规范 §10.1 写定唯一的字段规则，§6.2、§8.3、§8.6 规则 1 与 A6、§17.2、§19.2、§19.7、§19.10、data-model §4.11、MVP-EV-28 都引用它）：
  - 两个独立的判断：**启动的分类**（正常启动 / 恢复异常 / 没有启动进程，启动时一次确定）决定 `started_at` 与 `duration_ms`；**主进程的退出状态**（在 S3 之前可靠取得 / 之后取得 / 没有取得）决定 `exit_code`。
  - 正常启动：`started_at` 为 `t_start` 时的挂钟；`duration_ms` 为 `t_main − t_start` 或（没有取得时）`t_S3 − t_start` 的定义区间。
  - 恢复异常：没有 `t_start`，协议记下 `t_anom`；`started_at` 为 `t_anom` 时的挂钟；`duration_ms` 总是 `t_S3 − t_anom`，只是从观察到异常到进入 S3 的定义区间。进入 S3 之前对主进程句柄做一次零等待的查询；在 S3 之前可靠取得的自行退出，`exit_code` 保留它的退出码，**不因异常而被抹掉**；之后取得或没有取得为 NULL。后来取得的退出状态不反推开始时刻，也不使运行时长成为已知。
  - `exit_code` 的规则对正常启动与恢复异常相同（§6.2 的“正常退出”定义为在 S3 之前可靠取得的自行退出）；两组时间规则由启动的分类区分，不会同时适用。
  - `finished_at` 统一为写事务开始之前的挂钟；`head_end` / `tree_state_end` 在 D19（含恢复异常）时为 NULL / `unknown`。
  - 撤回第 6 版初稿中恢复异常 A6 无条件取 NULL 与 `t_S3 − t_start` 的写法。
- **N-092-1**（§19.7 的前提）：“没有其他进程恢复或挂起这个线程”只是推出“确定未运行”的**工程前提**，不是 D7 的豁免，也不是新的保证例外；前提显示不成立时保守报告、终止、确认并移交。
- **P-2**：启动之前的拒绝与 `SET 0` 收尾失败分开断言；后者的检查已经执行，保留的属性与资源状态如实记录，不算“没有副作用”。
- **Q-2**（§19.13，只登记）：按 Review 092 §6.2 写明 (a) 的字段（与 A6 的恢复异常两行相同）、与取消的先后（推荐保留 D8）、不保存退出码时为信息损失，以及 (a) 是新增的记录语义。A12 的记录继续待 Q-2；Q-2 只涉及恢复异常而终止已确认的情形，不掩盖 A6。
- **§110 的作者推演的更正**（§110 原文不改）：§110 第三个事件序列中“0 → 已运行、时刻未知”按本节改为“可能已运行”；“返回 1 → 已恢复，正常运行”理解为按协议视为已启动；第四个序列中的“可能已运行（或已运行）”同样改为“可能已运行”；A6 的字段按本节的统一规则。

**文档事件序列**（按修订后的条文逐步推演，不声称动态复现）：

| 序列 | 知识状态 | 所用规则 | 时间字段 | `exit_code` 的来源 | 记录 | `qualifying` | 批次的退出码 | 资源归属 |
|---|---|---|---|---|---|---|---|---|
| **1. 返回 0，没有证据证明用户指令执行过** | API：调用之前没有挂起；用户代码可能已运行；没有执行的证据；开始时刻未知；起点 `t_anom`；零等待查询没有退出 → 进入 S3；终止之后取得退出（S3 之后）；Job 的活动进程在 `T_kill` 内没有降为 0 | §8.5 S1 的恢复异常 → S3 → F1 → §8.6 规则 1 第 2 条（D19），A6 | `started_at` = `t_anom` 时的挂钟；`duration_ms = t_S3 − t_anom`（定义区间）；`finished_at` = 写事务之前的挂钟 | NULL（退出在 S3 之后取得，可能由终止造成） | 尝试记录 `error` / `context_changed`，`head_end` NULL，`tree_state_end = unknown`；说明“恢复异常、可能已执行、进程树没有确认结束、结束上下文没有读取” | 否 | 停止，6 | Job 句柄关闭（再次请求终止）；Runner 的进程句柄与 Go 的进程对象移交资源清单，不 `Wait`；线程句柄关闭。若终止已确认 → A12，记录待 Q-2，停止，6，句柄关闭、`Wait` 释放 |
| **2. 恢复异常，可靠取得主进程的退出码，但后代没有确认结束** | 例如返回 `-1`；零等待查询显示主进程已自行退出 7（S3 之前可靠取得）；进入 S3 之后 Job 的结束查询失败 → 保证范围内没有确认结束 | 同上：F1 → D19，A6 | `started_at` = `t_anom`；`duration_ms = t_S3 − t_anom`（不是运行时间：退出码不提供开始时刻）；`finished_at` 同上 | 7：进入 S3 之前从主进程句柄可靠取得的自行退出 | 同序列 1 | 否 | 停止，6 | 同序列 1 的未确认分支 |
| **3. 恢复异常，主进程的退出状态也未知** | 例如返回大于 1；零等待查询没有退出；之后对进程句柄的等待出错（观察失败，C-19）；保证范围内没有确认结束 | 同上：F1 → D19，A6 | `started_at` = `t_anom`；`duration_ms = t_S3 − t_anom`；`finished_at` 同上 | NULL（没有可靠取得） | 同序列 1 | 否 | 停止，6 | 同序列 1 的未确认分支 |
| **4a. 序列 1 ~ 3 之后写入失败**（B2） | U、T、C、O 保持 A6 的值 | §8.6 B2 | 不写入 | — | 这一次没有记录；说明仍写明可能已执行、没有确认结束、可能残留、结束上下文没有读取 | — | 停止，6 | 不变：进程句柄仍在资源清单中 |
| **4b. 之后执行项校验失败**（B3） | 同上 | §8.6 B3 | 回滚 | — | 没有记录；说明同上 | — | 停止，**1** | 不变 |
| **4c. 之后提交结果不确定**（B4） | 同上 | §8.6 B4 | 不确定 | — | 说明“这一次可能已经记录，也可能没有”；读不到记录不是重新执行的依据 | — | 停止，6 | 不变 |
| **对照：正常启动，可靠取得退出状态，后代没有确认结束** | 返回 1（或 Unix `Start`）；按协议已启动；`t_start`；主进程自行退出 3（S3 之前取得，D11 的终止）；后代没有确认结束 | D19，A6 | `started_at` = `t_start`；`duration_ms = t_main − t_start`（按协议起点计的运行时间） | 3 | 同序列 1 | 否 | 停止，6 | 进程句柄移交资源清单 |
| **对照：正常启动，退出状态没有取得** | 返回 1；观察失败 | D19，A6 | `duration_ms = t_S3 − t_start`（定义区间） | NULL | 同序列 1 | 否 | 停止，6 | 同上 |

**本轮修订的范围**：

- 相对审查对象 `25e59b9` 的新增提交：Review 092 的归档、规范草案、data-model 与 acceptance-criteria 的定向修订、本记录与活跃状态。
- Schema、示例、生产代码、迁移、原型代码、测试与 workflow 都没有改动；没有新的机制实验；不新增字段或枚举。

**保持不变**：

- 十四项、C-4 与 D21 是产品已决定；Q-1、Q-2 待负责人决定；
- P-1 ~ P-4 不授权实施；Review 091 B1 的关闭不重开；
- 规范是第 6 版草案，未冻结；生产实现未授权；WP2.5 与 M2 未验收；
- M-14、M-15、M-16、M-17 未解决；N-087-1 未修；
- §106 ~ §110 与历史审查原文没有改动。

**状态**：Review 091 B1 经 Review 092 关闭；B2 的残余 B2-R1、B2-R2 已修订，**待独立复审**；不宣布 B2 关闭。
