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
