# Codex Review 044：PR #20 首批规范与契约独立审查

- 日期：2026-10-18（Asia/Tokyo）。
- 仓库：`XKHoshizora/specmetry`；[PR #20](https://github.com/XKHoshizora/specmetry/pull/20)。
- 固定基准：`258fc50f5e94354892737614ab61eea564178e7c`。
- 固定目标：`4ef79c8f2c309393a94f58897d1f48cad15fee6f`。
- CI：[run 39048967831](https://github.com/XKHoshizora/specmetry/actions/runs/39048967831)，Attempt 1，8/8 SUCCESS。
- 范围：首批完整 WP2.1／WP2.2 的规范、交换契约、合法事实历史与独立复核器规格；必要时读取已验收的调用链。没有重新验收 M1，没有重新展开恢复方案审查，没有实现 M2。
- 方法：固定差异、规范、Schema、已有源码与测试断言的静态核查；GitHub 提交及 CI 元数据／日志核查；Windows 本机契约测试。

## 结论

**PR #20：NO-GO。存在一个必须在完整首批实现前解决的规格 BLOCKER：新增当前值设置输入与既有输出可复现性契约尚未统一。**

F1～F3 的已批准取舍已经落实。F4 的序号分配、共享事实保留和主要历史重建方案可成立；F5 的 ready 谓词、输入域及独立性要求也已经达到可执行规格的程度。阻塞不在这些规则需要全面重写，而在 TD-27 对输入与复现保证的影响还没有形成唯一、一致的契约。

**不要求修改 M1 的 init，不要求现在给设置增加 seq，也不要求实现 M2。** 保留设置作为当前值输入在技术上成立；需要在首批前补齐完整输入的重放约定，同步规范、Schema 说明和验收文字。是否将来给设置增加历史，可以与这项澄清分开处理。

本轮没有发现需要重开 PR #19、M1 或恢复方案的理由。准备报告的 GO 不受本结论影响。PR GO、规格可冻结和负责人授权实现是三个独立判断；当前既不能把完整 F4／F5 宣布冻结，也没有获得 M2 实现授权。

## 一、提交、范围与 CI 身份

开始及结束核查时，PR HEAD 均为固定目标，base 均为固定基准，PR 保持 Open。本地 HEAD 也为固定目标。开始时工作区干净，目标归档文件不存在。没有混入其他提交。

本轮三个提交：

| 提交 | 内容 |
|---|---|
| `38bb8a5` | 需求 flags、SOURCE_REMOVED 及对应 Schema／示例 |
| `5034f9d` | 负责人决定、F4／F5 规格及相关文档 |
| `4ef79c8` | 收尾表对恢复截止点的同步 |

实际差异为 14 个文件：

- `AGENTS.md`；
- `docs/acceptance-criteria.md`、`docs/architecture.md`、`docs/data-model.md`、`docs/implementation-plan.md`、`docs/m2-readiness.md`、`docs/prd.md`、`docs/review-resolution.md`；
- `schemas/common.v1.schema.json`、`schemas/status.v1.schema.json`；
- `schemas/examples/invalid-cases.json`、`schemas/examples/valid/status.exception.json`、`schemas/examples/valid/status.minimal.json`、新增的 `schemas/examples/valid/status.source-removed.json`。

没有生产代码、测试实现或迁移变化。`git diff --check` 通过。

| CI 项目 | 独立核对结果 |
|---|---|
| Event／HEAD | `pull_request`／`4ef79c8f2c309393a94f58897d1f48cad15fee6f` |
| Attempt／结果 | 1；`previous_attempt_url = null`；completed／success |
| Job | Go lint、三平台 Go tests、Web、三平台 Build with dashboard，8 个均 success |
| 实际检出的合并快照 | `554d0685f476919c6b8c4d704d2cd444f4bfad78` |
| 合并快照父提交 | `258fc50f5e94354892737614ab61eea564178e7c`、`4ef79c8f2c309393a94f58897d1f48cad15fee6f` |
| 合并快照与目标的内容树 | 均为 `54c4d049f5f70bb86b190c13daca5aea0f58ed7c` |

Windows、Ubuntu、macOS 三个 Go Job 的 checkout 日志均明确检出上述合并快照；三者都记录了 39 个合法示例通过、156 个非法反例被拒绝。Ubuntu Go Job 使用 Race Detector，需 root 的四项 CrossUser 测试另有 sudo 步骤运行并通过；普通步骤的条件跳过与后续补跑分别核查。

这组 CI 证明已提交仓库的检查结果，不证明尚未开发的推导引擎、复核器、M2 迁移或未提交的 Shell／恢复／Git 实验已经获得三平台验证。

## 二、BLOCKER

### B1：当前值设置输入与输出可复现性契约不一致，不能等到 M2 结束才澄清

**位置：**

- 新增的 [data-model §6.13，第 848～851 行](../data-model.md#L848)：`stage`、`sources_skipped`、`inventory_skipped` 是不随 `as_of_seq` 变化的当前值输入；历史前后两次推导均使用当前设置。
- 仍有效的 [data-model §6 开头](../data-model.md#L557)、[architecture §7](../architecture.md#L760)：输入列举没有这些设置，并仍声称依靠输出中的序号、快照等信息可以复现任何结论。
- [MVP-ST-14](../acceptance-criteria.md#L215)：仍声称输出中的 `as_of_seq`、`snapshot`、`sources[].current_sha256` 可以复现原结果。
- [status.v1 第 5 行](../../schemas/status.v1.schema.json#L5)：仍将可复现的输入概括为上述三类；[project 字段](../../schemas/status.v1.schema.json#L28)只输出 stage，不输出两个 skip 设置，且不允许任意附加字段。
- [TD-27](../architecture.md#L1313)：将是否为设置分配 seq 留到 M2 结束时设计复核，但没有区分“未来是否增加设置历史”和“当前完整输入／重放契约”这两个问题。

**源码事实：** [project.Plan.update](../../internal/project/init.go#L474)在第 489～509 行更新 stage／settings_json，没有分配序号。已有主 worktree 的 [ensureMain](../../internal/project/init.go#L522)只更新 last_seen_at；已有 worktree 不会走 [insertWorktree 的 NextSeq](../../internal/project/init.go#L594)。因此文档对 M1 实现的观察正确，不能通过假定 init 会增加 seq 消除问题。

**正常操作历史与逻辑反例：**

1. 已登记项目，仍在原 worktree；没有来源或提案，尚未跳过来源登记。按 [§6.12](../data-model.md#L826)，`init_progress = initialized`。
2. 对同一项目再次执行 `init --skip-sources`，不新登记 worktree、不创建新 Target。settings_json 改变，但事实序号不增加。
3. 现在按 §6.12 得到 `sources_registered`。两次的事实、as_of_seq、S、观测哈希和 stage 可以全部相同，唯独不在 status 输出中的 `sources_skipped` 改变。
4. 若稍后只拿旧输出标注的输入和当前数据库来重放，旧结果不能按现有承诺恢复。

stage 还有独立影响：在没有 Claim／关联提交时，[§6.5](../data-model.md#L650)会因 docs_only 与其他阶段得到不同进度。两个 skip 设置还影响初始化进度及盘点提示。这不是不影响结果的展示参数。

这是源码与规范共同给出的逻辑证明；本轮没有把尚未实现的 M2 status 当作已动态复现。也没有证明错误 ready 已发生：这里直接违反的是完整输入、历史解释和输出可复现性的冻结要求。

**影响：** WP2.1 的输入类型、重放用例、ST-14，以及 WP2.2 的测试前端会据此形成不同实现。当前值模型允许“同一事实截止点，在不同当前设置下有不同结果”；旧契约则给出了更强的输出重放承诺。仅把问题登记到 M2 末尾不能支持当前首批规格冻结。

**最小修正：**

1. 可以保留现有 init 和当前值设置模型，但要在 §6、architecture §7、F4 和 ST-14 中统一列出完整显式输入；明确 `--diff-since` 在同一次比较的两边固定使用同一份当前设置，其结果不是过去实际展示结果的回放。
2. 明确旧结果重放时如何提供当时的设置值，并使输出／重放约定与 Schema 说明一致。可通过补齐可重放的输入上下文实现，不必因此增加 init 的 seq 或设置历史。
3. 若选择保留“输出足以复现”的保证，就补齐所需信息；若选择放宽这项既有保证，则需要负责人明确确认，不能以“实施细节”或 TD-27 已登记代替批准。
4. 同步规定针对设置变化、相同事实截止点及 diff-since 的验收预期。这里只要求先冻结规则，不要求本 PR 编写 M2 实现或测试程序。

**截止判断：** 完整输入与复现语义必须在 WP2.1／WP2.2 前澄清；是否将来给设置增加 seq 可以另行评估。不要把这两个截止点机械绑定，也不需要重审 M1 的验收结论。

## 三、NON-BLOCKER

### N1：提交关联的创建生命周期应在 WP2.3 冻结时明示

[§4.12](../data-model.md#L456)与 [F4 的提交行](../data-model.md#L891)为 commit_links 选择了无独立 seq、经 commit_id 归属发现序号的模型。只要关联与首次保存的提交一起确定、后续扫描不向旧提交补写新关联，这个模型可以正确重建历史，纯推导的“发现序号大于 revision_seq”比较也成立。

目前文字没有明确写出上述生命周期；尤其在按 SHA 去重、历史改写后重扫，以及尚待决定的 Q7／正文匹配开关之间，需要避免将来出现“旧提交保留旧 seq，却在后来的扫描中补上没有时序的新关联”。否则 `as_of_seq` 无法区分补写前后。

最小补充是在 WP2.3／Q7 冻结时明确：旧提交是否完全跳过关联重算；若允许补写，新增关联如何记录发现时序。无新提交的一次扫描如何更新 scanned_seq 也应在该处说明。不要把一次扫描等同于一定存在新提交。

这不是已复现的生产缺陷：扫描尚未开发，Q7 也没有获批。现有 F4 可采用“提交与关联共同形成不可变发现事实”的实现方式，不需要现在增加迁移或扩展首批范围。本项不单独阻塞 PR；实现前应避免把未说明的补写行为引入已冻结的历史模型。

## 四、PRODUCT DECISION

### 1. §55 忠实保留了本次授权边界

[review-resolution §55](../review-resolution.md#L2710)与实际增量一致：

- 路线为完整 WP2.1／WP2.2；路线决定不等于实现授权。
- Q2 批准结构化输入身份、比较语义与错误边界；Git 读取机制仍在 WP2.3 前冻结。
- Q5 批准范围内需求验收率加 AC 状态计数。
- Q8 批准需求 flags、AC source_removed 和有范围筛选的非阻塞 SOURCE_REMOVED。
- 畸形输入采用选项 (a)；合法但不能 ready 的历史仍属于业务保证域。
- Q10 四项截止调整与 implementation-plan、architecture、PRD 的对应记录一致。

没有顺带批准 Q1、Q3、Q4、Q6、Q7、Q9、E1～E5、恢复退出码或交互草案。恢复仍是 M2 交付义务，仍须在 M4 自测前完成；U1／U2 在 WP2.10 前必须有明确结论；M0 N1 等依赖没有移除，安全保证没有因延期而降低。

### 2. TD-27 的需要决定部分

将设置作为完整显式的当前值输入，本身可以是对已有实现的准确技术澄清；本审查不要求为每项事实序号设计另立产品审批。

需要负责人介入的是：若为保留现有输出格式而放宽原有重放保证，或改变用户对历史／diff-since 的承诺，必须明确裁定。若以补齐可重放输入等方式保持既有保证，可先完成技术规格一致化，不必机械新增 PD 编号。B1 要求首批前消除的是契约冲突，并非强制选择设置历史方案。

除此之外，没有发现需要本轮新裁决的产品方向问题。既有未批准草案保持各自状态和已调整的截止点。

## 五、ACCEPTED RISK

**本轮没有新增接受风险。** 设置缺少 seq 导致的重放问题没有被本审查接受为风险；恢复例外也没有因本 PR 获得批准。

已批准的边界继续保留：MVP 只产生 observed 执行，不能自动沿用验收；通用域的合成隔离级用例不是 MVP 能力；畸形选项 (a) 不保证非法事实组合的业务结果，但不改变既有的缺失数据、版本不兼容等保守处理。

Windows 数据库创建偶发 Access is denied、doctor 的 Git Trace、M0 N1 及其他既有维护事项，不因这次 CI 绿色而关闭，也不被重新归类为已接受风险。

## 六、F1～F3、F4、F5 的具体核查

### F1／Q2：输入身份已落实

[§4.11.2](../data-model.md#L411)将保存原路径与查找规范化分开：查找采用反斜杠转换和 path.Clean，保存值与 M1 的 check_config_hash 不变。相等关系使用 `{path, mode, type, oid}` 集合，包含路径及可执行位／链接模式；不是只比较 OID。沿用条件与 inputs_changed 引用同一规则。

未声明 inputs、非 Git 的 NULL 记录，与“声明了 inputs 但缺少记录”的不相等分别处理。固定完整 head_start 后查找，路径不存在、所需提交／中间 Tree 读取失败、叶对象缺失及 gitlink 分开：

- 路径不存在／无提交：input_missing，不运行命令，记录失败语义；
- 必需对象无法安全读取：exit 6，不执行、不写执行记录；
- 已有条目的叶 Blob／Tree 缺失：仍可保存元数据；
- gitlink 不要求其提交对象存在于上级仓库。

符号链接、目录、LFS 指针及大小写语义有明确说明；不运行 LFS／过滤器，保持 PD-26 的禁用隐式 Fetch 和统一受控读取边界。[EV-10](../acceptance-criteria.md#L177)与 CX-19 的新增验收文字对应这些规则。Git 命令选择仍待 WP2.3，不需要本 PR 实现。

### F2／Q5 与 F3／Q8：语义和结构已落实

[主指标](../data-model.md#L811)的分子是 RS 中 accepted 的需求数，分母排除 waived，零分母为 null；另输出 AC 状态计数。没有将豁免计入已验收分子。

[需求／AC 标记](../data-model.md#L665)与 [SOURCE_REMOVED](../data-model.md#L795)区分了条目层级。警告对象是引用了已移除来源的条目，不是来源路径；有范围时按 RS／AS 筛选，Goal 仅在包含 RS 需求时产生；无范围时覆盖全部 active 条目。移除导致的 SCOPE_STALE 仍然阻塞，其他未移除来源的 SOURCE_CHANGED 也没有被替代。

新增需求 flags 为必填、去重且限定枚举；AC 的枚举增加 source_removed。common.gap 为 SOURCE_REMOVED 强制 blocking=false，并要求合法条目编号。Schema 不负责判定引用是否真的指向已移除来源、条目是否在范围、reason 是否按业务规范为 null、指标是否计算正确；这些仍须由 WP2.1 语义测试及后续真实输出测试证明。

SRC-04 的 M1／M2 标签修改只是把原文已有的 M2 状态推导责任写明，没有重开 M1，也没有新增延期。

### F4：主要事实结构、seq 与历史重建可成立，完整冻结受 B1 阻塞

新增序号方案具有明确的可执行含义：

- Claim 批次与 Claims 共用一次分配；V-C03 保证同一批次每个 AC 至多一条，所以按 AC 取最大 seq 没有并列；未映射工作仍有各自记录身份。
- 每次检查完成写入时单独分配 seq；顺序是事实写入顺序，不是启动时间。每次决定也单独分配，证据严格早于决定。
- 新发现提交可在一次扫描中共享 seq；共享不影响“发现序号大于 revision_seq”的比较。
- 一次事务可分配多个 seq；预演／失败回滚不消耗 seq；表内 UNIQUE 列与跨表、表内合法共享分开。
- Scope Items、Commit Links 使用复合标识，不要求每张表新增 ID。as_of_seq 到达共享序号时保留全部相关记录，不按 seq 去重。
- 当前决定、拒收解除、接受后失败及证据先后仍遵守各自严格比较，没有统一放宽为小于等于。

对历史重建的关键核查如下：

| 对象 | 核查结论 |
|---|---|
| 基准实体 | [Record.Snapshot](../../internal/baseline/record.go#L226)保存修订、状态、字段以及 AC 的父需求修订；Values 包含检查配置。F4 从 baseline_changes 的 after_json 重建，按 seq／change_index 排序，避免直接读取当前实体状态。正式 key、origin、AC 所属需求的不可变假设与现有写入规则相符；Requirement 的 goal 变化由快照保存。 |
| 配置与 revision_seq | 从历史配置计算哈希；取该修订首次出现的变更序号，避免 editorial 变化错误刷新修订起点。 |
| 来源 | 通过 registered_seq／removed_seq 和 source_acks 重建集合及确认哈希；第一次 ack 的 previous_sha256 可恢复初始登记哈希，避免将当前 sources.acked_sha256 误用于旧时刻。 |
| 提案与冲突 | 使用 reviewed_seq 判断过去的待审状态，冲突只看截止点前的基准变化；不能拿当前 review_state 直接充当历史。 |
| Question | 使用 created_seq／answered_seq 重建开闭状态。原 proposal_item_id 在 [关联回填](../../internal/baseline/apply.go#L753)后仍保留，所以可以按截止点时该条目的接受关系重建正式关联，不能直接沿用当前 related_id。 |
| Target／Scope | Target 用 created／archived 区间，Scope 按各 Target 的最大确认 seq；旧 Target 的范围不会直接继承给新 Target。 |
| Claim／执行／决定 | 保留全部记录、修订、配置、上下文、证据引用和时序，由推导各自选择当前值；没有用时间戳替代 seq。 |

这些部分没有发现“仅过滤当前投影的序号，就宣称恢复历史”的实质性缺陷。提交关联的后续写入边界见 N1，项目设置的完整重放边界见 B1。

正常未知输入没有被排为畸形：合法的 S=null、unknown 工作区、缺少输入记录和来源缺失仍按保守规则处理。合法的目标无法解析／非 Git 与 PD-26 的对象读取环境错误应继续区分；后者由调用方返回 exit 6，不能伪装为一次正常 not_assessed。

### F5：谓词与独立性要求可作为实现规格，整体输入冻结须跟随 F4

[acceptance-criteria §3.1](../acceptance-criteria.md#L332)的 R1～R9 在合法事实历史上与 ready 条件相符；R10 明确是证据等级输出一致性断言，不是新增业务门槛。

R6 对 manual、exception、normal 和 waived 分开处理，没有要求所有 AC 都有自动执行。其与决策表第 4／6 条的论证成立：合法 normal 决定先有同修订／配置的合格通过；material 配置变化使旧决定不再当前；接受后出现的失败由失败优先规则处理。畸形选项 (a) 没有因此放宽合法域的 ready 条件。

复核器与推导引擎使用两套独立原始转换器，不导入生产计算函数；保留全部执行／决定和共享序号记录，由双方自行按截止点过滤、重建和判断适用性。导入图检查也没有被冒充为排除共享错误假设的充分证据。

规格保留了拒收、接受后失败、父需求修订、过期范围、阻塞问题、来源变化／缺失、全豁免、错误快照及旧修订等合法反例，以及手动／例外／豁免组合、观测级正常接受、共享 seq 和通用隔离级沿用正例。允许结构性连带条件，但仍要求目标条件覆盖与归因；双向对照不能只在非 ready 样本上空转。

因此无需重写 F5 谓词。其 ready 判定部分可以保留；完整测试前端与输入契约仍须等 B1 的一致化，不能把同一个不完整重放假设复制到两套转换器后当作独立验证。

## 七、实际运行的验证与证据边界

本轮本机平台为 Windows／amd64，实际使用已经安装的 Go 1.27.1，以 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local 运行：

```text
go test -tags nowebui -mod=readonly ./internal/contract -count=1 -v
```

**结果：PASS，进程退出码 0。** 全部 Schema 编译通过，39 个合法示例通过，156 个非法反例被拒绝；错误信封、JSON Pointer、非法 JSON 及验证错误不回显输入等该包现有测试也通过。

初次通过环境中的 Go shim 启动时，版本管理器未能完成工具解析，Go 测试没有启动；随后直接使用已安装的真实 go.exe 执行上述离线检查并通过。没有下载工具链、安装依赖或修改用户配置；不能把第一次未启动的检查算成测试通过或代码失败。

[TestInvalidCasesFail](../../internal/contract/contract_test.go#L126)在施加每个 patch 前验证基础示例合法，再要求修改后为 invalid_input 且有 details。本轮同时核对了六例的单字段改动及对应约束：

| 反例 | 基础／修改 | 预期拒绝依据 |
|---|---|---|
| SCH-151 | 合法 status.source-removed；删除 requirement.flags | required |
| SCH-152 | 将 requirement.flags 改成 AC 专属 observed_evidence | flags enum |
| SCH-153 | 将 SOURCE_REMOVED.blocking 改成 true | common.gap 的 false 常量 |
| SCH-154 | 将 SOURCE_REMOVED.target_key 改成 null | string 类型 |
| SCH-155 | 将该 target_key 改成来源路径 | 条目编号 pattern |
| SCH-156 | 合法 gaps；替换为 blocking=true 的 SOURCE_REMOVED | gaps 复用同一 common.gap 约束 |

现有测试不逐例断言验证器内部 keyword；本轮通过基础示例预校验、单字段变异及约束静态核对确认拒绝原因，没有将作者未提交的临时实验描述为独立复现。

未运行完整 M1、动态攻击、网络／Helper 实验、恢复实验或新的 M2 程序。没有在本机复跑 Linux／macOS 或真实旧 Git；相关结论只限已核对的仓库 CI 与既有记录。没有用文档 CI 证明新运行时能力。

## 八、交付判断与下一步

| 问题 | 结论 |
|---|---|
| PR #20 是否 GO、可合并？ | **NO-GO**。先消除 B1 的规格冲突；不需要全面重做本 PR。 |
| F1～F3 是否完整落实已批准契约？ | **是**，未发现本轮实质性缺口。 |
| F4 是否具备冻结条件？ | seq、记录身份与主要历史重建规则可保留；**完整输入／重放契约尚不能冻结**。 |
| F5 是否具备冻结条件？ | ready 谓词、独立性和测试域规格可保留；整体输入前端冻结依赖 F4 的 B1 关闭。 |
| TD-27 是否必须首批前决定？ | 完整输入、diff-since 与复现保证必须先明确；不强制现在决定是否为设置增加 seq。若选择降低既有保证，须由负责人明确批准。 |
| 完整 WP2.1／WP2.2 还缺什么？ | B1 的最小规格修正及独立确认；随后由负责人明确授权实现。F6 的测试先行工作按冻结规格开展，不以本次审查代替实现证据。 |
| 是否授权 M2？ | **否。** 本审查不批准草案、不接受新风险、不启动实现。 |

建议交由 Claude Code 仅完成 B1 的规格一致化：保留 M1 init 的行为，明确完整设置输入和重放边界，同步相关规范、Schema 说明与 ST-14；若涉及保证放宽，先由负责人裁定。后续独立审查只需复核这项差异及直接关联内容。N1 可随 WP2.3／Q7 的冻结处理。

本轮仅新增本报告，使用仓库相对链接；未修改其他文件或历史审查，未提交、推送、合并。审查完成后暂停，等待负责人决定，不开始 M2。
