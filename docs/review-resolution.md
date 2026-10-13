# Codex 审查意见处理记录

> 状态：2026-10-14 · 对应设计版本 Draft v0.3
> 原始审查：[reviews/codex-review-001.md](reviews/codex-review-001.md)（第一轮，§1 ~ §4）、[reviews/codex-review-002.md](reviews/codex-review-002.md)（冻结审查，§5）、[reviews/codex-review-003.md](reviews/codex-review-003.md)（最终设计审查，§6）、[reviews/codex-review-004.md](reviews/codex-review-004.md)（M0 代码审查，§7）、[reviews/codex-review-005.md](reviews/codex-review-005.md)（M0 第二次代码审查，§8）、[reviews/codex-review-006.md](reviews/codex-review-006.md)（M0 第三次代码审查，§9）、[reviews/codex-review-007.md](reviews/codex-review-007.md)（M0 最终定向复审，§10）、[reviews/codex-review-008.md](reviews/codex-review-008.md)（M0 代码审查 GO，§11）、[reviews/codex-review-009.md](reviews/codex-review-009.md)（M1 设计审查 GO，§14）
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
