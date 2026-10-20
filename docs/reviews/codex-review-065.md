# Codex Review 065：WP2.4 Claim 规范及 claim-receipt.v1 独立审查

## 一、结论

**技术审查：NO-GO，1 项 BLOCKER。PR #29 当前不建议合并，整体规范尚不具备冻结条件。**

阻塞来自规范内部的退出码冲突，不是已复现的 Claim 生产实现缺陷：规范要求在写锁下先校验 `base_version`，但并发退役反例及测试计划又要求旧输入返回 exit 2。合法退役必然产生新的 Baseline Version，此时按既有规则应先返回 exit 5。同一合法历史不能同时满足两种验收结果，详见 B1。

`claim-receipt.v1` 本身未发现结构或条件分支缺陷。独立运行确认 24 个 Schema 编译、43 个合法示例通过、190 个非法反例拒绝；新增 SCH-166～190 均基于合法示例，实际拒绝原因符合描述。

E4（歧义前缀的对外处理）与 E6（回执内容范围）仍需负责人确认，属于产品决定，不能据技术检查代为批准。它们与 B1 分开处理。E1、E2、E3、E5 是符合既有契约的工程细化。

指定新 CI 正式 8/8 成功，但不消除规范冲突，也不证明尚未实现的 Claim 命令、安全写入、并发或回执构造已经通过测试。

## 二、固定身份、范围和方法

- 审查日期：2026-10-20，Asia/Tokyo。
- PR：[XKHoshizora/specmetry#29](https://github.com/XKHoshizora/specmetry/pull/29)。
- 固定 base：`7a361663ccb53f6a7db594a784c08ad54a24e4c8`。
- 固定 HEAD：`058b9a07ef3d62acae6ab51e45fb338f7e5820e3`。
- HEAD tree：`e479771e195c3a8d7a616361bf88a7bb442e00ce`。
- 开始及收尾核查时，远程 HEAD/base 均与指定对象相同，PR Open、未合并、`auto_merge = null`。未自行切换审查对象。
- 已读取适用的根 [AGENTS.md](../../AGENTS.md)；固定树中没有其他下级 AGENTS.md。

本轮审查整个 base→HEAD，而非只审查最后六个文档文件：

| 类别 | 文件 |
|---|---|
| 新规范 | [docs/wp2.4-claim.md](../wp2.4-claim.md) |
| 新交换契约 | [claim-receipt.v1.schema.json](../../schemas/claim-receipt.v1.schema.json) |
| 三个新合法示例 | [sync](../../schemas/examples/valid/claim-receipt.sync.json)、[audit](../../schemas/examples/valid/claim-receipt.audit.json)、[empty](../../schemas/examples/valid/claim-receipt.empty.json) |
| 反例及注册 | [invalid-cases.json](../../schemas/examples/invalid-cases.json)、[contract_test.go](../../internal/contract/contract_test.go) |
| 设计同步 | [AGENTS.md](../../AGENTS.md)、[acceptance-criteria.md](../acceptance-criteria.md)、[architecture.md](../architecture.md)、[data-model.md](../data-model.md)、[implementation-plan.md](../implementation-plan.md)、[prd.md](../prd.md) |
| 决定与维护 | [review-resolution.md](../review-resolution.md)、[m1-exit-evidence.md](../m1-exit-evidence.md) |

合计 15 文件，792 行新增、14 行删除。提交顺序为 `ae19899`（规范及契约）、`91e7e5f`（决定和维护登记）、`058b9a0`（D1～D3、生命周期与维护收口）。`91e7e5f`→HEAD 确实只有所述六个文档文件，但不能取代前一段新规范及新契约的完整审查。

没有 Claim 生产实现、迁移 SQL、Git Reader 实现、推导算法或 CLI 接线的变更。`internal/contract/contract_test.go` 的代码变化仅为新 Schema 名及示例前缀注册。没有把准备 PR 误当成工作包实现。

位置均以固定 HEAD 为准。源码追溯仅用于验证本轮承诺是否与现有机制兼容，没有重新验收 M1、WP2.1～WP2.3 或已关闭 B1。

## 三、BLOCKER

### B1：并发退役的 exit 2 预期，与版本优先检查及冻结契约矛盾

**位置：**

- [wp2.4-claim.md:83](../wp2.4-claim.md#L83)：`BEGIN IMMEDIATE` 后第一项检查 `base_version`，不一致立即 exit 5。
- [wp2.4-claim.md:272](../wp2.4-claim.md#L272)：读取后、写锁前，需求或目标 AC 被退役，要求 exit 2。
- [wp2.4-claim.md:331](../wp2.4-claim.md#L331)：两个连接构造上述退役竞争，仍要求提交方 exit 2。
- 对照 [data-model.md:1051](../data-model.md#L1051) 的 §7.1、V-C02，以及 [wp2.4-claim.md:277](../wp2.4-claim.md#L277) 的“确认期间基准被修订 → exit 5”。

**合法反例／源码级证明：**

1. 当前基准为 v1，REQ-001 和其 AC-001-1 都 active。调用方读取 v1，构造合法 audit：`base_version = 1`，范围含 REQ-001，目标修订为当前值。
2. Claim 完成上下文读取；manual 情形也可以已完成 v1 上的演练、正在等待确认。
3. 另一连接通过正常 Review 退役该需求或 AC，并提交 v2。现有 [baseline.Apply:126](../../internal/baseline/apply.go#L126) 将新版本设为 `current + 1`，为接受的修改插入 `baseline_versions`；[retireRecord:594](../../internal/baseline/apply.go#L594) 将退役绑定该版本。需求退役还在同一版本级联退役 AC。
4. Claim 随后取得写锁。按规范第一项，当前版本 2 不等于输入版本 1，必须返回 `stale_precondition / exit 5`，`current.base_version = 2`，位置 `/base_version`。它不能继续到 D2／active 校验后改报 exit 2。

该历史不依赖非法直接改库，也不需要用户篡改读取版本。若实现为了满足第 272／331 行而越过版本检查，便会违反 V-C02 和 §7.1；若忠实遵守第 83 行，计划中的指定用例又必然失败。现有 [proposal.Store:223](../../internal/proposal/store.go#L223) 也先检查基准版本，再检查目标有效性，不能用“与 propose 相同”解释这处差异。

**影响：** 首批实现无法获得唯一正确的错误分类及并发验收标准，可能诱导实施方用业务错误覆盖过期错误。当前尚无 Claim 实现，本发现不是声称现有 Claim 已经写错或发生部分提交。

**最小修正：**

- 保持既有版本优先顺序。将这两处正常并发退役用例改为 exit 5，并检查当前版本、无 Claim 业务写入和无新增 seq。
- 单独保留使用**当前基准版本**、但仍引用已退役／不存在需求或 AC 的输入用例，验证 D2／V-C03 的 exit 2 及精确 Pointer。
- D3 使用当前版本、active AC 实际属于范围外需求的输入，验证 `/claims/<i>/target` 的 exit 2；局部覆盖、`all` 和空 claims 正例继续保留。
- 明确“D2／D3 的 exit 2 在版本前置条件通过后适用”，同步 §13、§15 及相关描述。不必改冻结规则或撤销负责人对 D2／D3 的批准。若反而要让 D2 总是覆盖过期结果，则是另一个契约变更，不能由实施方自行决定。

## 四、NON-BLOCKER

### N1：失败总述的“任何数据”应统一限定为本次业务事实

[wp2.4-claim.md:68](../wp2.4-claim.md#L68) 将若干失败笼统写为“不写入任何数据”。§6.2 和 §10 已准确区分本次 Claim 事实、普通 Open 的既有行为及私有临时内容，因此不据此认定新增了绝对零副作用保证。

建议总述、回滚说明统一使用“本次 Claim 的业务事实及其 seq 不提交”，引用 §6.2／§10。失败后允许的数据库辅助行为和方案 B 保留不能被测试误写成数据目录逐字节不变。这是表述收口，不需要新的风险接受。

### N2：输出故障应说明可能留下部分 stdout，不能以读取状态证明没有提交

[wp2.4-claim.md:193](../wp2.4-claim.md#L193)～196 正确区分了提交前回执校验失败回滚，与 COMMIT 后输出失败保留事实、exit 6；§8 也明确重提不幂等、不自动重试。核心数据结果没有歧义。

建议为 §9.4 的“完整 error.v1”增加输出通道故障边界：若回执已经部分写出，stdout 可能截断，不能承诺还可在同一流追加一份完整错误文档并保持单文档合法性。应尽力在可用错误通道说明“可能已经提交，不要盲目重试”；不能把重新读取状态解释为能够证明原提交不存在。

现有 [writeJSON:25](../../internal/cli/output.go#L25) 在输出时才验证，且只检查底层 `Write` 的 error，没有检查短写计数；不能直接复用它来宣称新的提交前验证及输出失败保证已经实现。Claim 实现应预先构造、验证并最好编码好私有回执；提交后输出还要检查短写、显式错误，避免重新校验或向部分回执追加第二个 JSON。本轮不要求修复其他命令的输出助手，也没有构造一个尚不存在的 Claim CLI 失败证据。

## 五、既定决定与处理顺序

### Q6、manual 通道和确认

现有输入没有独立 CLI flag 指明批次类型，因此先读取受 8 MiB 限制的输入、严格验证 `claims.v1` 后识别 kind，是可实施的。规范没有错误承诺在读取文件之前便能拒绝 manual：无效 JSON 先 exit 2；识别为 manual 后，RequireHuman 在 DB、Git、代码及业务写入之前执行。不能直接照搬现有 propose 的完整处理顺序而把打开数据库放到此检查之前。

管道／非 TTY／已知 Agent 变量拒绝 manual，合法 sync/audit 不被连带禁止；Agent 变量只取识别结果，不保存其值。终端、环境变量和 `human:<name>` 都不是身份认证，§4 对边界的描述符合批准的 Q6／TD-13。

manual 的演练在回滚事务中完整执行，确认后另开写事务重新校验。`cli.confirm` 负责 JSON 提示走 stderr、stderr 必须可见、Guard 完整提示检查；取消／错误确认／EOF 不提交。规范保留现有确认文本按原值比较及终端安全处理，没有引入 `--yes`、Hook 或新认证机制。

实现测试还应覆盖提示短写、不读取确认输入，以及最终事务失败时连同首次 worktree 登记一起回滚；这目前是待实现的要求，不是本 PR 已运行的证据。

### D1～D3

| 已决定内容 | 本轮判断 |
|---|---|
| D1：实际工作区环境错误 exit 6，本次事实及 seq 不写入 | §5、§6.2、数据模型 §4.10、CLM-04 和测试计划一致；正常 undecidable→unknown、非 Git unknown 没有被取消 |
| D2：范围内不存在／退役需求 exit 2 | 规则及 Pointer 一致；B1 要修正它与版本过期同时发生时的优先级预期 |
| D3：audit AC 必须属于所声明需求，写锁下核对 | 处理顺序及 V-C05 一致；`all` 覆盖所有 active AC，局部和空 claims 仍合法；没有新增覆盖全部 AC 的条件 |

同一 AC 的父需求不能通过现有 `updateCriterion` 改换，规范对此的说明有依据。不能把 audit 的声明范围当成 Delivery Scope，也不能要求先做 Scope Confirmation 才允许 Claim。

### 事务与 seq

在写锁后按项目读取版本、active 状态、归属与修订，之后才登记 worktree、分配批次 seq，设计上可实现原子性。首次 worktree 有独立 seq，批次与全部 claims／unmapped_work 共用后一个 seq：一个事务可以分配两个 seq，不得把合法共享误判为重复。

批次内同一 AC 唯一、批次 seq 表内唯一，使“按 AC 取最新 Claim”没有同序号并列。空批次仍有批次 ID/seq；重复提交得到新批次，这与未提供幂等键的既有模型一致。尚未提供迁移 SQL，所以项目外键、append-only、REPLACE 防线和 manual/channel 约束仍需在实现阶段兑现。

## 六、代码上下文、引用及 E1～E6

### 上下文与 Git 分类

规范把采集放在写事务外，写锁保护数据库版本，不宣称 Git/磁盘与 SQL 构成原子快照。文件引用必须绑定同一次采集的 head；采集之后 HEAD 或工作区变化不自动使自述成为交付证据，也不能把 tree_state=clean 写成快照验证成功。

已静态核对现有 API：

- [CommitOf](../../internal/gitx/gitx.go#L214) 区分 ref 不存在／unborn 与指向无法读取的对象，但不能直接承担新前缀解析器。其歧义错误是环境错误，而且 `^{commit}` 的解析语义也不等于“前缀直接对应一个 commit 类型对象”。规范第 141 行已明确需要新受控函数，没有谎称现成 API 已足够。
- [Classify](../../internal/gitx/scan.go#L214) 只接受完整 commit ID：可靠缺失不只是一个退出码，而是结构化 batch-check、裸 ID 检查、stderr 是否异常及存储检查的组合。它不能直接处理短前缀或把非 commit 对象当作普通缺失。
- [proposal.CheckCodeRefs](../../internal/proposal/coderef.go#L37) 的现有实现会自行解析 HEAD。规范说复用其**规则**是合理的，但实施时不能直接调用它后宣称已经绑定先前采集的 head；需要显式传入固定 head 的检查接口或等效结构。
- 对象不存在、非 commit、歧义、损坏／不可读及 PD-26 拒绝必须分开。不能仅依据 rev-parse 的非零退出码或匹配原始 stderr 文本作出 `resolved=false`；原始 Git 错误也不能进回执、日志或 error.v1。

这是一项明确的后续实现任务，不因规范 PR 尚未提供该新函数就判为实现失败。新函数的可验证要求应覆盖结构化前缀枚举、非 commit、歧义与可靠缺失的证明、损坏 loose/pack/index/alternates、取消及对象变化；不确定时 exit 6，不能假成功。当前契约测试没有证明这些 Git 能力。

PD-26、部分克隆的 Code Ref 拒绝、普通文件与行号规则均未被改动。非 Git 文件仅限项目根内普通文件，链接解析后仍须在根内；正常 unknown 不等于绕过所需 Git 读取的安全门禁。不要为非 Git 的普通 unknown 无条件引入无必要的 Git 对象读取。

### E1～E6 裁定

| 项目 | 分类与判断 |
|---|---|
| E1：确认 `v<base_version>` | 合理工程细化，与现有确认机制相容；提示应明确批次 kind、数量及逐项信息。未映射工作也应有数量／引用的可理解展示，不显示摘要 |
| E2：manual/channel DB 约束 | 合理的第二道防线，落实 Q6；约束不等于证明操作者身份。迁移尚未实现 |
| E3：只看上一个 sync/audit | 符合 architecture §6.3。准确选择为同项目、同 worktree、排除 manual 后 seq 最大的那一批；若 dirty/unknown/head=null，不跳过它去找更早 clean；M2 回退 none，不生成 session_start |
| E4：歧义前缀 resolved=false | 技术上可实现，符合“只标记”的推荐方向，但有对外选择，见 PRODUCT DECISION P1 |
| E5：只让 Claim 在提交前校验回执 | 合理范围控制。没有借此修改 propose；其原有顺序只是准确披露，不代表本轮已统一或豁免其他问题 |
| E6：tests 只计数、audit_scope 不回显 | 技术结构及输入索引可支持，但属于回执内容范围选择，见 PRODUCT DECISION P2 |

E3 的例子：较早 sync clean(A)，最新 audit dirty(B)，其后还有 manual clean(C)。新 sync 的 previous_batch 候选是 B，结果 none；不能选 A 或 C。这没有改变已验收的进度或 ready 规则。

## 七、回执、Schema 与独立验证

### Schema 与业务责任的边界

新 Schema 为严格根对象，context、compare_from、claims、unmapped_work 及 refs 的各级对象都禁止额外字段；不提供 summary、测试名、verified 等字段。

- kind 只允许 sync/audit/manual；compare_from 当且仅当 sync 出现。
- compare_from 的 none 当且仅当 commit=null；其他来源必须有完整 SHA。
- unknown 断言只允许 audit；不得与 tree_state=unknown 这个独立上下文状态混淆。
- 空 claims、空 unmapped_work、无引用、仅未映射工作均有表达；refs 始终给 files/commits 两数组和 tests 数量。
- `session_start` 在 Schema 中为未来会话保留，但 M2 构造器不得产生它。规范已将这一限制列为业务测试，未声称 Schema 单独禁止。

Schema 不能证明 `index=i`、条数/顺序与输入一致、ID 互异、AC 属于当前项目、refs 数量和值来自对应行、共享 seq 或输入修订与当前修订相等。§9.2 已明确分工，独立补测也确认重复 index/claim_id 可以通过 Schema；这是业务校验责任，不能把 Schema 通过当作这些语义通过。

三份合法示例中，sync/audit 的 target、修订、断言、引用数量及缺省 refs 与既有 claims 输入示例逐项对应；空示例明确 null head 和 none 起点。示例内虚构的 batch/claim ID、seq、上下文不是数据库往返证据。回执足以识别批次、逐项 mapped/unmapped 结果；E6 选择确定后应继续保持其隐私边界。

未映射工作落库时，criterion、revision、assertion 的空值／适用字段约定，以及从分配的 claim_id 到输入下标的映射，应在迁移与构造器中明确。不能依赖 SQL 无 ORDER BY 的返回顺序，也不能凭 ULID 排序推导输入顺序；这不要求新增产品行为或本 PR 提前实现迁移。

### 实际运行

平台：Windows/amd64，Go `1.27.1`，Git `2.56.0.windows.2`。固定 HEAD 经 `git archive` 导出到仓库外；导出包 SHA-256 为 `9fd2f61b7fd6ef05e8ea07cb33a27183826db37c12bf452ded3209d0bf360681`。使用缓存依赖，`GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`。

1. 未添加临时检查前运行：

   ```text
   go test -tags nowebui -count=1 -timeout 3m -v ./internal/contract
   ```

   PASS，包 6.507 秒；24 个 Schema 编译、43 个合法示例、190 个非法反例，以及其余契约测试通过。

2. 只在隔离副本增加审查用测试，重新验证每个新增反例的基础文档合法、补丁后是 invalid_input、并检查实际 Details：

   ```text
   go test -tags nowebui -count=1 -timeout 3m -v -run '^TestReview065' ./internal/contract
   ```

   修正辅助检查的筛选条件后 PASS，包 6.624 秒。初次使用字符串范围筛选 SCH 编号，误把 SCH-17～19 也纳入，导致“28 而非 25”计数失败；改为精确筛选新 Schema 后重新通过。这是本轮临时检查程序的问题，不是 PR 测试或 Schema 缺陷，没有把该失败记成有效产品检出。

新增 25 个反例的实质拒绝依据：

| SCH | 实际位置／约束 |
|---|---|
| 166 | 根缺少 compare_from |
| 167 | audit 上 forbidden compare_from |
| 168 | `/claims/0/assertion` 禁止 unknown |
| 169～170 | `/compare_from/commit` 的 null/string 条件 |
| 171 | compare_from 完整 SHA 格式 |
| 172～173 | Claim/root 的额外 summary 字段 |
| 174 | `/claims/0/refs/tests` 必须为整数而非名称数组 |
| 175～176 | 缺少 refs／文件 exists |
| 177 | resolved 必须为 boolean |
| 178 | commit 前缀必须为小写 hex |
| 179 | tests 数量不得为负 |
| 180～181 | index 必填、不得为负 |
| 182 | unmapped_work 必填 |
| 183 | 未映射项禁止额外 summary |
| 184～186 | tree_state 枚举、完整 head、context 必填 |
| 187～188 | kind 枚举及输出 schema 判别值 |
| 189～190 | AC target 格式、禁止额外 verified |

各反例不是因无效基础示例而“假通过”。SCH-187 同时触发 kind 枚举和 compare_from 分支，不影响其未知 kind 必须拒绝的目标。现有注册测试没有永久断言每条反例的精确原因，但本轮独立逐项核对了这些原因。

额外正例覆盖三种 kind 的空批次、仅 unmapped_work；还确认 future session_start、重复 index/ID 的结构可被 Schema 接受，以验证文档所述的业务责任边界。没有把这些结构可接受样例说成合法 M2 持久化结果。

本轮没有 Claim 命令可运行；未伪造真实 Claim、PTY、Git 前缀解析、迁移、回滚、跨进程提交或故障输出已通过的证据。计划中的测试必须在实施 PR 兑现。

## 八、隐私、方案 B 与验收边界

§5 在持久化前扫描所有输入字符串、工作区登记路径，错误不给原文；§10 不回显摘要、测试名、环境变量值，保存的引用也不含文件内容。Schema 的严格字段能防止额外 summary 字段，但不能代替密钥扫描、错误脱敏或从数据库构造回执的测试。

actor 应复用现有经过扫描的显示名逻辑，而不是自行把配置值直接拼入持久化。Claim 的 manual 预览、文本结果与错误继续经过现有 termtext；JSON 保留原业务值，不能把显示转义写回数据库。

私有临时区的修订准确沿用 WP2.3 §14.3：可信清理按清单完成；身份/owner/清单异常、消费者未确认结束或完成记录持久化失败时保留并报告。后续 Scratch 依据实际完整记录认领，不把前次 Sync 错误说成已成功；不凭 PID、年龄或父进程死亡判断结束。没有扩大正常 Open、其他账户、合作并发或数据库保证。

验收映射覆盖 CLM-01～05、INV-01/02/04/06、WT-03、CX-10/12/13 和相关不变量；M3 会话和 session_start、WP2.7 输出、WP2.5 执行证据分工清楚。局部 audit 不是非法输入，只有实际覆盖全部当前 active AC 的当前修订时才支持完整盘点；空批次不是盘点完成的替代证明。

CLM-01/CX-12 的实施夹具仍应按冻结规则选取 AC：例如自动化 AC 有已确认检查但无执行证据，才应期待 claimed_without_evidence；没有检查时既有规则是 no_check。不能为满足过于简写的映射改动推导引擎。这里只核查映射适用性，不重新审计已验收算法。

## 九、PRODUCT DECISION

### P1：E4 的歧义前缀及回执原因

位置：[wp2.4-claim.md:141](../wp2.4-claim.md#L141)、[§14 E4](../wp2.4-claim.md#L314)。

推荐批准现方案：语法合法但有歧义的前缀记 `resolved=false`，保持 V-C06 的“只标记”语义；不存在、非 commit、歧义可统一为 unknown_commit，不强制增加细分原因字段。读取失败／受禁则必须 exit 6，不可同样记 false。

负责人也可以选择拒绝歧义或要求原因区分，但这会影响对外行为、回执及测试，需要先决定再冻结。当前未将该选择视为已批准，也没有把待决定本身列作技术缺陷。

### P2：E6 的最小回执内容

位置：[wp2.4-claim.md:170](../wp2.4-claim.md#L170)、[§14 E6](../wp2.4-claim.md#L315)。

推荐批准：tests 只返回数量、audit_scope 不回显；调用方已有原输入，batch ID、逐项 index/ID/target 和引用结果可供对应，且减小内容暴露。若负责人希望回执独立携带 audit 范围，或逐项测试名/占位索引结果，应明确字段、顺序和隐私约束，再同步 Schema 与例子。审查不代为批准。

D1～D3、Q6、Q9 的既定批准不重开。B1 按版本优先修正属于落实已有契约，无须另做产品决定。

## 十、ACCEPTED RISK

本轮不新增接受风险。只引用既有 §71 的 Git 私有临时区方案 B 局部边界：不合作同用户进程干预该命名空间（包括误操作）在排除范围内，Unix 精确窗口及 Clean 可能 false-clean 的后果已获批准；不能扩大到其他账户、正常合作并发、消费者崩溃后的保护、数据库、来源权限或隐私。

M-15、M-16、M-14 及其他维护事项均不是本轮接受的风险。Q9 内容选择和 E4/E6 待决也没有被转成风险接受。

## 十一、CI、旧失败及证据适用性

### 新 HEAD

[run 39424865715](https://github.com/XKHoshizora/specmetry/actions/runs/39424865715)：`pull_request`，attempt 1，正式 `completed/success`，8/8。已逐一读取八个 Job 的 checkout 日志：

| Job | ID | 结果 |
|---|---|---|
| Web | 118634752282 | success |
| Go tests (windows-latest) | 118634752724 | success |
| Go lint | 118634752768 | success |
| Go tests (ubuntu-latest) | 118634752829 | success |
| Go tests (macos-latest) | 118634752832 | success |
| Build with dashboard (macos-latest) | 118634927296 | success |
| Build with dashboard (ubuntu-latest) | 118634927346 | success |
| Build with dashboard (windows-latest) | 118634927369 | success |

全部实际检出 `c340ca695709ee7204d31a0a928c62dc96c39d54`。GitHub commit 元数据确认父提交为固定 base 与 HEAD；合并树 `e479771e195c3a8d7a616361bf88a7bb442e00ce` 与 HEAD 相同。不是拿另一个提交或树的结果用于本次审查。

三平台日志都实际 PASS `TestAllSchemasCompile`、`TestValidExamplesPass`、`TestInvalidCasesFail`，分别列出 43/190；不是按设计跳过。Ubuntu 普通 Go tests 使用 `-race`；另外的 Linux root 跨账户步骤八项 PASS，不带 `-race`。该 root 步骤在 Windows/macOS 不适用；普通 Linux 步骤跳过对应 root 用例。真实旧/unsafe Git、显式 M-14 诊断及 Windows 不可读权限用例等仍有设计跳过，不把整仓 CI 解释为所有条件全执行。

这是提交内容的三平台契约/回归证据，不是未提交实验或未实现 Claim 能力的三平台证明。本机仅独立运行上述 Windows 契约检查，没有重复整仓或平台压力测试。

### 必须保留的旧失败

独立核对 [run 39307739451](https://github.com/XKHoshizora/specmetry/actions/runs/39307739451) attempt 1：HEAD 为 `91e7e5fd402d55dda904e5c6e3f44c89d567abd4`，正式 completed/failure，7 成功、Windows Go tests 失败。

Windows job `118249164263` 的日志确认 checkout `60f8820e873fd237c78136be7be1cc30609bdb7d`；该快照父提交为 base 与旧 HEAD，tree `6b0a376e55b3388ec8115071315479906eb55776`。日志实际包含：

- `panic: test timed out after 10m0s`，当时列出的 `TestSourceHumanOnlyCommandsRefuseAgentInTerminal` 已运行 1 秒；不能据此称该单项运行了十分钟或已证明死锁。
- `TestSeqIsUniqueAcrossConnections`：`store_test.go:546`，SQLITE_BUSY。
- `TestSeqIsUniqueAcrossProcesses`：`store_test.go:571`，辅助进程 exit 1、SQLITE_BUSY。
- 同一失败运行的 43/190 契约示例检查也实际通过。

新旧 HEAD 间只有六份文档变化，没有修复这些生产或测试路径。新 CI 成功不证明旧失败根因是资源竞争、发生阶段已定位、M-15/M-16 已解决。§77 与维护登记准确保留失败、未解决及未接受风险的状态。本轮没有重跑 CI、修改 timeout、添加重试或更改 SQLite 错误映射。

## 十二、收尾与下一步

| 判断 | 本轮结论 |
|---|---|
| 技术审查 | **NO-GO**：先修 B1 的规范与测试预期冲突 |
| 新回执结构 | 技术检查通过；不是 Claim 端到端验收，E4/E6 对外选择仍待决定 |
| PR #29 当前是否可合并 | **不建议合并**；CI 虽已通过，规范阻塞尚存 |
| 整体规范是否具备冻结条件 | **尚未具备**：B1 修正、E4/E6 决定及同步后再确认 |
| 实现是否已授权 | **没有**；本审查不授权 WP2.4、WP2.5 或其他工作包 |

最小接手范围：

1. 实施方修正 B1，按版本优先分开“合法并发退役”与“当前版本下引用已退役对象”的验收；不要修改冻结的版本契约。
2. 负责人明确 E4/E6 的选择；同步新契约、说明及正反例中实际受影响的内容。N1/N2 可一并收口，不扩大到其他 CLI。
3. 对修订增量定向核查、合并身份及 CI 核验、负责人冻结规范并明确授权后，才可开始实现。
4. 实现时兑现受控前缀解析器、固定 head 的引用检查、迁移约束、完整业务回滚/首次 worktree 的 seq、回执与数据库行的对应关系、输出故障及真实通道测试。上述是工作包交付要求，不要求规范 PR 提前提交 Claim 实现。

本轮仅新增本未提交报告。未修改任何其他原仓库文件、历史审查、生产代码、测试或 Schema；未提交、推送、合并、改基或重跑 CI。没有宣布规范冻结、WP2.4/WP2.3/M2 验收，也没有关闭维护事项或启动后续工作。
