# Codex Review 034：M1 WP1.8 首次正式独立代码审查

- 审查日期：2026-10-17（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #14](https://github.com/XKHoshizora/specmetry/pull/14)
- 目标提交：`90beb48d3c5cc2d8b7a7cbbc045b0925ab793ead`
- 基准提交：`fbd6dff75dad0133a6811d2f1f7640616901a1ee`
- CI：[GitHub Actions 38939986285](https://github.com/XKHoshizora/specmetry/actions/runs/38939986285)
- 定位：WP1.8 的首次完整独立审查，不是定向补丁复审。
- 范围：Delivery Target、Delivery Scope、审核附带范围确认、迁移、事实不变量、并发和版本前置条件、来源集合哈希、豁免集合比较、交换 Schema、人工确认、隐私与直接回归。
- 依据：implementation-plan 的 WP1.8，PRD FR-TGT-01～07，data-model §4.2、§5.3、§5.5、§7.1，architecture §5.1、§6.1.7、§6.6、§9.1、§10.2，M1 相关验收标准，以及 scope.v1、target-scope.v1、review.v1。
- 方法：实际差异与必要调用链静态审查、三平台 CI 元数据及日志核查、隔离副本中的现有测试、独立 CLI 反例、独立连接竞争、临时触发器故障注入及逐行业务数据比对。

**最终结论：NO-GO。** 发现 **2 项 BLOCKER**，均为新增自由文本入口违反已冻结的隐私校验规则。Target/Scope 的主要事务与并发机制通过验证，但当前提交不宜验收或合并。

## BLOCKER

### B1：交互式 Scope 备注绕过 Schema，允许代码原文落盘

位置：

- [target.go:340](../../internal/cli/target.go#L340)：文件输入经过 Schema 校验后扫描全部字符串；交互式参数只做密钥扫描。
- [target.go:362](../../internal/cli/target.go#L362)：交互式输入直接构造 Input，随后只执行 `scope.Check`。
- [target.go:437](../../internal/cli/target.go#L437)：未经格式校验的备注直接进入 `Note`。
- [confirm.go:169](../../internal/scope/confirm.go#L169)：原样持久化备注。

`scope.Check` 检查业务前置条件，但不验证备注格式。`-m` 路径没有执行 `scope.v1` 对备注规定的禁止代码围栏、长度等限制。

**规格冲突：**

- [architecture.md:893](../architecture.md#L893) 明确要求：“所有自由文本都不能包含代码围栏（由 schema 强制）”，列举的自由文本包含备注。
- [scope.v1.schema.json:39](../../schemas/scope.v1.schema.json#L39) 的 `note` 引用 `plainText1000`。
- [common.v1.schema.json:82](../../schemas/common.v1.schema.json#L82) 要求备注长度为 1～1000 字符，并禁止代码围栏。

**独立复现：**

在隔离的已初始化项目中，已有 Baseline version 2 和有效 Target。将以下合成备注分别通过文件和 `-m` 提交：

````text
```go
func Review034SyntheticCodeCanary() {}
```
````

| 输入路径 | 实际结果 |
|---|---|
| `scope confirm --file … --json` | exit 2，确认前拒绝，无业务写入。 |
| `scope confirm -m … --json`，正确输入 `v2` 确认 | exit 0，备注原样保存到 `scope_confirmations.note`。 |

通过 SQL 读取确认了原值，并在实际 `specmetry.db` 文件中找到了该合成代码。反例使用合成文本，不涉及真实客户源码或凭据。

**实际影响：** 同一项业务输入因进入路径不同而受到不同的隐私保护；交互式入口能够把按契约应拒绝的代码围栏内容持久化。人工确认和终端转义不能代替写入内容校验。

另一个同源反例：1001 字符的 `-m` 备注先进入人工确认，之后才因数据库长度约束返回 **internal / exit 1**。事务虽然完整回滚，但输入错误分类也不正确，应在确认前按 invalid_input / exit 2 拒绝。

**最小修复：** 将交互式输入规范化为 `scope.v1` 文档，在显示确认前执行与文件输入相同的 Schema 校验；正确处理省略的可选字段。补充文件与参数输入的等价反例，要求均为 exit 2、无确认提示、无写入、seq 不变。

### B2：Target 名称同样允许持久化被禁止的代码围栏文本

位置：

- [target.go:68](../../internal/scope/target.go#L68)：`CheckName` 仅检查 UTF-8、空值、长度和控制字符。
- [target-scope.v1.schema.json:34](../../schemas/target-scope.v1.schema.json#L34)：名称约束也未禁止代码围栏。

**独立复现：**

在隔离项目执行 `target set`，ref 为 `refs/heads/review034-release`，名称为：

````text
```go func Review034TargetCodeCanary() {} ```
````

正确输入 ref 确认后：

- 命令返回 exit 0。
- 活动 `delivery_targets.name` 原样保存该值。
- `target show --json` 返回同一原值，并通过现有 `target-scope.v1` 校验。

**规格冲突与实际影响：** Target 名称是新增的自由文本，仍受 architecture §10.2 的统一隐私规则约束。当前名称校验和输出 Schema 都允许被该规则禁止的代码围栏文本进入持久化数据。单行限制和终端转义不能代替这项写入校验。

**最小修复：** 在名称校验中拒绝代码围栏，并同步输出 Schema。新增公开 CLI 反例，验证 exit 2、无确认提示、不创建或归档目标、seq 不变。

## NON-BLOCKER

### N1：跨项目需求外键测试可能由另一条外键提前拒绝

位置：[scope_test.go:87](../../internal/store/sqlite/scope_test.go#L87)。

该用例将属于项目 A 的 confirmation 和 requirement 配上项目 B，因此可能先因 confirmation 的项目外键失败，不能单独证明 requirement 的复合外键有效。

建议增加真实属于 B 的 Requirement，再用 A 的合法 confirmation/project 引用它。

**实际约束未发现缺陷：** 本轮已独立构造该前置条件，数据库正确拒绝。此项属于仓库测试的覆盖准确性问题，不是生产外键失效。

## PRODUCT DECISION

无新增必要产品决策。两项 BLOCKER 都有现成的明确契约依据，不需要重新讨论产品架构或改变隐私保证边界。

## ACCEPTED RISK

本轮未自行接受新风险。

M1 有效豁免集合为空、真实 Decision 生命周期及交付状态推导延期至 M2，均按已经批准的阶段边界评估，没有作为缺陷。没有将其他尚未解决的历史安全事项归类为已接受风险。

## Target 审查结果

除 B2 的自由文本校验外，主要行为符合既定设计：

- 数据库部分唯一索引保证每个项目最多一个活动 Target。
- 仅接受完整分支或标签 ref，再通过受控 Git 的 `check-ref-format` 判断合法性；远端 ref、短名称、非法 ref 和密钥输入被拒绝。
- `SetTarget` 在取得写锁后的事务内重新比较提示时的 Target ID，防止旧会话归档另一个会话的新目标。
- 归档旧目标与创建新目标共用一个 seq，并属于同一事务。
- 新目标没有旧目标的 Scope，历史目标和历史确认仍保留。
- Snapshot 在读取时通过 `CommitOf` 解析，不写入为固定事实；未解析 ref 与已有对象不可读取的情况被区分。
- 分支、annotated tag、未解析 ref、非 Git 项目及无默认目标的主要路径已有实际测试。
- 未改名的同 ref 操作不分配 seq，不改变 Scope；改名创建新目标。

### 同 ref no-op 的并发语义

位置：[target.go:244](../../internal/cli/target.go#L244)。

no-op 可以在读取当前目标的事务处线性化。它不执行写入，不会覆盖其后并发建立的新目标。响应表示本次读取时该目标已存在，不能承诺返回之后目标仍未变化；按已批准的无变更语义，这一时间边界不构成 BLOCKER。

## Scope 与 V-T01 / V-T02 审查结果

- `target show --json` 与 `scope show --json` 使用同一读取及文档构造逻辑，并有实际输出一致性检查。
- 没有目标、没有确认、已有确认及过期确认的主要输出正确。
- 默认选中全部 active must Requirement，允许明确修改清单；空数组、重复、退役或不存在的需求被拒绝。
- Scope Items 保存 Requirement 粒度及确认时的 Revision。
- `complete_scope`、排除的 pending/conflict 数量、`incomplete_docs` 声明均有对应业务检查。
- Source Set Hash 来自已确认来源集合；独立修改来源文件但不 ack，没有改变确认输入哈希，也没有被错误当作已确认的来源变更。
- 版本或已确认来源集合哈希变化时，Scope 正确标为 stale。
- 文件输入严格经过 `scope.v1` 和全部字符串密钥扫描；交互式备注的格式缺口见 B1。

### V-T01

M1 的 `effectiveWaivers` 明确返回空集合，符合当前尚未实现 Decision 的阶段边界。`WaiversUnder` 按所选需求筛选，`Check` 排序后精确比较 acknowledged IDs，并对缺失、额外或无效集合返回 exit 5 和当前集合。

合成非空集合测试验证了筛选、顺序无关和精确匹配；M1 的真实写入与输出验证了空集合语义。接口保留了 M2 查询有效豁免的连接点。本轮没有把合成集合测试描述为已经验证真实 Decision 生命周期、Revision 失效或最新决定推导。

### V-T02

写入前在 SQLite `BEGIN IMMEDIATE` 已取得写锁之后重新读取当前状态并比较确认 ID。首次 null、旧确认 ID、重新读取当前确认后成功等主要路径正确。

本轮额外独立启动两个 CLI 会话，使用同一份 C1 输入；在两者都已经显示确认并等待输入后，同时放行：

- 一个 exit 0。
- 另一个 exit 5。
- 失败方 `error.current.scope_confirmation_id` 等于胜出确认 ID。
- 只新增一份确认、只增加一个 seq，失败方没有残留写入。

该验证使用两次独立 CLI 初始化及数据库连接，没有以顺序调用代替竞争。

## Review + Scope 原子性

实现先执行 `baseline.Apply`，再在同一事务中执行 `review.ConfirmScope`，共享 Review 的 seq。需求 item_id 只允许解析为本次审核接受的新增 Requirement；错误类型、拒绝或未接受条目等会拒绝。Scope 绑定审核后的版本，pending/conflict 数量也按审核后的状态检查；全部拒绝时不创建空 Baseline Version，仍可以确认现有范围。

### 独立的晚期失败验证

在隔离副本中构造包含 Goal、Requirement、AC、Source Ref、Code Ref 和关联 Question 的提案，并通过公开 Review 命令提交 Scope。

临时 Scope Item 插入触发器仅在以下效果已经在事务中可见时才注入失败：Baseline Version、Goal、Requirement、AC、已审核 Item、Question 关联、Source Ref、Code Ref 及 Scope Header 均已经产生。这样，失败不能被“Baseline 尚未开始应用”这一前置条件缺口掩盖。

实际分别验证：

1. 成功演练后取消：全部事实不变。
2. 演练中的晚期 Scope Item 插入失败：未显示确认，全部事实回滚。
3. 演练成功后，在等待确认时由另一连接安装故障触发器：正式事务的晚期 Scope Item 插入失败，全部事实回滚。
4. 已有 Requirement 与 AC 的 material 更新应用后，后续 Scope 来源哈希检查失败：Revision 和相关事实全部回滚。

比对不是仅检查行数：独立读取相关业务表的全部行、全部列，覆盖 Baseline Versions、Changes、Proposal 状态、实体及 Revision、编号计数器、Question 关联、引用、Scope Header/Items 和全局 seq。

移除故障后正常重试成功，新 Requirement 仍获得首个应分配的编号，没有因失败或演练跳号。

### 确认期间被其他连接更新

演练成功后，由另一连接单独写入 C2，不推进 Baseline Version。旧 Review 随后正式提交：

- 返回 exit 5，指向 Scope 的过期前置条件，并返回 C2。
- Baseline、Proposal、编号、Revision、引用和历史保持 C2 提交后的完整状态。
- 重新读取 C2 后，原 Review 可以正常应用并确认新范围。

因此，V-T02 不只是独立 Scope 命令有效，附带在 Review 中的正式写事务也重新检查。

## 迁移、数据库与 Schema

- 目标增量仅新增迁移 0006，0001～0005 未修改。
- 独立使用前五份真实迁移建立已有数据库，写入项目、Target、Source、Baseline、Proposal、Question 等记录后升级到 6。
- 升级结果为 5→6，并产生迁移备份路径；旧业务行内容和业务 seq 保持不变。
- 跨项目 Target/Requirement 关联受到复合外键约束。
- Scope 历史 UPDATE/DELETE 受只追加触发器保护；独立验证 Scope Item 的 REPLACE、UPSERT 不能覆盖历史事实。
- Scope 至少一个 Requirement 由应用层的 Check 和写入调用链保证；没有把数据库表本身宣称为能独立强制这一跨行不变量。
- 现有 Purge 相关约束测试通过；没有扩大为重新验收整个 M0 或尚未完成的 M2 purge 条件。

Schema 条件约束、两个新增有效示例和旧示例均实际执行：**38 个有效示例、146 个非法反例全部通过**，包括 SCH-140～146。输出一致性测试额外检查 Target ID、当前 Scope Confirmation ID 等 Schema 无法表达的关系。

但绿色 Schema 测试未覆盖 B1 的交互式旁路，也未发现 B2 的名称格式规则缺失，不能据此宣称全部隐私契约已经满足。

## 人工确认、JSON 与直接回归

- `target set`、`scope confirm` 和 Review 附带 Scope 复用 `cli.confirm`。
- Agent、非 TTY、已知 Agent 环境变量、stdout 重定向等拒绝路径有真实命令与终端测试。
- 新操作已加入统一 JSON 确认矩阵：JSON 模式提示在 stderr，取消与读取失败时 stdout 为完整 `error.v1`。
- stderr 非终端、提示短写入等情况维持 exit 6、不读取确认、不写入。
- 终端转义继续有效，JSON 解码后保留业务原值；但转义不修复 B1/B2 的写入内容问题。
- 相关命令行脱敏和 Cobra 隐藏补全拒绝回归通过；没有重新展开历史安全审计。

## 测试证据与覆盖缺口

隔离目标副本中实际运行并通过的相关现有测试包括：

- Scope 包：`TestCheck`、`TestCheckWaivers`、`TestReadAndWrite`、`TestConfirmRace`、`TestSetTarget`。
- CLI：`TestScopeInReview`、`TestTargetShowAndSet`、`TestTargetWithoutOne`、`TestTargetSetWhileConfirming`、`TestScopeConfirm`、`TestScopeConfirmedMeanwhile`、`TestTargetAndScopeNeedAPerson`、`TestTargetAndScopeConfirmationNotShownWhole`。
- SQLite：`TestScopeTablesConstraints`、相关迁移备份及迁移序列检查。
- Git：`TestValidRefName`。
- Contract：全部 Schema 编译、38 个有效示例、146 个非法反例及相关错误脱敏检查。
- 直接回归：`TestConfirmationWithJSON`、`TestCommandLineStillWorks`、`TestShellCompletionIsRefused`。

额外独立验证涵盖两个隐私缺口、晚期 Scope 失败的全部回滚、已有实体 Revision 回滚、Review 确认期间 C2 更新、两个同时等待确认的 CLI 会话、未经 ack 的磁盘变化，以及真实 5→6 升级和正确前置条件下的跨项目需求关联拒绝。

两个隐私反例的“运行成功”是指成功复现当前违规行为，不代表对应产品验收通过。自建原子性测试最初的来源引用字段名不正确，修正测试自身前置条件后重新运行，最终晚期故障路径和全部回滚均实际验证通过；没有将前置条件失败视为成功的故障验证。

现有覆盖缺口需要在修复中补入仓库：交互式备注与文件输入的格式等价性、Target 名称的代码围栏拒绝，以及 NON-BLOCKER N1 所述更精确的外键反例。

## CI 与平台证据边界

已核查 [CI 38939986285](https://github.com/XKHoshizora/specmetry/actions/runs/38939986285) 的元数据及三平台日志：

- Head 为目标提交 `90beb48d3c5cc2d8b7a7cbbc045b0925ab793ead`。
- 8/8 Job 成功。
- 相关 Target、Scope、Review、并发及迁移约束测试在 Windows、Ubuntu、macOS 实际执行并通过。
- Ubuntu 的 Go 测试按工作流启用 Race Detector。

独立本地运行平台为 Windows；其他平台及 Linux Race Detector 的证据来自 CI，不描述为本地独立实测。CI 全绿没有替代安全和契约判断。

核查时 PR #14 为 Open、尚未合并，Head 与 Base 均对应本次审查范围。

## 验收范围与最终回答

MVP-TGT-01～08、TGT-10 的 M1 部分、TGT-11，以及 Target/Scope 的 CH-01/02 主要业务路径通过上述验证；但新增自由文本入口仍违反当前隐私与 Schema 契约，因此不能据此完成 WP1.8 代码验收。

MVP-TGT-09 的 status/SCOPE_STALE 差距、NO_SCOPE 的交付状态推导、真实 Decision 与有效豁免生命周期，以及 init_progress 的完整推导，按已批准计划在 M2 完成，不作为本轮新增 BLOCKER。

| 问题 | 结论 |
|---|---|
| Target/Scope 数据模型和行为是否符合冻结设计？ | 主要实现符合；新增自由文本校验存在 B1/B2 两项违约。 |
| V-T01、V-T02 的 M1 实现是否正确？ | **通过本轮验证。** M2 的真实豁免生命周期未提前宣称通过。 |
| Review + Scope 是否真正原子化？ | **是，独立故障及并发反例通过。** |
| 数据库迁移与 Schema 约束是否可靠？ | 迁移及关系约束通过；交互式输入校验与名称 Schema 仍需修复。 |
| 人工确认与 JSON 输出是否安全？ | 通道、完整提示和分流通过；写入内容校验尚不完整。 |
| M1 应由 WP1.8 完成的验收标准是否满足？ | **尚未全部满足。** |
| PR #14 是否可以正式验收并合并？ | **不可以。最终结论：NO-GO。** |

修复两个输入边界后即可进行定向复审，没有证据要求重构存储层。不自行宣布 M1 整体验收完成；最终里程碑退出条件仍由产品负责人单独确认。

审查结束时原仓库工作区干净，未修改代码、测试、配置或业务数据，未提交、推送或合并，未开始 M2。本报告根据用户随后明确授权单独保存，保存动作仅新增本文件。
