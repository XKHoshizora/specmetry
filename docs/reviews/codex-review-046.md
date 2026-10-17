# Codex Review 046：WP2.1／WP2.2 联合独立代码审查

- 日期：2026-10-18（Asia/Tokyo）。
- 仓库：`XKHoshizora/specmetry`。
- 冻结规范基准：`666efe2c907d518b91cd86710a18989efa5dfdb4`。
- PR #21（WP2.1）：[PR](https://github.com/XKHoshizora/specmetry/pull/21)，base `666efe2c907d518b91cd86710a18989efa5dfdb4`，HEAD `1dc17740a44c02f170dcf066d480a77fde4f2e24`。
- PR #22（WP2.2）：[PR](https://github.com/XKHoshizora/specmetry/pull/22)，base **`1dc17740a44c02f170dcf066d480a77fde4f2e24`**，HEAD `221df8877f331e6726bfec033fc20151450dbab3`。它是叠在 #21 上的 PR，不以 master 为本轮差异基准。
- 方法：完整阅读两个固定差异中的生产推导包、共享测试基座、生成器、转换器、复核器和测试；对照冻结规范与必要的 M1 写入调用链；核查 GitHub 元数据和 CI 日志；运行定向测试，并在隔离副本中进行独立原始事实及生成器合法性验证。
- 归档：仅新增本报告。全文使用仓库相对链接或公开 GitHub 链接；代码行号均对应上列固定 SHA。没有修改其他仓库文件，没有提交、推送或合并。

## 结论

| 对象 | 结论 | 原因 |
|---|---|---|
| PR #21／WP2.1 | **NO-GO** | 共享生成器把违反冻结写入规则的历史作为合法历史，影响本工作包性质测试的验收依据（B1）。 |
| PR #22／WP2.2 | **NO-GO** | 独立算法及主要定向反例检查成立，但双向对照依赖同一存在缺陷的合法历史生成器；不能据现有 600 个样本完成 F5 验收（同一 B1 的依赖影响）。 |

**本轮发现一个实质性验收阻塞，位于测试基座；没有独立复现出生产推导器错误产生 `ready` 的合法事实反例。** 这两句话需要分别保留：没有发现错误 `ready`，不能使不符合冻结输入域的测试证据自动有效；生成器有问题，也不能据此声称生产引擎已经发生错误交付判断。

无需重构推导器或复核器。最小修复集中在生成审核操作的合法性、固定应用顺序及对应的独立约束检查，随后重跑受影响的性质测试和双向对照。另记录两项非阻塞证据／接口问题，以及一处冻结文字的窄范围澄清事项。

## 一、身份、差异与范围

开始及结束核查时，两个 PR 都保持 Open，HEAD 和 base 均与任务指定值一致，没有把其他提交混入结论。本地 HEAD 为 #22 的固定目标；开始时工作区干净，本报告文件不存在。

实际差异：

| 差异 | 文件及内容 |
|---|---|
| `666efe2 → 1dc1774` | 22 个文件，新增 6,717 行、删除 2 行：生产推导包及测试、`factgen`／`toderive`、开发文档。共享测试基座属于 #21，已纳入本轮审查。 |
| `1dc1774 → 221df88` | 7 个文件，新增 1,346 行：`readycheck`、`fromdump`、隔离及对照测试、开发文档。没有修改 #21 的推导器或生成器。 |

冻结的 data-model、acceptance-criteria、Schema 没有变化；两个差异也没有新增 CLI、数据库迁移或生产写入模块。没有越界实现 WP2.3，没有批准其他 M2 草案或关闭维护项。

审查依据为 [data-model §6、§6.13～§6.14](../data-model.md)、[acceptance-criteria §3.1／F5](../acceptance-criteria.md) 和 [implementation-plan 的 WP2.1／WP2.2](../implementation-plan.md)。复用 [Review 045](codex-review-045.md) 已成立的规范准备结论，没有重新验收 M1。

## 二、BLOCKER

### B1：Random 生成的“合法历史”违反唯一 Target 与固定应用顺序，性质测试及双向对照的输入域不成立

**位置与归属：**

- #21 `1dc1774`：[factgen/random.go](../../internal/testutil/factgen/random.go) 第 149～223 行，尤其第 172 行开始的 Goal／Requirement 混合循环，以及第 194 行之后的 AC 操作循环。
- #21：[factgen/history.go](../../internal/testutil/factgen/history.go) 第 420～433 行，`Apply` 把每个 primary change 生成为同一 Proposal 的独立 accepted Item；第 439～451 行按调用顺序写入 `change_index`。
- #21：[derive/property_test.go](../../internal/domain/derive/property_test.go) 第 25、126 行及后续性质测试，把 `Random` 输出作为合法历史使用。
- #22 `221df88`：[readycheck/fromdump/compare_test.go](../../internal/testutil/readycheck/fromdump/compare_test.go) 第 41～88 行，600 个样本的双向对照使用同一个生成器，没有独立核对这里暴露的写入约束。

**冻结规则与真实调用链：**

1. data-model §8.3 的 V-P03：同一 Proposal 的同一 `target` 至多出现一次。[proposal.CheckStatic](../../internal/proposal/proposal.go) 第 159～172 行实际拒绝重复 Target；这也适用于 `propose --apply` 的输入。
2. data-model §5.5：Goal → Requirement → AC add → AC update → AC reaffirm → AC retire，步骤内部按 `item_index` 应用。[baseline.Apply](../../internal/baseline/apply.go) 第 155 行起按这些步骤执行，不能把任意回调顺序作为真实审核顺序。
3. data-model §6.14 与 F5 要求双向对照在生产写入能产生的合法事实历史上进行；畸形历史另行测试，不作同一业务结果保证。

**独立复现结果：**全部在固定 #22 代码树的隔离副本中执行；其中 `factgen` 和推导器文件与固定 #21 完全相同。没有调用生产推导结果来判定样本是否合法，也没有把“共享 seq”本身误判为问题。检查区分有 Proposal Item 的显式变更与无 Item 的级联变更。

| 使用的原有参数 | 反例 |
|---|---|
| #21 性质测试的原参数：`Steps:40, Deliver:0.5, Cutoff:0.2, Isolated:seed%3==0` | **seed 2**，截止点 18，seq 15：同一 Proposal `01JAAAAAAA000000000000001K` 的两个不同 Item，先 `reaffirm`、后 `update` 同一 AC `01JAAAAAAA000000000000000C`，`change_index` 为 7、8。它违反同一 Target 唯一，而且其 AC 应用阶段顺序也不符合 §5.5。 |
| #22 对照的原参数：`Steps:40, Deliver:0.55, Cutoff:0.15, Isolated:seed%2==1` | **seed 5**，截止点 20，seq 20：同一 Proposal `01JAAAAAAA000000000000001W` 的 Item `…0024`、`…0025` 两次更新同一 Requirement `…000R`；同一批次的 Item `…0026`、`…0027` 又两次更新同一 AC `…001A`。这是显式 Item，不是级联。 |
| #22 同上 | **seed 3**，截止点 26，seq 10：同一 Proposal `01JAAAAAAA000000000000000R` 的显式变更依次为 Goal add（index 0）、Requirement update（index 1）、Goal add（index 2）。真实应用引擎必须先应用两个 Goal，再应用 Requirement。 |

seed 5 两次 Requirement update 的快照内容甚至相同，但这不使它合法：禁止的是同一 Proposal 重复指定 Target，不仅是禁止产生不同值。

最小复验方式是调用上述 `factgen.Random`，只查看 `seq ≤ AsOf` 的 `BaselineChanges`，通过 `ProposalItemID` 联回 `ProposalItems`，按 `(ProposalID, EntityID)` 检查重复的显式 Target，并按 `change_index` 核对 §5.5 的步骤。临时检查在上述三个 seed 上明确失败；另一次日志核对确认了两条变更确属同一 Proposal 的不同 Item。

**实际影响：**

- #21 的“合法历史性质测试通过”不再是其所宣称输入域上的可靠证据。
- #22 的双方一致可能发生在冻结规范不保证业务结果的历史上；现有 `412 not ready / 188 ready` 分布也不能直接作为合法域覆盖分布。
- 不能简单删掉这些 seed，或用推导器／复核器的结果筛掉它们。那会继续缺少独立的写入合法性依据，并可能缩窄真正需要覆盖的合法历史。
- 此处没有证明两个实现存在错误 `ready`，也没有证明全部生成样本都非法；阻塞的是本轮明确要求的性质测试与 F5 验收证据。

**最小修复：**

1. 在生成一次审核／人工编辑时，先构造操作计划，保证显式 Target 不重复；保留合法的级联与同 seq 多行。
2. 按 §5.5 分阶段应用计划，阶段内保留 `item_index`；不要用 ULID 或任意回调顺序代替业务顺序。
3. 为生成器增加独立的合法性断言，至少覆盖本次两个已证实的约束及上述固定 seed；不要调用推导器或复核器判断合法性。`History` 可以继续服务显式畸形测试，但 `Random` 宣称的合法域必须得到约束。
4. 修复 #21 的共享基座后，重跑 #21 性质测试、#22 正反例及双向对照，重新记录 ready 分布。无需提前实现真实 M2 写入命令。

本项须在两个工作包正式验收前关闭。修复主要落在 #21；#22 的阻塞是可复现的共享依赖问题，不要求复制一套生成器来“解决”它。

## 三、NON-BLOCKER

### N1：两套转换器没有完整保留 F5 指定的决定时序元数据

位置：[factgen/dump.go](../../internal/testutil/factgen/dump.go) 第 224 行保留 `SeenDecisionID`；[toderive.go](../../internal/testutil/factgen/toderive/toderive.go) 第 119～128 行、[fromdump.go](../../internal/testutil/readycheck/fromdump/fromdump.go) 第 65～70 行均未映射它；两个目标 Decision 模型也没有该字段。

F5 的“两套转换器”明确列出了要保留 `seen_decision_id`。因此不能把当前转换器概括为保留了规范列出的全部时序信息。当前 ready 谓词不读取这个写入前置条件，且完整链仍在原始 Dump 中，本轮没有发现因此改变合法输入结果的路径，故不另升级为错误 ready 的 BLOCKER。

建议补齐原样字段及映射断言，保留 NULL 与具体前驱 ID 的区别。它可以不参与 ready 计算，也不要求推导器额外实现畸形输入选项 (b)。后续 SQLite 前端不能把这个遗漏继续带入“完整原始事实前端”的声明。

### N2：R10 的联合断言目前仅覆盖 ready 输出的等级

位置：[readycheck/check.go](../../internal/testutil/readycheck/check.go) 第 374～380 行，只有 `ready == true` 时比较 grade；[compare_test.go](../../internal/testutil/readycheck/fromdump/compare_test.go) 第 54～68 行也只在 ready 分支调用 `Consistent`。

这足以支持本包最关键的“ready 时不能夸大证据等级”检查；没有发现借此错误放行 ready 的路径。但不能据它宣称已对所有非 ready 输出独立核对等级。例如 AC 仍直接接受、项目因来源变化不能 ready 时，等级仍有业务含义。

建议明确帮助函数的适用范围，或补一个“非 ready，但仍有 normal 接受贡献等级”的输出一致性用例。这里是证据覆盖边界，不要求本轮开发 status CLI，也不是已经发现生产等级错误。

## 四、PRODUCT DECISION／需要的窄范围规范澄清

### P1：无 Scope 时，§6.10 的一句总述与项目级差距表存在文字冲突

冻结 data-model §6.10 的项目级表把 `SOURCE_CHANGED` 等列为阻塞；§6.8、F5 R9 也保留来源变化约束。但 §6.10 末尾又写“此时阻塞的只有项目级的 `NO_SCOPE`”。

当前实现采用：没有 Scope 时，条目级状态差距不阻塞，项目级差距仍按表阻塞。合法反例为：已有 Target、没有 Scope，且一个已登记来源的观测哈希变化；实现同时输出阻塞的 `NO_SCOPE` 和 `SOURCE_CHANGED`。这与“只有 NO_SCOPE”的绝对字面不一致。

作者 literal reading 2 是一种与项目级表及保守规则相容的解读，但不能称为冻结文字完全没有歧义。最小澄清为明确末句仅约束条目级状态差距，并说明其他项目级差距仍按表；若负责人要求绝对“仅 NO_SCOPE”，则须相应确定项目级展示规则。

**这不是新发现的错误 ready 路径**：两种解读下 `NO_SCOPE` 都使目标 `not_assessed`。本轮不擅自选择新的产品取舍，不因该句要求重构算法；建议负责人作一句限定范围的澄清，最晚在对外 status／gaps 契约落实前完成。

除此之外，没有发现必须新增产品功能或重新决定证据等级、沿用、设置重放等已冻结语义的必要。

## 五、WP2.1 生产实现的审查结果

以下是实际核查结果，不代表 B1 已关闭或工作包整体已验收。

| 领域 | 核查结果与边界 |
|---|---|
| 截止点与基准历史 | [history.go](../../internal/domain/derive/history.go) 使用截止点内最后的 `after_json` 快照重建状态，以 `seq/change_index` 选择，当前投影只承载不可变身份属性；`revision_seq` 从当前修订首次出现的变更取得。已有测试故意污染当前投影，验证旧快照不被替换。 |
| Target／Scope | Target 按创建、归档区间选择，Scope 只选该 Target 最新确认；不继承旧 Target 的 Scope。截止点没有活动 Target 时有效 S 为 NULL。RS／AS 只包含当前仍 active 的需求及 AC。 |
| 来源历史 | 重建登记／移除区间；截止点前最后 ack 生效，没有旧 ack 时用第一条确认的 previous hash 恢复登记值；按 ID 重算来源集合哈希。未经 ack 的磁盘变化只进入观测，不覆盖已确认哈希。 |
| 提案与问题 | 审核截止点之后的 Item 仍为 pending；Question 的实体关联由截止点内接受变更恢复，不直接信任当前回填结果；关闭时间晚于截止点的问题仍 open。 |
| 共享 seq | 转换器逐行保留，历史选择使用 change_index／记录身份；没有按 seq 去重。全库 LatestSeq 是截止点完整性的高水位，不错误要求本项目必须恰有同序号事实。B1 是生成的写入内容非法，不是共享 seq 本身非法。 |
| 合格执行、V、D、HIST | [criterion.go](../../internal/domain/derive/criterion.go) 检查非 NULL 且相等 HEAD、两端 clean、无 context_changed、合法等级；V 绑定当前 AC 修订、配置及 S。失败／错误不当作通过；D 按当前修订最大 seq；HIST 不把 Claim 当验收。 |
| 决策规则 | 11 条状态规则按优先级实现；父需求变化先于豁免，合格失败先于接受，拒收解除使用代码变化及严格后续证据。直接适用与隔离级沿用分开；MVP 观测级的新快照通过不能自动 accepted。未发现合法事实上的错误 ready 反例。 |
| 聚合、差距与指标 | 需求状态、目标状态、空范围／无 AC／全部豁免的保守结果、SOURCE_REMOVED 非阻塞与 SOURCE_CHANGED／SCOPE_STALE 阻塞分别处理。主指标及零分母遵守冻结定义。部分展示口径见下一节，不把它们当自动验收证据。 |
| 证据等级 | 只聚合范围内实际 accepted 的 normal 接受贡献；直接取对应证据，沿用取 S 的隔离级执行；manual／exception／waiver 不伪装为自动验证，observed 为较弱等级。MVP 与合成隔离级测试域分开。 |
| 纯度与别名 | 不仅检查导入白名单，也追踪实际调用：哈希写入仅进入内存 hash／Builder，没有文件、网络、数据库、子进程、时钟或随机源调用。结果中的列表、映射、视图独立构造；定向修改返回结果没有修改原输入。 |
| 必需输入与未知 | nil Snapshot／Settings 是缺项；空 Commit 是合法 NULL；缺少某来源观测与该来源观测为 NULL 分开。观测集合缺项、重复、多余会拒绝；没有静默从外部补齐。任意畸形事实的业务结果仍不在选项 (a) 保证范围。 |
| 重放与 ChangesSince | Result 记录 S、三项设置和来源观测。ChangesSince 单独计算 AC 状态／进度投影，不调用完整 Derive 后吞掉旧来源观测缺项；两次计算共享本次设置和 S，只改变事实截止点。其保证不是回放过去的磁盘或过去设置。 |

## 六、作者实现解释的逐项裁定

这里依据规范和调用链判断，不以“两个实现相同”或“已有测试通过”为充分理由。

### PR #21 的 12 条 Literal readings

| # | 裁定 | 依据／限定 |
|---|---|---|
| 1：进度只计直接关联 AC 的提交 | **符合冻结规则的保守解读** | §6.5 对 AC a 取事实；§4.12 允许关联到 Requirement 或 AC，但没有规定把 Requirement 的提交向全部子 AC 传播。需求聚合也不是传播提交的接口。只关联父需求的提交不应凭空证明每个子 AC 开始实施。以后增加传播属于另一个明确规则，不能用本轮测试一致性代替批准。 |
| 2：无 Scope 仍保留其他项目级阻塞 | **真实文字歧义** | 见 P1。实现与项目级表及总体保守规则相容；“只有 NO_SCOPE”需限定。两种解读不改变此时目标不能 ready。 |
| 3：无 active AC 时 coverage 为 null | **合理工程细节** | 比例的 0/0 未被定义为 100%；不产生 INVENTORY_INCOMPLETE，也不凭空满足 coverage=100%。docs_only／inventory_skipped 仍可按独立条件达到 inventoried；空 AC 的交付仍由 needs_criteria 等规则约束。 |
| 4：exception_acceptance 标记只看 §6.2 适用性 | **符合冻结规则** | snapshot 相等与最终 AC 状态是两层判断；parent_revised 或后续失败可以优先决定状态。保留例外标记不等于 accepted，也不提升 evidence_grade。 |
| 5：blocking_questions 为项目计数，exception_acceptances 为 AS 计数 | **合理工程口径，需如实标注** | §6.11 对这组字段只写“计数”，没有逐一规定范围。项目未关闭问题计数不等于本次 Scope 的 BLOCKING_QUESTION 差距数量；当前实现未用前者绕过或扩大 ready 谓词。建议对外接口说明两者区别，不能把该计数当阻塞差距数量。 |
| 6：旧截止点没有活动 Target 时 S=NULL | **符合冻结规则** | F4 的未知状态与 Target 区间规则明确支持，包括 diff 起点尚无 Target 的情形。 |
| 7：差距稳定顺序及按 Question／Source 分条 | **合理工程细节** | 满足“阻塞优先、类型、条目编号”的规则；稳定数字编号排序，没有用 ULID 判断业务时序。Question key 和来源 path 用于定位，不改变交付规则。 |
| 8：未规定的 next_action 文案 | **合理工程细节** | 提示文案不是新的状态谓词或产品行为，不需要为一句“排查结果不一致的检查”新增产品决定。 |
| 9：workspace_run 与 is_target_snapshot | **合理的工作区展示解释** | latest run 绑定当前修订、配置、worktree 及其 HEAD，可包含不合格执行，且输出 Qualifying；IsTargetSnapshot 仅表示 HEAD=S，dirty 时仍可能为 true。独立验证了这个事实。它不能单独被消费者解释成干净工作区或快照已验证，目标判断没有读取该布尔值作证据。 |
| 10：rule 7 只看当前决定，兜底 first_review | **符合冻结规则；兜底限于不受业务保证的组合** | §5.4 定义了当前决定，旧修订决定已不是 D；正常合法状态按 §6.3 的优先级给原因。未发现兜底使 accepted／ready 被错误推出的路径，不能把兜底结果视为畸形输入的业务承诺。 |
| 11：Decision summary 的等级 | **符合证据展示边界** | normal 或带证据的 rejection 可展示所依据执行的等级；manual／exception／waived 为 none。拒收展示证据不等于交付接受；aggregate 只计实际接受贡献。 |
| 12：未知等级不合格，非 pass 作为失败 | **符合保守规则** | 未知等级属于畸形；正常 error／fail 不被当作 pass。context_changed 另行排除，不能用未知值制造隔离证据。 |

### PR #22 的两条解释

| 解释 | 裁定 |
|---|---|
| R7 仅在非空 AS 全豁免时失败；空 AS 由 R4／R5 失败 | **合理的失败归因处理**。不会把空 AS 放行为 ready，也没有缩窄必须拒绝的输入集合。 |
| R8：pending Proposal Item 的问题不是“没有关联任何条目”的项目级问题 | **符合冻结谓词的字面及实体边界**。它有明确 Proposal Item 关联，尚不是 RS／AS 的正式条目；被接受后再按产生的实体关联判断。独立原始事实对照验证了该边界，不仅依赖双方相同解读。PR #22 描述把它交叉指向 #21 literal reading 2 不准确，实际对应的是问题关联规则；不影响实现。 |

上述解释中没有发现需要另行批准沿用、例外验收、空 AC 自动就绪或 Claim 升格的新语义。B1 不是解释选择，而是已冻结写入规则的实际违反。

## 七、独立复核器、转换器与测试的可信范围

### 1. 独立性

[readycheck](../../internal/testutil/readycheck/check.go) 有自己的事实模型、历史重建、来源集合哈希、配置哈希、执行筛选、决定适用性与输入比较。其生产逻辑只用标准库，没有导入引擎、scope、baseline 或经其他包间接使用它们的选择器。实际代码审查与导入隔离测试共同支持这一点；不只根据白名单得出结论。

[fromdump](../../internal/testutil/readycheck/fromdump/fromdump.go) 与 [toderive](../../internal/testutil/factgen/toderive/toderive.go) 分别解析快照及输入记录，不预先按截止点筛选，不把一方选出的当前决定、V 或适用性传给另一方。两者只映射各自计算需要的字段，不能声称完整无损复制原始账本，特别是 N1。

生成器使用生产 hash／快照辅助函数构造某些合法形状，不等于复核器复用这些计算；复核器确实独立重算。独立性仍不能补救 B1：共享的事实构造若错误，两边完全可以在同一个非法输入上同时通过。

### 2. 正反例及双向性质

已检查复核器的 19 个条件反例、rejected Item 不阻塞的对照、观测级 normal 正例、manual＋exception＋waiver 正例、共享 seq 正例和通用合成沿用正例。断言检查具体失败条件，允许结构上必然连带失败，没有只检查程序成功返回。

连接层确实同时检查两个方向：引擎 ready 而复核器失败、复核器 ready 而引擎不 ready 都会失败；也保证 ready 与非 ready、三种等级均出现，并检查不同设置不改变 ready。

这些机制成立，但**样本合法性的前提未成立**。当前 400／600 个样本的绿色结果，只能描述“现有生成输入上的结果”，不能保持作者“全部为合法历史”的结论。修复 B1 后须重新取得合法域分布。

对畸形输入，两方不崩溃、确定性的检查与合法域双向对照分开，符合选项 (a)。本轮不要求纯函数验证所有生产写入不变量，也不要求现在完成 SQLite 前端。

### 3. 测试先行

提交历史确认 #21 先有 `c4f9490` 的类型、桩、测试及基座，再有 `fb0d918` 引擎；#22 先有 `d36455d` 的模型、桩、转换器与测试，再有 `114e601` 复核器。之后分别有 lint 修订及 #22 合入 #21 的修订。

作者报告桩阶段 #21 为 53／56 失败，#22 为 6／10 失败。本轮核对了提交顺序，**没有独立运行这两个桩阶段**；这些失败数量保留为作者证据，不改写成独立复验。

## 八、本轮实际运行的验证

平台：Windows，本机 PowerShell，Go `1.27.1 windows/amd64`。使用真实 Go 可执行文件，`GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`-mod=readonly`，没有下载新依赖或调用远端。

### 1. 现有新增包测试

在固定 `221df88` 组合树运行：

```text
go test -tags nowebui -mod=readonly ./internal/domain/derive ./internal/testutil/factgen/... ./internal/testutil/readycheck/... -count=1
```

结果：

| 包 | 结果 |
|---|---|
| internal/domain/derive | PASS，独立简洁重跑显示 1.686s |
| internal/testutil/factgen | 无本包测试文件；由 derive／readycheck 测试使用 |
| internal/testutil/factgen/toderive | 无本包测试文件；由连接层使用 |
| internal/testutil/readycheck | PASS，0.086s |
| internal/testutil/readycheck/fromdump | PASS，1.095s |

也执行了同范围的 `-v` 运行，进程正常成功退出。#21 的推导器和 factgen 在 #22 差异中完全未修改，因此这里的相关包结果适用于 #21 固定源码；它不是声称另行检出了 #21，也不能用本机 Windows 结果冒充本机 Linux Race Detector。

### 2. 不使用 History／Random 的独立原始事实

在隔离副本新增临时测试，以字面原始行构造一个合法的 Goal／Requirement／AC、Proposal 与已接受 Items、共享 seq 的 Version／Changes／Scope，以及独立计算的配置哈希和来源集合哈希；再按规则构造边界。没有用生产引擎选择结果或生成器筛选样本。以下 **13 项全部 PASS**：

1. 同快照观测级 normal 接受为 ready／observed。
2. 全库高水位高于本项目最后事实，仍可合法推导。
3. 当前来源投影已有后续 ack，旧截止点仍恢复旧确认哈希。
4. 未确认磁盘变化阻止 ready。
5. NULL 来源观测为合法“文件缺失”，阻止 ready。
6. 缺少观测记录返回 InputError，不被当成文件缺失。
7. 接受后在同快照合格失败，变为 failing／regressed，目标不能 ready。
8. 后续拒收覆盖此前接受，目标不能 ready。
9. 新快照只有 observed pass，不沿用此前接受。
10. 通用合成隔离级 pass 可以依冻结条件沿用，等级 snapshot_isolated。
11. 问题关联 pending Item，不被伪装成无关联项目级问题。
12. dirty workspace run 不覆盖目标的合格 snapshot run；HEAD=S 的展示布尔值与执行合格性分开。
13. 修改返回的列表、Scope、观测、决定摘要及指标，没有修改原输入。

这些是代表性独立反例与正例，不是穷举证明，也不能替代 B1 所要求的合法生成域。

### 3. 生成器合法性审计

隔离临时测试的以下断言 **失败，符合发现 B1 的预期**：

- #22 原参数下检查显式 Target 唯一：seed 5 失败。
- #22 原参数下检查 Goal／Requirement 阶段顺序：seed 3 失败。
- #21 原参数下检查同 Proposal 的显式 Target 唯一：seed 2 失败。

随后只打印上述原始记录的诊断测试 PASS，确认了具体 Proposal、Item、Entity、seq 与 change_index；它不是把非法历史重新判为合法。

本轮没有进行新的网络、外部 Helper 或 Git 安全攻击实验，没有把临时测试移入原仓库。最初一次临时副本准备没有形成可用 Go module，Go 在测试执行前报 module 缺失；之后改用独立、可确认源码完整的隔离副本完成上述验证。该准备错误不是项目代码失败，也不计为测试通过。

## 九、CI 身份、合并快照与历史

两组指定 CI 都是 **Attempt 1，8／8 SUCCESS**。八个 Job 分别为 Web、Go lint、Windows／Ubuntu／macOS Go tests，以及三个平台的 embedded-dashboard Build。已读取 16 个 Job 的日志并核对检出身份。

| 对象 | workflow HEAD | 实际检出的 PR 合并快照 | 两个父提交 | 树 |
|---|---|---|---|---|
| [#21 CI 39066962753](https://github.com/XKHoshizora/specmetry/actions/runs/39066962753) | `1dc1774` | `f61e346ea86c5d5336ba7b30d4a65c5df89e0e19` | `666efe2`、`1dc1774` | `e5f9290bf2bc4fb81eb04344cf70b5c5c918a275` |
| [#22 CI 39067048337](https://github.com/XKHoshizora/specmetry/actions/runs/39067048337) | `221df88` | `61e54ddc47685a8bbbd7410767d74679581a3cfd` | `1dc1774`、`221df88` | `7505d5c43a88717e03117424d7f47ab9174f5537` |

两个合并快照的内容树分别与对应固定 HEAD 相同；没有混用 workflow HEAD 与 checkout SHA。#22 的验证是相对 #21 的组合树，不是相对 master 的独立功能分支。

三个平台日志均有 derive 性质测试结果 `103 ready / 297 not ready`；#22 另有对照结果 `412 not ready / 159 ready observed / 19 ready snapshot_isolated / 10 ready none`，R1～R9 都出现失败归因。上述新测试确实运行，没有把它们的 SKIP 当成 PASS。

Ubuntu Job 的实际脚本在 Linux 设置 `race=-race` 并执行完整 Go 测试，相关包成功；这是 GitHub Ubuntu Race Detector 证据。本机没有另行执行 Linux Race Detector，也没有独立重跑 macOS／Windows 以外的实际平台。

保留失败历史：[旧运行 39066356194](https://github.com/XKHoshizora/specmetry/actions/runs/39066356194)，HEAD `2e6e02b`，Go lint 失败、Windows Go tests 被取消；作者记录为两项 staticcheck QF1001／QF1002，后由 `1dc1774` 修订。不能把最新首轮 8／8 成功写成“整个 PR 从未失败”。

CI 证明提交中的现有检查成功，不证明本轮未提交的临时审计也通过；B1 恰是绿色 CI 未检查到的生成器前提。

## 十、ACCEPTED RISK 与未验证边界

本轮**没有新增接受风险**。B1、N1、N2、P1 都不归入接受风险，也没有关闭既有维护事项。

保留已批准的边界：

- MVP 只有 observed 执行；`snapshot_isolated` 仅用于通用合法域的合成单元测试，不代表 MVP 具备隔离执行或自动沿用能力。
- 畸形输入沿用已批准的选项 (a)：不保证业务结果；不崩溃及确定性与合法域的正确性交付保证分开。
- workspace 展示不代替目标快照证据。

后续仍需各自工作包验证，不能由本轮纯函数测试提前宣称通过：

| 后续边界 | 本轮没有证明的内容 |
|---|---|
| 数据库读取前端 | 一次真实读事务中的事实、高水位、设置取值与完整历史装配；跨项目隔离；实际缺项识别。 |
| Git Reader／观测 | S、工作区、inputs 及文件观测的真实取得，PD-26、路径与对象错误边界。 |
| 写入命令 | Claim、执行、Decision 的真实前置条件、seq 分配、事务回滚和并发保护。 |
| status／gaps CLI | 完整 Schema、字段映射、JSON 输出、diff_since_seq、真实命令重放。 |
| WP2.9 | SQLite 复核器前端、真实数据库双向对照、CX 集成与 E2E。 |

没有重新审查或验收 M1；没有将其他 M2 草案、恢复方案或维护问题因本轮 CI 绿色而关闭。

## 十一、逐项回答与接手建议

1. **PR #21 是否 GO？NO-GO。** 生产推导的主要规则、历史重建、纯度、重放及独立边界用例审查成立，但合法生成域性质测试存在 B1，尚不满足完整 WP2.1 验收证据。
2. **PR #22 是否 GO？NO-GO。** 复核器逻辑与隔离性未发现新的错误 ready；F5 双向对照受共享 B1 阻塞，尚不能完成 WP2.2 验收。
3. **共享基座的可信范围？** Dump 及两个转换器没有复用引擎筛选结果；History 可表达共享 seq、修订与失败历史；Random 的“生产合法历史”声明已被三个原参数 seed 反例推翻，须修复后重建覆盖证据。
4. **实现解释是否都需要负责人决定？** 不需要。大多数符合冻结规则或属于普通展示／诊断细节；无 Scope 的绝对“只有 NO_SCOPE”存在 P1 的局部文字冲突，需窄范围澄清。没有授权新的产品取舍。
5. **两个工作包是否已满足各自验收？** 当前均未满足，原因是本报告明确复现的测试输入域缺陷，不是要求提前完成后续 CLI、SQLite 或 E2E。
6. **后续合并与改基条件？** 先在 #21 修复共享基座、补合法性断言，#22 同步依赖并重跑本轮受影响测试；取得相应修订审查及 CI 后，才考虑先合并 #21，再将 #22 改基到实际 master。改基须核对差异仍只属于 WP2.2、共享源码与审查修订一致、无冲突引入新行为，并验证新的 CI／实际合并树。当前两个 PR 都不建议合并。

可交给 Claude Code 的接手任务：仅修复 `factgen.Random` 的审核操作计划，确保同 Proposal 显式 Target 唯一及 §5.5 应用顺序；补独立于 derive／readycheck 的生成合法性回归，覆盖本报告 seed 2、3、5，保留合法共享 seq、级联、正例及不能 ready 的历史。同步 #22 后重跑性质与双向对照，更新证据，不删除失败 seed 来规避、不修改冻结规则、不启动后续工作包。

**本次联合审查结束。原仓库除新增本报告外保持不变；没有提交、推送、合并或宣布 M2 整体验收。下一步等待负责人安排修复与复审。**
